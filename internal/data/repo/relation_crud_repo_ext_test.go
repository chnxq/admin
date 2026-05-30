package repo

import (
	"context"
	"io"
	"reflect"
	"sort"
	"testing"

	identityv1 "admin/api/gen/identity/v1"
	permissionv1 "admin/api/gen/permission/v1"
	"admin/internal/data/ent"
	"admin/internal/data/ent/permissionapi"
	"admin/internal/data/ent/permissionauditlog"
	"admin/internal/data/ent/permissionmenu"
	"admin/internal/data/ent/rolepermission"
	_ "admin/internal/data/ent/runtime"
	"admin/internal/data/ent/usercredential"
	"admin/internal/data/ent/userorgunit"
	"admin/internal/data/ent/userposition"
	"admin/internal/data/ent/userrole"

	entsql "entgo.io/ent/dialect/sql"
	paginationv1 "github.com/chnxq/x-crud/api/gen/pagination/v1"
	entCrud "github.com/chnxq/x-crud/entgo"
	crudviewer "github.com/chnxq/x-crud/viewer"
	"github.com/chnxq/x-utils/mapper"
	xlog "github.com/chnxq/xkitmod/log"
	_ "github.com/mattn/go-sqlite3"
)

func newRelationCrudEntClientForTest(t *testing.T, dbName string) (*entCrud.EntClient[*ent.Client], *ent.Client) {
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

func newRelationCrudLoggerForTest() *xlog.Helper {
	return xlog.NewHelper(xlog.NewStdLogger(io.Discard))
}

func relationCrudContext() context.Context {
	return relationCrudPlatformContext()
}

func relationCrudPlatformContext() context.Context {
	return crudviewer.WithContext(context.Background(), tenantScopeTestViewer{
		platform: true,
		tenant:   false,
		tenantID: 0,
	})
}

func relationCrudTenantContext(tenantID uint64) context.Context {
	return crudviewer.WithContext(context.Background(), tenantScopeTestViewer{
		platform: true,
		tenant:   true,
		tenantID: tenantID,
	})
}

func boolPtr(v bool) *bool {
	return &v
}

func newUserRepoForRelationCrudTest(entClient *entCrud.EntClient[*ent.Client]) *userRepo {
	repo := &userRepo{
		log:       newRelationCrudLoggerForTest(),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[identityv1.User, ent.User](),
	}
	repo.init()
	return repo
}

func newRoleRepoForRelationCrudTest(entClient *entCrud.EntClient[*ent.Client]) *roleRepo {
	repo := &roleRepo{
		log:       newRelationCrudLoggerForTest(),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[permissionv1.Role, ent.Role](),
	}
	repo.init()
	return repo
}

func newPermissionRepoForRelationCrudTest(entClient *entCrud.EntClient[*ent.Client]) *permissionRepo {
	repo := &permissionRepo{
		log:       newRelationCrudLoggerForTest(),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[permissionv1.Permission, ent.Permission](),
	}
	repo.init()
	return repo
}

func sortedUint32(values []uint32) []uint32 {
	out := append([]uint32(nil), values...)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func assertUint32Set(t *testing.T, name string, got []uint32, want []uint32) {
	t.Helper()
	if !reflect.DeepEqual(sortedUint32(got), sortedUint32(want)) {
		t.Fatalf("%s mismatch: got %v, want %v", name, got, want)
	}
}

func TestUserRepoCreatePersistsAndHydratesRelations(t *testing.T) {
	entClient, client := newRelationCrudEntClientForTest(t, "relation-crud-user")
	repo := newUserRepoForRelationCrudTest(entClient)
	ctx := relationCrudContext()

	orgEntity, err := client.OrgUnit.Create().SetName("研发中心").SetCode("rd").Save(ctx)
	if err != nil {
		t.Fatalf("create org unit failed: %v", err)
	}
	positionEntity, err := client.Position.Create().SetName("架构师").SetCode("architect").SetOrgUnitID(orgEntity.ID).Save(ctx)
	if err != nil {
		t.Fatalf("create position failed: %v", err)
	}
	roleEntity, err := client.Role.Create().SetName("管理员").SetCode("admin").Save(ctx)
	if err != nil {
		t.Fatalf("create role failed: %v", err)
	}

	username := "relation-user"
	password := "P@ssw0rd-1"
	if _, err := repo.Create(ctx, &identityv1.CreateUserRequest{
		Data: &identityv1.User{
			Username:    &username,
			OrgUnitIds:  []uint32{orgEntity.ID},
			PositionIds: []uint32{positionEntity.ID},
			RoleIds:     []uint32{roleEntity.ID},
		},
		Password: &password,
	}); err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	userEntity, err := client.User.Query().Only(ctx)
	if err != nil {
		t.Fatalf("load created user failed: %v", err)
	}

	if count, err := client.UserOrgUnit.Query().Where(userorgunit.UserIDEQ(userEntity.ID), userorgunit.OrgUnitIDEQ(orgEntity.ID)).Count(ctx); err != nil || count != 1 {
		t.Fatalf("expected one user org relation, count=%d err=%v", count, err)
	}
	if count, err := client.UserPosition.Query().Where(userposition.UserIDEQ(userEntity.ID), userposition.PositionIDEQ(positionEntity.ID)).Count(ctx); err != nil || count != 1 {
		t.Fatalf("expected one user position relation, count=%d err=%v", count, err)
	}
	if count, err := client.UserRole.Query().Where(userrole.UserIDEQ(userEntity.ID), userrole.RoleIDEQ(roleEntity.ID)).Count(ctx); err != nil || count != 1 {
		t.Fatalf("expected one user role relation, count=%d err=%v", count, err)
	}
	credential, err := client.UserCredential.Query().
		Where(
			usercredential.UserIDEQ(userEntity.ID),
			usercredential.IdentifierEQ(username),
			usercredential.CredentialTypeEQ(usercredential.CredentialTypePasswordHash),
		).
		Only(ctx)
	if err != nil {
		t.Fatalf("load user credential failed: %v", err)
	}
	if credential.Credential == nil || *credential.Credential == password {
		t.Fatalf("expected persisted credential to be normalized, got %+v", credential.Credential)
	}
	matched, _, err := VerifyPasswordCredential(password, *credential.Credential)
	if err != nil || !matched {
		t.Fatalf("expected normalized password to verify, matched=%v err=%v", matched, err)
	}

	dto, err := repo.Get(ctx, &identityv1.GetUserRequest{
		QueryBy: &identityv1.GetUserRequest_Id{Id: userEntity.ID},
	})
	if err != nil {
		t.Fatalf("get user failed: %v", err)
	}
	assertUint32Set(t, "org_unit_ids", dto.GetOrgUnitIds(), []uint32{orgEntity.ID})
	assertUint32Set(t, "position_ids", dto.GetPositionIds(), []uint32{positionEntity.ID})
	assertUint32Set(t, "role_ids", dto.GetRoleIds(), []uint32{roleEntity.ID})
	if dto.GetOrgUnitName() != "研发中心" || dto.GetPositionName() != "架构师" {
		t.Fatalf("expected relation names to be hydrated, got org=%q position=%q", dto.GetOrgUnitName(), dto.GetPositionName())
	}
	if !reflect.DeepEqual(dto.GetRoles(), []string{"admin"}) || !reflect.DeepEqual(dto.GetRoleNames(), []string{"管理员"}) {
		t.Fatalf("expected role code/name hydration, got codes=%v names=%v", dto.GetRoles(), dto.GetRoleNames())
	}
}

func TestUserRepoDeleteCleansRelationsAndCredentials(t *testing.T) {
	entClient, client := newRelationCrudEntClientForTest(t, "relation-crud-user-delete")
	repo := newUserRepoForRelationCrudTest(entClient)
	ctx := relationCrudContext()

	orgEntity, err := client.OrgUnit.Create().SetName("研发中心").SetCode("rd").Save(ctx)
	if err != nil {
		t.Fatalf("create org unit failed: %v", err)
	}
	positionEntity, err := client.Position.Create().SetName("架构师").SetCode("architect").SetOrgUnitID(orgEntity.ID).Save(ctx)
	if err != nil {
		t.Fatalf("create position failed: %v", err)
	}
	roleEntity, err := client.Role.Create().SetName("管理员").SetCode("admin").Save(ctx)
	if err != nil {
		t.Fatalf("create role failed: %v", err)
	}

	username := "delete-user"
	password := "P@ssw0rd-1"
	if _, err := repo.Create(ctx, &identityv1.CreateUserRequest{
		Data: &identityv1.User{
			Username:    &username,
			OrgUnitIds:  []uint32{orgEntity.ID},
			PositionIds: []uint32{positionEntity.ID},
			RoleIds:     []uint32{roleEntity.ID},
		},
		Password: &password,
	}); err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	userEntity, err := client.User.Query().Only(ctx)
	if err != nil {
		t.Fatalf("load created user failed: %v", err)
	}
	if _, err := repo.Delete(ctx, &identityv1.DeleteUserRequest{
		QueryBy: &identityv1.DeleteUserRequest_Id{Id: userEntity.ID},
	}); err != nil {
		t.Fatalf("delete user failed: %v", err)
	}

	if count, err := client.User.Query().Count(ctx); err != nil || count != 0 {
		t.Fatalf("expected no users after delete, count=%d err=%v", count, err)
	}
	if count, err := client.UserOrgUnit.Query().Where(userorgunit.UserIDEQ(userEntity.ID)).Count(ctx); err != nil || count != 0 {
		t.Fatalf("expected user org relations to be deleted, count=%d err=%v", count, err)
	}
	if count, err := client.UserPosition.Query().Where(userposition.UserIDEQ(userEntity.ID)).Count(ctx); err != nil || count != 0 {
		t.Fatalf("expected user position relations to be deleted, count=%d err=%v", count, err)
	}
	if count, err := client.UserRole.Query().Where(userrole.UserIDEQ(userEntity.ID)).Count(ctx); err != nil || count != 0 {
		t.Fatalf("expected user role relations to be deleted, count=%d err=%v", count, err)
	}
	if count, err := client.UserCredential.Query().Where(usercredential.UserIDEQ(userEntity.ID)).Count(ctx); err != nil || count != 0 {
		t.Fatalf("expected user credentials to be deleted, count=%d err=%v", count, err)
	}
}

func TestUserRepoTenantViewerCannotUpdateOrDeleteOtherTenantUser(t *testing.T) {
	entClient, client := newRelationCrudEntClientForTest(t, "relation-crud-user-tenant-guard")
	repo := newUserRepoForRelationCrudTest(entClient)
	seedCtx := relationCrudContext()
	ctx := relationCrudTenantContext(101)

	otherTenant := uint32(202)
	userEntity, err := client.User.Create().
		SetUsername("other-tenant-user").
		SetTenantID(otherTenant).
		Save(seedCtx)
	if err != nil {
		t.Fatalf("create other tenant user failed: %v", err)
	}

	nickname := "updated"
	if _, err := repo.Update(ctx, &identityv1.UpdateUserRequest{
		Id: userEntity.ID,
		Data: &identityv1.User{
			Nickname: &nickname,
		},
	}); !identityv1.IsForbidden(err) {
		t.Fatalf("expected forbidden update, got %v", err)
	}

	if _, err := repo.Delete(ctx, &identityv1.DeleteUserRequest{
		QueryBy: &identityv1.DeleteUserRequest_Id{Id: userEntity.ID},
	}); !identityv1.IsForbidden(err) {
		t.Fatalf("expected forbidden delete, got %v", err)
	}
}

func TestUserRepoTenantScopedRelationsRejectCrossTenantAssignments(t *testing.T) {
	entClient, client := newRelationCrudEntClientForTest(t, "relation-crud-user-cross-tenant-relations")
	repo := newUserRepoForRelationCrudTest(entClient)
	seedCtx := relationCrudContext()
	ctx := relationCrudTenantContext(101)

	ownTenant := uint32(101)
	otherTenant := uint32(202)
	orgEntity, err := client.OrgUnit.Create().SetName("other-org").SetCode("other-org").SetTenantID(otherTenant).Save(seedCtx)
	if err != nil {
		t.Fatalf("create other tenant org unit failed: %v", err)
	}
	positionEntity, err := client.Position.Create().
		SetName("other-position").
		SetCode("other-position").
		SetTenantID(otherTenant).
		SetOrgUnitID(orgEntity.ID).
		Save(seedCtx)
	if err != nil {
		t.Fatalf("create other tenant position failed: %v", err)
	}
	globalRole, err := client.Role.Create().SetName("global-role").SetCode("global-role").SetTenantID(0).Save(seedCtx)
	if err != nil {
		t.Fatalf("create global role failed: %v", err)
	}
	otherRole, err := client.Role.Create().SetName("other-role").SetCode("other-role").SetTenantID(otherTenant).Save(seedCtx)
	if err != nil {
		t.Fatalf("create other tenant role failed: %v", err)
	}

	usernameA := "cross-org"
	if _, err := repo.Create(ctx, &identityv1.CreateUserRequest{
		Data: &identityv1.User{
			Username:   &usernameA,
			TenantId:   &ownTenant,
			OrgUnitIds: []uint32{orgEntity.ID},
		},
	}); !identityv1.IsForbidden(err) {
		t.Fatalf("expected forbidden org-unit assignment, got %v", err)
	}

	usernameB := "cross-position"
	if _, err := repo.Create(ctx, &identityv1.CreateUserRequest{
		Data: &identityv1.User{
			Username:    &usernameB,
			TenantId:    &ownTenant,
			PositionIds: []uint32{positionEntity.ID},
		},
	}); !identityv1.IsForbidden(err) {
		t.Fatalf("expected forbidden position assignment, got %v", err)
	}

	usernameC := "cross-role"
	if _, err := repo.Create(ctx, &identityv1.CreateUserRequest{
		Data: &identityv1.User{
			Username: &usernameC,
			TenantId: &ownTenant,
			RoleIds:  []uint32{globalRole.ID, otherRole.ID},
		},
	}); !identityv1.IsForbidden(err) {
		t.Fatalf("expected forbidden role assignment, got %v", err)
	}
}

func TestRoleRepoCreatePersistsPermissionsAndAuditLog(t *testing.T) {
	entClient, client := newRelationCrudEntClientForTest(t, "relation-crud-role")
	repo := newRoleRepoForRelationCrudTest(entClient)
	ctx := relationCrudContext()

	permA, err := client.Permission.Create().SetName("用户读取").SetCode("system:user:read").Save(ctx)
	if err != nil {
		t.Fatalf("create permission A failed: %v", err)
	}
	permB, err := client.Permission.Create().SetName("用户更新").SetCode("system:user:update").Save(ctx)
	if err != nil {
		t.Fatalf("create permission B failed: %v", err)
	}

	roleName := "关系测试角色"
	roleCode := "relation-role"
	if _, err := repo.Create(ctx, &permissionv1.CreateRoleRequest{
		Data: &permissionv1.Role{
			Name:        &roleName,
			Code:        &roleCode,
			Permissions: []uint32{permA.ID, permB.ID},
		},
	}); err != nil {
		t.Fatalf("create role failed: %v", err)
	}

	roleEntity, err := client.Role.Query().Only(ctx)
	if err != nil {
		t.Fatalf("load created role failed: %v", err)
	}
	if count, err := client.RolePermission.Query().Where(rolepermission.RoleIDEQ(roleEntity.ID)).Count(ctx); err != nil || count != 2 {
		t.Fatalf("expected two role permissions, count=%d err=%v", count, err)
	}
	auditRows, err := client.PermissionAuditLog.Query().Where(permissionauditlog.TargetTypeEQ("role")).All(ctx)
	if err != nil {
		t.Fatalf("load permission audit logs failed: %v", err)
	}
	if len(auditRows) != 1 {
		t.Fatalf("expected one role audit log, got %d", len(auditRows))
	}
	if auditRows[0].Action == nil || *auditRows[0].Action != permissionauditlog.ActionCreate {
		t.Fatalf("expected create audit action, got %+v", auditRows[0].Action)
	}

	dto, err := repo.Get(ctx, &permissionv1.GetRoleRequest{
		QueryBy: &permissionv1.GetRoleRequest_Id{Id: roleEntity.ID},
	})
	if err != nil {
		t.Fatalf("get role failed: %v", err)
	}
	assertUint32Set(t, "role.permissions", dto.GetPermissions(), []uint32{permA.ID, permB.ID})
}

func TestRoleRepoDeleteCleansRelationsAndWritesAuditLog(t *testing.T) {
	entClient, client := newRelationCrudEntClientForTest(t, "relation-crud-role-delete")
	repo := newRoleRepoForRelationCrudTest(entClient)
	ctx := relationCrudContext()

	permEntity, err := client.Permission.Create().SetName("用户读取").SetCode("system:user:read").Save(ctx)
	if err != nil {
		t.Fatalf("create permission failed: %v", err)
	}
	userEntity, err := client.User.Create().SetUsername("role-delete-user").Save(ctx)
	if err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	roleName := "待删除角色"
	roleCode := "delete-role"
	if _, err := repo.Create(ctx, &permissionv1.CreateRoleRequest{
		Data: &permissionv1.Role{
			Name:        &roleName,
			Code:        &roleCode,
			Permissions: []uint32{permEntity.ID},
		},
	}); err != nil {
		t.Fatalf("create role failed: %v", err)
	}
	roleEntity, err := client.Role.Query().Only(ctx)
	if err != nil {
		t.Fatalf("load created role failed: %v", err)
	}
	if _, err := client.UserRole.Create().SetUserID(userEntity.ID).SetRoleID(roleEntity.ID).Save(ctx); err != nil {
		t.Fatalf("create user role failed: %v", err)
	}

	if _, err := repo.Delete(ctx, &permissionv1.DeleteRoleRequest{
		QueryBy: &permissionv1.DeleteRoleRequest_Id{Id: roleEntity.ID},
	}); err != nil {
		t.Fatalf("delete role failed: %v", err)
	}

	if count, err := client.Role.Query().Count(ctx); err != nil || count != 0 {
		t.Fatalf("expected no roles after delete, count=%d err=%v", count, err)
	}
	if count, err := client.RolePermission.Query().Where(rolepermission.RoleIDEQ(roleEntity.ID)).Count(ctx); err != nil || count != 0 {
		t.Fatalf("expected role permissions to be deleted, count=%d err=%v", count, err)
	}
	if count, err := client.UserRole.Query().Where(userrole.RoleIDEQ(roleEntity.ID)).Count(ctx); err != nil || count != 0 {
		t.Fatalf("expected user roles to be deleted, count=%d err=%v", count, err)
	}
	auditRows, err := client.PermissionAuditLog.Query().Where(permissionauditlog.TargetTypeEQ("role")).All(ctx)
	if err != nil {
		t.Fatalf("load permission audit logs failed: %v", err)
	}
	if len(auditRows) != 2 {
		t.Fatalf("expected create and delete role audit logs, got %d", len(auditRows))
	}
	if auditRows[1].Action == nil || *auditRows[1].Action != permissionauditlog.ActionDelete {
		t.Fatalf("expected delete audit action, got %+v", auditRows[1].Action)
	}
}

func TestRoleRepoTenantViewerSeesGlobalAndOwnRoleOnly(t *testing.T) {
	entClient, client := newRelationCrudEntClientForTest(t, "relation-crud-role-tenant-list")
	repo := newRoleRepoForRelationCrudTest(entClient)
	seedCtx := relationCrudContext()
	ctx := relationCrudTenantContext(101)

	globalRole, err := client.Role.Create().SetName("global").SetCode("global").SetTenantID(0).Save(seedCtx)
	if err != nil {
		t.Fatalf("create global role failed: %v", err)
	}
	ownTenant := uint32(101)
	if _, err := client.Role.Create().SetName("tenant-101").SetCode("tenant-101").SetTenantID(ownTenant).Save(seedCtx); err != nil {
		t.Fatalf("create own tenant role failed: %v", err)
	}
	otherTenant := uint32(202)
	otherRole, err := client.Role.Create().SetName("tenant-202").SetCode("tenant-202").SetTenantID(otherTenant).Save(seedCtx)
	if err != nil {
		t.Fatalf("create other tenant role failed: %v", err)
	}

	resp, err := repo.List(ctx, &paginationv1.PagingRequest{NoPaging: boolPtr(true)})
	if err != nil {
		t.Fatalf("list role failed: %v", err)
	}
	if len(resp.GetItems()) != 2 {
		t.Fatalf("expected 2 visible roles, got %d", len(resp.GetItems()))
	}
	if _, err := repo.Get(ctx, &permissionv1.GetRoleRequest{
		QueryBy: &permissionv1.GetRoleRequest_Id{Id: globalRole.ID},
	}); err != nil {
		t.Fatalf("get global role failed: %v", err)
	}
	if _, err := repo.Get(ctx, &permissionv1.GetRoleRequest{
		QueryBy: &permissionv1.GetRoleRequest_Id{Id: otherRole.ID},
	}); !permissionv1.IsForbidden(err) {
		t.Fatalf("expected forbidden when reading other tenant role, got %v", err)
	}
}

func TestRoleRepoTenantViewerCannotMutateGlobalRole(t *testing.T) {
	entClient, client := newRelationCrudEntClientForTest(t, "relation-crud-role-tenant-mutate")
	repo := newRoleRepoForRelationCrudTest(entClient)
	seedCtx := relationCrudContext()
	ctx := relationCrudTenantContext(101)

	roleEntity, err := client.Role.Create().SetName("global").SetCode("global").SetTenantID(0).Save(seedCtx)
	if err != nil {
		t.Fatalf("create global role failed: %v", err)
	}
	newName := "global-updated"
	if _, err := repo.Update(ctx, &permissionv1.UpdateRoleRequest{
		Id: roleEntity.ID,
		Data: &permissionv1.Role{
			Name: &newName,
			Code: stringPtr("global"),
			Type: permissionv1.Role_SYSTEM.Enum(),
		},
	}); !permissionv1.IsForbidden(err) {
		t.Fatalf("expected forbidden update, got %v", err)
	}
	if _, err := repo.Delete(ctx, &permissionv1.DeleteRoleRequest{
		QueryBy: &permissionv1.DeleteRoleRequest_Id{Id: roleEntity.ID},
	}); !permissionv1.IsForbidden(err) {
		t.Fatalf("expected forbidden delete, got %v", err)
	}
}

func TestRoleRepoCreateRejectsMissingPermissionIDs(t *testing.T) {
	entClient, _ := newRelationCrudEntClientForTest(t, "relation-crud-role-missing-permissions")
	repo := newRoleRepoForRelationCrudTest(entClient)
	ctx := relationCrudPlatformContext()

	name := "role-with-missing-permission"
	code := "role:missing:permission"
	if _, err := repo.Create(ctx, &permissionv1.CreateRoleRequest{
		Data: &permissionv1.Role{
			Name:        &name,
			Code:        &code,
			Permissions: []uint32{99999},
		},
	}); err == nil {
		t.Fatalf("expected create role to reject missing permission ids")
	}
}

func TestPermissionRepoTenantViewerCannotMutatePermission(t *testing.T) {
	entClient, client := newRelationCrudEntClientForTest(t, "relation-crud-permission-tenant-mutate")
	repo := newPermissionRepoForRelationCrudTest(entClient)
	seedCtx := relationCrudContext()
	ctx := relationCrudTenantContext(101)

	name := "tenant-create"
	code := "tenant:create"
	if _, err := repo.Create(ctx, &permissionv1.CreatePermissionRequest{
		Data: &permissionv1.Permission{
			Name: &name,
			Code: &code,
		},
	}); !permissionv1.IsForbidden(err) {
		t.Fatalf("expected forbidden create, got %v", err)
	}

	permissionEntity, err := client.Permission.Create().SetName("global").SetCode("global:view").Save(seedCtx)
	if err != nil {
		t.Fatalf("create permission failed: %v", err)
	}

	updatedName := "global-updated"
	if _, err := repo.Update(ctx, &permissionv1.UpdatePermissionRequest{
		Id: permissionEntity.ID,
		Data: &permissionv1.Permission{
			Name: &updatedName,
			Code: stringPtr("global:view"),
		},
	}); !permissionv1.IsForbidden(err) {
		t.Fatalf("expected forbidden update, got %v", err)
	}

	if _, err := repo.Delete(ctx, &permissionv1.DeletePermissionRequest{
		QueryBy: &permissionv1.DeletePermissionRequest_Id{Id: permissionEntity.ID},
	}); !permissionv1.IsForbidden(err) {
		t.Fatalf("expected forbidden delete, got %v", err)
	}
}

func TestPermissionRepoCreateRejectsMissingMenuOrAPIIDs(t *testing.T) {
	entClient, _ := newRelationCrudEntClientForTest(t, "relation-crud-permission-missing-relations")
	repo := newPermissionRepoForRelationCrudTest(entClient)
	ctx := relationCrudPlatformContext()

	name := "permission-with-missing-relations"
	code := "permission:missing:relations"
	if _, err := repo.Create(ctx, &permissionv1.CreatePermissionRequest{
		Data: &permissionv1.Permission{
			Name:    &name,
			Code:    &code,
			MenuIds: []uint32{99991},
		},
	}); err == nil {
		t.Fatalf("expected create permission to reject missing menu ids")
	}

	if _, err := repo.Create(ctx, &permissionv1.CreatePermissionRequest{
		Data: &permissionv1.Permission{
			Name:   &name,
			Code:   stringPtr("permission:missing:api"),
			ApiIds: []uint32{99992},
		},
	}); err == nil {
		t.Fatalf("expected create permission to reject missing api ids")
	}
}

func TestPermissionRepoCreatePersistsMenuApiRelationsAndAuditLog(t *testing.T) {
	entClient, client := newRelationCrudEntClientForTest(t, "relation-crud-permission")
	repo := newPermissionRepoForRelationCrudTest(entClient)
	ctx := relationCrudPlatformContext()

	menuEntity, err := client.Menu.Create().SetName("AuditLog").SetPath("/audit/logs").Save(ctx)
	if err != nil {
		t.Fatalf("create menu failed: %v", err)
	}
	apiEntity, err := client.Api.Create().SetModule("audit").SetPath("/admin/v1/audit/logs").SetMethod("GET").Save(ctx)
	if err != nil {
		t.Fatalf("create api failed: %v", err)
	}

	permissionName := "审计日志查看"
	permissionCode := "audit:log:view"
	if _, err := repo.Create(ctx, &permissionv1.CreatePermissionRequest{
		Data: &permissionv1.Permission{
			Name:    &permissionName,
			Code:    &permissionCode,
			MenuIds: []uint32{menuEntity.ID},
			ApiIds:  []uint32{apiEntity.ID},
		},
	}); err != nil {
		t.Fatalf("create permission failed: %v", err)
	}

	permissionEntity, err := client.Permission.Query().Only(ctx)
	if err != nil {
		t.Fatalf("load created permission failed: %v", err)
	}
	if count, err := client.PermissionMenu.Query().Where(permissionmenu.PermissionIDEQ(permissionEntity.ID), permissionmenu.MenuIDEQ(menuEntity.ID)).Count(ctx); err != nil || count != 1 {
		t.Fatalf("expected one permission menu relation, count=%d err=%v", count, err)
	}
	if count, err := client.PermissionApi.Query().Where(permissionapi.PermissionIDEQ(permissionEntity.ID), permissionapi.APIIDEQ(apiEntity.ID)).Count(ctx); err != nil || count != 1 {
		t.Fatalf("expected one permission api relation, count=%d err=%v", count, err)
	}
	auditRows, err := client.PermissionAuditLog.Query().Where(permissionauditlog.TargetTypeEQ("permission")).All(ctx)
	if err != nil {
		t.Fatalf("load permission audit logs failed: %v", err)
	}
	if len(auditRows) != 1 {
		t.Fatalf("expected one permission audit log, got %d", len(auditRows))
	}
	if auditRows[0].Action == nil || *auditRows[0].Action != permissionauditlog.ActionCreate {
		t.Fatalf("expected create audit action, got %+v", auditRows[0].Action)
	}

	dto, err := repo.Get(ctx, &permissionv1.GetPermissionRequest{
		QueryBy: &permissionv1.GetPermissionRequest_Id{Id: permissionEntity.ID},
	})
	if err != nil {
		t.Fatalf("get permission failed: %v", err)
	}
	assertUint32Set(t, "permission.menu_ids", dto.GetMenuIds(), []uint32{menuEntity.ID})
	assertUint32Set(t, "permission.api_ids", dto.GetApiIds(), []uint32{apiEntity.ID})
}

func TestPermissionRepoDeleteCleansRelationsAndWritesAuditLog(t *testing.T) {
	entClient, client := newRelationCrudEntClientForTest(t, "relation-crud-permission-delete")
	repo := newPermissionRepoForRelationCrudTest(entClient)
	ctx := relationCrudPlatformContext()

	menuEntity, err := client.Menu.Create().SetName("AuditLog").SetPath("/audit/logs").Save(ctx)
	if err != nil {
		t.Fatalf("create menu failed: %v", err)
	}
	apiEntity, err := client.Api.Create().SetModule("audit").SetPath("/admin/v1/audit/logs").SetMethod("GET").Save(ctx)
	if err != nil {
		t.Fatalf("create api failed: %v", err)
	}
	roleEntity, err := client.Role.Create().SetName("审计角色").SetCode("audit-role").Save(ctx)
	if err != nil {
		t.Fatalf("create role failed: %v", err)
	}

	permissionName := "审计日志查看"
	permissionCode := "audit:log:view"
	if _, err := repo.Create(ctx, &permissionv1.CreatePermissionRequest{
		Data: &permissionv1.Permission{
			Name:    &permissionName,
			Code:    &permissionCode,
			MenuIds: []uint32{menuEntity.ID},
			ApiIds:  []uint32{apiEntity.ID},
		},
	}); err != nil {
		t.Fatalf("create permission failed: %v", err)
	}
	permissionEntity, err := client.Permission.Query().Only(ctx)
	if err != nil {
		t.Fatalf("load created permission failed: %v", err)
	}
	if _, err := client.RolePermission.Create().SetRoleID(roleEntity.ID).SetPermissionID(permissionEntity.ID).Save(ctx); err != nil {
		t.Fatalf("create role permission failed: %v", err)
	}

	if _, err := repo.Delete(ctx, &permissionv1.DeletePermissionRequest{
		QueryBy: &permissionv1.DeletePermissionRequest_Id{Id: permissionEntity.ID},
	}); err != nil {
		t.Fatalf("delete permission failed: %v", err)
	}

	if count, err := client.Permission.Query().Count(ctx); err != nil || count != 0 {
		t.Fatalf("expected no permissions after delete, count=%d err=%v", count, err)
	}
	if count, err := client.PermissionMenu.Query().Where(permissionmenu.PermissionIDEQ(permissionEntity.ID)).Count(ctx); err != nil || count != 0 {
		t.Fatalf("expected permission menus to be deleted, count=%d err=%v", count, err)
	}
	if count, err := client.PermissionApi.Query().Where(permissionapi.PermissionIDEQ(permissionEntity.ID)).Count(ctx); err != nil || count != 0 {
		t.Fatalf("expected permission apis to be deleted, count=%d err=%v", count, err)
	}
	if count, err := client.RolePermission.Query().Where(rolepermission.PermissionIDEQ(permissionEntity.ID)).Count(ctx); err != nil || count != 0 {
		t.Fatalf("expected role permissions to be deleted, count=%d err=%v", count, err)
	}
	auditRows, err := client.PermissionAuditLog.Query().Where(permissionauditlog.TargetTypeEQ("permission")).All(ctx)
	if err != nil {
		t.Fatalf("load permission audit logs failed: %v", err)
	}
	if len(auditRows) != 2 {
		t.Fatalf("expected create and delete permission audit logs, got %d", len(auditRows))
	}
	if auditRows[1].Action == nil || *auditRows[1].Action != permissionauditlog.ActionDelete {
		t.Fatalf("expected delete audit action, got %+v", auditRows[1].Action)
	}
}

func TestPermissionRepoDeleteByCodeCleansRelations(t *testing.T) {
	entClient, client := newRelationCrudEntClientForTest(t, "relation-crud-permission-delete-code")
	repo := newPermissionRepoForRelationCrudTest(entClient)
	ctx := relationCrudPlatformContext()

	menuEntity, err := client.Menu.Create().SetName("AuditLog").SetPath("/audit/logs").Save(ctx)
	if err != nil {
		t.Fatalf("create menu failed: %v", err)
	}
	apiEntity, err := client.Api.Create().SetModule("audit").SetPath("/admin/v1/audit/logs").SetMethod("GET").Save(ctx)
	if err != nil {
		t.Fatalf("create api failed: %v", err)
	}
	roleEntity, err := client.Role.Create().SetName("审计角色").SetCode("audit-role").Save(ctx)
	if err != nil {
		t.Fatalf("create role failed: %v", err)
	}

	permissionName := "审计日志查看"
	permissionCode := "audit:log:view"
	if _, err := repo.Create(ctx, &permissionv1.CreatePermissionRequest{
		Data: &permissionv1.Permission{
			Name:    &permissionName,
			Code:    &permissionCode,
			MenuIds: []uint32{menuEntity.ID},
			ApiIds:  []uint32{apiEntity.ID},
		},
	}); err != nil {
		t.Fatalf("create permission failed: %v", err)
	}
	permissionEntity, err := client.Permission.Query().Only(ctx)
	if err != nil {
		t.Fatalf("load created permission failed: %v", err)
	}
	if _, err := client.RolePermission.Create().SetRoleID(roleEntity.ID).SetPermissionID(permissionEntity.ID).Save(ctx); err != nil {
		t.Fatalf("create role permission failed: %v", err)
	}

	if _, err := repo.Delete(ctx, &permissionv1.DeletePermissionRequest{
		QueryBy: &permissionv1.DeletePermissionRequest_Code{Code: permissionCode},
	}); err != nil {
		t.Fatalf("delete permission by code failed: %v", err)
	}

	if count, err := client.Permission.Query().Count(ctx); err != nil || count != 0 {
		t.Fatalf("expected no permissions after delete by code, count=%d err=%v", count, err)
	}
	if count, err := client.PermissionMenu.Query().Where(permissionmenu.PermissionIDEQ(permissionEntity.ID)).Count(ctx); err != nil || count != 0 {
		t.Fatalf("expected permission menus to be deleted, count=%d err=%v", count, err)
	}
	if count, err := client.PermissionApi.Query().Where(permissionapi.PermissionIDEQ(permissionEntity.ID)).Count(ctx); err != nil || count != 0 {
		t.Fatalf("expected permission apis to be deleted, count=%d err=%v", count, err)
	}
	if count, err := client.RolePermission.Query().Where(rolepermission.PermissionIDEQ(permissionEntity.ID)).Count(ctx); err != nil || count != 0 {
		t.Fatalf("expected role permissions to be deleted, count=%d err=%v", count, err)
	}
}

func TestPermissionRepoUpdateSkipsAuditWhenSnapshotUnchanged(t *testing.T) {
	entClient, client := newRelationCrudEntClientForTest(t, "relation-crud-permission-update-noaudit")
	repo := newPermissionRepoForRelationCrudTest(entClient)
	ctx := relationCrudPlatformContext()

	menuEntity, err := client.Menu.Create().SetName("AuditLog").SetPath("/audit/logs").Save(ctx)
	if err != nil {
		t.Fatalf("create menu failed: %v", err)
	}
	apiEntity, err := client.Api.Create().SetModule("audit").SetPath("/admin/v1/audit/logs").SetMethod("GET").Save(ctx)
	if err != nil {
		t.Fatalf("create api failed: %v", err)
	}

	permissionName := "审计日志查看"
	permissionCode := "audit:log:view"
	if _, err := repo.Create(ctx, &permissionv1.CreatePermissionRequest{
		Data: &permissionv1.Permission{
			Name:    &permissionName,
			Code:    &permissionCode,
			MenuIds: []uint32{menuEntity.ID},
			ApiIds:  []uint32{apiEntity.ID},
		},
	}); err != nil {
		t.Fatalf("create permission failed: %v", err)
	}

	permissionEntity, err := client.Permission.Query().Only(ctx)
	if err != nil {
		t.Fatalf("load created permission failed: %v", err)
	}
	current, err := repo.Get(ctx, &permissionv1.GetPermissionRequest{
		QueryBy: &permissionv1.GetPermissionRequest_Id{Id: permissionEntity.ID},
	})
	if err != nil {
		t.Fatalf("get current permission failed: %v", err)
	}

	if _, err := repo.Update(ctx, &permissionv1.UpdatePermissionRequest{
		Id: permissionEntity.ID,
		Data: &permissionv1.Permission{
			Name:        current.Name,
			Code:        current.Code,
			GroupId:     current.GroupId,
			MenuIds:     append([]uint32(nil), current.GetMenuIds()...),
			ApiIds:      append([]uint32(nil), current.GetApiIds()...),
			Description: current.Description,
			Status:      current.Status,
		},
	}); err != nil {
		t.Fatalf("update permission failed: %v", err)
	}

	auditRows, err := client.PermissionAuditLog.Query().Where(permissionauditlog.TargetTypeEQ("permission")).All(ctx)
	if err != nil {
		t.Fatalf("load permission audit logs failed: %v", err)
	}
	if len(auditRows) != 1 {
		t.Fatalf("expected unchanged update to skip audit log, got %d logs", len(auditRows))
	}
}

func TestPermissionRepoAuditDisabledContextSkipsAuditWrites(t *testing.T) {
	entClient, client := newRelationCrudEntClientForTest(t, "relation-crud-permission-audit-disabled")
	repo := newPermissionRepoForRelationCrudTest(entClient)
	ctx := WithPermissionAuditDisabled(relationCrudPlatformContext())

	permissionName := "审计日志查看"
	permissionCode := "audit:log:view"
	if _, err := repo.Create(ctx, &permissionv1.CreatePermissionRequest{
		Data: &permissionv1.Permission{
			Name:   &permissionName,
			Code:   &permissionCode,
			Status: permissionv1.Permission_ON.Enum(),
		},
	}); err != nil {
		t.Fatalf("create permission failed: %v", err)
	}

	permissionEntity, err := client.Permission.Query().Only(ctx)
	if err != nil {
		t.Fatalf("load created permission failed: %v", err)
	}

	if _, err := repo.Update(ctx, &permissionv1.UpdatePermissionRequest{
		Id: permissionEntity.ID,
		Data: &permissionv1.Permission{
			Name:   stringPtr("审计日志查看2"),
			Code:   &permissionCode,
			Status: permissionv1.Permission_ON.Enum(),
		},
	}); err != nil {
		t.Fatalf("update permission failed: %v", err)
	}

	if _, err := repo.Delete(ctx, &permissionv1.DeletePermissionRequest{
		QueryBy: &permissionv1.DeletePermissionRequest_Id{Id: permissionEntity.ID},
	}); err != nil {
		t.Fatalf("delete permission failed: %v", err)
	}

	auditRows, err := client.PermissionAuditLog.Query().Where(permissionauditlog.TargetTypeEQ("permission")).All(ctx)
	if err != nil {
		t.Fatalf("load permission audit logs failed: %v", err)
	}
	if len(auditRows) != 0 {
		t.Fatalf("expected disabled audit context to skip audit logs, got %d", len(auditRows))
	}
}
