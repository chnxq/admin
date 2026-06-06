package auditlogcleanup

import (
	"admin/internal/data/repo"
	taskruntime "admin/internal/task/runtime"
)

func NewAuditLogCleanupExecutor(
	apiAuditLogRepo repo.ApiAuditLogRepo,
	loginAuditLogRepo repo.LoginAuditLogRepo,
	permissionAuditLogRepo repo.PermissionAuditLogRepo,
) taskruntime.Executor {
	var store Store
	apiCleaner, apiOK := apiAuditLogRepo.(repo.ApiAuditLogCleaner)
	loginCleaner, loginOK := loginAuditLogRepo.(repo.LoginAuditLogCleaner)
	permissionCleaner, permissionOK := permissionAuditLogRepo.(repo.PermissionAuditLogCleaner)
	if apiOK || loginOK || permissionOK {
		store = NewAuditLogCleanupStore(apiCleaner, loginCleaner, permissionCleaner)
	}
	return NewExecutor(store)
}
