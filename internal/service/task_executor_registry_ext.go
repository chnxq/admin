package service

import (
	"admin/internal/data/repo"
	taskruntime "admin/internal/task"
)

func newTaskExecutorRegistry(
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

	return taskruntime.NewDefaultRegistry(
		taskruntime.NewRuntimeDeps(apiCleaner, loginCleaner, permissionCleaner),
	)
}
