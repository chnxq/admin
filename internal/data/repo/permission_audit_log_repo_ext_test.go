package repo

import (
	"context"
	"io"
	"testing"

	auditv1 "admin/api/gen/audit/v1"
	"admin/internal/data/ent"
	_ "admin/internal/data/ent/runtime"
	entsql "entgo.io/ent/dialect/sql"
	entCrud "github.com/chnxq/x-crud/entgo"
	crudviewer "github.com/chnxq/x-crud/viewer"
	xlog "github.com/chnxq/xkitmod/log"
	_ "github.com/mattn/go-sqlite3"
)

type permissionAuditTestViewer struct{}

func (permissionAuditTestViewer) UserID() uint64                    { return 1 }
func (permissionAuditTestViewer) TenantID() uint64                  { return 1 }
func (permissionAuditTestViewer) OrgUnitID() uint64                 { return 0 }
func (permissionAuditTestViewer) Permissions() []string             { return []string{"*"} }
func (permissionAuditTestViewer) Roles() []string                   { return []string{"system"} }
func (permissionAuditTestViewer) DataScope() []crudviewer.DataScope { return nil }
func (permissionAuditTestViewer) TraceID() string                   { return "" }
func (permissionAuditTestViewer) HasPermission(string, string) bool { return true }
func (permissionAuditTestViewer) IsPlatformContext() bool           { return true }
func (permissionAuditTestViewer) IsTenantContext() bool             { return true }
func (permissionAuditTestViewer) IsSystemContext() bool             { return true }
func (permissionAuditTestViewer) ShouldAudit() bool                 { return false }

func TestWritePermissionAuditLogEntity_FillsRequiredStringsWhenMissing(t *testing.T) {
	driver, err := entsql.Open("sqlite3", "file:permission-audit-log?mode=memory&cache=shared&_fk=1")
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

	entClient := entCrud.NewEntClient[*ent.Client](client, driver)
	logger := xlog.NewHelper(xlog.NewStdLogger(io.Discard))

	action := auditv1.PermissionAuditLog_CREATE
	log := &auditv1.PermissionAuditLog{
		Action:     &action,
		TargetType: stringPtr("permission"),
		TargetId:   stringPtr("1"),
	}

	ctx := crudviewer.WithContext(context.Background(), permissionAuditTestViewer{})
	if err := writePermissionAuditLogEntity(ctx, entClient, logger, log); err != nil {
		t.Fatalf("writePermissionAuditLogEntity returned error: %v", err)
	}

	rows, err := client.PermissionAuditLog.Query().All(ctx)
	if err != nil {
		t.Fatalf("query permission audit logs failed: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 permission audit log, got %d", len(rows))
	}
	if rows[0].IPAddress == nil || *rows[0].IPAddress != "" {
		t.Fatalf("expected ip_address to be persisted as empty string, got %+v", rows[0].IPAddress)
	}
	if rows[0].RequestID == nil || *rows[0].RequestID != "" {
		t.Fatalf("expected request_id to be persisted as empty string, got %+v", rows[0].RequestID)
	}
	if rows[0].Reason == nil || *rows[0].Reason != "创建权限对象" {
		t.Fatalf("expected default reason to be persisted, got %+v", rows[0].Reason)
	}
}

func TestWritePermissionAuditLogEntity_PersistsRequestMetaValues(t *testing.T) {
	driver, err := entsql.Open("sqlite3", "file:permission-audit-log-meta?mode=memory&cache=shared&_fk=1")
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

	entClient := entCrud.NewEntClient[*ent.Client](client, driver)
	logger := xlog.NewHelper(xlog.NewStdLogger(io.Discard))

	action := auditv1.PermissionAuditLog_UPDATE
	ip := "10.0.0.8"
	requestID := "req-123"
	reason := "manual adjust"
	log := &auditv1.PermissionAuditLog{
		Action:     &action,
		TargetType: stringPtr("role"),
		TargetId:   stringPtr("2"),
		IpAddress:  &ip,
		RequestId:  &requestID,
		Reason:     &reason,
	}

	ctx := crudviewer.WithContext(context.Background(), permissionAuditTestViewer{})
	if err := writePermissionAuditLogEntity(ctx, entClient, logger, log); err != nil {
		t.Fatalf("writePermissionAuditLogEntity returned error: %v", err)
	}

	rows, err := client.PermissionAuditLog.Query().All(ctx)
	if err != nil {
		t.Fatalf("query permission audit logs failed: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 permission audit log, got %d", len(rows))
	}
	if rows[0].IPAddress == nil || *rows[0].IPAddress != ip {
		t.Fatalf("expected ip_address=%q, got %+v", ip, rows[0].IPAddress)
	}
	if rows[0].RequestID == nil || *rows[0].RequestID != requestID {
		t.Fatalf("expected request_id=%q, got %+v", requestID, rows[0].RequestID)
	}
	if rows[0].Reason == nil || *rows[0].Reason != reason {
		t.Fatalf("expected reason=%q, got %+v", reason, rows[0].Reason)
	}
}

func TestDefaultPermissionAuditReason(t *testing.T) {
	action := auditv1.PermissionAuditLog_DELETE
	log := &auditv1.PermissionAuditLog{Action: &action}
	if got := defaultPermissionAuditReason(log); got != "删除权限对象" {
		t.Fatalf("unexpected default reason: %q", got)
	}
}
