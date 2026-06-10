package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	authenticationv1 "admin/api/gen/authentication/v1"
	identityv1 "admin/api/gen/identity/v1"
	"admin/internal/data/repo"
	paginationv1 "github.com/chnxq/x-crud/api/gen/pagination/v1"
	crudviewer "github.com/chnxq/x-crud/viewer"
	"github.com/chnxq/xkitmod/log"
	"github.com/chnxq/xkitpkg/app"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type generatedDataWithTenantRepo interface {
	TenantRepoProvider() repo.TenantRepo
}

type manualAuthFlowService struct {
	store *authFlowStore
	log   *log.Helper
}

type manualSocialAuthService struct {
	userRepo           repo.UserRepo
	userCredentialRepo repo.UserCredentialRepo
	tenantRepo         repo.TenantRepo
	credentialFinder   userCredentialFinder
	authFlowStore      *authFlowStore
	auth               *authConfig
	tokenStore         *tokenStore
	log                *log.Helper
}

func newManualSocialAuthService(appCtx *app.AppCtx, data GeneratedData) *manualSocialAuthService {
	service := &manualSocialAuthService{
		log: log.NewHelper(log.With(log.GetLogger(), "module", "social_auth/server")),
	}
	if provider, ok := data.(generatedDataWithUserRepo); ok {
		service.userRepo = provider.UserRepoProvider()
	}
	if provider, ok := data.(generatedDataWithUserCredentialRepo); ok {
		service.userCredentialRepo = provider.UserCredentialRepoProvider()
		if finder, ok := service.userCredentialRepo.(userCredentialFinder); ok {
			service.credentialFinder = finder
		}
	}
	if provider, ok := data.(generatedDataWithTenantRepo); ok {
		service.tenantRepo = provider.TenantRepoProvider()
	}
	if store, err := newAuthFlowStore(loadDataConfig(appCtx)); err == nil {
		service.authFlowStore = store
	} else {
		service.log.Errorf("chain=manual_social_auth.init init auth flow store failed: %s", err.Error())
	}
	if auth, err := loadAuthConfig(appCtx); err == nil {
		service.auth = auth
	} else {
		service.log.Errorf("chain=manual_social_auth.init load auth config failed: %s", err.Error())
	}
	if store, err := newTokenStore(loadDataConfig(appCtx)); err == nil {
		service.tokenStore = store
	} else {
		service.log.Errorf("chain=manual_social_auth.init init token store failed: %s", err.Error())
	}
	return service
}

func newManualAuthFlowServiceWithStore(appCtx *app.AppCtx) *manualAuthFlowService {
	service := &manualAuthFlowService{
		log: log.NewHelper(log.With(log.GetLogger(), "module", "auth_flow/server")),
	}
	if store, err := newAuthFlowStore(loadDataConfig(appCtx)); err == nil {
		service.store = store
	} else {
		service.log.Errorf("chain=manual_auth_flow.init init auth flow store failed: %s", err.Error())
	}
	return service
}

func (s *manualAuthFlowService) CreateAuthSession(_ context.Context, req *authenticationv1.CreateAuthSessionRequest) (*authenticationv1.AuthSession, error) {
	if s == nil || s.store == nil {
		return nil, authenticationv1.ErrorInternalServerError("auth flow service is unavailable")
	}
	if req == nil {
		return nil, authenticationv1.ErrorBadRequest("request is required")
	}
	sessionID := "as_" + newJWTID()
	sessionToken, err := generateRefreshToken()
	if err != nil {
		return nil, authenticationv1.ErrorInternalServerError("failed to create auth session")
	}
	expiresAt := time.Now().Add(authFlowSessionTTL)
	record := &authFlowSessionRecord{
		SessionID:    sessionID,
		SessionToken: sessionToken,
		Scene:        req.GetScene(),
		Status:       authenticationv1.AuthFlowStatus_AUTH_FLOW_PENDING,
		TenantID:     req.GetTenantId(),
		UserID:       req.GetUserId(),
		ProviderKey:  strings.TrimSpace(req.GetProviderKey()),
		Provider:     req.GetProvider(),
		ClientType:   req.GetClientType(),
		RedirectURI:  strings.TrimSpace(req.GetRedirectUri()),
		ExtraInfo:    strings.TrimSpace(req.GetExtraInfo()),
		ExpiresAt:    expiresAt,
	}
	if record.ProviderKey == "" {
		record.ProviderKey = providerKeyFromOAuthProvider(req.GetProvider())
	}
	record.QRCodeURL = buildAuthSessionQRCodeURL(sessionID, sessionToken, record.ProviderKey)
	record.DisplayHint = "等待扫码或授权完成"
	if err := s.store.SaveSession(record); err != nil {
		return nil, authenticationv1.ErrorInternalServerError("failed to persist auth session")
	}
	return authSessionDTO(record), nil
}

func (s *manualAuthFlowService) GetAuthSession(_ context.Context, req *authenticationv1.GetAuthSessionRequest) (*authenticationv1.AuthSession, error) {
	if s == nil || s.store == nil {
		return nil, authenticationv1.ErrorInternalServerError("auth flow service is unavailable")
	}
	record, err := s.store.GetSession(req.GetSessionId())
	if err != nil {
		return nil, authenticationv1.ErrorNotFound("auth session not found")
	}
	return authSessionDTO(record), nil
}

func (s *manualAuthFlowService) PollAuthSession(_ context.Context, req *authenticationv1.PollAuthSessionRequest) (*authenticationv1.AuthSession, error) {
	if s == nil || s.store == nil {
		return nil, authenticationv1.ErrorInternalServerError("auth flow service is unavailable")
	}
	record, err := s.store.GetSession(req.GetSessionId())
	if err != nil {
		return nil, authenticationv1.ErrorNotFound("auth session not found")
	}
	if token := strings.TrimSpace(req.GetSessionToken()); token != "" && token != record.SessionToken {
		return nil, authenticationv1.ErrorUnauthorized("invalid auth session token")
	}
	if time.Now().After(record.ExpiresAt) {
		record.Status = authenticationv1.AuthFlowStatus_AUTH_FLOW_EXPIRED
		_ = s.store.SaveSession(record)
	}
	return authSessionDTO(record), nil
}

func (s *manualAuthFlowService) ConfirmAuthSession(_ context.Context, req *authenticationv1.ConfirmAuthSessionRequest) (*authenticationv1.AuthSession, error) {
	if s == nil || s.store == nil {
		return nil, authenticationv1.ErrorInternalServerError("auth flow service is unavailable")
	}
	record, err := s.store.GetSession(req.GetSessionId())
	if err != nil {
		return nil, authenticationv1.ErrorNotFound("auth session not found")
	}
	if strings.TrimSpace(req.GetSessionToken()) != record.SessionToken {
		return nil, authenticationv1.ErrorUnauthorized("invalid auth session token")
	}
	if req.Approved != nil && !req.GetApproved() {
		record.Status = authenticationv1.AuthFlowStatus_AUTH_FLOW_CANCELED
		_ = s.store.SaveSession(record)
		return authSessionDTO(record), nil
	}
	now := time.Now()
	record.Status = authenticationv1.AuthFlowStatus_AUTH_FLOW_CONFIRMED
	record.ConfirmedAt = &now
	if req.UserId != nil {
		record.UserID = req.GetUserId()
	}
	if req.ExtraInfo != nil {
		record.ExtraInfo = req.GetExtraInfo()
	}
	if err := s.store.SaveSession(record); err != nil {
		return nil, authenticationv1.ErrorInternalServerError("failed to persist auth session")
	}
	return authSessionDTO(record), nil
}

func (s *manualAuthFlowService) CancelAuthSession(_ context.Context, req *authenticationv1.CancelAuthSessionRequest) (*emptypb.Empty, error) {
	if s == nil || s.store == nil {
		return nil, authenticationv1.ErrorInternalServerError("auth flow service is unavailable")
	}
	record, err := s.store.GetSession(req.GetSessionId())
	if err == nil {
		if token := strings.TrimSpace(req.GetSessionToken()); token != "" && token != record.SessionToken {
			return nil, authenticationv1.ErrorUnauthorized("invalid auth session token")
		}
	}
	_ = s.store.DeleteSession(req.GetSessionId())
	return &emptypb.Empty{}, nil
}

func (s *manualSocialAuthService) StartSocialLogin(ctx context.Context, req *authenticationv1.StartSocialLoginRequest) (*authenticationv1.StartSocialLoginResponse, error) {
	if s == nil || s.authFlowStore == nil {
		return nil, authenticationv1.ErrorInternalServerError("social auth service is unavailable")
	}
	authFlow := &manualAuthFlowService{store: s.authFlowStore, log: s.log}
	session, err := authFlow.CreateAuthSession(ctx, &authenticationv1.CreateAuthSessionRequest{
		Scene:       authenticationv1.AuthFlowScene_SOCIAL_LOGIN,
		TenantId:    req.TenantId,
		ProviderKey: req.ProviderKey,
		Provider:    optionalOAuthProvider(req.GetProvider()),
		ClientType:  req.ClientType,
		RedirectUri: req.RedirectUri,
	})
	if err != nil {
		return nil, err
	}
	return &authenticationv1.StartSocialLoginResponse{
		AuthorizationUrl: stringPtr(session.GetQrCodeUrl()),
		QrCodeUrl:        stringPtr(session.GetQrCodeUrl()),
		SessionId:        stringPtr(session.GetSessionId()),
		State:            stringPtr(session.GetSessionToken()),
		ExpiresAt:        session.ExpiresAt,
		DisplayHint:      stringPtr("请在手机端完成扫码或授权"),
	}, nil
}

func (s *manualSocialAuthService) CompleteSocialLogin(ctx context.Context, req *authenticationv1.CompleteSocialLoginRequest) (*authenticationv1.CompleteSocialLoginResponse, error) {
	if s == nil || s.authFlowStore == nil {
		return nil, authenticationv1.ErrorInternalServerError("social auth service is unavailable")
	}
	providerKey := strings.TrimSpace(req.GetProviderKey())
	if providerKey == "" {
		providerKey = providerKeyFromOAuthProvider(req.GetProvider())
	}
	sessionID := strings.TrimSpace(req.GetSessionId())
	if sessionID == "" {
		sessionID = "as_" + newJWTID()
	}
	accountID := strings.TrimSpace(req.GetCode())
	if accountID == "" {
		return nil, authenticationv1.ErrorBadRequest("social auth code is required")
	}

	record := &authFlowSessionRecord{
		SessionID:         sessionID,
		SessionToken:      strings.TrimSpace(req.GetState()),
		Scene:             authenticationv1.AuthFlowScene_SOCIAL_LOGIN,
		Status:            authenticationv1.AuthFlowStatus_AUTH_FLOW_PENDING,
		ProviderKey:       providerKey,
		Provider:          req.GetProvider(),
		ProviderAccountID: accountID,
		ClientType:        req.GetClientType(),
		RedirectURI:       strings.TrimSpace(req.GetRedirectUri()),
		ExpiresAt:         time.Now().Add(authFlowSessionTTL),
		Profile: &authFlowProviderProfile{
			ProviderKey:       providerKey,
			Provider:          req.GetProvider(),
			ProviderAccountID: accountID,
			Nickname:          defaultSocialNickname(providerKey, accountID),
			RawProfileJSON:    socialProfileJSON(providerKey, accountID),
		},
	}
	if existing, err := s.authFlowStore.GetSession(sessionID); err == nil && existing != nil {
		record.SessionToken = existing.SessionToken
		record.TenantID = existing.TenantID
		record.ExpiresAt = existing.ExpiresAt
	}
	if record.SessionToken == "" {
		token, err := generateRefreshToken()
		if err != nil {
			return nil, authenticationv1.ErrorInternalServerError("failed to create auth session token")
		}
		record.SessionToken = token
	}
	record.QRCodeURL = buildAuthSessionQRCodeURL(record.SessionID, record.SessionToken, record.ProviderKey)

	userDTO, err := s.findUserBySocialIdentity(ctx, record.TenantID, providerKey, accountID)
	if err != nil {
		s.log.Errorf("%s complete social login find bound user failed provider=%s account=%s: %s", authRequestLogPrefix(ctx, "manual_social_auth.complete"), providerKey, accountID, err.Error())
		return nil, err
	}
	if userDTO != nil {
		record.Status = authenticationv1.AuthFlowStatus_AUTH_FLOW_CONFIRMED
		now := time.Now()
		record.ConfirmedAt = &now
		record.UserID = userDTO.GetId()
		if err := s.authFlowStore.SaveSession(record); err != nil {
			return nil, authenticationv1.ErrorInternalServerError("failed to persist auth session")
		}
		loginReq := &authenticationv1.LoginRequest{
			ClientType: req.ClientType,
			DeviceId:   req.DeviceId,
		}
		resp, err := issueLoginResponse(userDTO, loginReq, s.auth, s.tokenStore)
		if err != nil {
			return nil, err
		}
		return &authenticationv1.CompleteSocialLoginResponse{
			Status: authenticationv1.SocialAuthStatus_SOCIAL_AUTH_BOUND,
			Login:  resp,
		}, nil
	}

	record.Status = authenticationv1.AuthFlowStatus_AUTH_FLOW_UNBOUND
	bindToken, err := generateRefreshToken()
	if err != nil {
		return nil, authenticationv1.ErrorInternalServerError("failed to create bind token")
	}
	record.BindToken = bindToken
	if err := s.authFlowStore.SaveSession(record); err != nil {
		return nil, authenticationv1.ErrorInternalServerError("failed to persist auth session")
	}
	if err := s.authFlowStore.SaveBindRecord(bindToken, &authFlowBindRecord{
		SessionID:   record.SessionID,
		TenantID:    record.TenantID,
		ProviderKey: record.ProviderKey,
		Provider:    record.Provider,
		Profile:     record.Profile,
		ExpiresAt:   record.ExpiresAt,
	}); err != nil {
		return nil, authenticationv1.ErrorInternalServerError("failed to persist bind record")
	}

	return &authenticationv1.CompleteSocialLoginResponse{
		Status: authenticationv1.SocialAuthStatus_SOCIAL_AUTH_UNBOUND,
		Pending: &authenticationv1.SocialAuthPendingBinding{
			SessionId:   record.SessionID,
			BindToken:   bindToken,
			Profile:     socialProfileDTO(record.Profile),
			ExpiresAt:   timestamppb.New(record.ExpiresAt),
			DisplayHint: stringPtr("尚未绑定本地账号，请选择绑定已有账号或创建新账号"),
		},
	}, nil
}

func (s *manualSocialAuthService) ExchangeMiniAppCode(context.Context, *authenticationv1.ExchangeMiniAppCodeRequest) (*authenticationv1.ExchangeMiniAppCodeResponse, error) {
	return nil, authenticationv1.ErrorNotImplemented("miniapp social auth is not implemented yet")
}

func (s *manualSocialAuthService) ConfirmBindOrRegister(ctx context.Context, req *authenticationv1.ConfirmBindOrRegisterRequest) (*authenticationv1.ConfirmBindOrRegisterResponse, error) {
	if s == nil || s.authFlowStore == nil || s.userCredentialRepo == nil || s.userRepo == nil {
		return nil, authenticationv1.ErrorInternalServerError("social auth service is unavailable")
	}
	bindRecord, err := s.authFlowStore.GetBindRecord(req.GetBindToken())
	if err != nil {
		return nil, authenticationv1.ErrorUnauthorized("bind token is invalid or expired")
	}

	var userDTO *identityv1.User
	switch req.GetOperation() {
	case authenticationv1.ConfirmBindOrRegisterOperation_BIND_EXISTING:
		existing := req.GetExisting()
		if existing == nil {
			return nil, authenticationv1.ErrorBadRequest("existing account payload is required")
		}
		if err := verifyCaptcha(existing.GetCaptchaId(), existing.GetCaptchaCode()); err != nil {
			return nil, err
		}
		identifier := existingAccountIdentifier(existing)
		if identifier == "" || strings.TrimSpace(existing.GetPassword()) == "" {
			return nil, authenticationv1.ErrorBadRequest("existing account credentials are required")
		}
		credential, err := s.credentialFinder.FindPasswordCredentialByIdentifier(ensureDefaultViewerContext(ctx), identifier)
		if err != nil {
			return nil, err
		}
		if credential == nil || credential.User == nil || credential.Credential == nil || credential.Credential.Credential == nil {
			return nil, authenticationv1.ErrorUserNotFound("user not found")
		}
		matched, _, err := repo.VerifyPasswordCredential(existing.GetPassword(), *credential.Credential.Credential)
		if err != nil {
			return nil, err
		}
		if !matched {
			return nil, authenticationv1.ErrorIncorrectPassword("incorrect password")
		}
		userDTO, err = s.userRepo.Get(ensureDefaultViewerContext(ctx), &identityv1.GetUserRequest{
			QueryBy: &identityv1.GetUserRequest_Id{Id: credential.User.ID},
		})
		if err != nil || userDTO == nil {
			return nil, authenticationv1.ErrorUserNotFound("user not found")
		}
	case authenticationv1.ConfirmBindOrRegisterOperation_REGISTER_NEW:
		registration := req.GetRegistration()
		if registration == nil {
			return nil, authenticationv1.ErrorBadRequest("registration payload is required")
		}
		registerResp, err := s.registerUser(ctx, registration, bindRecord.TenantID)
		if err != nil {
			return nil, err
		}
		userDTO, err = s.userRepo.Get(ensureDefaultViewerContext(ctx), &identityv1.GetUserRequest{
			QueryBy: &identityv1.GetUserRequest_Id{Id: registerResp.GetUserId()},
		})
		if err != nil || userDTO == nil {
			return nil, authenticationv1.ErrorUserNotFound("user not found")
		}
	default:
		return nil, authenticationv1.ErrorBadRequest("invalid bind operation")
	}

	if err := s.bindSocialCredential(ctx, userDTO, bindRecord); err != nil {
		return nil, err
	}
	loginResp, err := issueLoginResponse(userDTO, &authenticationv1.LoginRequest{
		ClientType: req.ClientType,
		DeviceId:   req.DeviceId,
	}, s.auth, s.tokenStore)
	if err != nil {
		return nil, err
	}
	_ = s.authFlowStore.DeleteBindRecord(req.GetBindToken())
	return &authenticationv1.ConfirmBindOrRegisterResponse{
		Status: authenticationv1.SocialAuthStatus_SOCIAL_AUTH_BOUND,
		UserId: uint32Ptr(userDTO.GetId()),
		Login:  loginResp,
	}, nil
}

func (s *manualSocialAuthService) findUserBySocialIdentity(ctx context.Context, tenantID uint32, providerKey, accountID string) (*identityv1.User, error) {
	if s.userCredentialRepo == nil || s.userRepo == nil {
		return nil, nil
	}
	listResp, err := s.userCredentialRepo.List(ensureDefaultViewerContext(ctx), buildSocialCredentialPagingRequest(tenantID, providerKey, accountID))
	if err != nil {
		return nil, err
	}
	if listResp == nil || len(listResp.GetItems()) == 0 {
		return nil, nil
	}
	item := listResp.GetItems()[0]
	if item == nil || item.GetUserId() == 0 {
		return nil, nil
	}
	return s.userRepo.Get(ensureDefaultViewerContext(ctx), &identityv1.GetUserRequest{
		QueryBy: &identityv1.GetUserRequest_Id{Id: item.GetUserId()},
	})
}

func buildSocialCredentialPagingRequest(tenantID uint32, providerKey, accountID string) *paginationv1.PagingRequest {
	return &paginationv1.PagingRequest{
		Limit: uint32Ptr(1),
		FilteringType: &paginationv1.PagingRequest_FilterExpr{
			FilterExpr: &paginationv1.FilterExpr{
				Type: paginationv1.ExprType_AND,
				Conditions: []*paginationv1.FilterCondition{
					{
						Field:      "provider",
						Op:         paginationv1.Operator_EQ,
						ValueOneof: &paginationv1.FilterCondition_Value{Value: providerKey},
					},
					{
						Field:      "provider_account_id",
						Op:         paginationv1.Operator_EQ,
						ValueOneof: &paginationv1.FilterCondition_Value{Value: accountID},
					},
					{
						Field:      "tenant_id",
						Op:         paginationv1.Operator_EQ,
						ValueOneof: &paginationv1.FilterCondition_Value{Value: fmt.Sprintf("%d", tenantID)},
					},
				},
			},
		},
	}
}

func (s *manualSocialAuthService) bindSocialCredential(ctx context.Context, userDTO *identityv1.User, bindRecord *authFlowBindRecord) error {
	if userDTO == nil || bindRecord == nil || bindRecord.Profile == nil {
		return authenticationv1.ErrorBadRequest("invalid bind record")
	}
	identityType := authenticationv1.UserCredential_SOCIAL_OAUTH
	credentialType := authenticationv1.UserCredential_OAUTH_TOKEN
	status := authenticationv1.UserCredential_ENABLED
	req := &authenticationv1.CreateUserCredentialRequest{
		Data: &authenticationv1.UserCredential{
			UserId:            uint32Ptr(userDTO.GetId()),
			TenantId:          uint32Ptr(firstNonZero(bindRecord.TenantID, userDTO.GetTenantId())),
			IdentityType:      &identityType,
			CredentialType:    &credentialType,
			Identifier:        stringPtr(bindRecord.Profile.ProviderAccountID),
			Credential:        stringPtr(bindRecord.Profile.RawProfileJSON),
			Status:            &status,
			Provider:          stringPtr(bindRecord.ProviderKey),
			ProviderAccountId: stringPtr(bindRecord.Profile.ProviderAccountID),
			ExtraInfo:         stringPtr(bindRecord.Profile.RawProfileJSON),
		},
	}
	_, err := s.userCredentialRepo.Create(ensureDefaultViewerContext(ctx), req)
	return err
}

func (s *manualSocialAuthService) registerUser(ctx context.Context, req *authenticationv1.RegisterUserRequest, defaultTenantID uint32) (*authenticationv1.RegisterUserResponse, error) {
	if req == nil {
		return nil, authenticationv1.ErrorBadRequest("request is required")
	}
	switch payload := req.GetRegisterBy().(type) {
	case *authenticationv1.RegisterUserRequest_ByUsername:
		if err := verifyCaptcha(payload.ByUsername.GetCaptchaId(), payload.ByUsername.GetCaptchaCode()); err != nil {
			return nil, err
		}
		username := strings.TrimSpace(payload.ByUsername.GetUsername())
		password := strings.TrimSpace(payload.ByUsername.GetPassword())
		if username == "" || password == "" {
			return nil, authenticationv1.ErrorBadRequest("username and password are required")
		}
		if existsResp, err := s.userRepo.UserExists(ensureDefaultViewerContext(ctx), &identityv1.UserExistsRequest{
			QueryBy: &identityv1.UserExistsRequest_Username{Username: username},
		}); err == nil && existsResp != nil && existsResp.GetExist() {
			return nil, authenticationv1.ErrorConflict("username already exists")
		}
		tenantID, err := s.resolveRegisterTenantID(ctx, req.GetTenantCode(), defaultTenantID)
		if err != nil {
			return nil, err
		}
		status := identityv1.User_NORMAL
		createReq := &identityv1.CreateUserRequest{
			Data: &identityv1.User{
				Username: &username,
				Email:    payload.ByUsername.Email,
				Mobile:   payload.ByUsername.Mobile,
				TenantId: tenantID,
				Status:   &status,
			},
			Password: &password,
		}
		if _, err := s.userRepo.Create(ensureDefaultViewerContext(ctx), createReq); err != nil {
			return nil, err
		}
		userDTO, err := s.userRepo.Get(ensureDefaultViewerContext(ctx), &identityv1.GetUserRequest{
			QueryBy: &identityv1.GetUserRequest_Username{Username: username},
		})
		if err != nil || userDTO == nil {
			return nil, authenticationv1.ErrorInternalServerError("failed to load registered user")
		}
		loginResp, err := issueLoginResponse(userDTO, &authenticationv1.LoginRequest{
			Identifier: &authenticationv1.LoginRequest_Username{Username: username},
			ClientType: req.ClientType,
			DeviceId:   req.DeviceId,
		}, s.auth, s.tokenStore)
		if err != nil {
			return nil, err
		}
		return &authenticationv1.RegisterUserResponse{
			UserId: userDTO.GetId(),
			Login:  loginResp,
		}, nil
	default:
		return nil, authenticationv1.ErrorNotImplemented("only username registration is implemented")
	}
}

func (s *manualSocialAuthService) resolveRegisterTenantID(ctx context.Context, tenantCode string, defaultTenantID uint32) (*uint32, error) {
	tenantCode = strings.TrimSpace(tenantCode)
	if tenantCode != "" && s.tenantRepo != nil {
		tenantDTO, err := s.tenantRepo.Get(ensureDefaultViewerContext(ctx), &identityv1.GetTenantRequest{
			QueryBy: &identityv1.GetTenantRequest_Code{Code: tenantCode},
		})
		if err != nil {
			return nil, err
		}
		if tenantDTO != nil {
			return uint32Ptr(tenantDTO.GetId()), nil
		}
	}
	if defaultTenantID > 0 {
		return uint32Ptr(defaultTenantID), nil
	}
	if viewer, ok := crudviewer.FromContext(ctx); ok && viewer != nil && viewer.IsTenantContext() {
		return uint32Ptr(uint32(viewer.TenantID())), nil
	}
	return nil, nil
}

func buildAuthSessionQRCodeURL(sessionID, sessionToken, providerKey string) string {
	return fmt.Sprintf("xadmin://social-auth?sessionId=%s&token=%s&provider=%s", sessionID, sessionToken, providerKey)
}

func authSessionDTO(record *authFlowSessionRecord) *authenticationv1.AuthSession {
	if record == nil {
		return nil
	}
	dto := &authenticationv1.AuthSession{
		SessionId:         record.SessionID,
		SessionToken:      stringPtr(record.SessionToken),
		Scene:             record.Scene,
		Status:            record.Status,
		ProviderKey:       optionalString(record.ProviderKey),
		Provider:          optionalOAuthProvider(record.Provider),
		ProviderAccountId: optionalString(record.ProviderAccountID),
		ClientType:        optionalClientType(record.ClientType),
		RedirectUri:       optionalString(record.RedirectURI),
		ExtraInfo:         optionalString(record.ExtraInfo),
		ExpiresAt:         timestamppb.New(record.ExpiresAt),
		QrCodeUrl:         optionalString(record.QRCodeURL),
		DisplayHint:       optionalString(record.DisplayHint),
	}
	if record.TenantID > 0 {
		dto.TenantId = uint32Ptr(record.TenantID)
	}
	if record.UserID > 0 {
		dto.UserId = uint32Ptr(record.UserID)
	}
	if record.ConfirmedAt != nil {
		dto.ConfirmedAt = timestamppb.New(*record.ConfirmedAt)
	}
	return dto
}

func socialProfileDTO(profile *authFlowProviderProfile) *authenticationv1.SocialProviderProfile {
	if profile == nil {
		return nil
	}
	return &authenticationv1.SocialProviderProfile{
		ProviderKey:       optionalString(profile.ProviderKey),
		Provider:          optionalOAuthProvider(profile.Provider),
		ProviderAccountId: optionalString(profile.ProviderAccountID),
		Nickname:          optionalString(profile.Nickname),
		Avatar:            optionalString(profile.Avatar),
		Email:             optionalString(profile.Email),
		Mobile:            optionalString(profile.Mobile),
		RawProfileJson:    optionalString(profile.RawProfileJSON),
	}
}

func providerKeyFromOAuthProvider(provider authenticationv1.OAuthProvider) string {
	switch provider {
	case authenticationv1.OAuthProvider_GITHUB:
		return "github"
	case authenticationv1.OAuthProvider_DINGTALK:
		return "dingtalk"
	case authenticationv1.OAuthProvider_WECHAT:
		return "wechat"
	case authenticationv1.OAuthProvider_ALIPAY:
		return "alipay"
	default:
		return strings.ToLower(provider.String())
	}
}

func defaultSocialNickname(providerKey, accountID string) string {
	if accountID == "" {
		return providerKey + "-user"
	}
	return providerKey + "-" + accountID
}

func socialProfileJSON(providerKey, accountID string) string {
	payload, _ := json.Marshal(map[string]string{
		"provider":          providerKey,
		"providerAccountId": accountID,
		"nickname":          defaultSocialNickname(providerKey, accountID),
	})
	return string(payload)
}

func existingAccountIdentifier(existing *authenticationv1.ExistingAccountBinding) string {
	if existing == nil {
		return ""
	}
	switch identifier := existing.GetIdentifier().(type) {
	case *authenticationv1.ExistingAccountBinding_Username:
		return strings.TrimSpace(identifier.Username)
	case *authenticationv1.ExistingAccountBinding_Email:
		return strings.TrimSpace(identifier.Email)
	case *authenticationv1.ExistingAccountBinding_Mobile:
		return strings.TrimSpace(identifier.Mobile)
	default:
		return ""
	}
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func stringPtr(value string) *string {
	return &value
}

func uint32Ptr(value uint32) *uint32 {
	return &value
}

func optionalOAuthProvider(value authenticationv1.OAuthProvider) *authenticationv1.OAuthProvider {
	return &value
}

func optionalClientType(value authenticationv1.ClientType) *authenticationv1.ClientType {
	return &value
}

func firstNonZero(values ...uint32) uint32 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}
