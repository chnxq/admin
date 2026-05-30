package repo

import (
	"context"
	"testing"

	identityv1 "admin/api/gen/identity/v1"

	crudviewer "github.com/chnxq/x-crud/viewer"
)

type tenantScopeTestViewer struct {
	platform bool
	tenant   bool
	tenantID uint64
}

func (v tenantScopeTestViewer) UserID() uint64                    { return 1 }
func (v tenantScopeTestViewer) TenantID() uint64                  { return v.tenantID }
func (v tenantScopeTestViewer) OrgUnitID() uint64                 { return 0 }
func (v tenantScopeTestViewer) Permissions() []string             { return nil }
func (v tenantScopeTestViewer) Roles() []string                   { return nil }
func (v tenantScopeTestViewer) DataScope() []crudviewer.DataScope { return nil }
func (v tenantScopeTestViewer) TraceID() string                   { return "" }
func (v tenantScopeTestViewer) HasPermission(string, string) bool { return false }
func (v tenantScopeTestViewer) IsPlatformContext() bool           { return v.platform }
func (v tenantScopeTestViewer) IsTenantContext() bool             { return v.tenant }
func (v tenantScopeTestViewer) IsSystemContext() bool             { return v.platform && !v.tenant }
func (v tenantScopeTestViewer) ShouldAudit() bool                 { return false }

func TestViewerTenantID_ReturnsNilWithoutViewer(t *testing.T) {
	if got := viewerTenantID(context.Background()); got != nil {
		t.Fatalf("expected nil tenant id, got %v", *got)
	}
}

func TestViewerTenantID_ReturnsNilForPurePlatformContext(t *testing.T) {
	ctx := crudviewer.WithContext(context.Background(), tenantScopeTestViewer{
		platform: true,
		tenant:   false,
		tenantID: 101,
	})
	if got := viewerTenantID(ctx); got != nil {
		t.Fatalf("expected nil tenant id for platform context, got %v", *got)
	}
}

func TestViewerTenantID_ReturnsTenantForTenantContext(t *testing.T) {
	ctx := crudviewer.WithContext(context.Background(), tenantScopeTestViewer{
		platform: false,
		tenant:   true,
		tenantID: 101,
	})
	got := viewerTenantID(ctx)
	if got == nil || *got != 101 {
		t.Fatalf("expected tenant id 101, got %+v", got)
	}
}

func TestViewerTenantID_ReturnsTenantForPlatformTenantContext(t *testing.T) {
	ctx := crudviewer.WithContext(context.Background(), tenantScopeTestViewer{
		platform: true,
		tenant:   true,
		tenantID: 202,
	})
	got := viewerTenantID(ctx)
	if got == nil || *got != 202 {
		t.Fatalf("expected tenant id 202, got %+v", got)
	}
}

func TestEnsureTenantAccessible_AllowsPlatformContext(t *testing.T) {
	resourceTenantID := uint32(101)
	if err := ensureTenantAccessible(context.Background(), &resourceTenantID); err != nil {
		t.Fatalf("expected platform context to bypass tenant check, got %v", err)
	}
}

func TestEnsureTenantAccessible_AllowsSameTenant(t *testing.T) {
	ctx := crudviewer.WithContext(context.Background(), tenantScopeTestViewer{
		tenant:   true,
		tenantID: 101,
	})
	resourceTenantID := uint32(101)
	if err := ensureTenantAccessible(ctx, &resourceTenantID); err != nil {
		t.Fatalf("expected same tenant access to pass, got %v", err)
	}
}

func TestEnsureTenantAccessible_RejectsCrossTenant(t *testing.T) {
	ctx := crudviewer.WithContext(context.Background(), tenantScopeTestViewer{
		tenant:   true,
		tenantID: 101,
	})
	resourceTenantID := uint32(202)
	err := ensureTenantAccessible(ctx, &resourceTenantID)
	if !identityv1.IsForbidden(err) {
		t.Fatalf("expected forbidden error, got %v", err)
	}
}

func TestEnsureTenantAccessible_RejectsTenantAccessToGlobalResource(t *testing.T) {
	ctx := crudviewer.WithContext(context.Background(), tenantScopeTestViewer{
		tenant:   true,
		tenantID: 101,
	})
	if err := ensureTenantAccessible(ctx, nil); !identityv1.IsForbidden(err) {
		t.Fatalf("expected forbidden error for nil tenant resource, got %v", err)
	}
}

func TestEnsureHybridTenantAccessible_AllowsGlobalResourceForTenantViewer(t *testing.T) {
	ctx := crudviewer.WithContext(context.Background(), tenantScopeTestViewer{
		tenant:   true,
		tenantID: 101,
	})
	if err := ensureHybridTenantAccessible(ctx, nil); err != nil {
		t.Fatalf("expected tenant viewer to read global resource, got %v", err)
	}
}

func TestEnsureHybridTenantAccessible_AllowsZeroTenantResourceForTenantViewer(t *testing.T) {
	ctx := crudviewer.WithContext(context.Background(), tenantScopeTestViewer{
		tenant:   true,
		tenantID: 101,
	})
	resourceTenantID := uint32(0)
	if err := ensureHybridTenantAccessible(ctx, &resourceTenantID); err != nil {
		t.Fatalf("expected tenant viewer to read zero-tenant resource, got %v", err)
	}
}

func TestEnsureHybridTenantAccessible_RejectsOtherTenantResource(t *testing.T) {
	ctx := crudviewer.WithContext(context.Background(), tenantScopeTestViewer{
		tenant:   true,
		tenantID: 101,
	})
	resourceTenantID := uint32(202)
	if err := ensureHybridTenantAccessible(ctx, &resourceTenantID); !identityv1.IsForbidden(err) {
		t.Fatalf("expected forbidden error, got %v", err)
	}
}

func TestEnsureHybridTenantMutable_RejectsGlobalResourceForTenantViewer(t *testing.T) {
	ctx := crudviewer.WithContext(context.Background(), tenantScopeTestViewer{
		tenant:   true,
		tenantID: 101,
	})
	if err := ensureHybridTenantMutable(ctx, nil); !identityv1.IsForbidden(err) {
		t.Fatalf("expected forbidden error, got %v", err)
	}
}

func TestEnsureHybridTenantMutable_RejectsZeroTenantResourceForTenantViewer(t *testing.T) {
	ctx := crudviewer.WithContext(context.Background(), tenantScopeTestViewer{
		tenant:   true,
		tenantID: 101,
	})
	resourceTenantID := uint32(0)
	if err := ensureHybridTenantMutable(ctx, &resourceTenantID); !identityv1.IsForbidden(err) {
		t.Fatalf("expected forbidden error, got %v", err)
	}
}

func TestEnsureHybridTenantMutable_AllowsSameTenantResource(t *testing.T) {
	ctx := crudviewer.WithContext(context.Background(), tenantScopeTestViewer{
		tenant:   true,
		tenantID: 101,
	})
	resourceTenantID := uint32(101)
	if err := ensureHybridTenantMutable(ctx, &resourceTenantID); err != nil {
		t.Fatalf("expected same-tenant mutation to pass, got %v", err)
	}
}

func TestEnsurePlatformOnlyMutable_AllowsPlatformContext(t *testing.T) {
	if err := ensurePlatformOnlyMutable(context.Background()); err != nil {
		t.Fatalf("expected platform context to pass, got %v", err)
	}
}

func TestEnsurePlatformOnlyMutable_RejectsTenantContext(t *testing.T) {
	ctx := crudviewer.WithContext(context.Background(), tenantScopeTestViewer{
		tenant:   true,
		tenantID: 101,
	})
	if err := ensurePlatformOnlyMutable(ctx); !identityv1.IsForbidden(err) {
		t.Fatalf("expected forbidden error, got %v", err)
	}
}

func TestResolveCreateTenantID_AllowsPlatformRequestedTenant(t *testing.T) {
	requested := uint32(303)
	got, err := resolveCreateTenantID(context.Background(), &requested)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got == nil || *got != 303 {
		t.Fatalf("expected requested tenant id 303, got %+v", got)
	}
}

func TestResolveCreateTenantID_DefaultsToViewerTenant(t *testing.T) {
	ctx := crudviewer.WithContext(context.Background(), tenantScopeTestViewer{
		tenant:   true,
		tenantID: 101,
	})
	got, err := resolveCreateTenantID(ctx, nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got == nil || *got != 101 {
		t.Fatalf("expected viewer tenant id 101, got %+v", got)
	}
}

func TestResolveCreateTenantID_RejectsCrossTenantRequest(t *testing.T) {
	ctx := crudviewer.WithContext(context.Background(), tenantScopeTestViewer{
		tenant:   true,
		tenantID: 101,
	})
	requested := uint32(202)
	if _, err := resolveCreateTenantID(ctx, &requested); !identityv1.IsForbidden(err) {
		t.Fatalf("expected forbidden error, got %v", err)
	}
}
