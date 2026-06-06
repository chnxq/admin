package auditlogcleanup

import (
	"context"
	"time"
)

const (
	CleanupAuditLogsInvokeTarget = "system:cleanup:audit-logs"
)

type Store interface {
	CleanupAPIBefore(ctx context.Context, tenantID *uint32, before time.Time) (int, error)
	CleanupLoginBefore(ctx context.Context, tenantID *uint32, before time.Time) (int, error)
	CleanupPermissionBefore(ctx context.Context, tenantID *uint32, before time.Time) (int, error)
}
