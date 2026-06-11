package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
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
	roleRepo           repo.RoleRepo
	orgUnitRepo        repo.OrgUnitRepo
	positionRepo       repo.PositionRepo
	credentialFinder   userCredentialFinder
	authFlowStore      *authFlowStore
	auth               *authConfig
	social             *socialAuthConfig
	tokenStore         *tokenStore
	registration       *registrationDefaultsResolver
	log                *log.Helper
}

type manualOAuthService struct {
	userCredentialRepo repo.UserCredentialRepo
	authFlowStore      *authFlowStore
	social             *socialAuthConfig
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
	if provider, ok := data.(generatedDataWithRoleRepo); ok {
		service.roleRepo = provider.RoleRepoProvider()
	}
	if provider, ok := data.(generatedDataWithOrgUnitRepo); ok {
		service.orgUnitRepo = provider.OrgUnitRepoProvider()
	}
	if provider, ok := data.(generatedDataWithPositionRepo); ok {
		service.positionRepo = provider.PositionRepoProvider()
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
	if social, err := loadSocialAuthConfig(appCtx); err == nil {
		service.social = social
	} else {
		service.log.Errorf("chain=manual_social_auth.init load social auth config failed: %s", err.Error())
	}
	if store, err := newTokenStore(loadDataConfig(appCtx)); err == nil {
		service.tokenStore = store
	} else {
		service.log.Errorf("chain=manual_social_auth.init init token store failed: %s", err.Error())
	}
	service.registration = newRegistrationDefaultsResolver(appCtx, data)
	return service
}

func newManualOAuthService(appCtx *app.AppCtx, data GeneratedData) *manualOAuthService {
	service := &manualOAuthService{
		log: log.NewHelper(log.With(log.GetLogger(), "module", "oauth/server")),
	}
	if provider, ok := data.(generatedDataWithUserCredentialRepo); ok {
		service.userCredentialRepo = provider.UserCredentialRepoProvider()
	}
	if store, err := newAuthFlowStore(loadDataConfig(appCtx)); err == nil {
		service.authFlowStore = store
	} else {
		service.log.Errorf("chain=manual_oauth.init init auth flow store failed: %s", err.Error())
	}
	if social, err := loadSocialAuthConfig(appCtx); err == nil {
		service.social = social
	} else {
		service.log.Errorf("chain=manual_oauth.init load social auth config failed: %s", err.Error())
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
	providerKey := normalizeOAuthProviderKey(strings.TrimSpace(req.GetProviderKey()), req.GetProvider())
	authFlow := &manualAuthFlowService{store: s.authFlowStore, log: s.log}
	session, err := authFlow.CreateAuthSession(ctx, &authenticationv1.CreateAuthSessionRequest{
		Scene:       authenticationv1.AuthFlowScene_SOCIAL_LOGIN,
		TenantId:    req.TenantId,
		ProviderKey: stringPtr(providerKey),
		Provider:    optionalOAuthProvider(req.GetProvider()),
		ClientType:  req.ClientType,
		RedirectUri: req.RedirectUri,
	})
	if err != nil {
		return nil, err
	}
	authorizationURL := session.GetQrCodeUrl()
	if providerKey == "github" || providerKey == "dingtalk_web" || providerKey == "wechat_web" {
		authorizationURL, err = s.buildAuthorizationURL(req, session)
		if err != nil {
			return nil, err
		}
	}
	return &authenticationv1.StartSocialLoginResponse{
		AuthorizationUrl: stringPtr(authorizationURL),
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
	providerKey := normalizeOAuthProviderKey(strings.TrimSpace(req.GetProviderKey()), req.GetProvider())
	sessionID := strings.TrimSpace(req.GetSessionId())
	if sessionID == "" {
		if existing, err := s.authFlowStore.FindSessionByToken(strings.TrimSpace(req.GetState())); err == nil && existing != nil {
			sessionID = existing.SessionID
		} else {
			sessionID = "as_" + newJWTID()
		}
	}
	code := strings.TrimSpace(req.GetCode())
	if code == "" {
		return nil, authenticationv1.ErrorBadRequest("social auth code is required")
	}
	sessionRecord, err := s.resolveAuthSessionForComplete(req, sessionID)
	if err != nil {
		return nil, err
	}
	profile, err := s.resolveSocialProfile(ctx, req, providerKey, code, sessionRecord)
	if err != nil {
		return nil, err
	}
	if profile == nil || strings.TrimSpace(profile.ProviderAccountID) == "" {
		return nil, authenticationv1.ErrorUnauthorized("failed to resolve social identity")
	}

	record := &authFlowSessionRecord{
		SessionID:         sessionID,
		SessionToken:      strings.TrimSpace(req.GetState()),
		Scene:             authenticationv1.AuthFlowScene_SOCIAL_LOGIN,
		Status:            authenticationv1.AuthFlowStatus_AUTH_FLOW_PENDING,
		ProviderKey:       providerKey,
		Provider:          req.GetProvider(),
		ProviderAccountID: profile.ProviderAccountID,
		ClientType:        req.GetClientType(),
		RedirectURI:       strings.TrimSpace(req.GetRedirectUri()),
		ExpiresAt:         time.Now().Add(authFlowSessionTTL),
		Profile:           profile,
	}
	if sessionRecord != nil {
		existing := sessionRecord
		record.SessionToken = existing.SessionToken
		record.TenantID = existing.TenantID
		record.ExpiresAt = existing.ExpiresAt
		if strings.TrimSpace(record.RedirectURI) == "" {
			record.RedirectURI = existing.RedirectURI
		}
	}
	if record.SessionToken == "" {
		token, err := generateRefreshToken()
		if err != nil {
			return nil, authenticationv1.ErrorInternalServerError("failed to create auth session token")
		}
		record.SessionToken = token
	}
	record.QRCodeURL = buildAuthSessionQRCodeURL(record.SessionID, record.SessionToken, record.ProviderKey)

	userDTO, err := s.findUserBySocialIdentity(ctx, record.TenantID, providerKey, profile.ProviderAccountID)
	if err != nil {
		s.log.Errorf("%s complete social login find bound user failed provider=%s account=%s: %s", authRequestLogPrefix(ctx, "manual_social_auth.complete"), providerKey, profile.ProviderAccountID, err.Error())
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

func (s *manualOAuthService) ListLinkedAccounts(ctx context.Context, _ *authenticationv1.ListLinkedAccountsRequest) (*authenticationv1.ListLinkedAccountsResponse, error) {
	if s == nil || s.userCredentialRepo == nil {
		return nil, authenticationv1.ErrorInternalServerError("oauth service is unavailable")
	}
	viewer, ok := crudviewer.FromContext(ctx)
	if !ok || viewer == nil || viewer.UserID() == 0 {
		return nil, authenticationv1.ErrorUnauthorized("user is not authenticated")
	}
	resp, err := s.userCredentialRepo.List(ctx, &paginationv1.PagingRequest{
		NoPaging: boolPtr(true),
		FilteringType: &paginationv1.PagingRequest_FilterExpr{
			FilterExpr: &paginationv1.FilterExpr{
				Type: paginationv1.ExprType_AND,
				Conditions: []*paginationv1.FilterCondition{
					{
						Field:      "user_id",
						Op:         paginationv1.Operator_EQ,
						ValueOneof: &paginationv1.FilterCondition_Value{Value: fmt.Sprintf("%d", viewer.UserID())},
					},
					{
						Field:      "identity_type",
						Op:         paginationv1.Operator_EQ,
						ValueOneof: &paginationv1.FilterCondition_Value{Value: authenticationv1.UserCredential_SOCIAL_OAUTH.String()},
					},
					{
						Field:      "status",
						Op:         paginationv1.Operator_NEQ,
						ValueOneof: &paginationv1.FilterCondition_Value{Value: authenticationv1.UserCredential_REMOVED.String()},
					},
				},
			},
		},
	})
	if err != nil {
		s.log.Errorf("%s list linked oauth accounts failed: %s", authRequestLogPrefix(ctx, "manual_oauth.list"), err.Error())
		return nil, err
	}
	result := &authenticationv1.ListLinkedAccountsResponse{
		Items: make([]*authenticationv1.UserCredential, 0, len(resp.GetItems())),
		Total: resp.GetTotal(),
	}
	for _, item := range resp.GetItems() {
		if item == nil {
			continue
		}
		result.Items = append(result.Items, item)
	}
	return result, nil
}

func (s *manualOAuthService) ListProviders(context.Context, *authenticationv1.ListProvidersRequest) (*authenticationv1.ListProvidersResponse, error) {
	items := make([]*authenticationv1.ProviderMetadata, 0, 6)
	if s != nil && s.social != nil {
		items = append(items,
			providerMetadataFromConfig(authenticationv1.OAuthProvider_GITHUB, "github", "GitHub", s.social.GitHub),
			providerMetadataFromConfig(authenticationv1.OAuthProvider_DINGTALK, "dingtalk_web", "钉钉", s.social.DingTalkWeb),
			providerMetadataFromConfig(authenticationv1.OAuthProvider_WECHAT, "wechat_web", "微信网页", s.social.WeChatWeb),
			providerMetadataPlaceholder(authenticationv1.OAuthProvider_WECHAT, "wechat_miniapp", "微信小程序"),
			providerMetadataPlaceholder(authenticationv1.OAuthProvider_ALIPAY, "alipay", "支付宝"),
			providerMetadataPlaceholder(authenticationv1.OAuthProvider_DOUYIN, "douyin", "抖音"),
		)
		return &authenticationv1.ListProvidersResponse{Items: items}, nil
	}
	if s != nil && s.social != nil {
		items = append(items, providerMetadataFromConfig(authenticationv1.OAuthProvider_GITHUB, "github", "GitHub", s.social.GitHub))
		items = append(items, providerMetadataFromConfig(authenticationv1.OAuthProvider_WECHAT, "wechat_web", "微信网页", s.social.WeChatWeb))
	}
	items = append(items,
		providerMetadataPlaceholder(authenticationv1.OAuthProvider_DINGTALK, "dingtalk", "钉钉"),
		providerMetadataPlaceholder(authenticationv1.OAuthProvider_WECHAT, "wechat_web", "微信网页"),
		providerMetadataPlaceholder(authenticationv1.OAuthProvider_WECHAT, "wechat_miniapp", "微信小程序"),
		providerMetadataPlaceholder(authenticationv1.OAuthProvider_ALIPAY, "alipay", "支付宝"),
		providerMetadataPlaceholder(authenticationv1.OAuthProvider_DOUYIN, "douyin", "抖音"),
	)
	return &authenticationv1.ListProvidersResponse{Items: items}, nil
}

func (s *manualOAuthService) StartLinkOAuth(ctx context.Context, req *authenticationv1.StartLinkOAuthRequest) (*authenticationv1.StartLinkOAuthResponse, error) {
	if s == nil || s.authFlowStore == nil {
		return nil, authenticationv1.ErrorInternalServerError("oauth service is unavailable")
	}
	viewer, ok := crudviewer.FromContext(ctx)
	if !ok || viewer == nil || viewer.UserID() == 0 {
		return nil, authenticationv1.ErrorUnauthorized("user is not authenticated")
	}
	providerKey := strings.TrimSpace(req.GetProviderCustom())
	providerKey = normalizeOAuthProviderKey(providerKey, req.GetProvider())
	sessionID := "as_" + newJWTID()
	sessionToken, err := generateRefreshToken()
	if err != nil {
		return nil, authenticationv1.ErrorInternalServerError("failed to create oauth session")
	}
	record := &authFlowSessionRecord{
		SessionID:    sessionID,
		SessionToken: sessionToken,
		Scene:        authenticationv1.AuthFlowScene_SOCIAL_BIND,
		Status:       authenticationv1.AuthFlowStatus_AUTH_FLOW_PENDING,
		TenantID:     uint32(viewer.TenantID()),
		UserID:       uint32(viewer.UserID()),
		ProviderKey:  providerKey,
		Provider:     req.GetProvider(),
		ClientType:   authenticationv1.ClientType_admin,
		RedirectURI:  strings.TrimSpace(req.GetRedirectUri()),
		ExpiresAt:    time.Now().Add(authFlowSessionTTL),
		DisplayHint:  "请完成第三方账号授权以绑定当前登录用户",
	}
	if record.RedirectURI == "" && s.social != nil {
		switch providerKey {
		case "github":
			record.RedirectURI = s.social.GitHub.RedirectURI
		case "dingtalk_web":
			record.RedirectURI = s.social.DingTalkWeb.RedirectURI
		case "wechat_web":
			record.RedirectURI = s.social.WeChatWeb.RedirectURI
		}
	}
	if err := s.authFlowStore.SaveSession(record); err != nil {
		return nil, authenticationv1.ErrorInternalServerError("failed to persist oauth session")
	}
	startReq := &authenticationv1.StartSocialLoginRequest{
		Provider:    req.GetProvider(),
		ProviderKey: stringPtr(providerKey),
		RedirectUri: stringPtr(record.RedirectURI),
	}
	session := authSessionDTO(record)
	authorizationURL, err := (&manualSocialAuthService{social: s.social}).buildAuthorizationURL(startReq, session)
	if err != nil {
		return nil, err
	}
	return &authenticationv1.StartLinkOAuthResponse{
		AuthorizationUrl: stringPtr(authorizationURL),
		OperationId:      stringPtr(record.SessionID),
		ExpiresAt:        timestamppb.New(record.ExpiresAt),
		DisplayHint:      stringPtr(record.DisplayHint),
	}, nil
}

func (s *manualOAuthService) ConfirmLinkOAuth(ctx context.Context, req *authenticationv1.ConfirmLinkOAuthRequest) (*authenticationv1.ConfirmLinkOAuthResponse, error) {
	if s == nil || s.authFlowStore == nil || s.userCredentialRepo == nil {
		return nil, authenticationv1.ErrorInternalServerError("oauth service is unavailable")
	}
	viewer, ok := crudviewer.FromContext(ctx)
	if !ok || viewer == nil || viewer.UserID() == 0 {
		return nil, authenticationv1.ErrorUnauthorized("user is not authenticated")
	}
	operationID := strings.TrimSpace(req.GetOperationId())
	var (
		record *authFlowSessionRecord
		err    error
	)
	if operationID != "" {
		record, err = s.authFlowStore.GetSession(operationID)
	} else {
		record, err = s.authFlowStore.FindSessionByToken(strings.TrimSpace(req.GetState()))
	}
	if err != nil || record == nil {
		return nil, authenticationv1.ErrorUnauthorized("oauth session is invalid or expired")
	}
	if record.UserID != uint32(viewer.UserID()) {
		return nil, authenticationv1.ErrorForbidden("oauth session does not belong to current user")
	}
	code := strings.TrimSpace(req.GetCode())
	if code == "" {
		return nil, authenticationv1.ErrorBadRequest("oauth code is required")
	}
	profile, err := (&manualSocialAuthService{
		authFlowStore: s.authFlowStore,
		social:        s.social,
		log:           s.log,
	}).resolveSocialProfile(ctx, &authenticationv1.CompleteSocialLoginRequest{
		Provider:    req.GetProvider(),
		ProviderKey: stringPtr(firstNonEmpty(record.ProviderKey, req.GetProviderCustom())),
		RedirectUri: stringPtr(firstNonEmpty(record.RedirectURI, req.GetProviderCustom())),
	}, firstNonEmpty(record.ProviderKey, providerKeyFromOAuthProvider(req.GetProvider())), code, record)
	if err != nil {
		return nil, err
	}
	if profile == nil || strings.TrimSpace(profile.ProviderAccountID) == "" {
		return nil, authenticationv1.ErrorUnauthorized("failed to resolve social identity")
	}

	existing, err := s.findLinkedCredentialByProvider(ctx, firstNonEmpty(record.ProviderKey, providerKeyFromOAuthProvider(req.GetProvider())), profile.ProviderAccountID)
	if err != nil {
		s.log.Errorf("%s find linked oauth credential failed: %s", authRequestLogPrefix(ctx, "manual_oauth.confirm"), err.Error())
		return nil, err
	}
	if existing != nil {
		if existing.GetUserId() != uint32(viewer.UserID()) {
			return nil, authenticationv1.ErrorConflict("third-party account is already linked to another user")
		}
		return &authenticationv1.ConfirmLinkOAuthResponse{
			Account: existing,
		}, nil
	}

	identityType := authenticationv1.UserCredential_SOCIAL_OAUTH
	credentialType := authenticationv1.UserCredential_OAUTH_TOKEN
	status := authenticationv1.UserCredential_ENABLED
	createReq := &authenticationv1.CreateUserCredentialRequest{
		Data: &authenticationv1.UserCredential{
			UserId:            uint32Ptr(uint32(viewer.UserID())),
			TenantId:          uint32Ptr(uint32(viewer.TenantID())),
			IdentityType:      &identityType,
			CredentialType:    &credentialType,
			Identifier:        stringPtr(profile.ProviderAccountID),
			Credential:        stringPtr(profile.RawProfileJSON),
			Status:            &status,
			Provider:          stringPtr(firstNonEmpty(record.ProviderKey, providerKeyFromOAuthProvider(req.GetProvider()))),
			ProviderAccountId: stringPtr(profile.ProviderAccountID),
			ExtraInfo:         stringPtr(profile.RawProfileJSON),
		},
	}
	if _, err := s.userCredentialRepo.Create(ensureDefaultViewerContext(ctx), createReq); err != nil {
		s.log.Errorf("%s create linked oauth credential failed user_id=%d: %s", authRequestLogPrefix(ctx, "manual_oauth.confirm"), viewer.UserID(), err.Error())
		return nil, err
	}
	created, err := s.findLinkedCredentialByProvider(ctx, firstNonEmpty(record.ProviderKey, providerKeyFromOAuthProvider(req.GetProvider())), profile.ProviderAccountID)
	if err != nil {
		return nil, err
	}
	record.Status = authenticationv1.AuthFlowStatus_AUTH_FLOW_CONFIRMED
	now := time.Now()
	record.ConfirmedAt = &now
	record.ProviderAccountID = profile.ProviderAccountID
	record.Profile = profile
	_ = s.authFlowStore.SaveSession(record)
	return &authenticationv1.ConfirmLinkOAuthResponse{
		Account: created,
	}, nil
}

func (s *manualOAuthService) UnlinkOAuth(ctx context.Context, req *authenticationv1.UnlinkOAuthRequest) (*emptypb.Empty, error) {
	if s == nil || s.userCredentialRepo == nil {
		return nil, authenticationv1.ErrorInternalServerError("oauth service is unavailable")
	}
	viewer, ok := crudviewer.FromContext(ctx)
	if !ok || viewer == nil || viewer.UserID() == 0 {
		return nil, authenticationv1.ErrorUnauthorized("user is not authenticated")
	}
	var target *authenticationv1.UserCredential
	var err error
	if req.GetCredentialId() != "" {
		target, err = s.userCredentialRepo.Get(ctx, &authenticationv1.GetUserCredentialRequest{
			QueryBy: &authenticationv1.GetUserCredentialRequest_Id{Id: parseUint32Safe(req.GetCredentialId())},
		})
	} else {
		target, err = s.findCurrentUserLinkedCredentialByProvider(ctx, uint32(viewer.UserID()), firstNonEmpty(req.GetProviderCustom(), providerKeyFromOAuthProvider(req.GetProvider())))
	}
	if err != nil {
		s.log.Errorf("%s load linked oauth credential failed: %s", authRequestLogPrefix(ctx, "manual_oauth.unlink"), err.Error())
		return nil, err
	}
	if target == nil || target.GetId() == 0 {
		return nil, authenticationv1.ErrorNotFound("linked oauth account not found")
	}
	if target.GetUserId() != uint32(viewer.UserID()) {
		return nil, authenticationv1.ErrorForbidden("linked oauth account does not belong to current user")
	}
	_, err = s.userCredentialRepo.Delete(ctx, &authenticationv1.DeleteUserCredentialRequest{
		QueryBy: &authenticationv1.DeleteUserCredentialRequest_Id{Id: target.GetId()},
	})
	if err != nil {
		s.log.Errorf("%s unlink oauth credential failed credential_id=%d: %s", authRequestLogPrefix(ctx, "manual_oauth.unlink"), target.GetId(), err.Error())
		return nil, err
	}
	return &emptypb.Empty{}, nil
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
		requestTenantCode := req.GetTenantCode()
		if strings.TrimSpace(requestTenantCode) == "" && defaultTenantID > 0 {
			requestTenantCode = ""
		}
		assignment, err := s.resolveRegistrationAssignment(ctx, requestTenantCode, defaultTenantID)
		if err != nil {
			return nil, err
		}
		status := identityv1.User_NORMAL
		createReq := &identityv1.CreateUserRequest{
			Data: &identityv1.User{
				Username:    &username,
				Email:       payload.ByUsername.Email,
				Mobile:      payload.ByUsername.Mobile,
				TenantId:    uint32Ptr(assignment.TenantID),
				RoleIds:     assignment.RoleIDs,
				OrgUnitIds:  assignment.OrgUnitIDs,
				PositionIds: assignment.PositionIDs,
				Status:      &status,
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

func (s *manualSocialAuthService) resolveRegistrationAssignment(ctx context.Context, tenantCode string, defaultTenantID uint32) (*registrationAssignment, error) {
	if s.registration == nil {
		return &registrationAssignment{TenantID: defaultTenantID}, nil
	}
	if defaultTenantID > 0 {
		return s.registration.resolveRegistrationAssignmentForTenantID(ensureDefaultViewerContext(ctx), defaultTenantID)
	}
	return s.registration.resolveRegistrationAssignment(ensureDefaultViewerContext(ctx), tenantCode)
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
		return "dingtalk_web"
	case authenticationv1.OAuthProvider_WECHAT:
		return "wechat_web"
	case authenticationv1.OAuthProvider_ALIPAY:
		return "alipay"
	default:
		return strings.ToLower(provider.String())
	}
}

func normalizeOAuthProviderKey(providerKey string, provider authenticationv1.OAuthProvider) string {
	switch strings.TrimSpace(providerKey) {
	case "":
		if strings.TrimSpace(providerKey) == "" {
			return providerKeyFromOAuthProvider(provider)
		}
		return strings.TrimSpace(providerKey)
	case "dingtalk":
		return "dingtalk_web"
	case "wechat":
		return "wechat_web"
	case "github", "dingtalk_web", "alipay", "douyin", "wechat_web", "wechat_miniapp":
		return strings.TrimSpace(providerKey)
	default:
		return strings.TrimSpace(providerKey)
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

type socialProviderConfig interface {
	getAuthURL() string
	getTokenURL() string
	getScopes() []string
}

func (c githubSocialProviderConfig) getAuthURL() string {
	return c.AuthURL
}

func (c githubSocialProviderConfig) getTokenURL() string {
	return c.TokenURL
}

func (c githubSocialProviderConfig) getScopes() []string {
	return c.Scopes
}

func (c wechatSocialProviderConfig) getAuthURL() string {
	return c.AuthURL
}

func (c wechatSocialProviderConfig) getTokenURL() string {
	return c.TokenURL
}

func (c wechatSocialProviderConfig) getScopes() []string {
	return c.Scopes
}

func (c dingtalkSocialProviderConfig) getAuthURL() string {
	return c.AuthURL
}

func (c dingtalkSocialProviderConfig) getTokenURL() string {
	return c.TokenURL
}

func (c dingtalkSocialProviderConfig) getScopes() []string {
	return c.Scopes
}

func providerMetadataFromConfig(provider authenticationv1.OAuthProvider, providerCustom, displayName string, cfg socialProviderConfig) *authenticationv1.ProviderMetadata {
	return &authenticationv1.ProviderMetadata{
		Provider:              provider,
		ProviderCustom:        providerCustom,
		DisplayName:           displayName,
		AuthorizationEndpoint: cfg.getAuthURL(),
		TokenEndpoint:         cfg.getTokenURL(),
		DefaultScopes:         append([]string(nil), cfg.getScopes()...),
	}
}

func providerMetadataPlaceholder(provider authenticationv1.OAuthProvider, providerCustom, displayName string) *authenticationv1.ProviderMetadata {
	return &authenticationv1.ProviderMetadata{
		Provider:       provider,
		ProviderCustom: providerCustom,
		DisplayName:    displayName,
	}
}

type socialAuthConfig struct {
	GitHub      githubSocialProviderConfig
	DingTalkWeb dingtalkSocialProviderConfig
	WeChatWeb   wechatSocialProviderConfig
}

type githubSocialProviderConfig struct {
	AuthURL      string
	ClientID     string
	ClientSecret string
	Enabled      bool
	RedirectURI  string
	Scopes       []string
	TokenURL     string
	UserAPIURL   string
}

type wechatSocialProviderConfig struct {
	AuthURL      string
	ClientID     string
	ClientSecret string
	Enabled      bool
	RedirectURI  string
	Scopes       []string
	TokenURL     string
	UserAPIURL   string
}

type dingtalkSocialProviderConfig struct {
	AuthURL      string
	ClientID     string
	ClientSecret string
	Enabled      bool
	RedirectURI  string
	Scopes       []string
	TokenURL     string
	UserAPIURL   string
}

type githubAccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	Error       string `json:"error"`
	Description string `json:"error_description"`
}

type githubUserProfile struct {
	AvatarURL string `json:"avatar_url"`
	Email     string `json:"email"`
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
}

type wechatAccessTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in"`
	OpenID       string `json:"openid"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	UnionID      string `json:"unionid"`
	ErrCode      int64  `json:"errcode"`
	ErrMsg       string `json:"errmsg"`
}

type wechatUserProfile struct {
	OpenID     string `json:"openid"`
	Nickname   string `json:"nickname"`
	Sex        int32  `json:"sex"`
	Province   string `json:"province"`
	City       string `json:"city"`
	Country    string `json:"country"`
	HeadImgURL string `json:"headimgurl"`
	UnionID    string `json:"unionid"`
	ErrCode    int64  `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

type dingtalkAccessTokenResponse struct {
	AccessToken  string `json:"accessToken"`
	ExpireIn     int64  `json:"expireIn"`
	RefreshToken string `json:"refreshToken"`
}

type dingtalkUserProfile struct {
	AvatarURL string `json:"avatarUrl"`
	Email     string `json:"email"`
	Mobile    string `json:"mobile"`
	Nick      string `json:"nick"`
	OpenID    string `json:"openId"`
	StateCode string `json:"stateCode"`
	UnionID   string `json:"unionId"`
}

func loadSocialAuthConfig(appCtx *app.AppCtx) (*socialAuthConfig, error) {
	if appCtx == nil || appCtx.GetConfig() == nil || appCtx.GetConfig().GetServer() == nil {
		return nil, fmt.Errorf("app config is missing")
	}
	restCfg := appCtx.GetConfig().GetServer().GetRest()
	defaultGitHubRedirectURI := "http://localhost:5666/auth/social/callback/github"
	defaultDingTalkRedirectURI := "http://localhost:5666/auth/social/callback/dingtalk"
	defaultWeChatWebRedirectURI := "http://localhost:5666/auth/social/callback/wechat"
	if restCfg != nil {
		if address := strings.TrimSpace(restCfg.GetAddr()); address != "" {
			if strings.HasPrefix(address, ":") {
				defaultGitHubRedirectURI = "http://localhost" + address + "/auth/social/callback/github"
				defaultDingTalkRedirectURI = "http://localhost" + address + "/auth/social/callback/dingtalk"
				defaultWeChatWebRedirectURI = "http://localhost" + address + "/auth/social/callback/wechat"
			} else if strings.HasPrefix(address, "http://") || strings.HasPrefix(address, "https://") {
				base := strings.TrimRight(address, "/")
				defaultGitHubRedirectURI = base + "/auth/social/callback/github"
				defaultDingTalkRedirectURI = base + "/auth/social/callback/dingtalk"
				defaultWeChatWebRedirectURI = base + "/auth/social/callback/wechat"
			}
		}
	}
	cfg := &socialAuthConfig{}
	authn := appCtx.GetConfig().GetAuthn()
	if authn != nil && authn.GetSocialAuth() != nil && authn.GetSocialAuth().GetGithub() != nil {
		github := authn.GetSocialAuth().GetGithub()
		cfg.GitHub = githubSocialProviderConfig{
			AuthURL:      strings.TrimSpace(github.GetAuthUrl()),
			ClientID:     strings.TrimSpace(github.GetClientId()),
			ClientSecret: strings.TrimSpace(github.GetClientSecret()),
			Enabled:      github.GetEnabled(),
			RedirectURI:  strings.TrimSpace(github.GetRedirectUri()),
			Scopes:       append([]string(nil), github.GetScopes()...),
			TokenURL:     strings.TrimSpace(github.GetTokenUrl()),
			UserAPIURL:   strings.TrimSpace(github.GetUserApiUrl()),
		}
	}
	if authn != nil && authn.GetSocialAuth() != nil && authn.GetSocialAuth().GetDingtalk() != nil {
		dingtalk := authn.GetSocialAuth().GetDingtalk()
		cfg.DingTalkWeb = dingtalkSocialProviderConfig{
			AuthURL:      strings.TrimSpace(dingtalk.GetAuthUrl()),
			ClientID:     strings.TrimSpace(dingtalk.GetClientId()),
			ClientSecret: strings.TrimSpace(dingtalk.GetClientSecret()),
			Enabled:      dingtalk.GetEnabled(),
			RedirectURI:  strings.TrimSpace(dingtalk.GetRedirectUri()),
			Scopes:       append([]string(nil), dingtalk.GetScopes()...),
			TokenURL:     strings.TrimSpace(dingtalk.GetTokenUrl()),
			UserAPIURL:   strings.TrimSpace(dingtalk.GetUserApiUrl()),
		}
	}
	if authn != nil && authn.GetSocialAuth() != nil && authn.GetSocialAuth().GetWechatWeb() != nil {
		wechat := authn.GetSocialAuth().GetWechatWeb()
		cfg.WeChatWeb = wechatSocialProviderConfig{
			AuthURL:      strings.TrimSpace(wechat.GetAuthUrl()),
			ClientID:     strings.TrimSpace(wechat.GetClientId()),
			ClientSecret: strings.TrimSpace(wechat.GetClientSecret()),
			Enabled:      wechat.GetEnabled(),
			RedirectURI:  strings.TrimSpace(wechat.GetRedirectUri()),
			Scopes:       append([]string(nil), wechat.GetScopes()...),
			TokenURL:     strings.TrimSpace(wechat.GetTokenUrl()),
			UserAPIURL:   strings.TrimSpace(wechat.GetUserApiUrl()),
		}
	}
	if cfg.GitHub.AuthURL == "" {
		cfg.GitHub.AuthURL = "https://github.com/login/oauth/authorize"
	}
	if cfg.GitHub.TokenURL == "" {
		cfg.GitHub.TokenURL = "https://github.com/login/oauth/access_token"
	}
	if cfg.GitHub.UserAPIURL == "" {
		cfg.GitHub.UserAPIURL = "https://api.github.com/user"
	}
	if len(cfg.GitHub.Scopes) == 0 {
		cfg.GitHub.Scopes = []string{"read:user", "user:email"}
	}
	if cfg.GitHub.ClientID == "" {
		cfg.GitHub.ClientID = strings.TrimSpace(getenvFirst("ADMIN_SOCIAL_GITHUB_CLIENT_ID", "GITHUB_CLIENT_ID"))
	}
	if cfg.GitHub.ClientSecret == "" {
		cfg.GitHub.ClientSecret = strings.TrimSpace(getenvFirst("ADMIN_SOCIAL_GITHUB_CLIENT_SECRET", "GITHUB_CLIENT_SECRET"))
	}
	if cfg.GitHub.RedirectURI == "" {
		cfg.GitHub.RedirectURI = strings.TrimSpace(getenvFirst("ADMIN_SOCIAL_GITHUB_REDIRECT_URI"))
	}
	if cfg.GitHub.RedirectURI == "" {
		cfg.GitHub.RedirectURI = defaultGitHubRedirectURI
	}
	if !cfg.GitHub.Enabled {
		cfg.GitHub.Enabled = cfg.GitHub.ClientID != "" && cfg.GitHub.ClientSecret != ""
	}

	if cfg.DingTalkWeb.AuthURL == "" {
		cfg.DingTalkWeb.AuthURL = "https://login.dingtalk.com/oauth2/auth"
	}
	if cfg.DingTalkWeb.TokenURL == "" {
		cfg.DingTalkWeb.TokenURL = "https://api.dingtalk.com/v1.0/oauth2/userAccessToken"
	}
	if cfg.DingTalkWeb.UserAPIURL == "" {
		cfg.DingTalkWeb.UserAPIURL = "https://api.dingtalk.com/v1.0/contact/users/me"
	}
	if len(cfg.DingTalkWeb.Scopes) == 0 {
		cfg.DingTalkWeb.Scopes = []string{"openid"}
	}
	if cfg.DingTalkWeb.ClientID == "" {
		cfg.DingTalkWeb.ClientID = strings.TrimSpace(getenvFirst("ADMIN_SOCIAL_DINGTALK_CLIENT_ID", "DINGTALK_CLIENT_ID"))
	}
	if cfg.DingTalkWeb.ClientSecret == "" {
		cfg.DingTalkWeb.ClientSecret = strings.TrimSpace(getenvFirst("ADMIN_SOCIAL_DINGTALK_CLIENT_SECRET", "DINGTALK_CLIENT_SECRET"))
	}
	if cfg.DingTalkWeb.RedirectURI == "" {
		cfg.DingTalkWeb.RedirectURI = strings.TrimSpace(getenvFirst("ADMIN_SOCIAL_DINGTALK_REDIRECT_URI", "DINGTALK_REDIRECT_URI"))
	}
	if cfg.DingTalkWeb.RedirectURI == "" {
		cfg.DingTalkWeb.RedirectURI = defaultDingTalkRedirectURI
	}
	if !cfg.DingTalkWeb.Enabled {
		cfg.DingTalkWeb.Enabled = cfg.DingTalkWeb.ClientID != "" && cfg.DingTalkWeb.ClientSecret != ""
	}

	if cfg.WeChatWeb.AuthURL == "" {
		cfg.WeChatWeb.AuthURL = "https://open.weixin.qq.com/connect/qrconnect"
	}
	if cfg.WeChatWeb.TokenURL == "" {
		cfg.WeChatWeb.TokenURL = "https://api.weixin.qq.com/sns/oauth2/access_token"
	}
	if cfg.WeChatWeb.UserAPIURL == "" {
		cfg.WeChatWeb.UserAPIURL = "https://api.weixin.qq.com/sns/userinfo"
	}
	if len(cfg.WeChatWeb.Scopes) == 0 {
		cfg.WeChatWeb.Scopes = []string{"snsapi_login"}
	}
	if cfg.WeChatWeb.ClientID == "" {
		cfg.WeChatWeb.ClientID = strings.TrimSpace(getenvFirst("ADMIN_SOCIAL_WECHAT_WEB_CLIENT_ID", "WECHAT_WEB_CLIENT_ID", "WECHAT_CLIENT_ID"))
	}
	if cfg.WeChatWeb.ClientSecret == "" {
		cfg.WeChatWeb.ClientSecret = strings.TrimSpace(getenvFirst("ADMIN_SOCIAL_WECHAT_WEB_CLIENT_SECRET", "WECHAT_WEB_CLIENT_SECRET", "WECHAT_CLIENT_SECRET"))
	}
	if cfg.WeChatWeb.RedirectURI == "" {
		cfg.WeChatWeb.RedirectURI = strings.TrimSpace(getenvFirst("ADMIN_SOCIAL_WECHAT_WEB_REDIRECT_URI", "WECHAT_WEB_REDIRECT_URI"))
	}
	if cfg.WeChatWeb.RedirectURI == "" {
		cfg.WeChatWeb.RedirectURI = defaultWeChatWebRedirectURI
	}
	if !cfg.WeChatWeb.Enabled {
		cfg.WeChatWeb.Enabled = cfg.WeChatWeb.ClientID != "" && cfg.WeChatWeb.ClientSecret != ""
	}
	return cfg, nil
}

func getenvFirst(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(strings.TrimSpace(strings.Trim(os.Getenv(key), "\""))); value != "" {
			return value
		}
	}
	return ""
}

func (s *manualSocialAuthService) buildAuthorizationURL(req *authenticationv1.StartSocialLoginRequest, session *authenticationv1.AuthSession) (string, error) {
	providerKey := firstNonEmpty(strings.TrimSpace(req.GetProviderKey()), providerKeyFromOAuthProvider(req.GetProvider()))
	switch providerKey {
	case "github":
		return s.buildGitHubAuthorizationURL(req, session)
	case "dingtalk_web":
		return s.buildDingTalkAuthorizationURL(req, session)
	case "wechat_web":
		return s.buildWeChatWebAuthorizationURL(req, session)
	default:
		return "", authenticationv1.ErrorNotImplemented("this social provider is not implemented yet")
	}
}

func (s *manualSocialAuthService) buildGitHubAuthorizationURL(req *authenticationv1.StartSocialLoginRequest, session *authenticationv1.AuthSession) (string, error) {
	if s == nil || s.social == nil || !s.social.GitHub.Enabled {
		return "", authenticationv1.ErrorInternalServerError("github social auth is not configured")
	}
	redirectURI := strings.TrimSpace(req.GetRedirectUri())
	if redirectURI == "" {
		redirectURI = s.social.GitHub.RedirectURI
	}
	query := url.Values{}
	query.Set("client_id", s.social.GitHub.ClientID)
	query.Set("redirect_uri", redirectURI)
	query.Set("state", session.GetSessionToken())
	query.Set("scope", strings.Join(s.social.GitHub.Scopes, " "))
	query.Set("allow_signup", "true")
	return s.social.GitHub.AuthURL + "?" + query.Encode(), nil
}

func (s *manualSocialAuthService) buildDingTalkAuthorizationURL(req *authenticationv1.StartSocialLoginRequest, session *authenticationv1.AuthSession) (string, error) {
	if s == nil || s.social == nil || !s.social.DingTalkWeb.Enabled {
		return "", authenticationv1.ErrorInternalServerError("dingtalk social auth is not configured")
	}
	redirectURI := strings.TrimSpace(req.GetRedirectUri())
	if redirectURI == "" {
		redirectURI = s.social.DingTalkWeb.RedirectURI
	}
	query := url.Values{}
	query.Set("client_id", s.social.DingTalkWeb.ClientID)
	query.Set("redirect_uri", redirectURI)
	query.Set("response_type", "code")
	query.Set("scope", strings.Join(s.social.DingTalkWeb.Scopes, " "))
	query.Set("prompt", "consent")
	query.Set("state", session.GetSessionToken())
	authorizationURL := s.social.DingTalkWeb.AuthURL + "?" + query.Encode()
	if s.log != nil {
		s.log.Infof("chain=manual_social_auth.dingtalk_web action=build_authorization_url client_id=%s redirect_uri=%s url=%s", s.social.DingTalkWeb.ClientID, redirectURI, authorizationURL)
	}
	return authorizationURL, nil
}

func (s *manualSocialAuthService) buildWeChatWebAuthorizationURL(req *authenticationv1.StartSocialLoginRequest, session *authenticationv1.AuthSession) (string, error) {
	if s == nil || s.social == nil || !s.social.WeChatWeb.Enabled {
		return "", authenticationv1.ErrorInternalServerError("wechat web social auth is not configured")
	}
	redirectURI := strings.TrimSpace(req.GetRedirectUri())
	if redirectURI == "" {
		redirectURI = s.social.WeChatWeb.RedirectURI
	}
	query := url.Values{}
	query.Set("appid", s.social.WeChatWeb.ClientID)
	query.Set("redirect_uri", redirectURI)
	query.Set("response_type", "code")
	// WeChat website login only accepts snsapi_login here.
	query.Set("scope", "snsapi_login")
	query.Set("state", session.GetSessionToken())
	authorizationURL := s.social.WeChatWeb.AuthURL + "?" + query.Encode() + "#wechat_redirect"
	if s.log != nil {
		s.log.Infof("chain=manual_social_auth.wechat_web action=build_authorization_url appid=%s redirect_uri=%s url=%s", s.social.WeChatWeb.ClientID, redirectURI, authorizationURL)
	}
	return authorizationURL, nil
}

func (s *manualSocialAuthService) resolveAuthSessionForComplete(req *authenticationv1.CompleteSocialLoginRequest, sessionID string) (*authFlowSessionRecord, error) {
	if s == nil || s.authFlowStore == nil || strings.TrimSpace(sessionID) == "" {
		return nil, nil
	}
	record, err := s.authFlowStore.GetSession(sessionID)
	if err != nil {
		if strings.TrimSpace(req.GetState()) == "" {
			return nil, authenticationv1.ErrorUnauthorized("social auth session is invalid or expired")
		}
		return nil, nil
	}
	if state := strings.TrimSpace(req.GetState()); state != "" && record != nil && record.SessionToken != "" && state != record.SessionToken {
		return nil, authenticationv1.ErrorUnauthorized("social auth state mismatch")
	}
	return record, nil
}

func (s *manualSocialAuthService) resolveSocialProfile(ctx context.Context, req *authenticationv1.CompleteSocialLoginRequest, providerKey, code string, sessionRecord *authFlowSessionRecord) (*authFlowProviderProfile, error) {
	switch firstNonEmpty(providerKey, providerKeyFromOAuthProvider(req.GetProvider())) {
	case "github":
		return s.resolveGitHubProfile(ctx, req, code, sessionRecord)
	case "dingtalk_web":
		return s.resolveDingTalkProfile(ctx, req, code, sessionRecord)
	case "wechat_web":
		return s.resolveWeChatWebProfile(ctx, req, code, sessionRecord)
	default:
		return &authFlowProviderProfile{
			ProviderKey:       providerKey,
			Provider:          req.GetProvider(),
			ProviderAccountID: code,
			Nickname:          defaultSocialNickname(providerKey, code),
			RawProfileJSON:    socialProfileJSON(providerKey, code),
		}, nil
	}
}

func (s *manualSocialAuthService) resolveWeChatWebProfile(ctx context.Context, req *authenticationv1.CompleteSocialLoginRequest, code string, sessionRecord *authFlowSessionRecord) (*authFlowProviderProfile, error) {
	if s == nil || s.social == nil || !s.social.WeChatWeb.Enabled {
		return nil, authenticationv1.ErrorInternalServerError("wechat web social auth is not configured")
	}
	redirectURI := strings.TrimSpace(req.GetRedirectUri())
	if redirectURI == "" && sessionRecord != nil {
		redirectURI = strings.TrimSpace(sessionRecord.RedirectURI)
	}
	if redirectURI == "" {
		redirectURI = s.social.WeChatWeb.RedirectURI
	}
	tokenResp, err := s.exchangeWeChatWebAccessToken(ctx, code, redirectURI)
	if err != nil {
		return nil, err
	}
	profile, rawProfile, err := s.fetchWeChatWebProfile(ctx, tokenResp)
	if err != nil {
		return nil, err
	}
	accountID := firstNonEmpty(strings.TrimSpace(profile.UnionID), strings.TrimSpace(profile.OpenID), strings.TrimSpace(tokenResp.UnionID), strings.TrimSpace(tokenResp.OpenID))
	if accountID == "" {
		return nil, authenticationv1.ErrorUnauthorized("wechat account id is empty")
	}
	return &authFlowProviderProfile{
		ProviderKey:       "wechat_web",
		Provider:          authenticationv1.OAuthProvider_WECHAT,
		ProviderAccountID: accountID,
		Nickname:          strings.TrimSpace(profile.Nickname),
		Avatar:            strings.TrimSpace(profile.HeadImgURL),
		RawProfileJSON:    rawProfile,
	}, nil
}

func (s *manualSocialAuthService) resolveDingTalkProfile(ctx context.Context, req *authenticationv1.CompleteSocialLoginRequest, code string, sessionRecord *authFlowSessionRecord) (*authFlowProviderProfile, error) {
	if s == nil || s.social == nil || !s.social.DingTalkWeb.Enabled {
		return nil, authenticationv1.ErrorInternalServerError("dingtalk social auth is not configured")
	}
	redirectURI := strings.TrimSpace(req.GetRedirectUri())
	if redirectURI == "" && sessionRecord != nil {
		redirectURI = strings.TrimSpace(sessionRecord.RedirectURI)
	}
	if redirectURI == "" {
		redirectURI = s.social.DingTalkWeb.RedirectURI
	}
	accessToken, err := s.exchangeDingTalkAccessToken(ctx, code, redirectURI)
	if err != nil {
		return nil, err
	}
	profile, rawProfile, err := s.fetchDingTalkProfile(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	accountID := firstNonEmpty(strings.TrimSpace(profile.UnionID), strings.TrimSpace(profile.OpenID))
	if accountID == "" {
		return nil, authenticationv1.ErrorUnauthorized("dingtalk account id is empty")
	}
	return &authFlowProviderProfile{
		ProviderKey:       "dingtalk_web",
		Provider:          authenticationv1.OAuthProvider_DINGTALK,
		ProviderAccountID: accountID,
		Nickname:          strings.TrimSpace(profile.Nick),
		Avatar:            strings.TrimSpace(profile.AvatarURL),
		Email:             strings.TrimSpace(profile.Email),
		Mobile:            strings.TrimSpace(profile.Mobile),
		RawProfileJSON:    rawProfile,
	}, nil
}

func (s *manualSocialAuthService) resolveGitHubProfile(ctx context.Context, req *authenticationv1.CompleteSocialLoginRequest, code string, sessionRecord *authFlowSessionRecord) (*authFlowProviderProfile, error) {
	if s == nil || s.social == nil || !s.social.GitHub.Enabled {
		return nil, authenticationv1.ErrorInternalServerError("github social auth is not configured")
	}
	redirectURI := strings.TrimSpace(req.GetRedirectUri())
	if redirectURI == "" && sessionRecord != nil {
		redirectURI = strings.TrimSpace(sessionRecord.RedirectURI)
	}
	if redirectURI == "" {
		redirectURI = s.social.GitHub.RedirectURI
	}
	token, err := s.exchangeGitHubAccessToken(ctx, code, redirectURI)
	if err != nil {
		return nil, err
	}
	profile, rawProfile, err := s.fetchGitHubProfile(ctx, token)
	if err != nil {
		return nil, err
	}
	accountID := fmt.Sprintf("%d", profile.ID)
	nickname := strings.TrimSpace(profile.Name)
	if nickname == "" {
		nickname = strings.TrimSpace(profile.Login)
	}
	return &authFlowProviderProfile{
		ProviderKey:       "github",
		Provider:          authenticationv1.OAuthProvider_GITHUB,
		ProviderAccountID: accountID,
		Nickname:          nickname,
		Avatar:            strings.TrimSpace(profile.AvatarURL),
		Email:             strings.TrimSpace(profile.Email),
		RawProfileJSON:    rawProfile,
	}, nil
}

func (s *manualSocialAuthService) exchangeGitHubAccessToken(ctx context.Context, code, redirectURI string) (string, error) {
	values := url.Values{}
	values.Set("client_id", s.social.GitHub.ClientID)
	values.Set("client_secret", s.social.GitHub.ClientSecret)
	values.Set("code", code)
	if strings.TrimSpace(redirectURI) != "" {
		values.Set("redirect_uri", redirectURI)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.social.GitHub.TokenURL, strings.NewReader(values.Encode()))
	if err != nil {
		return "", authenticationv1.ErrorInternalServerError("failed to build github token request")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", authenticationv1.ErrorInternalServerError("github token exchange failed")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", authenticationv1.ErrorInternalServerError("failed to read github token response")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.log.Errorf("chain=manual_social_auth.github status=token_exchange code=%d body=%s", resp.StatusCode, string(body))
		return "", authenticationv1.ErrorUnauthorized("github token exchange failed")
	}
	var tokenResp githubAccessTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", authenticationv1.ErrorInternalServerError("failed to decode github token response")
	}
	if strings.TrimSpace(tokenResp.AccessToken) == "" {
		if strings.TrimSpace(tokenResp.Description) != "" {
			return "", authenticationv1.ErrorUnauthorized("github token exchange rejected by provider")
		}
		return "", authenticationv1.ErrorUnauthorized("github access token is empty")
	}
	return tokenResp.AccessToken, nil
}

func (s *manualSocialAuthService) exchangeDingTalkAccessToken(ctx context.Context, code, redirectURI string) (string, error) {
	payload, err := json.Marshal(map[string]string{
		"clientId":     s.social.DingTalkWeb.ClientID,
		"clientSecret": s.social.DingTalkWeb.ClientSecret,
		"code":         code,
		"grantType":    "authorization_code",
		"redirectUri":  redirectURI,
	})
	if err != nil {
		return "", authenticationv1.ErrorInternalServerError("failed to encode dingtalk token request")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.social.DingTalkWeb.TokenURL, strings.NewReader(string(payload)))
	if err != nil {
		return "", authenticationv1.ErrorInternalServerError("failed to build dingtalk token request")
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", authenticationv1.ErrorInternalServerError("dingtalk token exchange failed")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", authenticationv1.ErrorInternalServerError("failed to read dingtalk token response")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.log.Errorf("chain=manual_social_auth.dingtalk_web status=token_exchange code=%d body=%s", resp.StatusCode, string(body))
		return "", authenticationv1.ErrorUnauthorized("dingtalk token exchange failed")
	}
	var tokenResp dingtalkAccessTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", authenticationv1.ErrorInternalServerError("failed to decode dingtalk token response")
	}
	if strings.TrimSpace(tokenResp.AccessToken) == "" {
		return "", authenticationv1.ErrorUnauthorized("dingtalk access token is empty")
	}
	return tokenResp.AccessToken, nil
}

func (s *manualSocialAuthService) fetchGitHubProfile(ctx context.Context, accessToken string) (*githubUserProfile, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.social.GitHub.UserAPIURL, nil)
	if err != nil {
		return nil, "", authenticationv1.ErrorInternalServerError("failed to build github profile request")
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("User-Agent", "xadmin-social-auth")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", authenticationv1.ErrorInternalServerError("github profile request failed")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", authenticationv1.ErrorInternalServerError("failed to read github profile response")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.log.Errorf("chain=manual_social_auth.github status=fetch_profile code=%d body=%s", resp.StatusCode, string(body))
		return nil, "", authenticationv1.ErrorUnauthorized("github profile request failed")
	}
	var profile githubUserProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, "", authenticationv1.ErrorInternalServerError("failed to decode github profile")
	}
	if profile.ID == 0 {
		return nil, "", authenticationv1.ErrorUnauthorized("github account id is empty")
	}
	return &profile, string(body), nil
}

func (s *manualSocialAuthService) fetchDingTalkProfile(ctx context.Context, accessToken string) (*dingtalkUserProfile, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.social.DingTalkWeb.UserAPIURL, nil)
	if err != nil {
		return nil, "", authenticationv1.ErrorInternalServerError("failed to build dingtalk profile request")
	}
	req.Header.Set("x-acs-dingtalk-access-token", accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", authenticationv1.ErrorInternalServerError("dingtalk profile request failed")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", authenticationv1.ErrorInternalServerError("failed to read dingtalk profile response")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.log.Errorf("chain=manual_social_auth.dingtalk_web status=fetch_profile code=%d body=%s", resp.StatusCode, string(body))
		return nil, "", authenticationv1.ErrorUnauthorized("dingtalk profile request failed")
	}
	var profile dingtalkUserProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, "", authenticationv1.ErrorInternalServerError("failed to decode dingtalk profile")
	}
	if strings.TrimSpace(profile.UnionID) == "" && strings.TrimSpace(profile.OpenID) == "" {
		return nil, "", authenticationv1.ErrorUnauthorized("dingtalk account id is empty")
	}
	return &profile, string(body), nil
}

func (s *manualSocialAuthService) exchangeWeChatWebAccessToken(ctx context.Context, code, redirectURI string) (*wechatAccessTokenResponse, error) {
	query := url.Values{}
	query.Set("appid", s.social.WeChatWeb.ClientID)
	query.Set("secret", s.social.WeChatWeb.ClientSecret)
	query.Set("code", code)
	query.Set("grant_type", "authorization_code")
	if strings.TrimSpace(redirectURI) != "" {
		query.Set("redirect_uri", redirectURI)
	}
	endpoint := s.social.WeChatWeb.TokenURL + "?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, authenticationv1.ErrorInternalServerError("failed to build wechat token request")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, authenticationv1.ErrorInternalServerError("wechat token exchange failed")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, authenticationv1.ErrorInternalServerError("failed to read wechat token response")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.log.Errorf("chain=manual_social_auth.wechat_web status=token_exchange code=%d body=%s", resp.StatusCode, string(body))
		return nil, authenticationv1.ErrorUnauthorized("wechat token exchange failed")
	}
	var tokenResp wechatAccessTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, authenticationv1.ErrorInternalServerError("failed to decode wechat token response")
	}
	if tokenResp.ErrCode != 0 {
		s.log.Errorf("chain=manual_social_auth.wechat_web status=token_exchange errcode=%d errmsg=%s", tokenResp.ErrCode, tokenResp.ErrMsg)
		return nil, authenticationv1.ErrorUnauthorized("wechat token exchange rejected by provider")
	}
	if strings.TrimSpace(tokenResp.AccessToken) == "" || strings.TrimSpace(tokenResp.OpenID) == "" {
		return nil, authenticationv1.ErrorUnauthorized("wechat access token is empty")
	}
	return &tokenResp, nil
}

func (s *manualSocialAuthService) fetchWeChatWebProfile(ctx context.Context, tokenResp *wechatAccessTokenResponse) (*wechatUserProfile, string, error) {
	if tokenResp == nil {
		return nil, "", authenticationv1.ErrorUnauthorized("wechat token response is empty")
	}
	query := url.Values{}
	query.Set("access_token", tokenResp.AccessToken)
	query.Set("openid", tokenResp.OpenID)
	query.Set("lang", "zh_CN")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.social.WeChatWeb.UserAPIURL+"?"+query.Encode(), nil)
	if err != nil {
		return nil, "", authenticationv1.ErrorInternalServerError("failed to build wechat profile request")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", authenticationv1.ErrorInternalServerError("wechat profile request failed")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", authenticationv1.ErrorInternalServerError("failed to read wechat profile response")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.log.Errorf("chain=manual_social_auth.wechat_web status=fetch_profile code=%d body=%s", resp.StatusCode, string(body))
		return nil, "", authenticationv1.ErrorUnauthorized("wechat profile request failed")
	}
	var profile wechatUserProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, "", authenticationv1.ErrorInternalServerError("failed to decode wechat profile")
	}
	if profile.ErrCode != 0 {
		s.log.Errorf("chain=manual_social_auth.wechat_web status=fetch_profile errcode=%d errmsg=%s", profile.ErrCode, profile.ErrMsg)
		return nil, "", authenticationv1.ErrorUnauthorized("wechat profile request rejected by provider")
	}
	if strings.TrimSpace(profile.OpenID) == "" {
		return nil, "", authenticationv1.ErrorUnauthorized("wechat openid is empty")
	}
	return &profile, string(body), nil
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

func boolPtr(value bool) *bool {
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func parseUint32Safe(value string) uint32 {
	var result uint32
	_, _ = fmt.Sscanf(strings.TrimSpace(value), "%d", &result)
	return result
}

func (s *manualOAuthService) findLinkedCredentialByProvider(ctx context.Context, providerKey, accountID string) (*authenticationv1.UserCredential, error) {
	if s == nil || s.userCredentialRepo == nil {
		return nil, nil
	}
	conditions := []*paginationv1.FilterCondition{
		{
			Field:      "provider",
			Op:         paginationv1.Operator_EQ,
			ValueOneof: &paginationv1.FilterCondition_Value{Value: providerKey},
		},
		{
			Field:      "identity_type",
			Op:         paginationv1.Operator_EQ,
			ValueOneof: &paginationv1.FilterCondition_Value{Value: authenticationv1.UserCredential_SOCIAL_OAUTH.String()},
		},
	}
	if strings.TrimSpace(accountID) != "" {
		conditions = append(conditions, &paginationv1.FilterCondition{
			Field:      "provider_account_id",
			Op:         paginationv1.Operator_EQ,
			ValueOneof: &paginationv1.FilterCondition_Value{Value: accountID},
		})
	}
	resp, err := s.userCredentialRepo.List(ensureDefaultViewerContext(ctx), &paginationv1.PagingRequest{
		Limit: uint32Ptr(1),
		FilteringType: &paginationv1.PagingRequest_FilterExpr{
			FilterExpr: &paginationv1.FilterExpr{
				Type:       paginationv1.ExprType_AND,
				Conditions: conditions,
			},
		},
	})
	if err != nil || resp == nil || len(resp.GetItems()) == 0 {
		return nil, err
	}
	return resp.GetItems()[0], nil
}

func (s *manualOAuthService) findCurrentUserLinkedCredentialByProvider(ctx context.Context, userID uint32, providerKey string) (*authenticationv1.UserCredential, error) {
	if s == nil || s.userCredentialRepo == nil {
		return nil, nil
	}
	resp, err := s.userCredentialRepo.List(ensureDefaultViewerContext(ctx), &paginationv1.PagingRequest{
		Limit: uint32Ptr(1),
		FilteringType: &paginationv1.PagingRequest_FilterExpr{
			FilterExpr: &paginationv1.FilterExpr{
				Type: paginationv1.ExprType_AND,
				Conditions: []*paginationv1.FilterCondition{
					{
						Field:      "user_id",
						Op:         paginationv1.Operator_EQ,
						ValueOneof: &paginationv1.FilterCondition_Value{Value: fmt.Sprintf("%d", userID)},
					},
					{
						Field:      "provider",
						Op:         paginationv1.Operator_EQ,
						ValueOneof: &paginationv1.FilterCondition_Value{Value: providerKey},
					},
					{
						Field:      "identity_type",
						Op:         paginationv1.Operator_EQ,
						ValueOneof: &paginationv1.FilterCondition_Value{Value: authenticationv1.UserCredential_SOCIAL_OAUTH.String()},
					},
				},
			},
		},
	})
	if err != nil || resp == nil || len(resp.GetItems()) == 0 {
		return nil, err
	}
	return resp.GetItems()[0], nil
}

func uint32Value(value *uint32) uint32 {
	if value == nil {
		return 0
	}
	return *value
}
