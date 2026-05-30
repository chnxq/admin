package repo

import (
	"context"
	"io"
	"testing"

	identityv1 "admin/api/gen/identity/v1"
	permissionv1 "admin/api/gen/permission/v1"
	resourcev1 "admin/api/gen/resource/v1"
	"admin/internal/data/ent"
	_ "admin/internal/data/ent/runtime"

	entsql "entgo.io/ent/dialect/sql"
	entCrud "github.com/chnxq/x-crud/entgo"
	crudviewer "github.com/chnxq/x-crud/viewer"
	"github.com/chnxq/x-utils/mapper"
	xlog "github.com/chnxq/xkitmod/log"
	_ "github.com/mattn/go-sqlite3"
)

func newPlatformResourceEntClientForTest(t *testing.T, dbName string) (*entCrud.EntClient[*ent.Client], *ent.Client) {
	t.Helper()

	driver, err := entsql.Open("sqlite3", "file:"+dbName+"?mode=memory&cache=shared&_fk=1")
	if err != nil {
		t.Fatalf("open sqlite driver failed: %v", err)
	}
	client := ent.NewClient(ent.Driver(driver))
	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("create schema failed: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
		_ = driver.Close()
	})

	return entCrud.NewEntClient[*ent.Client](client, driver), client
}

func newPlatformResourceLoggerForTest() *xlog.Helper {
	return xlog.NewHelper(xlog.NewStdLogger(io.Discard))
}

func platformRepoCtx() context.Context {
	return crudviewer.WithContext(context.Background(), tenantScopeTestViewer{
		platform: true,
		tenant:   false,
		tenantID: 0,
	})
}

func tenantRepoCtx() context.Context {
	return crudviewer.WithContext(context.Background(), tenantScopeTestViewer{
		platform: true,
		tenant:   true,
		tenantID: 101,
	})
}

func newApiRepoForPlatformTest(entClient *entCrud.EntClient[*ent.Client]) *apiRepo {
	repo := &apiRepo{
		log:       newPlatformResourceLoggerForTest(),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[resourcev1.Api, ent.Api](),
	}
	repo.init()
	return repo
}

func newMenuRepoForPlatformTest(entClient *entCrud.EntClient[*ent.Client]) *menuRepo {
	repo := &menuRepo{
		log:       newPlatformResourceLoggerForTest(),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[resourcev1.Menu, ent.Menu](),
	}
	repo.init()
	return repo
}

func newTenantRepoForPlatformTest(entClient *entCrud.EntClient[*ent.Client]) *tenantRepo {
	repo := &tenantRepo{
		log:       newPlatformResourceLoggerForTest(),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[identityv1.Tenant, ent.Tenant](),
	}
	repo.init()
	return repo
}

func newPermissionGroupRepoForPlatformTest(entClient *entCrud.EntClient[*ent.Client]) *permissionGroupRepo {
	repo := &permissionGroupRepo{
		log:       newPlatformResourceLoggerForTest(),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[permissionv1.PermissionGroup, ent.PermissionGroup](),
	}
	repo.init()
	return repo
}

func TestApiRepoRejectsTenantMutationsAndSync(t *testing.T) {
	entClient, client := newPlatformResourceEntClientForTest(t, "platform-resource-api")
	repo := newApiRepoForPlatformTest(entClient)
	tenantCtx := tenantRepoCtx()
	platformCtx := platformRepoCtx()

	name := "audit"
	if _, err := repo.Create(tenantCtx, &resourcev1.CreateApiRequest{
		Data: &resourcev1.Api{Module: &name, Path: &name, Method: &name},
	}); !identityv1.IsForbidden(err) {
		t.Fatalf("expected tenant create forbidden, got %v", err)
	}
	if err := repo.SyncApisFromOpenAPI(tenantCtx); !identityv1.IsForbidden(err) {
		t.Fatalf("expected tenant sync forbidden, got %v", err)
	}

	path := "/admin/v1/test"
	method := "GET"
	module := "test"
	if _, err := repo.Create(platformCtx, &resourcev1.CreateApiRequest{
		Data: &resourcev1.Api{
			Path:   &path,
			Method: &method,
			Module: &module,
			Status: resourcev1.Api_ON.Enum(),
		},
	}); err != nil {
		t.Fatalf("platform create api failed: %v", err)
	}
	apiEntity, err := client.Api.Query().Only(platformCtx)
	if err != nil {
		t.Fatalf("load api failed: %v", err)
	}
	if _, err := repo.Update(tenantCtx, &resourcev1.UpdateApiRequest{
		Id:   apiEntity.ID,
		Data: &resourcev1.Api{Path: &path, Method: &method, Module: &module, Status: resourcev1.Api_ON.Enum()},
	}); !identityv1.IsForbidden(err) {
		t.Fatalf("expected tenant update forbidden, got %v", err)
	}
	if _, err := repo.Delete(tenantCtx, &resourcev1.DeleteApiRequest{
		QueryBy: &resourcev1.DeleteApiRequest_Id{Id: apiEntity.ID},
	}); !identityv1.IsForbidden(err) {
		t.Fatalf("expected tenant delete forbidden, got %v", err)
	}
}

func TestMenuRepoRejectsTenantMutations(t *testing.T) {
	entClient, client := newPlatformResourceEntClientForTest(t, "platform-resource-menu")
	repo := newMenuRepoForPlatformTest(entClient)
	tenantCtx := tenantRepoCtx()
	platformCtx := platformRepoCtx()

	name := "MenuA"
	path := "/menu-a"
	if _, err := repo.Create(tenantCtx, &resourcev1.CreateMenuRequest{
		Data: &resourcev1.Menu{Name: &name, Path: &path, Status: resourcev1.Menu_ON.Enum()},
	}); !identityv1.IsForbidden(err) {
		t.Fatalf("expected tenant create forbidden, got %v", err)
	}

	if _, err := repo.Create(platformCtx, &resourcev1.CreateMenuRequest{
		Data: &resourcev1.Menu{Name: &name, Path: &path, Status: resourcev1.Menu_ON.Enum()},
	}); err != nil {
		t.Fatalf("platform create menu failed: %v", err)
	}
	menuEntity, err := client.Menu.Query().Only(platformCtx)
	if err != nil {
		t.Fatalf("load menu failed: %v", err)
	}
	if _, err := repo.Update(tenantCtx, &resourcev1.UpdateMenuRequest{
		Id:   menuEntity.ID,
		Data: &resourcev1.Menu{Name: &name, Path: &path, Status: resourcev1.Menu_ON.Enum()},
	}); !identityv1.IsForbidden(err) {
		t.Fatalf("expected tenant update forbidden, got %v", err)
	}
	if _, err := repo.Delete(tenantCtx, &resourcev1.DeleteMenuRequest{
		QueryBy: &resourcev1.DeleteMenuRequest_Id{Id: menuEntity.ID},
	}); !identityv1.IsForbidden(err) {
		t.Fatalf("expected tenant delete forbidden, got %v", err)
	}
}

func TestTenantRepoRejectsTenantMutations(t *testing.T) {
	entClient, client := newPlatformResourceEntClientForTest(t, "platform-resource-tenant")
	repo := newTenantRepoForPlatformTest(entClient)
	tenantCtx := tenantRepoCtx()
	platformCtx := platformRepoCtx()

	name := "tenant-a"
	code := "tenant-a"
	if _, err := repo.Create(tenantCtx, &identityv1.CreateTenantRequest{
		Data: &identityv1.Tenant{Name: &name, Code: &code},
	}); !identityv1.IsForbidden(err) {
		t.Fatalf("expected tenant create forbidden, got %v", err)
	}

	if _, err := repo.Create(platformCtx, &identityv1.CreateTenantRequest{
		Data: &identityv1.Tenant{Name: &name, Code: &code},
	}); err != nil {
		t.Fatalf("platform create tenant failed: %v", err)
	}
	tenantEntity, err := client.Tenant.Query().Only(platformCtx)
	if err != nil {
		t.Fatalf("load tenant failed: %v", err)
	}
	if _, err := repo.Update(tenantCtx, &identityv1.UpdateTenantRequest{
		Id:   tenantEntity.ID,
		Data: &identityv1.Tenant{Name: &name, Code: &code},
	}); !identityv1.IsForbidden(err) {
		t.Fatalf("expected tenant update forbidden, got %v", err)
	}
	if _, err := repo.Delete(tenantCtx, &identityv1.DeleteTenantRequest{
		QueryBy: &identityv1.DeleteTenantRequest_Id{Id: tenantEntity.ID},
	}); !identityv1.IsForbidden(err) {
		t.Fatalf("expected tenant delete forbidden, got %v", err)
	}
}

func TestPermissionGroupRepoRejectsTenantMutations(t *testing.T) {
	entClient, client := newPlatformResourceEntClientForTest(t, "platform-resource-permission-group")
	repo := newPermissionGroupRepoForPlatformTest(entClient)
	tenantCtx := tenantRepoCtx()
	platformCtx := platformRepoCtx()

	name := "group-a"
	module := "system"
	if _, err := repo.Create(tenantCtx, &permissionv1.CreatePermissionGroupRequest{
		Data: &permissionv1.PermissionGroup{Name: &name, Module: &module, Status: permissionv1.PermissionGroup_ON.Enum()},
	}); !identityv1.IsForbidden(err) {
		t.Fatalf("expected tenant create forbidden, got %v", err)
	}

	if _, err := repo.Create(platformCtx, &permissionv1.CreatePermissionGroupRequest{
		Data: &permissionv1.PermissionGroup{Name: &name, Module: &module, Status: permissionv1.PermissionGroup_ON.Enum()},
	}); err != nil {
		t.Fatalf("platform create permission group failed: %v", err)
	}
	groupEntity, err := client.PermissionGroup.Query().Only(platformCtx)
	if err != nil {
		t.Fatalf("load permission group failed: %v", err)
	}
	if _, err := repo.Update(tenantCtx, &permissionv1.UpdatePermissionGroupRequest{
		Id:   groupEntity.ID,
		Data: &permissionv1.PermissionGroup{Name: &name, Module: &module, Status: permissionv1.PermissionGroup_ON.Enum()},
	}); !identityv1.IsForbidden(err) {
		t.Fatalf("expected tenant update forbidden, got %v", err)
	}
	if _, err := repo.Delete(tenantCtx, &permissionv1.DeletePermissionGroupRequest{
		QueryBy: &permissionv1.DeletePermissionGroupRequest_Id{Id: groupEntity.ID},
	}); !identityv1.IsForbidden(err) {
		t.Fatalf("expected tenant delete forbidden, got %v", err)
	}
}
