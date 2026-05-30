// Code generated from: xkit-template.
// generated at        2026-05-25 16:32:33 CST.

package server

import (
	"fmt"

	"github.com/chnxq/xkitpkg/app"
	httptransport "github.com/chnxq/xkitpkg/transport/http"
)

type GeneratedData interface{}

func NewHTTPServer(appCtx *app.AppCtx, services GeneratedHTTPServices, data GeneratedData) (*httptransport.Server, error) {
	opts, err := HTTPServerOptions(appCtx, data)
	if err != nil {
		return nil, fmt.Errorf("http server options: %w", err)
	}
	srv := httptransport.NewServer(opts...)
	RegisterGeneratedHTTPServices(srv, services)
	RegisterManualHTTPServices(srv, appCtx)
	RegisterManualHTTPServicesWithData(srv, appCtx, data)
	RegisterConfiguredHTTPHandlers(srv, appCtx)
	return srv, nil
}
