// Code generated from: xkit-template.
// generated at        2026-05-25 16:32:33 CST.

package bootstrap

import (
	"github.com/chnxq/xkitpkg/app"
	"github.com/chnxq/xkitpkg/transport"

	"admin/internal/server"
)

type ManualServers struct {
	Servers []transport.Server
	Asynq   any
}

func NewManualServers(appCtx *app.AppCtx) (*ManualServers, func(), error) {
	bundle := &ManualServers{Servers: make([]transport.Server, 0, 2)}

	asynqServer, err := server.NewAsynqServer(appCtx)
	if err != nil {
		return nil, func() {}, err
	}
	if asynqServer != nil {
		bundle.Servers = append(bundle.Servers, asynqServer)
		bundle.Asynq = asynqServer
	}

	sseServer, err := server.NewSSEServer(appCtx)
	if err != nil {
		return nil, func() {}, err
	}
	if sseServer != nil {
		bundle.Servers = append(bundle.Servers, sseServer)
	}

	return bundle, func() {}, nil
}
