// Code generated from: xkit-template.
// generated at        2026-05-25 16:32:33 CST.

package server

import (
	"fmt"

	"github.com/chnxq/xkitpkg/app"
	grpctransport "github.com/chnxq/xkitpkg/transport/grpc"
	"google.golang.org/grpc"
)

type ModuleGRPCRegistrar interface {
	RegisterGRPC(grpc.ServiceRegistrar)
}

func NewGRPCServer(appCtx *app.AppCtx, services GeneratedGRPCServices, data GeneratedData, modules []ModuleGRPCRegistrar) (*grpctransport.Server, error) {
	opts, err := GRPCServerOptions(appCtx, data)
	if err != nil {
		return nil, fmt.Errorf("grpc server options: %w", err)
	}
	srv := grpctransport.NewServer(opts...)
	RegisterGeneratedGRPCServices(srv, services)
	for _, module := range modules {
		if module == nil {
			continue
		}
		module.RegisterGRPC(srv)
	}
	//RegisterManualGRPCServices(srv, appCtx)
	return srv, nil
}
