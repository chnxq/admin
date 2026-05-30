// Code generated from: xkit-template.
// generated at        2026-05-25 16:32:33 CST.

package bootstrap

import (
	"github.com/chnxq/xkitpkg/app"
	"github.com/chnxq/xkitpkg/transport"

	"admin/internal/server"
)

func NewManualServers(appCtx *app.AppCtx) ([]transport.Server, func(), error) {
	servers := make([]transport.Server, 0, 2)

	asynqServer, err := server.NewAsynqServer(appCtx)
	if err != nil {
		return nil, func() {}, err
	}
	if asynqServer != nil {
		servers = append(servers, asynqServer)
	}

	sseServer, err := server.NewSSEServer(appCtx)
	if err != nil {
		return nil, func() {}, err
	}
	if sseServer != nil {
		servers = append(servers, sseServer)
	}

	return servers, func() {}, nil
}
