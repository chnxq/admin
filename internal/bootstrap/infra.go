// Code generated from: xkit-template.
// generated at        2026-06-20 12:52:15 CST.

package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/chnxq/xkitmod/log"
	kregistry "github.com/chnxq/xkitmod/registry"
	"github.com/chnxq/xkitpkg/app"
	conf "github.com/chnxq/xkitpkg/conf/v1"
	"github.com/chnxq/xkitpkg/logger"
	"github.com/chnxq/xkitpkg/registry"
	"github.com/chnxq/xkitpkg/tracer"
)

func NewLogger(serverConfig *conf.ServerConfig) (log.Logger, func(), error) {
	if serverConfig == nil || serverConfig.GetLogger() == nil {
		return log.DefaultLogger, func() {}, nil
	}
	serverLog := logger.NewLoggerProvider(serverConfig.GetLogger(), nil)
	if serverLog == nil {
		return log.DefaultLogger, func() {}, nil
	}
	return serverLog, func() {}, nil
}

func NewRegistrar(serverConfig *conf.ServerConfig) (kregistry.Registrar, func(), error) {
	if serverConfig == nil || serverConfig.Registry == nil || serverConfig.Registry.GetType() == "" {
		return nil, func() {}, nil
	}

	registrar, err := registry.NewRegistrar(serverConfig.Registry)
	if err != nil {
		return nil, func() {}, fmt.Errorf("new registrar: %w", err)
	}
	return registrar, func() {}, nil
}

func NewTracer(appCtx *app.AppCtx) (func(), error) {
	if appCtx == nil {
		return func() {}, nil
	}

	helper := appCtx.NewLoggerHelper("tracer")
	serverConfig := appCtx.GetConfig()
	if serverConfig == nil || serverConfig.GetTrace() == nil {
		helper.Warn("no trace config found")
		return func() {}, nil
	}

	_, shutdown, err := tracer.NewTracerProviderWithShutdown(
		appCtx.AppContext(),
		serverConfig.GetTrace(),
		appCtx.GetAppInfo(),
	)
	if err != nil {
		helper.Errorf("init tracer failed: %v", err)
		return func() {}, fmt.Errorf("init tracer: %w", err)
	}

	helper.Infof("Tracer initialized: %v", serverConfig.GetTrace())
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := shutdown(ctx); err != nil {
			helper.Errorf("shutdown tracer failed: %v", err)
		}
	}, nil
}
