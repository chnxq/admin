package bootstrap

import (
	"admin/internal/server"
	xdev "admin/modules/xdev"
	modulehost "admin/shared/modulehost"
	"github.com/chnxq/xkitpkg/app"
	httptransport "github.com/chnxq/xkitpkg/transport/http"
	"google.golang.org/grpc"
)

type hostModuleRuntime struct {
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

func registeredHostModules() []modulehost.Module {
	return []modulehost.Module{
		&xdev.Module{},
	}
}

func loadHostModules(appCtx *app.AppCtx, cleanup *CleanupStack) ([]hostModuleRuntime, error) {
	if appCtx == nil {
		return nil, nil
	}

	modules := registeredHostModules()
	runtimes := make([]hostModuleRuntime, 0, len(modules))
	for _, module := range modules {
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
		registerHTTP: func(srv *httptransport.Server) {
			module.RegisterHTTP(srv, moduleServices)
		},
		registerGRPC: func(registrar grpc.ServiceRegistrar) {
			module.RegisterGRPC(registrar, moduleServices)
		},
	}, nil
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
