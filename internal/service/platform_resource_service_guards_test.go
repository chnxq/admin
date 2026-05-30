package service

import (
	"context"
	"testing"

	identityv1 "admin/api/gen/identity/v1"
	permissionv1 "admin/api/gen/permission/v1"
	resourcev1 "admin/api/gen/resource/v1"

	paginationv1 "github.com/chnxq/x-crud/api/gen/pagination/v1"
	crudviewer "github.com/chnxq/x-crud/viewer"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type platformResourceTestViewer struct {
	platform bool
	tenant   bool
	tenantID uint64
}

func (v platformResourceTestViewer) UserID() uint64                    { return 1 }
func (v platformResourceTestViewer) TenantID() uint64                  { return v.tenantID }
func (v platformResourceTestViewer) OrgUnitID() uint64                 { return 0 }
func (v platformResourceTestViewer) Permissions() []string             { return []string{"*"} }
func (v platformResourceTestViewer) Roles() []string                   { return []string{"system"} }
func (v platformResourceTestViewer) DataScope() []crudviewer.DataScope { return nil }
func (v platformResourceTestViewer) TraceID() string                   { return "" }
func (v platformResourceTestViewer) HasPermission(string, string) bool { return true }
func (v platformResourceTestViewer) IsPlatformContext() bool           { return v.platform }
func (v platformResourceTestViewer) IsTenantContext() bool             { return v.tenant }
func (v platformResourceTestViewer) IsSystemContext() bool             { return v.platform && !v.tenant }
func (v platformResourceTestViewer) ShouldAudit() bool                 { return false }

func platformOnlyCtx() context.Context {
	return crudviewer.WithContext(context.Background(), platformResourceTestViewer{
		platform: true,
		tenant:   false,
		tenantID: 0,
	})
}

func tenantCtx() context.Context {
	return crudviewer.WithContext(context.Background(), platformResourceTestViewer{
		platform: true,
		tenant:   true,
		tenantID: 101,
	})
}

type fakeApiRepo struct {
	createCalls int
	updateCalls int
	deleteCalls int
	syncCalls   int
}

func (r *fakeApiRepo) List(context.Context, *paginationv1.PagingRequest) (*resourcev1.ListApiResponse, error) {
	return &resourcev1.ListApiResponse{}, nil
}
func (r *fakeApiRepo) Get(context.Context, *resourcev1.GetApiRequest) (*resourcev1.Api, error) {
	return &resourcev1.Api{}, nil
}
func (r *fakeApiRepo) Create(context.Context, *resourcev1.CreateApiRequest) (*emptypb.Empty, error) {
	r.createCalls++
	return &emptypb.Empty{}, nil
}
func (r *fakeApiRepo) Update(context.Context, *resourcev1.UpdateApiRequest) (*emptypb.Empty, error) {
	r.updateCalls++
	return &emptypb.Empty{}, nil
}
func (r *fakeApiRepo) Delete(context.Context, *resourcev1.DeleteApiRequest) (*emptypb.Empty, error) {
	r.deleteCalls++
	return &emptypb.Empty{}, nil
}
func (r *fakeApiRepo) SyncApisFromOpenAPI(context.Context) error {
	r.syncCalls++
	return nil
}
func (r *fakeApiRepo) GetOpenAPIRouteData(context.Context) (*resourcev1.ListApiResponse, error) {
	return &resourcev1.ListApiResponse{}, nil
}

type fakeMenuRepo struct {
	createCalls int
	updateCalls int
	deleteCalls int
}

func (r *fakeMenuRepo) List(context.Context, *paginationv1.PagingRequest) (*resourcev1.ListMenuResponse, error) {
	return &resourcev1.ListMenuResponse{}, nil
}
func (r *fakeMenuRepo) Get(context.Context, *resourcev1.GetMenuRequest) (*resourcev1.Menu, error) {
	return &resourcev1.Menu{}, nil
}
func (r *fakeMenuRepo) Create(context.Context, *resourcev1.CreateMenuRequest) (*emptypb.Empty, error) {
	r.createCalls++
	return &emptypb.Empty{}, nil
}
func (r *fakeMenuRepo) Update(context.Context, *resourcev1.UpdateMenuRequest) (*emptypb.Empty, error) {
	r.updateCalls++
	return &emptypb.Empty{}, nil
}
func (r *fakeMenuRepo) Delete(context.Context, *resourcev1.DeleteMenuRequest) (*emptypb.Empty, error) {
	r.deleteCalls++
	return &emptypb.Empty{}, nil
}

type fakeTenantRepo struct {
	createCalls int
	updateCalls int
	deleteCalls int
}

func (r *fakeTenantRepo) List(context.Context, *paginationv1.PagingRequest) (*identityv1.ListTenantResponse, error) {
	return &identityv1.ListTenantResponse{}, nil
}
func (r *fakeTenantRepo) Get(context.Context, *identityv1.GetTenantRequest) (*identityv1.Tenant, error) {
	return &identityv1.Tenant{}, nil
}
func (r *fakeTenantRepo) Create(context.Context, *identityv1.CreateTenantRequest) (*emptypb.Empty, error) {
	r.createCalls++
	return &emptypb.Empty{}, nil
}
func (r *fakeTenantRepo) Update(context.Context, *identityv1.UpdateTenantRequest) (*emptypb.Empty, error) {
	r.updateCalls++
	return &emptypb.Empty{}, nil
}
func (r *fakeTenantRepo) Delete(context.Context, *identityv1.DeleteTenantRequest) (*emptypb.Empty, error) {
	r.deleteCalls++
	return &emptypb.Empty{}, nil
}

type fakePermissionGroupRepo struct {
	createCalls int
	updateCalls int
	deleteCalls int
}

func (r *fakePermissionGroupRepo) List(context.Context, *paginationv1.PagingRequest) (*permissionv1.ListPermissionGroupResponse, error) {
	return &permissionv1.ListPermissionGroupResponse{}, nil
}
func (r *fakePermissionGroupRepo) Get(context.Context, *permissionv1.GetPermissionGroupRequest) (*permissionv1.PermissionGroup, error) {
	return &permissionv1.PermissionGroup{}, nil
}
func (r *fakePermissionGroupRepo) Create(context.Context, *permissionv1.CreatePermissionGroupRequest) (*emptypb.Empty, error) {
	r.createCalls++
	return &emptypb.Empty{}, nil
}
func (r *fakePermissionGroupRepo) Update(context.Context, *permissionv1.UpdatePermissionGroupRequest) (*emptypb.Empty, error) {
	r.updateCalls++
	return &emptypb.Empty{}, nil
}
func (r *fakePermissionGroupRepo) Delete(context.Context, *permissionv1.DeletePermissionGroupRequest) (*emptypb.Empty, error) {
	r.deleteCalls++
	return &emptypb.Empty{}, nil
}

func TestApiServicePlatformGuard(t *testing.T) {
	repo := &fakeApiRepo{}
	svc := &ApiService{apiRepo: repo}

	if _, err := svc.Create(tenantCtx(), &resourcev1.CreateApiRequest{Data: &resourcev1.Api{}}); !identityv1.IsForbidden(err) {
		t.Fatalf("expected tenant create forbidden, got %v", err)
	}
	if repo.createCalls != 0 {
		t.Fatalf("expected tenant create to stop before repo")
	}
	if _, err := svc.Create(platformOnlyCtx(), &resourcev1.CreateApiRequest{Data: &resourcev1.Api{}}); err != nil {
		t.Fatalf("expected platform create to pass, got %v", err)
	}
	if repo.createCalls != 1 {
		t.Fatalf("expected platform create to call repo once, got %d", repo.createCalls)
	}

	if _, err := svc.SyncApis(tenantCtx(), &emptypb.Empty{}); !identityv1.IsForbidden(err) {
		t.Fatalf("expected tenant sync forbidden, got %v", err)
	}
	if repo.syncCalls != 0 {
		t.Fatalf("expected tenant sync to stop before repo")
	}
	if _, err := svc.SyncApis(platformOnlyCtx(), &emptypb.Empty{}); err != nil {
		t.Fatalf("expected platform sync to pass, got %v", err)
	}
	if repo.syncCalls != 1 {
		t.Fatalf("expected platform sync to call repo once, got %d", repo.syncCalls)
	}
}

func TestMenuServicePlatformGuard(t *testing.T) {
	repo := &fakeMenuRepo{}
	svc := &MenuService{menuRepo: repo}

	if _, err := svc.Update(tenantCtx(), &resourcev1.UpdateMenuRequest{Data: &resourcev1.Menu{}}); !identityv1.IsForbidden(err) {
		t.Fatalf("expected tenant update forbidden, got %v", err)
	}
	if repo.updateCalls != 0 {
		t.Fatalf("expected tenant update to stop before repo")
	}
	if _, err := svc.Update(platformOnlyCtx(), &resourcev1.UpdateMenuRequest{Data: &resourcev1.Menu{}}); err != nil {
		t.Fatalf("expected platform update to pass, got %v", err)
	}
	if repo.updateCalls != 1 {
		t.Fatalf("expected platform update to call repo once, got %d", repo.updateCalls)
	}
}

func TestTenantServicePlatformGuard(t *testing.T) {
	repo := &fakeTenantRepo{}
	svc := &TenantService{tenantRepo: repo}

	if _, err := svc.Delete(tenantCtx(), &identityv1.DeleteTenantRequest{}); !identityv1.IsForbidden(err) {
		t.Fatalf("expected tenant delete forbidden, got %v", err)
	}
	if repo.deleteCalls != 0 {
		t.Fatalf("expected tenant delete to stop before repo")
	}
	if _, err := svc.Delete(platformOnlyCtx(), &identityv1.DeleteTenantRequest{}); err != nil {
		t.Fatalf("expected platform delete to pass, got %v", err)
	}
	if repo.deleteCalls != 1 {
		t.Fatalf("expected platform delete to call repo once, got %d", repo.deleteCalls)
	}
}

func TestPermissionGroupServicePlatformGuard(t *testing.T) {
	repo := &fakePermissionGroupRepo{}
	svc := &PermissionGroupService{permissionGroupRepo: repo}

	if _, err := svc.Create(tenantCtx(), &permissionv1.CreatePermissionGroupRequest{Data: &permissionv1.PermissionGroup{}}); !identityv1.IsForbidden(err) {
		t.Fatalf("expected tenant create forbidden, got %v", err)
	}
	if repo.createCalls != 0 {
		t.Fatalf("expected tenant create to stop before repo")
	}
	if _, err := svc.Create(platformOnlyCtx(), &permissionv1.CreatePermissionGroupRequest{Data: &permissionv1.PermissionGroup{}}); err != nil {
		t.Fatalf("expected platform create to pass, got %v", err)
	}
	if repo.createCalls != 1 {
		t.Fatalf("expected platform create to call repo once, got %d", repo.createCalls)
	}
}
