// Code generated from: xkit-template.
// generated at        2026-05-25 16:32:33 CST.

package bootstrap

import (
	"context"
	"fmt"

	databootstrap "admin/internal/data/bootstrap"
	taskruntime "admin/internal/task"
	"github.com/chnxq/xkitpkg/app"
	conf "github.com/chnxq/xkitpkg/conf/v1"
	"github.com/chnxq/xkitpkg/config"
	"github.com/chnxq/xkitpkg/transport"
)

type Options struct {
	Name       string
	Version    string
	BuildTime  string
	GitCommit  string
	ConfigPath string
}

func Initialize(ctx context.Context, opts Options) (*app.App, func(), error) {
	opts = normalizeOptions(opts)

	serverConfig, err := loadServerConfig(opts.ConfigPath)
	if err != nil {
		return nil, func() {}, err
	}

	cleanup := &CleanupStack{}
	appCtx, err := newAppContext(ctx, opts, serverConfig, cleanup)
	if err != nil {
		cleanup.Run()
		return nil, func() {}, err
	}

	if err := initDataResources(appCtx, cleanup); err != nil {
		cleanup.Run()
		return nil, func() {}, err
	}

	servers, err := newTransportServers(appCtx, cleanup)
	if err != nil {
		cleanup.Run()
		return nil, func() {}, err
	}

	return app.NewApp(appCtx, servers...), cleanup.Run, nil
}

func normalizeOptions(opts Options) Options {
	if opts.Name == "" {
		opts.Name = "Admin"
	}
	if opts.Version == "" {
		opts.Version = "dev"
	}
	if opts.ConfigPath == "" {
		opts.ConfigPath = "./configs"
	}
	return opts
}

func loadServerConfig(configPath string) (*conf.ServerConfig, error) {
	if err := config.LoadServerConfig(configPath); err != nil {
		return nil, fmt.Errorf("load server config: %w", err)
	}
	return config.GetServerConfig(), nil
}

func newAppContext(ctx context.Context, opts Options, serverConfig *conf.ServerConfig, cleanup *CleanupStack) (*app.AppCtx, error) {
	logger, loggerCleanup, err := NewLogger(serverConfig)
	cleanup.Add(loggerCleanup)
	if err != nil {
		return nil, err
	}

	registrar, registrarCleanup, err := NewRegistrar(serverConfig)
	cleanup.Add(registrarCleanup)
	if err != nil {
		return nil, err
	}

	appCtx := app.NewAppCtx(ctx, &conf.AppInfo{
		Project:   "admin",
		AppId:     "service/admin",
		Name:      opts.Name,
		Version:   opts.Version,
		BuildId:   fmt.Sprintf("%s-%s", opts.Version, opts.BuildTime),
		GitCommit: opts.GitCommit,
	}, serverConfig, logger, registrar)

	tracerCleanup, err := NewTracer(appCtx)
	cleanup.Add(tracerCleanup)
	if err != nil {
		return nil, err
	}
	registerRuntimeConfigAppliers(appCtx)
	return appCtx, nil
}

func initDataResources(appCtx *app.AppCtx, cleanup *CleanupStack) error {
	dataResources, dataCleanup, err := databootstrap.NewDataResources(appCtx)
	cleanup.Add(dataCleanup)
	if err != nil {
		return err
	}
	_ = dataResources
	return nil
}

func newTransportServers(appCtx *app.AppCtx, cleanup *CleanupStack) ([]transport.Server, error) {
	components, generatedCleanup, err := NewGeneratedComponents(appCtx)
	cleanup.Add(generatedCleanup)
	if err != nil {
		return nil, err
	}

	generatedServers, err := components.Servers(appCtx)
	if err != nil {
		return nil, err
	}

	manualBundle, manualCleanup, err := NewManualServers(appCtx)
	cleanup.Add(manualCleanup)
	if err != nil {
		return nil, err
	}
	if cleanupTaskRuntime, err := registerTaskRuntime(appCtx.AppContext(), components); err != nil {
		return nil, err
	} else {
		cleanup.Add(cleanupTaskRuntime)
	}

	servers := make([]transport.Server, 0, len(generatedServers)+len(manualBundle.Servers))
	servers = append(servers, generatedServers...)
	servers = append(servers, manualBundle.Servers...)
	return servers, nil
}

func registerTaskRuntime(ctx context.Context, components *GeneratedComponents) (func(), error) {
	if components == nil || components.Services == nil {
		return func() {}, nil
	}
	return taskruntime.RegisterServices(
		ctx,
		components.Services.Task.TaskScheduler(),
		components.Services.TaskGroup.TaskScheduler(),
		components.Services.Task.TaskLogger(),
	)
}
