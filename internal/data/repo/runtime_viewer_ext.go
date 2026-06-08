package repo

import (
	"context"

	crudviewer "github.com/chnxq/x-crud/viewer"
)

type runtimeViewerContext struct{}

func withRuntimeViewerContext(ctx context.Context) context.Context {
	if viewer, ok := crudviewer.FromContext(ctx); ok && viewer != nil {
		return ctx
	}
	return crudviewer.WithContext(ctx, runtimeViewerContext{})
}

func (runtimeViewerContext) UserID() uint64        { return 0 }
func (runtimeViewerContext) TenantID() uint64      { return 0 }
func (runtimeViewerContext) OrgUnitID() uint64     { return 0 }
func (runtimeViewerContext) Permissions() []string { return nil }
func (runtimeViewerContext) Roles() []string       { return []string{"system"} }
func (runtimeViewerContext) DataScope() []crudviewer.DataScope {
	return []crudviewer.DataScope{{ScopeType: crudviewer.ScopeTypeAll}}
}
func (runtimeViewerContext) TraceID() string { return "" }
func (runtimeViewerContext) HasPermission(_, _ string) bool {
	return true
}
func (runtimeViewerContext) IsPlatformContext() bool { return true }
func (runtimeViewerContext) IsTenantContext() bool   { return false }
func (runtimeViewerContext) IsSystemContext() bool   { return true }
func (runtimeViewerContext) ShouldAudit() bool       { return false }
