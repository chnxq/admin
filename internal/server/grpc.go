// Code generated from: xkit-template.
// generated at        2026-05-25 16:32:33 CST.

package server

import (
	"fmt"

	"github.com/chnxq/xkitpkg/app"
	grpctransport "github.com/chnxq/xkitpkg/transport/grpc"
)

func NewGRPCServer(appCtx *app.AppCtx, services GeneratedGRPCServices, data GeneratedData) (*grpctransport.Server, error) {
	opts, err := GRPCServerOptions(appCtx, data)
	if err != nil {
		return nil, fmt.Errorf("grpc server options: %w", err)
	}
	srv := grpctransport.NewServer(opts...)
	RegisterGeneratedGRPCServices(srv, services)
	//RegisterManualGRPCServices(srv, appCtx)
	return srv, nil
}
