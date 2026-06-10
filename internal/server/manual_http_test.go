package server

import (
	"context"
	"net/http"
	"testing"
	"time"

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

type recordingRegisterUserRepo struct {
	createdReq  *identityv1.CreateUserRequest
	createdUser *identityv1.User
}

func (r *recordingRegisterUserRepo) List(context.Context, *paginationv1.PagingRequest) (*identityv1.ListUserResponse, error) {
	return nil, nil
}

func (r *recordingRegisterUserRepo) Get(_ context.Context, req *identityv1.GetUserRequest) (*identityv1.User, error) {
	if r.createdUser == nil || req == nil {
		return nil, nil
	}
	switch q := req.GetQueryBy().(type) {
	case *identityv1.GetUserRequest_Id:
		if r.createdUser.GetId() == q.Id {
			return r.createdUser, nil
		}
	case *identityv1.GetUserRequest_Username:
		if r.createdUser.GetUsername() == q.Username {
			return r.createdUser, nil
		}
	}
	return nil, nil
}

func (r *recordingRegisterUserRepo) Create(_ context.Context, req *identityv1.CreateUserRequest) (*emptypb.Empty, error) {
	r.createdReq = req
	username := ""
	if req != nil && req.GetData() != nil {
		username = req.GetData().GetUsername()
	}
	r.createdUser = &identityv1.User{
		Id:       ptr(uint32(99)),
		Username: &username,
		TenantId: req.GetData().TenantId,
		RoleIds:  append([]uint32(nil), req.GetData().GetRoleIds()...),
		Status:   identityv1.User_NORMAL.Enum(),
	}
	return &emptypb.Empty{}, nil
}

func (r *recordingRegisterUserRepo) Update(context.Context, *identityv1.UpdateUserRequest) (*emptypb.Empty, error) {
	return nil, nil
}

func (r *recordingRegisterUserRepo) Delete(context.Context, *identityv1.DeleteUserRequest) (*emptypb.Empty, error) {
	return nil, nil
}

func (r *recordingRegisterUserRepo) UserExists(context.Context, *identityv1.UserExistsRequest) (*identityv1.UserExistsResponse, error) {
	return &identityv1.UserExistsResponse{Exist: false}, nil
}

type recordingProfileUserRepo struct {
	user       *identityv1.User
	updatedReq *identityv1.UpdateUserRequest
}

func (r *recordingProfileUserRepo) List(context.Context, *paginationv1.PagingRequest) (*identityv1.ListUserResponse, error) {
	return nil, nil
}

func (r *recordingProfileUserRepo) Get(context.Context, *identityv1.GetUserRequest) (*identityv1.User, error) {
	return r.user, nil
}

func (r *recordingProfileUserRepo) Create(context.Context, *identityv1.CreateUserRequest) (*emptypb.Empty, error) {
	return nil, nil
}

func (r *recordingProfileUserRepo) Update(_ context.Context, req *identityv1.UpdateUserRequest) (*emptypb.Empty, error) {
	r.updatedReq = req
	return &emptypb.Empty{}, nil
}

func (r *recordingProfileUserRepo) Delete(context.Context, *identityv1.DeleteUserRequest) (*emptypb.Empty, error) {
	return nil, nil
}

func (r *recordingProfileUserRepo) UserExists(context.Context, *identityv1.UserExistsRequest) (*identityv1.UserExistsResponse, error) {
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

type recordingRegisterRoleRepo struct {
	role        *permissionv1.Role
	lastListReq *paginationv1.PagingRequest
}

func (r *recordingRegisterRoleRepo) List(_ context.Context, req *paginationv1.PagingRequest) (*permissionv1.ListRoleResponse, error) {
	r.lastListReq = req
	if r.role == nil {
		return &permissionv1.ListRoleResponse{}, nil
	}
	return &permissionv1.ListRoleResponse{
		Items: []*permissionv1.Role{r.role},
	}, nil
}

func (r *recordingRegisterRoleRepo) Get(context.Context, *permissionv1.GetRoleRequest) (*permissionv1.Role, error) {
	return nil, nil
}

func (r *recordingRegisterRoleRepo) Create(context.Context, *permissionv1.CreateRoleRequest) (*emptypb.Empty, error) {
	return nil, nil
}

func (r *recordingRegisterRoleRepo) Update(context.Context, *permissionv1.UpdateRoleRequest) (*emptypb.Empty, error) {
	return nil, nil
}

func (r *recordingRegisterRoleRepo) Delete(context.Context, *permissionv1.DeleteRoleRequest) (*emptypb.Empty, error) {
	return nil, nil
}

type recordingRegisterOrgUnitRepo struct {
	item        *identityv1.OrgUnit
	lastListReq *paginationv1.PagingRequest
}

func (r *recordingRegisterOrgUnitRepo) List(_ context.Context, req *paginationv1.PagingRequest) (*identityv1.ListOrgUnitResponse, error) {
	r.lastListReq = req
	if r.item == nil {
		return &identityv1.ListOrgUnitResponse{}, nil
	}
	return &identityv1.ListOrgUnitResponse{Items: []*identityv1.OrgUnit{r.item}}, nil
}

func (r *recordingRegisterOrgUnitRepo) Get(context.Context, *identityv1.GetOrgUnitRequest) (*identityv1.OrgUnit, error) {
	return nil, nil
}

func (r *recordingRegisterOrgUnitRepo) Create(context.Context, *identityv1.CreateOrgUnitRequest) (*emptypb.Empty, error) {
	return nil, nil
}

func (r *recordingRegisterOrgUnitRepo) Update(context.Context, *identityv1.UpdateOrgUnitRequest) (*emptypb.Empty, error) {
	return nil, nil
}

func (r *recordingRegisterOrgUnitRepo) Delete(context.Context, *identityv1.DeleteOrgUnitRequest) (*emptypb.Empty, error) {
	return nil, nil
}

type recordingRegisterPositionRepo struct {
	item        *identityv1.Position
	lastListReq *paginationv1.PagingRequest
}

func (r *recordingRegisterPositionRepo) List(_ context.Context, req *paginationv1.PagingRequest) (*identityv1.ListPositionResponse, error) {
	r.lastListReq = req
	if r.item == nil {
		return &identityv1.ListPositionResponse{}, nil
	}
	return &identityv1.ListPositionResponse{Items: []*identityv1.Position{r.item}}, nil
}

func (r *recordingRegisterPositionRepo) Get(context.Context, *identityv1.GetPositionRequest) (*identityv1.Position, error) {
	return nil, nil
}

func (r *recordingRegisterPositionRepo) Create(context.Context, *identityv1.CreatePositionRequest) (*emptypb.Empty, error) {
	return nil, nil
}

func (r *recordingRegisterPositionRepo) Update(context.Context, *identityv1.UpdatePositionRequest) (*emptypb.Empty, error) {
	return nil, nil
}

func (r *recordingRegisterPositionRepo) Delete(context.Context, *identityv1.DeletePositionRequest) (*emptypb.Empty, error) {
	return nil, nil
}

type stubTenantRepo struct {
	tenantByCode map[string]*identityv1.Tenant
}

func (r stubTenantRepo) List(context.Context, *paginationv1.PagingRequest) (*identityv1.ListTenantResponse, error) {
	items := make([]*identityv1.Tenant, 0, len(r.tenantByCode))
	for _, item := range r.tenantByCode {
		if item == nil {
			continue
		}
		items = append(items, item)
	}
	return &identityv1.ListTenantResponse{
		Items: items,
		Total: uint64(len(items)),
	}, nil
}

func (r stubTenantRepo) Get(_ context.Context, req *identityv1.GetTenantRequest) (*identityv1.Tenant, error) {
	if req == nil {
		return nil, nil
	}
	if q, ok := req.GetQueryBy().(*identityv1.GetTenantRequest_Id); ok {
		for _, item := range r.tenantByCode {
			if item != nil && item.GetId() == q.Id {
				return item, nil
			}
		}
		return nil, nil
	}
	if q, ok := req.GetQueryBy().(*identityv1.GetTenantRequest_Code); ok {
		return r.tenantByCode[q.Code], nil
	}
	return nil, nil
}

func (r stubTenantRepo) Create(context.Context, *identityv1.CreateTenantRequest) (*emptypb.Empty, error) {
	return nil, nil
}

func (r stubTenantRepo) Update(context.Context, *identityv1.UpdateTenantRequest) (*emptypb.Empty, error) {
	return nil, nil
}

func (r stubTenantRepo) Delete(context.Context, *identityv1.DeleteTenantRequest) (*emptypb.Empty, error) {
	return nil, nil
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
var _ repo.UserRepo = (*recordingRegisterUserRepo)(nil)
var _ repo.UserRepo = (*recordingProfileUserRepo)(nil)
var _ repo.RolePermissionReader = stubRolePermissionReader{}
var _ repo.RolePermissionReader = (*recordingRolePermissionReader)(nil)
var _ repo.RoleRepo = (*recordingRegisterRoleRepo)(nil)
var _ repo.OrgUnitRepo = (*recordingRegisterOrgUnitRepo)(nil)
var _ repo.PositionRepo = (*recordingRegisterPositionRepo)(nil)
var _ repo.TenantRepo = stubTenantRepo{}
var _ menuNavigationStore = stubMenuNavigationStore{}
var _ repo.UserCredentialRepo = (*stubUserCredentialRepo)(nil)
var _ userCredentialFinder = (*stubUserCredentialRepo)(nil)

func TestManualAdminPortalServiceGetNavigationReturnsEmptyWhenUserHasNoMenuPermissions(t *testing.T) {
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
	if len(resp.GetItems()) != 0 {
		t.Fatalf("expected no routes for roleless menu permission set, got %+v", resp.GetItems())
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

func TestApplyRegistrationDefaultsFromConf(t *testing.T) {
	defaults := registrationDefaults{
		DefaultTenantID:     1,
		DefaultTenantCode:   "default",
		DefaultRoleCode:     "GUEST",
		DefaultOrgUnitCode:  "UNDEFINED",
		DefaultPositionCode: "UNDEFINED",
	}
	got := applyRegistrationDefaultsFromConf(defaults, &conf.Authentication_Registration{
		DefaultTenantId:     9,
		DefaultTenantCode:   "tenant9",
		DefaultRoleCode:     "VISITOR",
		DefaultOrgUnitCode:  "ORG9",
		DefaultPositionCode: "POS9",
	})

	if got.DefaultTenantID != 9 {
		t.Fatalf("expected default tenant id 9, got %d", got.DefaultTenantID)
	}
	if got.DefaultTenantCode != "tenant9" {
		t.Fatalf("expected default tenant code tenant9, got %q", got.DefaultTenantCode)
	}
	if got.DefaultRoleCode != "VISITOR" {
		t.Fatalf("expected default role code VISITOR, got %q", got.DefaultRoleCode)
	}
	if got.DefaultOrgUnitCode != "ORG9" {
		t.Fatalf("expected default org unit code ORG9, got %q", got.DefaultOrgUnitCode)
	}
	if got.DefaultPositionCode != "POS9" {
		t.Fatalf("expected default position code POS9, got %q", got.DefaultPositionCode)
	}
}

func TestManualAuthenticationServiceRegisterUserAssignsDefaultUserRole(t *testing.T) {
	captchaID, captchaCode := mustGenerateCaptcha(t)
	userRepo := &recordingRegisterUserRepo{}
	roleRepo := &recordingRegisterRoleRepo{
		role: &permissionv1.Role{
			Id:       ptr(uint32(3)),
			Code:     ptr("GUEST"),
			TenantId: ptr(uint32(1)),
		},
	}
	orgUnitRepo := &recordingRegisterOrgUnitRepo{
		item: &identityv1.OrgUnit{
			Id:       ptr(uint32(11)),
			Code:     ptr("UNDEFINED"),
			TenantId: ptr(uint32(1)),
		},
	}
	positionRepo := &recordingRegisterPositionRepo{
		item: &identityv1.Position{
			Id:       ptr(uint32(12)),
			Code:     ptr("UNDEFINED"),
			TenantId: ptr(uint32(1)),
		},
	}
	service := &manualAuthenticationService{
		userRepo:     userRepo,
		tenantRepo:   stubTenantRepo{tenantByCode: map[string]*identityv1.Tenant{"default": {Id: ptr(uint32(1)), Code: ptr("default")}}},
		roleRepo:     roleRepo,
		orgUnitRepo:  orgUnitRepo,
		positionRepo: positionRepo,
		auth:         testAuthConfig(t),
		tokenStore:   mustNewTestTokenStore(t),
		log:          testAppCtx().NewLoggerHelper("authentication/test"),
	}
	service.registration = &registrationDefaultsResolver{
		tenantRepo:   service.tenantRepo,
		roleRepo:     service.roleRepo,
		orgUnitRepo:  service.orgUnitRepo,
		positionRepo: service.positionRepo,
		defaults: registrationDefaults{
			DefaultTenantID:     1,
			DefaultTenantCode:   "default",
			DefaultRoleCode:     "GUEST",
			DefaultOrgUnitCode:  "UNDEFINED",
			DefaultPositionCode: "UNDEFINED",
		},
	}

	resp, err := service.RegisterUser(context.Background(), &authenticationv1.RegisterUserRequest{
		ClientType: authenticationv1.ClientType_admin.Enum(),
		RegisterBy: &authenticationv1.RegisterUserRequest_ByUsername{
			ByUsername: &authenticationv1.RegisterByUsernameRequest{
				Username:    "xq",
				Password:    "123456A!",
				CaptchaId:   &captchaID,
				CaptchaCode: &captchaCode,
			},
		},
	})
	if err != nil {
		t.Fatalf("RegisterUser returned error: %v", err)
	}
	if resp == nil || resp.GetUserId() != 99 {
		t.Fatalf("expected registered user id 99, got %+v", resp)
	}
	if userRepo.createdReq == nil || userRepo.createdReq.GetData() == nil {
		t.Fatalf("expected create request to be captured")
	}
	if got := userRepo.createdReq.GetData().GetTenantId(); got != 1 {
		t.Fatalf("expected default tenant id 1, got %d", got)
	}
	if got := userRepo.createdReq.GetData().GetRoleIds(); len(got) != 1 || got[0] != 3 {
		t.Fatalf("expected GUEST role id [3], got %+v", got)
	}
	if got := userRepo.createdReq.GetData().GetOrgUnitIds(); len(got) != 1 || got[0] != 11 {
		t.Fatalf("expected default org unit id [11], got %+v", got)
	}
	if got := userRepo.createdReq.GetData().GetPositionIds(); len(got) != 1 || got[0] != 12 {
		t.Fatalf("expected default position id [12], got %+v", got)
	}
	if roleRepo.lastListReq == nil || roleRepo.lastListReq.GetFilterExpr() == nil {
		t.Fatalf("expected role lookup request captured")
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
	if err != nil {
		t.Fatalf("expected rotated access token accepted during grace period, got %v", err)
	}

	time.Sleep(accessTokenRefreshGraceTTL + 20*time.Millisecond)
	_, err = handler(ctx, nil)
	if err == nil || !authenticationv1.IsUnauthorized(err) {
		t.Fatalf("expected rotated access token rejected after grace period, got %v", err)
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

func TestIsProtectedServerRequestAllowsPublicRegister(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "http://localhost/admin/v1/register", nil)
	if err != nil {
		t.Fatalf("new request failed: %v", err)
	}
	tr := &testHTTPTransport{
		req:          req,
		pathTemplate: "/admin/v1/register",
	}
	ctx := transport.NewServerContext(context.Background(), tr)
	if isProtectedServerRequest(ctx) {
		t.Fatalf("expected /admin/v1/register to be public")
	}
}

func TestIsProtectedServerRequestAllowsPublicSocialAuthStart(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "http://localhost/admin/v1/social-auth:start", nil)
	if err != nil {
		t.Fatalf("new request failed: %v", err)
	}
	tr := &testHTTPTransport{
		req:          req,
		pathTemplate: "/admin/v1/social-auth:start",
	}
	ctx := transport.NewServerContext(context.Background(), tr)
	if isProtectedServerRequest(ctx) {
		t.Fatalf("expected /admin/v1/social-auth:start to be public")
	}
}

func TestIsProtectedServerRequestAllowsPublicAuthSessionPollByPathTemplate(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://localhost/admin/v1/auth-sessions/session-1/poll", nil)
	if err != nil {
		t.Fatalf("new request failed: %v", err)
	}
	tr := &testHTTPTransport{
		req:          req,
		pathTemplate: "/admin/v1/auth-sessions/{session_id}/poll",
	}
	ctx := transport.NewServerContext(context.Background(), tr)
	if isProtectedServerRequest(ctx) {
		t.Fatalf("expected auth session poll path template to be public")
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

func (d *testGeneratedData) GetAppCtx() *app.AppCtx                              { return d.appCtx }
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
