// Code generated from: xkit-template.
// generated at        2026-05-25 16:32:33 CST.

package server

import (
	"github.com/chnxq/xkitpkg/app"
	conf "github.com/chnxq/xkitpkg/conf/v1"
	"github.com/chnxq/xkitpkg/middleware"
	serverutils "github.com/chnxq/xkitpkg/server_utils"
)

func serverConfig(appCtx *app.AppCtx) *conf.ServerConfig {
	return serverutils.ServerConfig(appCtx)
}

func restConfig(appCtx *app.AppCtx) *conf.Server_REST {
	return serverutils.RestConfig(appCtx)
}

func grpcConfig(appCtx *app.AppCtx) *conf.Server_GRPC {
	return serverutils.GrpcConfig(appCtx)
}

func commonServerMiddlewares(appCtx *app.AppCtx, cfg *conf.Middleware) []middleware.Middleware {
	return serverutils.CommonServerMiddlewares(appCtx, cfg)
}
