package service

import (
	"context"

	taskv1 "admin/api/gen/task/v1"
	"admin/internal/data/repo"
	taskruntime "admin/internal/task"
)

func newTaskExecutorRegistry(
	taskRepo repo.TaskRepo,
	apiAuditLogRepo repo.ApiAuditLogRepo,
	loginAuditLogRepo repo.LoginAuditLogRepo,
	permissionAuditLogRepo repo.PermissionAuditLogRepo,
) *taskruntime.Registry {
	var apiCleaner taskruntime.ApiAuditLogCleaner
	if value, ok := apiAuditLogRepo.(repo.ApiAuditLogCleaner); ok {
		apiCleaner = value
	}

	var loginCleaner taskruntime.LoginAuditLogCleaner
	if value, ok := loginAuditLogRepo.(repo.LoginAuditLogCleaner); ok {
		loginCleaner = value
	}

	var permissionCleaner taskruntime.PermissionAuditLogCleaner
	if value, ok := permissionAuditLogRepo.(repo.PermissionAuditLogCleaner); ok {
		permissionCleaner = value
	}

	var taskSummaryProvider taskruntime.TaskSummaryProvider
	if value, ok := taskRepo.(interface {
		ListTasksForRuntime(ctx context.Context, tenantID *uint32) ([]*taskv1.Task, error)
	}); ok {
		taskSummaryProvider = value
	}

	return taskruntime.NewDefaultRegistry(
		taskruntime.NewRuntimeDeps(apiCleaner, loginCleaner, permissionCleaner, taskSummaryProvider),
	)
}
