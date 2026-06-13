// Code generated from: xkit-template.
// generated at        2026-05-01 09:46:07 CST.

package server

import (
	"fmt"

	swaggerUI "github.com/chnxq/x-swagger"
	"github.com/chnxq/xkitpkg/app"
	conf "github.com/chnxq/xkitpkg/conf/v1"
	httptransport "github.com/chnxq/xkitpkg/transport/http"
	httppprof "github.com/chnxq/xkitpkg/transport/http/pprof"
	"github.com/gorilla/handlers"

	"admin/cmd/server/assets"
)

func HTTPServerOptions(appCtx *app.AppCtx, data GeneratedData) ([]httptransport.ServerOption, error) {
	cfg := restConfig(appCtx)
	opts := []httptransport.ServerOption{
		httptransport.Address(":8000"),
	}

	if cfg != nil {
		if cfg.GetNetwork() != "" {
			opts = append(opts, httptransport.Network(cfg.GetNetwork()))
		}
		if cfg.GetAddr() != "" {
			opts = append(opts, httptransport.Address(cfg.GetAddr()))
		}
		if cfg.GetTimeout() != nil {
			opts = append(opts, httptransport.Timeout(cfg.GetTimeout().AsDuration()))
		}
		if cfg.GetTls() != nil {
			tlsConfig, err := loadServerTLSConfig(cfg.GetTls())
			if err != nil {
				return nil, fmt.Errorf("load rest tls config: %w", err)
			}
			if tlsConfig != nil {
				opts = append(opts, httptransport.TLSConfig(tlsConfig))
			}
		}
	}

	var cfgMiddleware *conf.Middleware
	if cfg != nil {
		cfgMiddleware = cfg.GetMiddleware()
		if appCtx != nil {
			appCtx.NewLoggerHelper("transport/http").Debugf(
				"REST middleware config: logging=%t recovery=%t tracing=%t validate=%t metadata=%t breaker=%t limiter=%t db_logging=%t",
				cfgMiddleware.GetEnableLogging(),
				cfgMiddleware.GetEnableRecovery(),
				cfgMiddleware.GetEnableTracing(),
				cfgMiddleware.GetEnableValidate(),
				cfgMiddleware.GetEnableMetadata(),
				cfgMiddleware.GetEnableCircuitBreaker(),
				cfgMiddleware.GetLimiter() != nil,
				cfg.GetEnableDbLogging(),
			)
		}
	}
	middlewares := commonServerMiddlewares(appCtx, cfgMiddleware)
	if authViewer := authViewerMiddleware(data); authViewer != nil {
		middlewares = append(middlewares, authViewer)
	}
	middlewares = append(middlewares, HTTPMiddlewares(appCtx)...)
	if cfg != nil && cfg.GetEnableDbLogging() {
		middlewares = append(middlewares, databaseLoggingMiddleware(data))
	}
	if len(middlewares) > 0 {
		opts = append(opts, httptransport.Middleware(middlewares...))
	}
	if filters := restConfigFilters(cfg); len(filters) > 0 {
		opts = append(opts, httptransport.Filter(filters...))
	}
	return opts, nil
}

func RegisterConfiguredHTTPHandlers(srv *httptransport.Server, appCtx *app.AppCtx) {
	if srv == nil {
		return
	}
	cfg := restConfig(appCtx)
	if cfg == nil {
		return
	}

	if cfg.GetEnablePprof() {
		handler := httppprof.NewHandler()
		srv.Handle("/debug/pprof", handler)
		srv.HandlePrefix("/debug/pprof/", handler)
	}

	if cfg.GetEnableSwagger() && len(assets.OpenApiData) > 0 {
		swaggerUI.RegisterSwaggerUIServerWithOption(
			srv,
			swaggerUI.WithTitle("API文档"),
			swaggerUI.WithBasePath("/docs/"),
			swaggerUI.WithMemoryData(assets.OpenApiData, "yaml"),
		)
	}
}

func restConfigFilters(cfg *conf.Server_REST) []httptransport.FilterFunc {
	if cfg == nil || cfg.GetCors() == nil {
		return nil
	}

	cors := cfg.GetCors()
	return []httptransport.FilterFunc{
		handlers.CORS(
			handlers.AllowedHeaders(cors.GetHeaders()),
			handlers.AllowedMethods(cors.GetMethods()),
			handlers.AllowedOrigins(cors.GetOrigins()),
		),
	}
}
