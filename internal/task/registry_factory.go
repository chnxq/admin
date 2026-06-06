package task

func NewDefaultRegistry(deps RuntimeDeps) *Registry {
	return MustNewRegistry(
		NewCleanupAuditLogsExecutor(deps),
	)
}

func NewRuntimeDeps(
	apiCleaner ApiAuditLogCleaner,
	loginCleaner LoginAuditLogCleaner,
	permissionCleaner PermissionAuditLogCleaner,
) RuntimeDeps {
	return RuntimeDeps{
		ApiAuditLogCleaner:        apiCleaner,
		LoginAuditLogCleaner:      loginCleaner,
		PermissionAuditLogCleaner: permissionCleaner,
	}
}
