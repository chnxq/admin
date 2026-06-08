package auditlogcleanup

import (
	"context"
	"strings"
	"testing"
	"time"

	taskv1 "admin/api/gen/task/v1"
	taskruntime "admin/internal/task/runtime"
)

type fakeAPIAuditLogCleaner struct {
	affected int
	err      error
	before   time.Time
	tenantID *uint32
}

func (f *fakeAPIAuditLogCleaner) CleanupApiAuditLogsBefore(_ context.Context, tenantID *uint32, before time.Time) (int, error) {
	f.tenantID = tenantID
	f.before = before
	return f.affected, f.err
}

type fakeLoginAuditLogCleaner struct {
	affected int
	err      error
}

func (f *fakeLoginAuditLogCleaner) CleanupLoginAuditLogsBefore(_ context.Context, _ *uint32, _ time.Time) (int, error) {
	return f.affected, f.err
}

type fakePermissionAuditLogCleaner struct {
	affected int
	err      error
}

func (f *fakePermissionAuditLogCleaner) CleanupPermissionAuditLogsBefore(_ context.Context, _ *uint32, _ time.Time) (int, error) {
	return f.affected, f.err
}

func TestCleanupAuditLogsExecutor_ValidateRejectsInvalidJSON(t *testing.T) {
	executor := NewExecutor(nil)
	err := executor.Validate(context.Background(), taskruntime.ValidationRequest{Raw: "{bad json"})
	if err == nil || !strings.Contains(err.Error(), "invalid cleanup args") {
		t.Fatalf("expected invalid cleanup args error, got %v", err)
	}
}

func TestCleanupAuditLogsExecutor_ValidateRejectsUnsupportedTarget(t *testing.T) {
	executor := NewExecutor(nil)
	err := executor.Validate(context.Background(), taskruntime.ValidationRequest{
		Raw: `{"expireHours":24,"targets":["api","other"]}`,
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported cleanup target") {
		t.Fatalf("expected unsupported target error, got %v", err)
	}
}

func TestCleanupAuditLogsExecutor_ExecuteReturnsAggregatedResult(t *testing.T) {
	apiCleaner := &fakeAPIAuditLogCleaner{affected: 2}
	loginCleaner := &fakeLoginAuditLogCleaner{affected: 3}
	permissionCleaner := &fakePermissionAuditLogCleaner{affected: 5}
	executor := NewExecutor(NewAuditLogCleanupStore(
		apiCleaner,
		loginCleaner,
		permissionCleaner,
	))

	tenantID := uint32(101)
	result, err := executor.Execute(context.Background(), taskruntime.ExecuteRequest{
		Task: &taskv1.Task{
			TenantId: &tenantID,
			Args:     stringPtr(`{"expireHours":24,"targets":["api","login","permission"]}`),
		},
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !strings.Contains(result, `"totalDeleted":10`) {
		t.Fatalf("expected totalDeleted=10, got %s", result)
	}
	if apiCleaner.tenantID == nil || *apiCleaner.tenantID != 101 {
		t.Fatalf("expected tenant id 101 to be forwarded, got %+v", apiCleaner.tenantID)
	}
	if apiCleaner.before.IsZero() {
		t.Fatalf("expected cleanup cutoff time to be set")
	}
}

func TestParseCleanupAuditLogInput(t *testing.T) {
	payload, err := ParseInput(`{"expireHours":12,"targets":["api"]}`)
	if err != nil {
		t.Fatalf("ParseCleanupAuditLogInput failed: %v", err)
	}
	if payload.ExpireHours != 12 || len(payload.Targets) != 1 || payload.Targets[0] != "api" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func stringPtr(value string) *string {
	return &value
}
