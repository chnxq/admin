// Code generated from: xkit-template.
// generated at        2026-05-01 09:46:07 CST.

package server

import (
	"context"

	authenticationv1 "admin/api/gen/authentication/v1"
	crudviewer "github.com/chnxq/x-crud/viewer"
	"github.com/chnxq/xkitpkg/app"
	"github.com/chnxq/xkitpkg/middleware"
)

func HTTPMiddlewares(appCtx *app.AppCtx) []middleware.Middleware {
	_ = appCtx
	return []middleware.Middleware{
		defaultViewerMiddleware(),
	}
}

func GRPCMiddlewares(appCtx *app.AppCtx) []middleware.Middleware {
	_ = appCtx
	return []middleware.Middleware{
		defaultViewerMiddleware(),
	}
}

func defaultViewerMiddleware() middleware.Middleware {
	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			if viewer, ok := crudviewer.FromContext(ctx); ok && viewer != nil {
				return next(ctx, req)
			}
			if isProtectedServerRequest(ctx) {
				return nil, authenticationv1.ErrorUnauthorized("authentication required")
			}
			ctx = crudviewer.WithContext(ctx, defaultViewerContext{})
			return next(ctx, req)
		}
	}
}

type defaultViewerContext struct{}

func (defaultViewerContext) UserID() uint64        { return 0 }
func (defaultViewerContext) TenantID() uint64      { return 0 }
func (defaultViewerContext) OrgUnitID() uint64     { return 0 }
func (defaultViewerContext) Permissions() []string { return nil }
func (defaultViewerContext) Roles() []string       { return []string{"system"} }
func (defaultViewerContext) DataScope() []crudviewer.DataScope {
	return []crudviewer.DataScope{{ScopeType: crudviewer.ScopeTypeAll}}
}
func (defaultViewerContext) TraceID() string { return "" }
func (defaultViewerContext) HasPermission(_, _ string) bool {
	return false
}
func (defaultViewerContext) IsPlatformContext() bool { return true }
func (defaultViewerContext) IsTenantContext() bool   { return false }
func (defaultViewerContext) IsSystemContext() bool   { return true }
func (defaultViewerContext) ShouldAudit() bool       { return false }
