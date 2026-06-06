package bootstrap

import (
	"context"
	"fmt"
	"strings"
	"time"

	paginationv1 "github.com/chnxq/x-crud/api/gen/pagination/v1"
	entCrud "github.com/chnxq/x-crud/entgo"
	crudviewer "github.com/chnxq/x-crud/viewer"
	"github.com/chnxq/xkitpkg/app"

	"admin/internal/data/ent"
	"admin/internal/data/ent/membership"
	"admin/internal/data/ent/membershiprole"
	"admin/internal/data/ent/orgunit"
	"admin/internal/data/ent/permission"
	"admin/internal/data/ent/permissiongroup"
	"admin/internal/data/ent/position"
	"admin/internal/data/ent/role"
	"admin/internal/data/ent/rolepermission"
	enttask "admin/internal/data/ent/task"
	enttaskgroup "admin/internal/data/ent/taskgroup"
	"admin/internal/data/ent/tenant"
	"admin/internal/data/ent/user"
	"admin/internal/data/ent/usercredential"
	"admin/internal/data/ent/userorgunit"
	"admin/internal/data/ent/userposition"
	"admin/internal/data/ent/userrole"
	"admin/internal/data/repo"
	"admin/internal/service"
)

const (
	defaultTenantCode            = "default"
	platformTenantID             = uint32(0)
	platformTenantName           = "XAdmin平台"
	adminUsername                = "admin"
	platformUsername             = "platform_admin"
	normalUsername               = "user"
	defaultPassword              = "123456"
	permissionGroupModuleFeature = "permission:view:feature"
	permissionGroupModuleExport  = "permission:view:service:export"
)

type defaultDataSeed struct {
	appCtx    *app.AppCtx
	entClient *entCrud.EntClient[*ent.Client]
	now       time.Time
}

type seedViewerContext struct{}

type seedRoleSpec struct {
	code          string
	name          string
	description   string
	roleType      role.Type
	isProtected   bool
	sortOrder     uint32
	permissionIDs []uint32
}

type seedUserSpec struct {
	username      string
	nickname      string
	realname      string
	email         string
	mobile        string
	telephone     string
	description   string
	tenantID      uint32
	orgUnitID     uint32
	positionID    uint32
	roleID        uint32
	primaryRoleID uint32
	password      string
}

func ensureDefaultData(appCtx *app.AppCtx, entClient *entCrud.EntClient[*ent.Client]) error {
	if appCtx == nil || entClient == nil || entClient.Client() == nil {
		return nil
	}

	seed := &defaultDataSeed{
		appCtx:    appCtx,
		entClient: entClient,
		now:       time.Now(),
	}
	return seed.run()
}

func (s *defaultDataSeed) run() error {
	ctx := crudviewer.WithContext(s.appCtx.AppContext(), seedViewerContext{})

	if err := s.syncResources(ctx); err != nil {
		return err
	}

	if err := s.ensureTaskSeeds(ctx); err != nil {
		return err
	}

	externalSeeded, err := s.hasExternalSeedData(ctx)
	if err != nil {
		return err
	}
	if externalSeeded {
		return s.reconcileExistingSeedData(ctx)
	}

	tenantEntity, err := s.ensureDefaultTenant(ctx)
	if err != nil {
		return err
	}

	rootOrg, deptOrg, err := s.ensureOrgUnits(ctx, tenantEntity.ID)
	if err != nil {
		return err
	}

	adminPosition, staffPosition, err := s.ensurePositions(ctx, tenantEntity.ID, deptOrg.ID)
	if err != nil {
		return err
	}

	permissionIDs, err := s.collectDefaultAdminPermissionIDs(ctx)
	if err != nil {
		return err
	}
	if _, err := s.ensurePlatformSuperRole(ctx, permissionIDs); err != nil {
		return err
	}
	normalPermissionIDs, err := s.collectNormalUserPermissionIDs(ctx)
	if err != nil {
		return err
	}

	superRole, normalRole, err := s.ensureRoles(ctx, tenantEntity.ID, permissionIDs, normalPermissionIDs)
	if err != nil {
		return err
	}
	platformSuperRole, err := s.ensurePlatformSuperRole(ctx, permissionIDs)
	if err != nil {
		return err
	}

	platformUser, err := s.ensureUser(ctx, seedUserSpec{
		username:      platformUsername,
		nickname:      "平台超级管理员",
		realname:      "平台超级管理员",
		email:         "platform.admin@example.com",
		mobile:        "13800000000",
		telephone:     "010-10000000",
		description:   "平台级超级管理员",
		tenantID:      platformTenantID,
		roleID:        platformSuperRole.ID,
		primaryRoleID: platformSuperRole.ID,
		password:      defaultPassword,
	})
	if err != nil {
		return err
	}

	adminUser, err := s.ensureUser(ctx, seedUserSpec{
		username:      adminUsername,
		nickname:      "超级管理员",
		realname:      "超级管理员",
		email:         "admin@example.com",
		mobile:        "13800000001",
		telephone:     "010-10000001",
		description:   "系统超级管理员",
		tenantID:      tenantEntity.ID,
		orgUnitID:     rootOrg.ID,
		positionID:    adminPosition.ID,
		roleID:        superRole.ID,
		primaryRoleID: superRole.ID,
		password:      defaultPassword,
	})
	if err != nil {
		return err
	}

	if err := s.replaceUserRoles(ctx, platformTenantID, platformUser.ID, []uint32{platformSuperRole.ID}); err != nil {
		return err
	}

	normalUser, err := s.ensureUser(ctx, seedUserSpec{
		username:      normalUsername,
		nickname:      "普通用户",
		realname:      "普通用户",
		email:         "user@example.com",
		mobile:        "13800000002",
		telephone:     "010-10000002",
		description:   "系统普通用户",
		tenantID:      tenantEntity.ID,
		orgUnitID:     deptOrg.ID,
		positionID:    staffPosition.ID,
		roleID:        normalRole.ID,
		primaryRoleID: normalRole.ID,
		password:      defaultPassword,
	})
	if err != nil {
		return err
	}

	if err := s.bindUserRelations(ctx, tenantEntity.ID, adminUser.ID, rootOrg.ID, adminPosition.ID, []uint32{superRole.ID}); err != nil {
		return err
	}
	if err := s.bindUserRelations(ctx, tenantEntity.ID, normalUser.ID, deptOrg.ID, staffPosition.ID, []uint32{normalRole.ID}); err != nil {
		return err
	}

	if err := s.ensureMembership(ctx, tenantEntity.ID, adminUser.ID, rootOrg.ID, adminPosition.ID, superRole.ID); err != nil {
		return err
	}
	if err := s.ensureMembership(ctx, tenantEntity.ID, normalUser.ID, deptOrg.ID, staffPosition.ID, normalRole.ID); err != nil {
		return err
	}

	if tenantEntity.AdminUserID == nil || *tenantEntity.AdminUserID != adminUser.ID {
		if _, err := s.entClient.Client().Tenant.UpdateOneID(tenantEntity.ID).
			SetAdminUserID(adminUser.ID).
			SetUpdatedAt(s.now).
			SetUpdatedBy(0).
			Save(ctx); err != nil {
			return fmt.Errorf("update default tenant admin user: %w", err)
		}
	}

	return nil
}

func (s *defaultDataSeed) hasExternalSeedData(ctx context.Context) (bool, error) {
	tenants, err := s.entClient.Client().Tenant.Query().
		Order(tenant.ByID()).
		All(ctx)
	if err != nil {
		return false, fmt.Errorf("list tenants for external seed detection: %w", err)
	}
	for _, item := range tenants {
		if item == nil {
			continue
		}
		if code := normalizeString(item.Code); code != "" && !strings.EqualFold(code, defaultTenantCode) {
			return true, nil
		}
	}

	users, err := s.entClient.Client().User.Query().
		Order(user.ByID()).
		All(ctx)
	if err != nil {
		return false, fmt.Errorf("list users for external seed detection: %w", err)
	}
	for _, item := range users {
		if item == nil {
			continue
		}
		username := normalizeString(item.Username)
		if username == "" {
			continue
		}
		if !strings.EqualFold(username, adminUsername) &&
			!strings.EqualFold(username, normalUsername) &&
			!strings.EqualFold(username, platformUsername) {
			return true, nil
		}
	}

	return false, nil
}

func (s *defaultDataSeed) reconcileExistingSeedData(ctx context.Context) error {
	tenants, err := s.entClient.Client().Tenant.Query().
		Order(tenant.ByID()).
		All(ctx)
	if err != nil {
		return fmt.Errorf("list existing tenants: %w", err)
	}
	if len(tenants) == 0 {
		return nil
	}

	permissionIDs, err := s.collectDefaultAdminPermissionIDs(ctx)
	if err != nil {
		return err
	}
	normalPermissionIDs, err := s.collectNormalUserPermissionIDs(ctx)
	if err != nil {
		return err
	}

	for _, tenantEntity := range tenants {
		if tenantEntity == nil {
			continue
		}

		superRole, normalRole, err := s.ensureRoles(ctx, tenantEntity.ID, permissionIDs, normalPermissionIDs)
		if err != nil {
			return err
		}

		users, err := s.entClient.Client().User.Query().
			Where(user.TenantIDEQ(tenantEntity.ID)).
			Order(user.ByID()).
			All(ctx)
		if err != nil {
			return fmt.Errorf("list users for tenant %d: %w", tenantEntity.ID, err)
		}

		for _, userEntity := range users {
			if userEntity == nil {
				continue
			}

			primaryRoleID := normalRole.ID
			if s.isTenantAdminUser(tenantEntity, userEntity) {
				primaryRoleID = superRole.ID
			}

			if err := s.replaceUserRoles(ctx, tenantEntity.ID, userEntity.ID, []uint32{primaryRoleID}); err != nil {
				return err
			}
			if err := s.reconcileMembershipRole(ctx, tenantEntity.ID, userEntity.ID, primaryRoleID); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *defaultDataSeed) reconcileMembershipRole(ctx context.Context, tenantID, userID, primaryRoleID uint32) error {
	entity, err := s.entClient.Client().Membership.Query().
		Where(
			membership.TenantIDEQ(tenantID),
			membership.UserIDEQ(userID),
		).
		Only(ctx)
	if ent.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("query membership user=%d tenant=%d: %w", userID, tenantID, err)
	}

	builder := s.entClient.Client().Membership.UpdateOneID(entity.ID).
		SetRoleID(primaryRoleID).
		SetIsPrimary(true).
		SetStatus(membership.StatusActive).
		SetAssignedAt(s.now).
		SetAssignedBy(0).
		SetUpdatedAt(s.now).
		SetUpdatedBy(0)
	if _, err := builder.Save(ctx); err != nil {
		return fmt.Errorf("update membership role user=%d tenant=%d: %w", userID, tenantID, err)
	}

	return s.replaceMembershipRoles(ctx, tenantID, entity.ID, []uint32{primaryRoleID})
}

func (s *defaultDataSeed) isTenantAdminUser(tenantEntity *ent.Tenant, userEntity *ent.User) bool {
	if tenantEntity == nil || userEntity == nil {
		return false
	}
	if tenantEntity.AdminUserID != nil && *tenantEntity.AdminUserID == userEntity.ID {
		return true
	}

	username := strings.ToLower(normalizeString(userEntity.Username))
	if username == "" {
		return false
	}
	return strings.Contains(username, "admin")
}

func (s *defaultDataSeed) syncResources(ctx context.Context) error {
	menuRepo := repo.NewMenuRepo(s.appCtx, s.entClient)
	if err := menuRepo.SyncDefaultNavigation(ctx); err != nil {
		return fmt.Errorf("sync default menus: %w", err)
	}

	apiRepo := repo.NewApiRepo(s.appCtx, s.entClient)
	if err := apiRepo.SyncApisFromOpenAPI(ctx); err != nil {
		return fmt.Errorf("sync openapi apis: %w", err)
	}

	permissionRepo := repo.NewPermissionRepo(s.appCtx, s.entClient)
	permissionGroupRepo := repo.NewPermissionGroupRepo(s.appCtx, s.entClient)
	roleRepo := repo.NewRoleRepo(s.appCtx, s.entClient)
	permissionService := service.NewPermissionService(s.appCtx, permissionRepo, roleRepo, permissionGroupRepo, menuRepo, apiRepo)
	if _, err := permissionService.SyncPermissions(ctx, nil); err != nil {
		return fmt.Errorf("sync permissions: %w", err)
	}

	return nil
}

func (s *defaultDataSeed) ensureDefaultTenant(ctx context.Context) (*ent.Tenant, error) {
	entity, err := s.entClient.Client().Tenant.Query().
		Where(tenant.CodeEQ(defaultTenantCode)).
		Only(ctx)
	if err == nil {
		updated := s.entClient.Client().Tenant.UpdateOneID(entity.ID).
			SetName("默认租户").
			SetCode(defaultTenantCode).
			SetType(tenant.TypePaid).
			SetStatus(tenant.StatusOn).
			SetAuditStatus(tenant.AuditStatusApproved).
			SetSubscriptionPlan("default").
			SetUpdatedAt(s.now).
			SetUpdatedBy(0)
		return updated.Save(ctx)
	}
	if !ent.IsNotFound(err) {
		return nil, fmt.Errorf("query default tenant: %w", err)
	}

	return s.entClient.Client().Tenant.Create().
		SetName("默认租户").
		SetCode(defaultTenantCode).
		SetType(tenant.TypePaid).
		SetStatus(tenant.StatusOn).
		SetAuditStatus(tenant.AuditStatusApproved).
		SetSubscriptionPlan("default").
		SetCreatedAt(s.now).
		SetCreatedBy(0).
		SetUpdatedAt(s.now).
		SetUpdatedBy(0).
		Save(ctx)
}

func (s *defaultDataSeed) ensureOrgUnits(ctx context.Context, tenantID uint32) (*ent.OrgUnit, *ent.OrgUnit, error) {
	root, err := s.ensureOrgUnit(ctx, tenantID, nil, "XAdmin 平台", "XADMIN", orgunit.TypeCompany, 1, "平台根组织")
	if err != nil {
		return nil, nil, err
	}
	dept, err := s.ensureOrgUnit(ctx, tenantID, &root.ID, "研发部", "RD", orgunit.TypeDepartment, 1, "默认业务部门")
	if err != nil {
		return nil, nil, err
	}
	return root, dept, nil
}

func (s *defaultDataSeed) ensureOrgUnit(ctx context.Context, tenantID uint32, parentID *uint32, name, code string, orgType orgunit.Type, sortOrder uint32, description string) (*ent.OrgUnit, error) {
	query := s.entClient.Client().OrgUnit.Query().
		Where(orgunit.TenantIDEQ(tenantID), orgunit.CodeEQ(code))

	entity, err := query.Only(ctx)
	if err == nil {
		builder := s.entClient.Client().OrgUnit.UpdateOneID(entity.ID).
			SetName(name).
			SetCode(code).
			SetType(orgType).
			SetStatus(orgunit.StatusOn).
			SetSortOrder(sortOrder).
			SetDescription(description).
			SetUpdatedAt(s.now).
			SetUpdatedBy(0)
		if parentID != nil {
			builder.SetParentID(*parentID)
		} else {
			builder.ClearParentID()
		}
		return builder.Save(ctx)
	}
	if !ent.IsNotFound(err) {
		return nil, fmt.Errorf("query org unit %s: %w", code, err)
	}

	builder := s.entClient.Client().OrgUnit.Create().
		SetTenantID(tenantID).
		SetName(name).
		SetCode(code).
		SetType(orgType).
		SetStatus(orgunit.StatusOn).
		SetSortOrder(sortOrder).
		SetDescription(description).
		SetCreatedAt(s.now).
		SetCreatedBy(0).
		SetUpdatedAt(s.now).
		SetUpdatedBy(0)
	if parentID != nil {
		builder.SetParentID(*parentID)
	}
	return builder.Save(ctx)
}

func (s *defaultDataSeed) ensurePositions(ctx context.Context, tenantID, deptOrgID uint32) (*ent.Position, *ent.Position, error) {
	adminPosition, err := s.ensurePosition(ctx, tenantID, deptOrgID, "系统管理员", "SYS_ADMIN", position.TypeManager, 1, "系统管理岗位")
	if err != nil {
		return nil, nil, err
	}
	staffPosition, err := s.ensurePosition(ctx, tenantID, deptOrgID, "普通员工", "STAFF", position.TypeRegular, 2, "普通员工岗位")
	if err != nil {
		return nil, nil, err
	}
	return adminPosition, staffPosition, nil
}

func (s *defaultDataSeed) ensurePosition(ctx context.Context, tenantID, orgUnitID uint32, name, code string, positionType position.Type, sortOrder uint32, description string) (*ent.Position, error) {
	entity, err := s.entClient.Client().Position.Query().
		Where(position.TenantIDEQ(tenantID), position.CodeEQ(code)).
		Only(ctx)
	if err == nil {
		return s.entClient.Client().Position.UpdateOneID(entity.ID).
			SetName(name).
			SetCode(code).
			SetOrgUnitID(orgUnitID).
			SetType(positionType).
			SetStatus(position.StatusOn).
			SetSortOrder(sortOrder).
			SetDescription(description).
			SetUpdatedAt(s.now).
			SetUpdatedBy(0).
			Save(ctx)
	}
	if !ent.IsNotFound(err) {
		return nil, fmt.Errorf("query position %s: %w", code, err)
	}

	return s.entClient.Client().Position.Create().
		SetTenantID(tenantID).
		SetName(name).
		SetCode(code).
		SetOrgUnitID(orgUnitID).
		SetType(positionType).
		SetStatus(position.StatusOn).
		SetSortOrder(sortOrder).
		SetDescription(description).
		SetCreatedAt(s.now).
		SetCreatedBy(0).
		SetUpdatedAt(s.now).
		SetUpdatedBy(0).
		Save(ctx)
}

func (s *defaultDataSeed) ensureTaskSeeds(ctx context.Context) error {
	groupEntity, err := s.entClient.Client().TaskGroup.Query().
		Where(
			enttaskgroup.TenantIDEQ(platformTenantID),
			enttaskgroup.GroupNameEQ("系统维护"),
		).
		Only(ctx)
	if err == nil {
		groupEntity, err = s.entClient.Client().TaskGroup.UpdateOneID(groupEntity.ID).
			SetGroupName("系统维护").
			SetRemark("系统内置维护任务").
			SetUpdatedAt(s.now).
			SetUpdatedBy(0).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("update task group seed: %w", err)
		}
	} else if ent.IsNotFound(err) {
		groupEntity, err = s.entClient.Client().TaskGroup.Create().
			SetTenantID(platformTenantID).
			SetGroupName("系统维护").
			SetRemark("系统内置维护任务").
			SetCreatedAt(s.now).
			SetCreatedBy(0).
			SetUpdatedAt(s.now).
			SetUpdatedBy(0).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("create task group seed: %w", err)
		}
	} else {
		return fmt.Errorf("query task group seed: %w", err)
	}

	groupID := uint64(groupEntity.ID)
	if err := s.ensureTaskSeed(ctx, taskSeedSpec{
		groupID:        groupID,
		taskName:       "删除过期日志",
		taskType:       enttask.TaskTypeFunction,
		cronExpression: "0 0 3 * * *",
		invokeTarget:   "system:cleanup:audit-logs",
		args:           `{"expireHours":720,"targets":["api","login","permission"]}`,
		retry:          0,
		concurrent:     false,
		status:         enttask.StatusStopped,
		remark:         "按小时参数清理 API/登录/权限审计日志",
	}); err != nil {
		return err
	}
	if err := s.ensureTaskSeed(ctx, taskSeedSpec{
		groupID:        groupID,
		taskName:       "任务运行概览",
		taskType:       enttask.TaskTypeFunction,
		cronExpression: "0 0 8 * * 1",
		invokeTarget:   "system:task:runtime-summary",
		args:           `{"tenantScope":"global"}`,
		retry:          0,
		concurrent:     false,
		status:         enttask.StatusStopped,
		remark:         "每周汇总所有内置任务运行状态，作为内部执行器示例",
	}); err != nil {
		return err
	}
	return nil
}

type taskSeedSpec struct {
	groupID        uint64
	taskName       string
	taskType       enttask.TaskType
	cronExpression string
	invokeTarget   string
	args           string
	retry          uint32
	concurrent     bool
	status         enttask.Status
	remark         string
}

func (s *defaultDataSeed) ensureTaskSeed(ctx context.Context, spec taskSeedSpec) error {
	taskEntity, err := s.entClient.Client().Task.Query().
		Where(
			enttask.TenantIDEQ(platformTenantID),
			enttask.GroupIDEQ(spec.groupID),
			enttask.TaskNameEQ(spec.taskName),
		).
		Only(ctx)
	if err == nil {
		_, err = s.entClient.Client().Task.UpdateOneID(taskEntity.ID).
			SetTaskName(spec.taskName).
			SetGroupID(spec.groupID).
			SetTaskType(spec.taskType).
			SetCronExpression(spec.cronExpression).
			SetInvokeTarget(spec.invokeTarget).
			SetArgs(spec.args).
			SetRetry(spec.retry).
			SetConcurrent(spec.concurrent).
			SetStatus(spec.status).
			SetRemark(spec.remark).
			SetUpdatedAt(s.now).
			SetUpdatedBy(0).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("update task seed %s: %w", spec.taskName, err)
		}
		return nil
	}
	if !ent.IsNotFound(err) {
		return fmt.Errorf("query task seed %s: %w", spec.taskName, err)
	}

	if _, err := s.entClient.Client().Task.Create().
		SetTenantID(platformTenantID).
		SetTaskName(spec.taskName).
		SetGroupID(spec.groupID).
		SetTaskType(spec.taskType).
		SetCronExpression(spec.cronExpression).
		SetInvokeTarget(spec.invokeTarget).
		SetArgs(spec.args).
		SetRetry(spec.retry).
		SetConcurrent(spec.concurrent).
		SetStatus(spec.status).
		SetRemark(spec.remark).
		SetCreatedAt(s.now).
		SetCreatedBy(0).
		SetUpdatedAt(s.now).
		SetUpdatedBy(0).
		Save(ctx); err != nil {
		return fmt.Errorf("create task seed %s: %w", spec.taskName, err)
	}

	return nil
}

func (s *defaultDataSeed) collectPermissionIDs(ctx context.Context) ([]uint32, error) {
	items, err := s.entClient.Client().Permission.Query().
		Where(permission.StatusEQ(permission.StatusOn)).
		Order(permission.ByID()).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list permissions: %w", err)
	}

	ids := make([]uint32, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		ids = append(ids, item.ID)
	}
	return ids, nil
}

func (s *defaultDataSeed) collectDefaultAdminPermissionIDs(ctx context.Context) ([]uint32, error) {
	groups, err := s.entClient.Client().PermissionGroup.Query().
		Where(permissiongroup.StatusEQ(permissiongroup.StatusOn)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list permission groups for default admin role: %w", err)
	}

	groupParentByID := make(map[uint32]uint32, len(groups))
	var featureRootID uint32
	var exportGroupID uint32
	for _, item := range groups {
		if item == nil {
			continue
		}
		groupParentByID[item.ID] = uint32Value(item.ParentID)
		switch strings.TrimSpace(stringValue(item.Module)) {
		case permissionGroupModuleFeature:
			if featureRootID != 0 && featureRootID != item.ID {
				return nil, fmt.Errorf("multiple active feature root permission groups detected: %d and %d", featureRootID, item.ID)
			}
			featureRootID = item.ID
		case permissionGroupModuleExport:
			if exportGroupID != 0 && exportGroupID != item.ID {
				return nil, fmt.Errorf("multiple active export permission groups detected: %d and %d", exportGroupID, item.ID)
			}
			exportGroupID = item.ID
		}
	}

	targetGroupIDs := map[uint32]struct{}{}
	if featureRootID != 0 {
		targetGroupIDs[featureRootID] = struct{}{}
		for groupID, parentID := range groupParentByID {
			if parentID == featureRootID {
				targetGroupIDs[groupID] = struct{}{}
			}
		}
	}
	if exportGroupID != 0 {
		targetGroupIDs[exportGroupID] = struct{}{}
		for groupID, parentID := range groupParentByID {
			if parentID == exportGroupID {
				targetGroupIDs[groupID] = struct{}{}
			}
		}
	}
	if len(targetGroupIDs) == 0 {
		return nil, nil
	}

	items, err := s.entClient.Client().Permission.Query().
		Where(permission.StatusEQ(permission.StatusOn)).
		Order(permission.ByID()).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list permissions for default admin role: %w", err)
	}

	ids := make([]uint32, 0, len(items))
	for _, item := range items {
		if item == nil || item.GroupID == nil {
			continue
		}
		if _, ok := targetGroupIDs[*item.GroupID]; !ok {
			continue
		}
		ids = append(ids, item.ID)
	}
	return uniqueUint32s(ids), nil
}

func (s *defaultDataSeed) collectNormalUserPermissionIDs(ctx context.Context) ([]uint32, error) {
	resp, err := repo.NewPermissionRepo(s.appCtx, s.entClient).List(ctx, &paginationv1.PagingRequest{
		NoPaging: boolPtr(true),
	})
	if err != nil {
		return nil, fmt.Errorf("list permissions for normal role: %w", err)
	}

	ids := make([]uint32, 0)
	for _, item := range resp.GetItems() {
		if item == nil || item.GetId() == 0 {
			continue
		}
		code := strings.TrimSpace(item.GetCode())
		if code == "" {
			continue
		}
		if strings.HasPrefix(code, "dashboard:") || strings.HasPrefix(code, "analytics:") || strings.HasPrefix(code, "workspace:") {
			ids = append(ids, item.GetId())
			continue
		}
		if strings.HasPrefix(code, "system:user:view") {
			ids = append(ids, item.GetId())
		}
	}

	if len(ids) > 0 {
		return uniqueUint32s(ids), nil
	}

	items, err := s.entClient.Client().Permission.Query().
		Where(permission.StatusEQ(permission.StatusOn)).
		Order(permission.ByID()).
		Limit(6).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("fallback list permissions for normal role: %w", err)
	}
	for _, item := range items {
		if item == nil {
			continue
		}
		ids = append(ids, item.ID)
	}
	return uniqueUint32s(ids), nil
}

func uint32Value(value *uint32) uint32 {
	if value == nil {
		return 0
	}
	return *value
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (s *defaultDataSeed) ensureRoles(ctx context.Context, tenantID uint32, superPermissionIDs, normalPermissionIDs []uint32) (*ent.Role, *ent.Role, error) {
	superRole, err := s.ensureRole(ctx, tenantID, seedRoleSpec{
		code:          "SUPER_ADMIN",
		name:          "超级管理员",
		description:   "拥有全部权限的系统角色",
		roleType:      role.TypeSystem,
		isProtected:   true,
		sortOrder:     1,
		permissionIDs: superPermissionIDs,
	})
	if err != nil {
		return nil, nil, err
	}
	normalRole, err := s.ensureRole(ctx, tenantID, seedRoleSpec{
		code:          "USER",
		name:          "普通用户",
		description:   "普通用户默认角色",
		roleType:      role.TypeTenant,
		isProtected:   true,
		sortOrder:     2,
		permissionIDs: normalPermissionIDs,
	})
	if err != nil {
		return nil, nil, err
	}
	return superRole, normalRole, nil
}

func (s *defaultDataSeed) ensurePlatformSuperRole(ctx context.Context, permissionIDs []uint32) (*ent.Role, error) {
	return s.ensureRole(ctx, platformTenantID, seedRoleSpec{
		code:          "PLATFORM_SUPER_ADMIN",
		name:          "平台超级管理员",
		description:   "拥有平台级全部权限的系统角色",
		roleType:      role.TypeSystem,
		isProtected:   true,
		sortOrder:     0,
		permissionIDs: permissionIDs,
	})
}

func (s *defaultDataSeed) ensureRole(ctx context.Context, tenantID uint32, spec seedRoleSpec) (*ent.Role, error) {
	entity, err := s.entClient.Client().Role.Query().
		Where(role.TenantIDEQ(tenantID), role.CodeEQ(spec.code)).
		Only(ctx)
	if err == nil {
		entity, err = s.entClient.Client().Role.UpdateOneID(entity.ID).
			SetCode(spec.code).
			SetName(spec.name).
			SetDescription(spec.description).
			SetIsProtected(spec.isProtected).
			SetType(spec.roleType).
			SetStatus(role.StatusOn).
			SetSortOrder(spec.sortOrder).
			SetUpdatedAt(s.now).
			SetUpdatedBy(0).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("update role %s: %w", spec.code, err)
		}
	} else if ent.IsNotFound(err) {
		entity, err = s.entClient.Client().Role.Create().
			SetTenantID(tenantID).
			SetCode(spec.code).
			SetName(spec.name).
			SetDescription(spec.description).
			SetIsProtected(spec.isProtected).
			SetType(spec.roleType).
			SetStatus(role.StatusOn).
			SetSortOrder(spec.sortOrder).
			SetCreatedAt(s.now).
			SetCreatedBy(0).
			SetUpdatedAt(s.now).
			SetUpdatedBy(0).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("create role %s: %w", spec.code, err)
		}
	} else {
		return nil, fmt.Errorf("query role %s: %w", spec.code, err)
	}

	if err := s.replaceRolePermissions(ctx, tenantID, entity.ID, spec.permissionIDs); err != nil {
		return nil, err
	}

	return entity, nil
}

func (s *defaultDataSeed) replaceRolePermissions(ctx context.Context, tenantID, roleID uint32, permissionIDs []uint32) error {
	if _, err := s.entClient.Client().RolePermission.Delete().
		Where(rolepermission.RoleIDEQ(roleID)).
		Exec(ctx); err != nil {
		return fmt.Errorf("clear role permissions for role %d: %w", roleID, err)
	}

	for _, permissionID := range uniqueUint32s(permissionIDs) {
		if permissionID == 0 {
			continue
		}
		if _, err := s.entClient.Client().RolePermission.Create().
			SetTenantID(tenantID).
			SetRoleID(roleID).
			SetPermissionID(permissionID).
			SetEffect(rolepermission.EffectAllow).
			SetPriority(0).
			SetStatus(rolepermission.StatusOn).
			SetCreatedAt(s.now).
			SetCreatedBy(0).
			SetUpdatedAt(s.now).
			SetUpdatedBy(0).
			Save(ctx); err != nil {
			return fmt.Errorf("create role permission role=%d permission=%d: %w", roleID, permissionID, err)
		}
	}

	return nil
}

func (s *defaultDataSeed) ensureUser(ctx context.Context, spec seedUserSpec) (*ent.User, error) {
	entity, err := s.entClient.Client().User.Query().
		Where(user.TenantIDEQ(spec.tenantID), user.UsernameEQ(spec.username)).
		Only(ctx)
	if err == nil {
		entity, err = s.entClient.Client().User.UpdateOneID(entity.ID).
			SetNillableNickname(&spec.nickname).
			SetNillableRealname(&spec.realname).
			SetNillableEmail(&spec.email).
			SetNillableMobile(&spec.mobile).
			SetNillableTelephone(&spec.telephone).
			SetNillableDescription(&spec.description).
			SetNillableGender(userGenderPtr(user.GenderSecret)).
			SetNillableStatus(userStatusPtr(user.StatusNormal)).
			SetUpdatedAt(s.now).
			SetUpdatedBy(0).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("update user %s: %w", spec.username, err)
		}
	} else if ent.IsNotFound(err) {
		entity, err = s.entClient.Client().User.Create().
			SetTenantID(spec.tenantID).
			SetUsername(spec.username).
			SetNickname(spec.nickname).
			SetRealname(spec.realname).
			SetEmail(spec.email).
			SetMobile(spec.mobile).
			SetTelephone(spec.telephone).
			SetDescription(spec.description).
			SetGender(user.GenderSecret).
			SetStatus(user.StatusNormal).
			SetCreatedAt(s.now).
			SetCreatedBy(0).
			SetUpdatedAt(s.now).
			SetUpdatedBy(0).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("create user %s: %w", spec.username, err)
		}
	} else {
		return nil, fmt.Errorf("query user %s: %w", spec.username, err)
	}

	if err := s.ensureUserCredential(ctx, spec.tenantID, entity.ID, spec.username, spec.password); err != nil {
		return nil, err
	}
	return entity, nil
}

func (s *defaultDataSeed) ensureUserCredential(ctx context.Context, tenantID, userID uint32, username, password string) error {
	normalizedPassword, err := repo.NormalizePasswordCredential(password)
	if err != nil {
		return fmt.Errorf("normalize credential for %s: %w", username, err)
	}

	entity, err := s.entClient.Client().UserCredential.Query().
		Where(
			usercredential.UserIDEQ(userID),
			usercredential.IdentityTypeEQ(usercredential.IdentityTypeUsername),
			usercredential.IdentifierEQ(username),
			usercredential.CredentialTypeEQ(usercredential.CredentialTypePasswordHash),
		).
		Only(ctx)
	if err == nil {
		_, err = s.entClient.Client().UserCredential.UpdateOneID(entity.ID).
			SetCredential(string(normalizedPassword)).
			SetStatus(usercredential.StatusEnabled).
			SetIsPrimary(true).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("update credential for %s: %w", username, err)
		}
		return nil
	}
	if !ent.IsNotFound(err) {
		return fmt.Errorf("query credential for %s: %w", username, err)
	}

	if _, err := s.entClient.Client().UserCredential.Create().
		SetTenantID(tenantID).
		SetUserID(userID).
		SetIdentityType(usercredential.IdentityTypeUsername).
		SetIdentifier(username).
		SetCredentialType(usercredential.CredentialTypePasswordHash).
		SetCredential(string(normalizedPassword)).
		SetIsPrimary(true).
		SetStatus(usercredential.StatusEnabled).
		SetCreatedAt(s.now).
		SetUpdatedAt(s.now).
		Save(ctx); err != nil {
		return fmt.Errorf("create credential for %s: %w", username, err)
	}
	return nil
}

func (s *defaultDataSeed) bindUserRelations(ctx context.Context, tenantID, userID, orgUnitID, positionID uint32, roleIDs []uint32) error {
	if err := s.replaceUserOrgUnits(ctx, tenantID, userID, orgUnitID, positionID); err != nil {
		return err
	}
	if err := s.replaceUserPositions(ctx, tenantID, userID, positionID); err != nil {
		return err
	}
	if err := s.replaceUserRoles(ctx, tenantID, userID, roleIDs); err != nil {
		return err
	}
	return nil
}

func (s *defaultDataSeed) replaceUserOrgUnits(ctx context.Context, tenantID, userID, orgUnitID, positionID uint32) error {
	if _, err := s.entClient.Client().UserOrgUnit.Delete().
		Where(userorgunit.UserIDEQ(userID)).
		Exec(ctx); err != nil {
		return fmt.Errorf("clear user org units for user %d: %w", userID, err)
	}

	if _, err := s.entClient.Client().UserOrgUnit.Create().
		SetTenantID(tenantID).
		SetUserID(userID).
		SetOrgUnitID(orgUnitID).
		SetPositionID(positionID).
		SetStatus(userorgunit.StatusActive).
		SetIsPrimary(true).
		SetAssignedAt(s.now).
		SetAssignedBy(0).
		SetCreatedAt(s.now).
		SetCreatedBy(0).
		SetUpdatedAt(s.now).
		SetUpdatedBy(0).
		Save(ctx); err != nil {
		return fmt.Errorf("create user org unit for user %d: %w", userID, err)
	}
	return nil
}

func (s *defaultDataSeed) replaceUserPositions(ctx context.Context, tenantID, userID, positionID uint32) error {
	if _, err := s.entClient.Client().UserPosition.Delete().
		Where(userposition.UserIDEQ(userID)).
		Exec(ctx); err != nil {
		return fmt.Errorf("clear user positions for user %d: %w", userID, err)
	}

	if _, err := s.entClient.Client().UserPosition.Create().
		SetTenantID(tenantID).
		SetUserID(userID).
		SetPositionID(positionID).
		SetStatus(userposition.StatusActive).
		SetIsPrimary(true).
		SetAssignedAt(s.now).
		SetAssignedBy(0).
		SetCreatedAt(s.now).
		SetCreatedBy(0).
		SetUpdatedAt(s.now).
		SetUpdatedBy(0).
		Save(ctx); err != nil {
		return fmt.Errorf("create user position for user %d: %w", userID, err)
	}
	return nil
}

func (s *defaultDataSeed) replaceUserRoles(ctx context.Context, tenantID, userID uint32, roleIDs []uint32) error {
	if _, err := s.entClient.Client().UserRole.Delete().
		Where(userrole.UserIDEQ(userID)).
		Exec(ctx); err != nil {
		return fmt.Errorf("clear user roles for user %d: %w", userID, err)
	}

	roleIDs = uniqueUint32s(roleIDs)
	for index, roleID := range roleIDs {
		if roleID == 0 {
			continue
		}
		if _, err := s.entClient.Client().UserRole.Create().
			SetTenantID(tenantID).
			SetUserID(userID).
			SetRoleID(roleID).
			SetStatus(userrole.StatusActive).
			SetIsPrimary(index == 0).
			SetAssignedAt(s.now).
			SetAssignedBy(0).
			SetCreatedAt(s.now).
			SetCreatedBy(0).
			SetUpdatedAt(s.now).
			SetUpdatedBy(0).
			Save(ctx); err != nil {
			return fmt.Errorf("create user role for user %d role %d: %w", userID, roleID, err)
		}
	}
	return nil
}

func (s *defaultDataSeed) ensureMembership(ctx context.Context, tenantID, userID, orgUnitID, positionID, roleID uint32) error {
	entity, err := s.entClient.Client().Membership.Query().
		Where(
			membership.TenantIDEQ(tenantID),
			membership.UserIDEQ(userID),
		).
		Only(ctx)
	if err == nil {
		entity, err = s.entClient.Client().Membership.UpdateOneID(entity.ID).
			SetOrgUnitID(orgUnitID).
			SetPositionID(positionID).
			SetRoleID(roleID).
			SetIsPrimary(true).
			SetStatus(membership.StatusActive).
			SetAssignedAt(s.now).
			SetAssignedBy(0).
			SetJoinedAt(s.now).
			SetUpdatedAt(s.now).
			SetUpdatedBy(0).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("update membership user=%d: %w", userID, err)
		}
	} else if ent.IsNotFound(err) {
		entity, err = s.entClient.Client().Membership.Create().
			SetTenantID(tenantID).
			SetUserID(userID).
			SetOrgUnitID(orgUnitID).
			SetPositionID(positionID).
			SetRoleID(roleID).
			SetIsPrimary(true).
			SetStatus(membership.StatusActive).
			SetAssignedAt(s.now).
			SetAssignedBy(0).
			SetJoinedAt(s.now).
			SetCreatedAt(s.now).
			SetCreatedBy(0).
			SetUpdatedAt(s.now).
			SetUpdatedBy(0).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("create membership user=%d: %w", userID, err)
		}
	} else {
		return fmt.Errorf("query membership user=%d: %w", userID, err)
	}

	return s.replaceMembershipRoles(ctx, tenantID, entity.ID, []uint32{roleID})
}

func (s *defaultDataSeed) replaceMembershipRoles(ctx context.Context, tenantID, membershipID uint32, roleIDs []uint32) error {
	if _, err := s.entClient.Client().MembershipRole.Delete().
		Where(membershiprole.MembershipIDEQ(membershipID)).
		Exec(ctx); err != nil {
		return fmt.Errorf("clear membership roles membership=%d: %w", membershipID, err)
	}

	roleIDs = uniqueUint32s(roleIDs)
	for index, roleID := range roleIDs {
		if roleID == 0 {
			continue
		}
		if _, err := s.entClient.Client().MembershipRole.Create().
			SetTenantID(tenantID).
			SetMembershipID(membershipID).
			SetRoleID(roleID).
			SetStatus(membershiprole.StatusActive).
			SetIsPrimary(index == 0).
			SetAssignedAt(s.now).
			SetAssignedBy(0).
			SetCreatedAt(s.now).
			SetCreatedBy(0).
			SetUpdatedAt(s.now).
			SetUpdatedBy(0).
			Save(ctx); err != nil {
			return fmt.Errorf("create membership role membership=%d role=%d: %w", membershipID, roleID, err)
		}
	}
	return nil
}

func uniqueUint32s(values []uint32) []uint32 {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[uint32]struct{}, len(values))
	out := make([]uint32, 0, len(values))
	for _, value := range values {
		if value == 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func boolPtr(value bool) *bool {
	return &value
}

func userGenderPtr(value user.Gender) *user.Gender {
	return &value
}

func userStatusPtr(value user.Status) *user.Status {
	return &value
}

func normalizeString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func (seedViewerContext) UserID() uint64        { return 0 }
func (seedViewerContext) TenantID() uint64      { return 0 }
func (seedViewerContext) OrgUnitID() uint64     { return 0 }
func (seedViewerContext) Permissions() []string { return nil }
func (seedViewerContext) Roles() []string       { return []string{"system"} }
func (seedViewerContext) DataScope() []crudviewer.DataScope {
	return []crudviewer.DataScope{{ScopeType: crudviewer.ScopeTypeAll}}
}
func (seedViewerContext) TraceID() string { return "" }
func (seedViewerContext) HasPermission(_, _ string) bool {
	return true
}
func (seedViewerContext) IsPlatformContext() bool { return true }
func (seedViewerContext) IsTenantContext() bool   { return false }
func (seedViewerContext) IsSystemContext() bool   { return true }
func (seedViewerContext) ShouldAudit() bool       { return false }
