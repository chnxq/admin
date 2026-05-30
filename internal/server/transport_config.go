// Code generated from: xkit-template.
// generated at        2026-05-25 16:32:33 CST.

package server

import (
	"github.com/chnxq/xkitpkg/app"
	conf "github.com/chnxq/xkitpkg/conf/v1"
	"github.com/chnxq/xkitpkg/middleware"
	"github.com/chnxq/xkitpkg/middleware/logging"
	"github.com/chnxq/xkitpkg/middleware/metadata"
	"github.com/chnxq/xkitpkg/middleware/ratelimit"
	"github.com/chnxq/xkitpkg/middleware/recovery"
	"github.com/chnxq/xkitpkg/middleware/tracing"
	"github.com/chnxq/xkitpkg/middleware/validate"
)

func serverConfig(appCtx *app.AppCtx) *conf.ServerConfig {
	if appCtx == nil {
		return nil
	}
	return appCtx.GetConfig()
}

func restConfig(appCtx *app.AppCtx) *conf.Server_REST {
	cfg := serverConfig(appCtx)
	if cfg == nil || cfg.Server == nil {
		return nil
	}
	return cfg.Server.Rest
}

func grpcConfig(appCtx *app.AppCtx) *conf.Server_GRPC {
	cfg := serverConfig(appCtx)
	if cfg == nil || cfg.Server == nil {
		return nil
	}
	return cfg.Server.Grpc
}

func commonServerMiddlewares(appCtx *app.AppCtx, cfg *conf.Middleware) []middleware.Middleware {
	if cfg == nil {
		return nil
	}

	ms := make([]middleware.Middleware, 0, 7)
	if cfg.GetEnableRecovery() {
		ms = append(ms, recovery.Recovery())
	}
	if cfg.GetEnableTracing() {
		ms = append(ms, tracing.Server())
	}
	if cfg.GetEnableValidate() {
		ms = append(ms, validate.ProtoValidate())
	}
	if cfg.GetEnableMetadata() {
		ms = append(ms, metadata.Server())
	}
	if cfg.GetLimiter() != nil {
		ms = append(ms, ratelimit.Server())
	}
	if cfg.GetEnableCircuitBreaker() {
		ms = append(ms, serverCircuitBreakerMiddleware())
	}
	if cfg.GetEnableLogging() && appCtx != nil {
		ms = append(ms, logging.Server(appCtx.GetLogger()))
	}
	return ms
}
