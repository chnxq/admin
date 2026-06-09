package server

import (
	"context"
	"net/http"
	"testing"

	"github.com/mojocn/base64Captcha"

	adminv1 "admin/api/gen/admin/v1"
	authenticationv1 "admin/api/gen/authentication/v1"
	identityv1 "admin/api/gen/identity/v1"
	permissionv1 "admin/api/gen/permission/v1"
	resourcev1 "admin/api/gen/resource/v1"
	"admin/internal/data/ent"
	"admin/internal/data/ent/user"
	"admin/internal/data/repo"
	paginationv1 "github.com/chnxq/x-crud/api/gen/pagination/v1"
	crudviewer "github.com/chnxq/x-crud/viewer"
	"github.com/chnxq/xkitpkg/app"
	conf "github.com/chnxq/xkitpkg/conf/v1"
	transport "github.com/chnxq/xkitpkg/transport"
	"google.golang.org/protobuf/types/known/emptypb"
)

type testPortalViewer struct {
	userID uint64
}

func (v testPortalViewer) UserID() uint64                    { return v.userID }
func (v testPortalViewer) TenantID() uint64                  { return 0 }
func (v testPortalViewer) OrgUnitID() uint64                 { return 0 }
func (v testPortalViewer) Permissions() []string             { return nil }
func (v testPortalViewer) Roles() []string                   { return nil }
func (v testPortalViewer) DataScope() []crudviewer.DataScope { return nil }
func (v testPortalViewer) TraceID() string                   { return "" }
func (v testPortalViewer) HasPermission(string, string) bool { return false }
func (v testPortalViewer) IsPlatformContext() bool           { return true }
func (v testPortalViewer) IsTenantContext() bool             { return false }
func (v testPortalViewer) IsSystemContext() bool             { return true }
func (v testPortalViewer) ShouldAudit() bool                 { return false }

type stubUserRepo struct {
	user *identityv1.User
}

func (r stubUserRepo) List(context.Context, *paginationv1.PagingRequest) (*identityv1.ListUserResponse, error) {
	return nil, nil
}

func (r stubUserRepo) Get(context.Context, *identityv1.GetUserRequest) (*identityv1.User, error) {
	return r.user, nil
}

func (r stubUserRepo) Create(context.Context, *identityv1.CreateUserRequest) (*emptypb.Empty, error) {
	return nil, nil
}

func (r stubUserRepo) Update(context.Context, *identityv1.UpdateUserRequest) (*emptypb.Empty, error) {
	return nil, nil
}

func (r stubUserRepo) Delete(context.Context, *identityv1.DeleteUserRequest) (*emptypb.Empty, error) {
	return nil, nil
}

func (r stubUserRepo) UserExists(context.Context, *identityv1.UserExistsRequest) (*identityv1.UserExistsResponse, error) {
	return nil, nil
}

type stubUserRepoWithRoleReader struct {
	stubUserRepo
	roleIDs []uint32
}

func (r stubUserRepoWithRoleReader) ListRoleIDsByUserID(context.Context, uint32) ([]uint32, error) {
	return append([]uint32(nil), r.roleIDs...), nil
}

type stubAuthUserRepo struct {
	users map[uint32]*identityv1.User
}

func (r stubAuthUserRepo) List(context.Context, *paginationv1.PagingRequest) (*identityv1.ListUserResponse, error) {
	return nil, nil
}

func (r stubAuthUserRepo) Get(_ context.Context, req *identityv1.GetUserRequest) (*identityv1.User, error) {
	switch q := req.GetQueryBy().(type) {
	case *identityv1.GetUserRequest_Id:
		return r.users[q.Id], nil
	case *identityv1.GetUserRequest_Username:
		for _, item := range r.users {
			if item != nil && item.GetUsername() == q.Username {
				return item, nil
			}
		}
	}
	return nil, nil
}

func (r stubAuthUserRepo) Create(context.Context, *identityv1.CreateUserRequest) (*emptypb.Empty, error) {
	return nil, nil
}

func (r stubAuthUserRepo) Update(context.Context, *identityv1.UpdateUserRequest) (*emptypb.Empty, error) {
	return nil, nil
}

func (r stubAuthUserRepo) Delete(context.Context, *identityv1.DeleteUserRequest) (*emptypb.Empty, error) {
	return nil, nil
}

func (r stubAuthUserRepo) UserExists(context.Context, *identityv1.UserExistsRequest) (*identityv1.UserExistsResponse, error) {
	return nil, nil
}

type stubUserCredentialRepo struct {
	record         *repo.UserCredentialWithUser
	lastUpgradeID  uint32
	lastUpgradePwd string
}

type testHeader struct {
	header http.Header
}

func (h testHeader) Get(key string) string {
	return h.header.Get(key)
}

func (h testHeader) Set(key, value string) {
	h.header.Set(key, value)
}

func (h testHeader) Add(key, value string) {
	h.header.Add(key, value)
}

func (h testHeader) Keys() []string {
	keys := make([]string, 0, len(h.header))
	for key := range h.header {
		keys = append(keys, key)
	}
	return keys
}

func (h testHeader) Values(key string) []string {
	return h.header.Values(key)
}

type testHTTPTransport struct {
	req          *http.Request
	pathTemplate string
}

func (t *testHTTPTransport) Kind() transport.Kind { return transport.KindHTTP }
func (t *testHTTPTransport) Endpoint() string     { return "http://localhost:8000" }
func (t *testHTTPTransport) Operation() string    { return t.pathTemplate }
func (t *testHTTPTransport) RequestHeader() transport.Header {
	if t.req == nil {
		return testHeader{header: make(http.Header)}
	}
	return testHeader{header: t.req.Header}
}
func (t *testHTTPTransport) ReplyHeader() transport.Header {
	return testHeader{header: make(http.Header)}
}
func (t *testHTTPTransport) Request() *http.Request { return t.req }
func (t *testHTTPTransport) PathTemplate() string   { return t.pathTemplate }

func (r *stubUserCredentialRepo) List(context.Context, *paginationv1.PagingRequest) (*authenticationv1.ListUserCredentialResponse, error) {
	return nil, nil
}

func (r *stubUserCredentialRepo) Count(context.Context, *paginationv1.PagingRequest) (*authenticationv1.CountUserCredentialResponse, error) {
	return nil, nil
}

func (r *stubUserCredentialRepo) Get(context.Context, *authenticationv1.GetUserCredentialRequest) (*authenticationv1.UserCredential, error) {
	return nil, nil
}

func (r *stubUserCredentialRepo) Create(context.Context, *authenticationv1.CreateUserCredentialRequest) (*emptypb.Empty, error) {
	return nil, nil
}

func (r *stubUserCredentialRepo) Update(context.Context, *authenticationv1.UpdateUserCredentialRequest) (*emptypb.Empty, error) {
	return nil, nil
}

func (r *stubUserCredentialRepo) Delete(context.Context, *authenticationv1.DeleteUserCredentialRequest) (*emptypb.Empty, error) {
	return nil, nil
}

func (r *stubUserCredentialRepo) ResetCredential(context.Context, *authenticationv1.ResetCredentialRequest) (*emptypb.Empty, error) {
	return nil, nil
}

func (r *stubUserCredentialRepo) FindPasswordCredentialByIdentifier(context.Context, string) (*repo.UserCredentialWithUser, error) {
	return r.record, nil
}

func (r *stubUserCredentialRepo) UpgradePasswordCredential(_ context.Context, credentialID uint32, plain string) error {
	r.lastUpgradeID = credentialID
	r.lastUpgradePwd = plain
	return nil
}

type stubRolePermissionReader struct {
	menuIDs []uint32
}

func (r stubRolePermissionReader) ListPermissionIDsByRoleIDs(context.Context, []uint32) ([]uint32, error) {
	return nil, nil
}

func (r stubRolePermissionReader) GetRolesPermissionMenuIDs(context.Context, []uint32) ([]uint32, error) {
	return append([]uint32(nil), r.menuIDs...), nil
}

type recordingRolePermissionReader struct {
	menuIDs     []uint32
	lastRoleIDs []uint32
}

func (r *recordingRolePermissionReader) ListPermissionIDsByRoleIDs(_ context.Context, roleIDs []uint32) ([]uint32, error) {
	r.lastRoleIDs = append([]uint32(nil), roleIDs...)
	return nil, nil
}

func (r *recordingRolePermissionReader) GetRolesPermissionMenuIDs(_ context.Context, roleIDs []uint32) ([]uint32, error) {
	r.lastRoleIDs = append([]uint32(nil), roleIDs...)
	return append([]uint32(nil), r.menuIDs...), nil
}

type stubMenuNavigationStore struct {
	items []*resourcev1.MenuRouteItem
}

func (s stubMenuNavigationStore) ListNavigationRoutes(context.Context) ([]*resourcev1.MenuRouteItem, error) {
	return append([]*resourcev1.MenuRouteItem(nil), s.items...), nil
}

func (s stubMenuNavigationStore) ListNavigationRoutesByIDs(context.Context, []uint32) ([]*resourcev1.MenuRouteItem, error) {
	return append([]*resourcev1.MenuRouteItem(nil), s.items...), nil
}

var _ repo.UserRepo = stubUserRepo{}
var _ repo.UserRepo = stubUserRepoWithRoleReader{}
var _ repo.UserRoleIDReader = stubUserRepoWithRoleReader{}
var _ repo.UserRepo = stubAuthUserRepo{}
var _ repo.RolePermissionReader = stubRolePermissionReader{}
var _ repo.RolePermissionReader = (*recordingRolePermissionReader)(nil)
var _ menuNavigationStore = stubMenuNavigationStore{}
var _ repo.UserCredentialRepo = (*stubUserCredentialRepo)(nil)
var _ userCredentialFinder = (*stubUserCredentialRepo)(nil)

func TestManualAdminPortalServiceGetNavigationFallsBackWhenUserHasNoMenuPermissions(t *testing.T) {
	service := &manualAdminPortalService{
		menuStore: stubMenuNavigationStore{
			items: []*resourcev1.MenuRouteItem{
				{Name: ptr("SystemUser"), Path: ptr("/system/user")},
			},
		},
		roleRepo: stubRolePermissionReader{menuIDs: nil},
		userRepo: stubUserRepo{
			user: &identityv1.User{RoleIds: []uint32{1}},
		},
	}

	ctx := crudviewer.WithContext(context.Background(), testPortalViewer{userID: 1})
	resp, err := service.GetNavigation(ctx, &emptypb.Empty{})
	if err != nil {
		t.Fatalf("GetNavigation returned error: %v", err)
	}
	if resp == nil {
		t.Fatalf("GetNavigation returned nil response")
	}
	if len(resp.GetItems()) == 0 {
		t.Fatalf("expected default routes fallback")
	}
	if got := resp.GetItems()[0].GetPath(); got != "/dashboard" {
		t.Fatalf("expected fallback dashboard route, got %q", got)
	}
}

func TestManualAdminPortalServiceGetNavigationReturnsAuthorizedRoutesOnly(t *testing.T) {
	wantItems := []*resourcev1.MenuRouteItem{
		{Name: ptr("SystemUser"), Path: ptr("/system/user")},
	}
	service := &manualAdminPortalService{
		menuStore: stubMenuNavigationStore{items: wantItems},
		roleRepo:  stubRolePermissionReader{menuIDs: []uint32{101}},
		userRepo: stubUserRepo{
			user: &identityv1.User{RoleIds: []uint32{1}},
		},
	}

	ctx := crudviewer.WithContext(context.Background(), testPortalViewer{userID: 1})
	resp, err := service.GetNavigation(ctx, &emptypb.Empty{})
	if err != nil {
		t.Fatalf("GetNavigation returned error: %v", err)
	}
	if resp == nil {
		t.Fatalf("GetNavigation returned nil response")
	}
	if len(resp.GetItems()) != len(wantItems) {
		t.Fatalf("expected %d routes, got %d", len(wantItems), len(resp.GetItems()))
	}
	if got := resp.GetItems()[0].GetPath(); got != wantItems[0].GetPath() {
		t.Fatalf("expected route path %q, got %q", wantItems[0].GetPath(), got)
	}
}

func TestManualAdminPortalServiceGetNavigationUsesUserRoleReader(t *testing.T) {
	wantItems := []*resourcev1.MenuRouteItem{
		{Name: ptr("Analytics"), Path: ptr("/analytics")},
	}
	roleRepo := &recordingRolePermissionReader{
		menuIDs: []uint32{101},
	}
	service := &manualAdminPortalService{
		menuStore:      stubMenuNavigationStore{items: wantItems},
		roleRepo:       roleRepo,
		userRoleReader: stubUserRepoWithRoleReader{roleIDs: []uint32{7}},
		userRepo: stubUserRepo{
			user: &identityv1.User{RoleIds: nil},
		},
	}

	ctx := crudviewer.WithContext(context.Background(), testPortalViewer{userID: 1})
	resp, err := service.GetNavigation(ctx, &emptypb.Empty{})
	if err != nil {
		t.Fatalf("GetNavigation returned error: %v", err)
	}
	if got := roleRepo.lastRoleIDs; len(got) != 1 || got[0] != 7 {
		t.Fatalf("expected role ids from user role reader, got %+v", got)
	}
	if len(resp.GetItems()) != 1 || resp.GetItems()[0].GetPath() != "/analytics" {
		t.Fatalf("expected analytics route, got %+v", resp.GetItems())
	}
}

func TestDefaultNavigationRoutesStillProvidesFallbackShapeForInitialData(t *testing.T) {
	resp := defaultNavigationRoutes()
	if resp == nil {
		t.Fatalf("defaultNavigationRoutes returned nil")
	}
	if len(resp.GetItems()) == 0 {
		t.Fatalf("defaultNavigationRoutes returned no items")
	}
	if resp.GetItems()[0].GetPath() == "" {
		t.Fatalf("defaultNavigationRoutes returned item without path")
	}
}

func TestManualAuthenticationServiceLoginRejectsIncorrectPassword(t *testing.T) {
	captchaID, captchaCode := mustGenerateCaptcha(t)
	service := &manualAuthenticationService{
		userRepo: stubAuthUserRepo{
			users: map[uint32]*identityv1.User{
				1: {
					Id:       ptr(uint32(1)),
					Username: ptr("admin"),
					Roles:    []string{"SUPER_ADMIN"},
				},
			},
		},
		credentialFinder: &stubUserCredentialRepo{
			record: &repo.UserCredentialWithUser{
				Credential: &ent.UserCredential{
					ID:         9,
					Credential: ptr("correct"),
				},
				User: &ent.User{
					ID:       1,
					Username: ptr("admin"),
					Status:   ptr(user.StatusNormal),
				},
			},
		},
		auth: testAuthConfig(t),
	}

	_, err := service.Login(context.Background(), &authenticationv1.LoginRequest{
		GrantType: authenticationv1.GrantType_password,
		ClientId:  ptr(captchaID),
		Code:      ptr(captchaCode),
		Identifier: &authenticationv1.LoginRequest_Username{
			Username: "admin",
		},
		Password: ptr("wrong"),
	})
	if err == nil || !authenticationv1.IsIncorrectPassword(err) {
		t.Fatalf("expected incorrect password error, got %v", err)
	}
}

func TestManualAuthenticationServiceLoginUpgradesLegacyPlainPassword(t *testing.T) {
	captchaID, captchaCode := mustGenerateCaptcha(t)
	credentialRepo := &stubUserCredentialRepo{
		record: &repo.UserCredentialWithUser{
			Credential: &ent.UserCredential{
				ID:         11,
				Credential: ptr("123456"),
			},
			User: &ent.User{
				ID:       1,
				Username: ptr("admin"),
				Status:   ptr(user.StatusNormal),
			},
		},
	}
	service := &manualAuthenticationService{
		userRepo: stubAuthUserRepo{
			users: map[uint32]*identityv1.User{
				1: {
					Id:       ptr(uint32(1)),
					Username: ptr("admin"),
					Roles:    []string{"SUPER_ADMIN"},
					Status:   identityv1.User_NORMAL.Enum(),
				},
			},
		},
		credentialFinder: credentialRepo,
		auth:             testAuthConfig(t),
		tokenStore:       mustNewTestTokenStore(t),
	}

	resp, err := service.Login(context.Background(), &authenticationv1.LoginRequest{
		GrantType: authenticationv1.GrantType_password,
		ClientId:  ptr(captchaID),
		Code:      ptr(captchaCode),
		Identifier: &authenticationv1.LoginRequest_Username{
			Username: "admin",
		},
		Password: ptr("123456"),
	})
	if err != nil {
		t.Fatalf("expected login success, got %v", err)
	}
	if resp == nil || resp.GetAccessToken() == "" || resp.GetRefreshToken() == "" {
		t.Fatalf("expected signed tokens, got %+v", resp)
	}
	if credentialRepo.lastUpgradeID != 11 || credentialRepo.lastUpgradePwd != "123456" {
		t.Fatalf("expected legacy password upgrade, got id=%d pwd=%q", credentialRepo.lastUpgradeID, credentialRepo.lastUpgradePwd)
	}
}

func TestManualAuthenticationServiceRefreshTokenRotatesTokenPair(t *testing.T) {
	captchaID, captchaCode := mustGenerateCaptcha(t)
	service := newTestAuthenticationService(t)

	loginResp, err := service.Login(context.Background(), &authenticationv1.LoginRequest{
		GrantType: authenticationv1.GrantType_password,
		ClientId:  ptr(captchaID),
		Code:      ptr(captchaCode),
		Identifier: &authenticationv1.LoginRequest_Username{
			Username: "admin",
		},
		Password: ptr("123456"),
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	refreshToken := loginResp.GetRefreshToken()

	refreshResp, err := service.RefreshToken(context.Background(), &authenticationv1.LoginRequest{
		GrantType:    authenticationv1.GrantType_refresh_token,
		RefreshToken: &refreshToken,
	})
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	if refreshResp.GetAccessToken() == "" || refreshResp.GetRefreshToken() == "" {
		t.Fatalf("expected rotated tokens, got %+v", refreshResp)
	}
	if refreshResp.GetRefreshToken() == refreshToken {
		t.Fatalf("expected refresh token rotation")
	}

	_, err = service.RefreshToken(context.Background(), &authenticationv1.LoginRequest{
		GrantType:    authenticationv1.GrantType_refresh_token,
		RefreshToken: &refreshToken,
	})
	if err == nil || !authenticationv1.IsRefreshTokenNotFound(err) {
		t.Fatalf("expected old refresh token rejected, got %v", err)
	}
}

func TestAuthViewerMiddlewareRejectsRotatedAccessToken(t *testing.T) {
	captchaID, captchaCode := mustGenerateCaptcha(t)
	service := newTestAuthenticationService(t)

	loginResp, err := service.Login(context.Background(), &authenticationv1.LoginRequest{
		GrantType: authenticationv1.GrantType_password,
		ClientId:  ptr(captchaID),
		Code:      ptr(captchaCode),
		Identifier: &authenticationv1.LoginRequest_Username{
			Username: "admin",
		},
		Password: ptr("123456"),
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	oldAccessToken := loginResp.GetAccessToken()
	refreshToken := loginResp.GetRefreshToken()

	_, err = service.RefreshToken(context.Background(), &authenticationv1.LoginRequest{
		GrantType:    authenticationv1.GrantType_refresh_token,
		RefreshToken: &refreshToken,
	})
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}

	data := &testGeneratedData{
		appCtx:     testAppCtx(),
		userRepo:   service.userRepo,
		roleRepo:   testRoleRepo{},
		permRepo:   testPermissionRepo{},
		credential: nil,
		tokenStore: service.tokenStore,
	}
	handler := authViewerMiddleware(data)(func(ctx context.Context, req any) (any, error) {
		return &emptypb.Empty{}, nil
	})
	req, err := http.NewRequest(http.MethodGet, "http://localhost/admin/v1/me", nil)
	if err != nil {
		t.Fatalf("new request failed: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+oldAccessToken)
	ctx := transport.NewServerContext(context.Background(), &testHTTPTransport{
		req:          req,
		pathTemplate: "/admin/v1/me",
	})
	_, err = handler(ctx, nil)
	if err == nil || !authenticationv1.IsUnauthorized(err) {
		t.Fatalf("expected revoked access token rejected, got %v", err)
	}
}

func TestManualAuthenticationServiceLogoutRevokesCurrentTokenPair(t *testing.T) {
	captchaID, captchaCode := mustGenerateCaptcha(t)
	service := newTestAuthenticationService(t)

	loginResp, err := service.Login(context.Background(), &authenticationv1.LoginRequest{
		GrantType: authenticationv1.GrantType_password,
		ClientId:  ptr(captchaID),
		Code:      ptr(captchaCode),
		Identifier: &authenticationv1.LoginRequest_Username{
			Username: "admin",
		},
		Password: ptr("123456"),
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, "http://localhost/admin/v1/logout", nil)
	if err != nil {
		t.Fatalf("new request failed: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+loginResp.GetAccessToken())
	ctx := transport.NewServerContext(context.Background(), &testHTTPTransport{
		req:          req,
		pathTemplate: "/admin/v1/logout",
	})

	if _, err := service.Logout(ctx, &emptypb.Empty{}); err != nil {
		t.Fatalf("logout failed: %v", err)
	}
	refreshToken := loginResp.GetRefreshToken()
	_, err = service.RefreshToken(context.Background(), &authenticationv1.LoginRequest{
		GrantType:    authenticationv1.GrantType_refresh_token,
		RefreshToken: &refreshToken,
	})
	if err == nil || !authenticationv1.IsRefreshTokenNotFound(err) {
		t.Fatalf("expected logout revoked refresh token, got %v", err)
	}
}

func TestVerifyCaptchaRejectsInvalidCode(t *testing.T) {
	driver := newCaptchaDriver()
	gen := base64Captcha.NewCaptcha(driver, captchaStore)
	id, _, _, err := gen.Generate()
	if err != nil {
		t.Fatalf("generate captcha failed: %v", err)
	}
	if err := verifyCaptcha(id, "xxxx"); err == nil || !authenticationv1.IsUnauthorized(err) {
		t.Fatalf("expected unauthorized on invalid captcha, got %v", err)
	}
}

func TestVerifyCaptchaAcceptsCorrectCode(t *testing.T) {
	driver := newCaptchaDriver()
	gen := base64Captcha.NewCaptcha(driver, captchaStore)
	id, _, answer, err := gen.Generate()
	if err != nil {
		t.Fatalf("generate captcha failed: %v", err)
	}
	if err := verifyCaptcha(id, answer); err != nil {
		t.Fatalf("expected captcha pass, got %v", err)
	}
}

func mustGenerateCaptcha(t *testing.T) (string, string) {
	t.Helper()
	driver := newCaptchaDriver()
	gen := base64Captcha.NewCaptcha(driver, captchaStore)
	id, _, answer, err := gen.Generate()
	if err != nil {
		t.Fatalf("generate captcha failed: %v", err)
	}
	return id, answer
}

func TestParseAndValidateTokenRejectsForgedBearerToken(t *testing.T) {
	_, err := parseAndValidateToken("access.admin.fake", testAuthConfig(t), tokenCategoryAccess)
	if err == nil || !authenticationv1.IsUnauthorized(err) {
		t.Fatalf("expected unauthorized error, got %v", err)
	}
}

func TestIsProtectedServerRequestRequiresAdminAPIs(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://localhost/admin/v1/me", nil)
	if err != nil {
		t.Fatalf("new request failed: %v", err)
	}
	tr := &testHTTPTransport{
		req:          req,
		pathTemplate: "/admin/v1/me",
	}
	ctx := transport.NewServerContext(context.Background(), tr)
	if !isProtectedServerRequest(ctx) {
		t.Fatalf("expected /admin/v1/me to be protected")
	}
}

func TestIsProtectedServerRequestAllowsPublicLogout(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "http://localhost/admin/v1/logout", nil)
	if err != nil {
		t.Fatalf("new request failed: %v", err)
	}
	tr := &testHTTPTransport{
		req:          req,
		pathTemplate: "/admin/v1/logout",
	}
	ctx := transport.NewServerContext(context.Background(), tr)
	if isProtectedServerRequest(ctx) {
		t.Fatalf("expected /admin/v1/logout to be public")
	}
}

func testAuthConfig(t *testing.T) *authConfig {
	t.Helper()
	cfg, err := loadAuthConfig(testAppCtx())
	if err != nil {
		t.Fatalf("load auth config failed: %v", err)
	}
	return cfg
}

func testAppCtx() *app.AppCtx {
	return app.NewAppCtx(context.Background(), &conf.AppInfo{Name: "test"}, &conf.ServerConfig{
		Server: &conf.Server{
			Rest: &conf.Server_REST{},
		},
		Data: &conf.Data{},
		Authn: &conf.Authentication{
			Type: "jwt",
			Jwt: &conf.Authentication_Jwt{
				Method: "HS256",
				Key:    "test-secret",
			},
		},
	}, nil, nil)
}

func mustNewTestTokenStore(t *testing.T) *tokenStore {
	t.Helper()
	store, err := newStandaloneTokenStore(&conf.Data{})
	if err != nil {
		t.Fatalf("new token store failed: %v", err)
	}
	return store
}

func newTestAuthenticationService(t *testing.T) *manualAuthenticationService {
	t.Helper()
	credentialRepo := &stubUserCredentialRepo{
		record: &repo.UserCredentialWithUser{
			Credential: &ent.UserCredential{
				ID:         11,
				Credential: ptr("123456"),
			},
			User: &ent.User{
				ID:       1,
				Username: ptr("admin"),
				Status:   ptr(user.StatusNormal),
			},
		},
	}
	return &manualAuthenticationService{
		userRepo: stubAuthUserRepo{
			users: map[uint32]*identityv1.User{
				1: {
					Id:       ptr(uint32(1)),
					Username: ptr("admin"),
					Roles:    []string{"SUPER_ADMIN"},
					Status:   identityv1.User_NORMAL.Enum(),
				},
			},
		},
		credentialFinder: credentialRepo,
		auth:             testAuthConfig(t),
		tokenStore:       mustNewTestTokenStore(t),
	}
}

type testGeneratedData struct {
	appCtx     *app.AppCtx
	userRepo   repo.UserRepo
	roleRepo   repo.RoleRepo
	permRepo   repo.PermissionRepo
	credential repo.UserCredentialRepo
	tokenStore *tokenStore
}

func (d *testGeneratedData) GetAppCtx() *app.AppCtx                             { return d.appCtx }
func (d *testGeneratedData) UserRepoProvider() repo.UserRepo                     { return d.userRepo }
func (d *testGeneratedData) RoleRepoProvider() repo.RoleRepo                     { return d.roleRepo }
func (d *testGeneratedData) PermissionRepoProvider() repo.PermissionRepo         { return d.permRepo }
func (d *testGeneratedData) UserCredentialRepoProvider() repo.UserCredentialRepo { return d.credential }
func (d *testGeneratedData) TokenStoreProvider() *tokenStore                     { return d.tokenStore }

type testRoleRepo struct{}

func (testRoleRepo) List(context.Context, *paginationv1.PagingRequest) (*permissionv1.ListRoleResponse, error) {
	return nil, nil
}
func (testRoleRepo) Get(context.Context, *permissionv1.GetRoleRequest) (*permissionv1.Role, error) {
	return nil, nil
}
func (testRoleRepo) Create(context.Context, *permissionv1.CreateRoleRequest) (*emptypb.Empty, error) {
	return nil, nil
}
func (testRoleRepo) Update(context.Context, *permissionv1.UpdateRoleRequest) (*emptypb.Empty, error) {
	return nil, nil
}
func (testRoleRepo) Delete(context.Context, *permissionv1.DeleteRoleRequest) (*emptypb.Empty, error) {
	return nil, nil
}
func (testRoleRepo) ListPermissionIDsByRoleIDs(context.Context, []uint32) ([]uint32, error) {
	return nil, nil
}
func (testRoleRepo) GetRolesPermissionMenuIDs(context.Context, []uint32) ([]uint32, error) {
	return nil, nil
}

type testPermissionRepo struct{}

func (testPermissionRepo) List(context.Context, *paginationv1.PagingRequest) (*permissionv1.ListPermissionResponse, error) {
	return nil, nil
}
func (testPermissionRepo) Get(context.Context, *permissionv1.GetPermissionRequest) (*permissionv1.Permission, error) {
	return nil, nil
}
func (testPermissionRepo) Create(context.Context, *permissionv1.CreatePermissionRequest) (*emptypb.Empty, error) {
	return nil, nil
}
func (testPermissionRepo) Update(context.Context, *permissionv1.UpdatePermissionRequest) (*emptypb.Empty, error) {
	return nil, nil
}
func (testPermissionRepo) Delete(context.Context, *permissionv1.DeletePermissionRequest) (*emptypb.Empty, error) {
	return nil, nil
}
func (testPermissionRepo) GetPermissionCodesByIDs(context.Context, []uint32) ([]string, error) {
	return nil, nil
}

var _ = adminv1.ListRouteResponse{}
