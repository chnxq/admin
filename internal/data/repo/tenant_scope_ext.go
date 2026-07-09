package repo

import (
	"context"
	"strings"

	identityv1 "admin/api/gen/identity/v1"
	"admin/internal/data/ent"
	"admin/internal/data/ent/tenant"

	crudviewer "github.com/chnxq/x-crud/viewer"
)

const (
	platformTenantID   uint32 = 0
	platformTenantName        = "XAdmin平台"
)

func displayTenantName(tenantID *uint32, tenantName *string) *string {
	if tenantID != nil && *tenantID == platformTenantID {
		return tenantNamePtr(platformTenantName)
	}
	if tenantName == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*tenantName)
	if trimmed == "" {
		return nil
	}
	return tenantNamePtr(trimmed)
}

func collectTenantIDs(values ...*uint32) []uint32 {
	if len(values) == 0 {
		return nil
	}
	result := make([]uint32, 0, len(values))
	seen := make(map[uint32]struct{}, len(values))
	for _, value := range values {
		if value == nil || *value == platformTenantID {
			continue
		}
		if _, ok := seen[*value]; ok {
			continue
		}
		seen[*value] = struct{}{}
		result = append(result, *value)
	}
	return result
}

func loadTenantNameMap(ctx context.Context, client *ent.Client, tenantIDs []uint32) map[uint32]string {
	if client == nil || len(tenantIDs) == 0 {
		return nil
	}
	rows, err := client.Tenant.Query().
		Where(tenant.IDIn(tenantIDs...)).
		All(ctx)
	if err != nil {
		return nil
	}
	result := make(map[uint32]string, len(rows))
	for _, row := range rows {
		if row == nil || row.Name == nil {
			continue
		}
		result[row.ID] = *row.Name
	}
	return result
}

func tenantNameFromMap(tenantNameMap map[uint32]string, tenantID *uint32) *string {
	if tenantID == nil {
		return nil
	}
	if *tenantID == platformTenantID {
		return tenantNamePtr(platformTenantName)
	}
	if len(tenantNameMap) == 0 {
		return nil
	}
	name, ok := tenantNameMap[*tenantID]
	if !ok || name == "" {
		return nil
	}
	return tenantNamePtr(name)
}

func resolvedTenantDisplayName(tenantNameMap map[uint32]string, tenantID uint32) string {
	if tenantID == platformTenantID {
		return platformTenantName
	}
	if len(tenantNameMap) == 0 {
		return ""
	}
	tenantName, ok := tenantNameMap[tenantID]
	if !ok {
		return ""
	}
	return tenantName
}

func tenantNamePtr(value string) *string {
	return &value
}

func tenantIDPtr(value uint32) *uint32 {
	return &value
}

func viewerTenantID(ctx context.Context) *uint32 {
	viewer, ok := crudviewer.FromContext(ctx)
	if !ok || viewer == nil {
		return nil
	}
	if viewer.IsPlatformContext() && !viewer.IsTenantContext() {
		return nil
	}
	tenantID := uint32(viewer.TenantID())
	if tenantID == platformTenantID {
		return nil
	}
	return &tenantID
}

func ensureTenantAccessible(ctx context.Context, resourceTenantID *uint32) error {
	viewerTenant := viewerTenantID(ctx)
	if viewerTenant == nil {
		return nil
	}
	if resourceTenantID == nil || *resourceTenantID != *viewerTenant {
		return identityv1.ErrorForbidden("cross-tenant access is forbidden")
	}
	return nil
}

func viewerWriteTenantID(ctx context.Context) *uint32 {
	viewer, ok := crudviewer.FromContext(ctx)
	if !ok || viewer == nil {
		return tenantIDPtr(platformTenantID)
	}
	tenantID := uint32(viewer.TenantID())
	return &tenantID
}

func ensureTenantMutable(ctx context.Context, resourceTenantID *uint32) error {
	viewerTenant := viewerWriteTenantID(ctx)
	if viewerTenant == nil {
		return identityv1.ErrorForbidden("cross-tenant access is forbidden")
	}
	if *viewerTenant == platformTenantID && resourceTenantID == nil {
		return nil
	}
	if resourceTenantID == nil || *resourceTenantID != *viewerTenant {
		return identityv1.ErrorForbidden("cross-tenant access is forbidden")
	}
	return nil
}

func applyHybridTenantScope[T interface{ Where(...func(*T)) }](builder any, tenantID *uint32) {
	// placeholder to keep file focused on tenant policy helpers
}

func hybridTenantVisible(resourceTenantID *uint32, viewerTenant *uint32) bool {
	if viewerTenant == nil {
		return true
	}
	if resourceTenantID == nil || *resourceTenantID == 0 {
		return true
	}
	return *resourceTenantID == *viewerTenant
}

func ensureHybridTenantAccessible(ctx context.Context, resourceTenantID *uint32) error {
	viewerTenant := viewerTenantID(ctx)
	if hybridTenantVisible(resourceTenantID, viewerTenant) {
		return nil
	}
	return identityv1.ErrorForbidden("cross-tenant access is forbidden")
}

func ensureHybridTenantMutable(ctx context.Context, resourceTenantID *uint32) error {
	viewerTenant := viewerTenantID(ctx)
	if viewerTenant == nil {
		return nil
	}
	if resourceTenantID != nil && *resourceTenantID == *viewerTenant {
		return nil
	}
	return identityv1.ErrorForbidden("cross-tenant access is forbidden")
}

func ensurePlatformOnlyMutable(ctx context.Context) error {
	if viewerTenantID(ctx) != nil {
		return identityv1.ErrorForbidden("cross-tenant access is forbidden")
	}
	return nil
}

func resolveCreateTenantID(ctx context.Context, requestedTenantID *uint32) (*uint32, error) {
	viewerTenant := viewerTenantID(ctx)
	if viewerTenant == nil {
		return requestedTenantID, nil
	}
	if requestedTenantID == nil {
		return viewerTenant, nil
	}
	if *requestedTenantID != *viewerTenant {
		return nil, identityv1.ErrorForbidden("cross-tenant access is forbidden")
	}
	return requestedTenantID, nil
}
