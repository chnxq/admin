package task

func NewDefaultRegistry(deps RuntimeDeps) *Registry {
	return MustNewRegistry(
		NewCleanupAuditLogsExecutor(deps),
		NewTaskRuntimeSummaryExecutor(deps),
	)
}

func NewRuntimeDeps(
	apiCleaner ApiAuditLogCleaner,
	loginCleaner LoginAuditLogCleaner,
	permissionCleaner PermissionAuditLogCleaner,
	taskSummaryProvider TaskSummaryProvider,
) RuntimeDeps {
	return RuntimeDeps{
		ApiAuditLogCleaner:        apiCleaner,
		LoginAuditLogCleaner:      loginCleaner,
		PermissionAuditLogCleaner: permissionCleaner,
		TaskSummaryProvider:       taskSummaryProvider,
	}
}
