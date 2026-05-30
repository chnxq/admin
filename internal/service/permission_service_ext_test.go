package service

import (
	"context"
	"strings"
	"testing"

	permissionv1 "admin/api/gen/permission/v1"

	crudviewer "github.com/chnxq/x-crud/viewer"
)

type permissionServiceTestViewer struct {
	platform bool
	tenant   bool
	tenantID uint64
}

func (v permissionServiceTestViewer) UserID() uint64                    { return 1 }
func (v permissionServiceTestViewer) TenantID() uint64                  { return v.tenantID }
func (v permissionServiceTestViewer) OrgUnitID() uint64                 { return 0 }
func (v permissionServiceTestViewer) Permissions() []string             { return []string{"*"} }
func (v permissionServiceTestViewer) Roles() []string                   { return []string{"system"} }
func (v permissionServiceTestViewer) DataScope() []crudviewer.DataScope { return nil }
func (v permissionServiceTestViewer) TraceID() string                   { return "" }
func (v permissionServiceTestViewer) HasPermission(string, string) bool { return true }
func (v permissionServiceTestViewer) IsPlatformContext() bool           { return v.platform }
func (v permissionServiceTestViewer) IsTenantContext() bool             { return v.tenant }
func (v permissionServiceTestViewer) IsSystemContext() bool             { return v.platform && !v.tenant }
func (v permissionServiceTestViewer) ShouldAudit() bool                 { return false }

func TestPermissionServiceSyncPermissions_RejectsTenantContext(t *testing.T) {
	svc := &PermissionService{}
	ctx := crudviewer.WithContext(context.Background(), permissionServiceTestViewer{
		platform: true,
		tenant:   true,
		tenantID: 101,
	})

	err := svc.syncPermissions(ctx)
	if !permissionv1.IsForbidden(err) {
		t.Fatalf("expected forbidden error, got %v", err)
	}
}

func TestPermissionServiceSyncPermissions_AllowsPlatformContextPastTenantGate(t *testing.T) {
	svc := &PermissionService{}
	ctx := crudviewer.WithContext(context.Background(), permissionServiceTestViewer{
		platform: true,
		tenant:   false,
		tenantID: 0,
	})

	err := svc.syncPermissions(ctx)
	if err == nil {
		t.Fatalf("expected incomplete dependency error")
	}
	if permissionv1.IsForbidden(err) {
		t.Fatalf("expected platform context to pass tenant gate, got forbidden: %v", err)
	}
	if !strings.Contains(err.Error(), "dependencies are incomplete") {
		t.Fatalf("expected dependency error, got %v", err)
	}
}
