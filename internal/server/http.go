// Code generated from: xkit-template.
// generated at        2026-05-25 16:32:33 CST.

package server

import (
	"fmt"

	"github.com/chnxq/xkitpkg/app"
	httptransport "github.com/chnxq/xkitpkg/transport/http"
)

type GeneratedData interface{}

type ModuleHTTPRegistrar interface {
	RegisterHTTP(*httptransport.Server)
}

func NewHTTPServer(appCtx *app.AppCtx, services GeneratedHTTPServices, data GeneratedData, modules []ModuleHTTPRegistrar) (*httptransport.Server, error) {
	opts, err := HTTPServerOptions(appCtx, data)
	if err != nil {
		return nil, fmt.Errorf("http server options: %w", err)
	}
	srv := httptransport.NewServer(opts...)
	RegisterGeneratedHTTPServices(srv, services)
	for _, module := range modules {
		if module == nil {
			continue
		}
		module.RegisterHTTP(srv)
	}
	RegisterManualHTTPServices(srv, appCtx)
	RegisterManualHTTPServicesWithData(srv, appCtx, data)
	RegisterConfiguredHTTPHandlers(srv, appCtx)
	return srv, nil
}
