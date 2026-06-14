// Code generated from: xkit-template.
// generated at        2026-05-01 09:46:07 CST.

package server

import (
	"fmt"

	"github.com/chnxq/xkitpkg/app"
	serverutils "github.com/chnxq/xkitpkg/server_utils"
	grpctransport "github.com/chnxq/xkitpkg/transport/grpc"
)

func GRPCServerOptions(appCtx *app.AppCtx, data GeneratedData) ([]grpctransport.ServerOption, error) {
	cfg := grpcConfig(appCtx)
	opts := []grpctransport.ServerOption{
		grpctransport.Address(":9000"),
	}

	if cfg != nil {
		if cfg.GetNetwork() != "" {
			opts = append(opts, grpctransport.Network(cfg.GetNetwork()))
		}
		if cfg.GetAddr() != "" {
			opts = append(opts, grpctransport.Address(cfg.GetAddr()))
		}
		if cfg.GetTimeout() != nil {
			opts = append(opts, grpctransport.Timeout(cfg.GetTimeout().AsDuration()))
		}
		if cfg.GetTls() != nil {
			tlsConfig, err := serverutils.LoadServerTLSConfig(cfg.GetTls())
			if err != nil {
				return nil, fmt.Errorf("load grpc tls config: %w", err)
			}
			if tlsConfig != nil {
				opts = append(opts, grpctransport.TLSConfig(tlsConfig))
			}
		}
		if appCtx != nil && cfg.GetMiddleware() != nil {
			appCtx.NewLoggerHelper("transport/grpc").Debugf(
				"GRPC middleware config: logging=%t recovery=%t tracing=%t validate=%t metadata=%t breaker=%t limiter=%t",
				cfg.GetMiddleware().GetEnableLogging(),
				cfg.GetMiddleware().GetEnableRecovery(),
				cfg.GetMiddleware().GetEnableTracing(),
				cfg.GetMiddleware().GetEnableValidate(),
				cfg.GetMiddleware().GetEnableMetadata(),
				cfg.GetMiddleware().GetEnableCircuitBreaker(),
				cfg.GetMiddleware().GetLimiter() != nil,
			)
		}
	}

	middlewares := commonServerMiddlewares(appCtx, cfg.GetMiddleware())
	if authViewer := authViewerMiddleware(data); authViewer != nil {
		middlewares = append(middlewares, authViewer)
	}
	middlewares = append(middlewares, GRPCMiddlewares(appCtx)...)
	if len(middlewares) > 0 {
		opts = append(opts, grpctransport.Middleware(middlewares...))
	}
	return opts, nil
}
