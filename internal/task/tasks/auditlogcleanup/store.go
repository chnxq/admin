package auditlogcleanup

import (
	"context"
	"time"

	"admin/internal/data/repo"
)

type auditLogCleanupStore struct {
	apiCleaner        repo.ApiAuditLogCleaner
	loginCleaner      repo.LoginAuditLogCleaner
	permissionCleaner repo.PermissionAuditLogCleaner
}

func NewAuditLogCleanupStore(
	apiCleaner repo.ApiAuditLogCleaner,
	loginCleaner repo.LoginAuditLogCleaner,
	permissionCleaner repo.PermissionAuditLogCleaner,
) Store {
	return &auditLogCleanupStore{
		apiCleaner:        apiCleaner,
		loginCleaner:      loginCleaner,
		permissionCleaner: permissionCleaner,
	}
}

func (s *auditLogCleanupStore) CleanupAPIBefore(ctx context.Context, tenantID *uint32, before time.Time) (int, error) {
	if s == nil || s.apiCleaner == nil {
		return 0, nil
	}
	return s.apiCleaner.CleanupApiAuditLogsBefore(ctx, tenantID, before)
}

func (s *auditLogCleanupStore) CleanupLoginBefore(ctx context.Context, tenantID *uint32, before time.Time) (int, error) {
	if s == nil || s.loginCleaner == nil {
		return 0, nil
	}
	return s.loginCleaner.CleanupLoginAuditLogsBefore(ctx, tenantID, before)
}

func (s *auditLogCleanupStore) CleanupPermissionBefore(ctx context.Context, tenantID *uint32, before time.Time) (int, error) {
	if s == nil || s.permissionCleaner == nil {
		return 0, nil
	}
	return s.permissionCleaner.CleanupPermissionAuditLogsBefore(ctx, tenantID, before)
}
