// Code generated from: xkit-template.
// generated at        2026-06-20 12:52:15 CST.

package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/chnxq/xkitpkg/app"
	conf "github.com/chnxq/xkitpkg/conf/v1"
	kconfig "github.com/chnxq/xkitpkg/config"
	"github.com/chnxq/xkitpkg/tracer"
)

func registerRuntimeConfigAppliers(appCtx *app.AppCtx) {
	if appCtx == nil {
		return
	}
	kconfig.RegisterRuntimeConfigApplier("trace", func(cfg *conf.ServerConfig) (bool, error) {
		return applyRuntimeTraceConfig(appCtx, cfg)
	})
}

func applyRuntimeTraceConfig(appCtx *app.AppCtx, cfg *conf.ServerConfig) (bool, error) {
	if cfg == nil || cfg.GetTrace() == nil || appCtx == nil || appCtx.GetAppInfo() == nil {
		return false, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, shutdown, err := tracer.NewTracerProviderWithShutdown(ctx, cfg.GetTrace(), appCtx.GetAppInfo()); err != nil {
		return false, fmt.Errorf("reload tracer provider: %w", err)
	} else if shutdown != nil {
		_ = shutdown
	}
	return true, nil
}
