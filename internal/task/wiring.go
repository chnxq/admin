package task

import (
	"context"

	"admin/internal/data/repo"
	"admin/internal/service"
)

func ConfigureServices(
	taskService *service.TaskService,
	taskGroupService *service.TaskGroupService,
	taskRepo repo.TaskRepo,
	taskGroupRepo repo.TaskGroupRepo,
	taskLogRepo repo.TaskLogRepo,
	apiAuditLogRepo repo.ApiAuditLogRepo,
	loginAuditLogRepo repo.LoginAuditLogRepo,
	permissionAuditLogRepo repo.PermissionAuditLogRepo,
) error {
	bundle, err := NewRuntimeBundleFromRepos(
		taskRepo,
		taskLogRepo,
		apiAuditLogRepo,
		loginAuditLogRepo,
		permissionAuditLogRepo,
	)
	if err != nil {
		return err
	}
	return service.BindTaskServices(
		taskService,
		taskGroupService,
		taskRepo,
		taskGroupRepo,
		bundle.Runner,
		bundle.Scheduler,
	)
}

func RegisterServices(
	ctx context.Context,
	taskService *service.TaskService,
	taskGroupService *service.TaskGroupService,
) (func(), error) {
	return service.RegisterTaskScheduler(ctx, taskService, taskGroupService)
}
