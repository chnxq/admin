package bootstrap

import (
	"context"

	"admin/internal/server"
	_ "admin/modules"
	modulehost "admin/shared/modulehost"

	"github.com/chnxq/xkitpkg/app"
	httptransport "github.com/chnxq/xkitpkg/transport/http"
	"google.golang.org/grpc"
)

type hostModuleRuntime struct {
	module       modulehost.Module
	moduleData   any
	moduleSrv    any
	registerHTTP func(*httptransport.Server)
	registerGRPC func(grpc.ServiceRegistrar)
}

func (r hostModuleRuntime) RegisterHTTP(srv *httptransport.Server) {
	if r.registerHTTP != nil {
		r.registerHTTP(srv)
	}
}

func (r hostModuleRuntime) RegisterGRPC(registrar grpc.ServiceRegistrar) {
	if r.registerGRPC != nil {
		r.registerGRPC(registrar)
	}
}

func loadHostModules(appCtx *app.AppCtx, cleanup *CleanupStack) ([]hostModuleRuntime, error) {
	if appCtx == nil {
		return nil, nil
	}

	hostModules := modulehost.GetRegisteredHostModules()
	runtimes := make([]hostModuleRuntime, 0, len(hostModules))
	for _, module := range hostModules {
		if module == nil {
			continue
		}
		runtime, err := initHostModule(module, appCtx, cleanup)
		if err != nil {
			return nil, err
		}
		if runtime != nil {
			runtimes = append(runtimes, *runtime)
		}
	}
	return runtimes, nil
}

func initHostModule(module modulehost.Module, appCtx *app.AppCtx, cleanup *CleanupStack) (*hostModuleRuntime, error) {
	moduleData, moduleCleanup, err := module.RegisterData(appCtx)
	if moduleCleanup != nil {
		cleanup.Add(moduleCleanup)
	}
	if err != nil {
		return nil, err
	}

	moduleServices := module.RegisterServices(appCtx, moduleData)
	return &hostModuleRuntime{
		module:     module,
		moduleData: moduleData,
		moduleSrv:  moduleServices,
		registerHTTP: func(srv *httptransport.Server) {
			module.RegisterHTTP(srv, moduleServices)
		},
		registerGRPC: func(registrar grpc.ServiceRegistrar) {
			module.RegisterGRPC(registrar, moduleServices)
		},
	}, nil
}

func SyncHostModuleResources(ctx context.Context, appCtx *app.AppCtx, runtimes []hostModuleRuntime) error {
	for _, runtime := range runtimes {
		if runtime.module == nil {
			continue
		}
		if err := runtime.module.SyncResources(ctx, appCtx, runtime.moduleData, nil); err != nil {
			return err
		}
	}
	return nil
}

func SeedHostModulesDefaultData(ctx context.Context, appCtx *app.AppCtx, runtimes []hostModuleRuntime) error {
	for _, runtime := range runtimes {
		if runtime.module == nil {
			continue
		}
		if err := runtime.module.SeedDefaultData(ctx, appCtx, runtime.moduleData); err != nil {
			return err
		}
	}
	return nil
}

func LoadHostModulesForSeed(appCtx *app.AppCtx) ([]hostModuleRuntime, error) {
	return loadHostModules(appCtx, nil)
}

func hostModuleHTTPRegistrars(runtimes []hostModuleRuntime) []server.ModuleHTTPRegistrar {
	if len(runtimes) == 0 {
		return nil
	}
	items := make([]server.ModuleHTTPRegistrar, 0, len(runtimes))
	for _, runtime := range runtimes {
		items = append(items, runtime)
	}
	return items
}

func hostModuleGRPCRegistrars(runtimes []hostModuleRuntime) []server.ModuleGRPCRegistrar {
	if len(runtimes) == 0 {
		return nil
	}
	items := make([]server.ModuleGRPCRegistrar, 0, len(runtimes))
	for _, runtime := range runtimes {
		items = append(items, runtime)
	}
	return items
}
