// Code generated from: xkit-template.
// generated at        2026-05-01 09:46:07 CST.

package server

import (
	"sync"

	"github.com/chnxq/xkitpkg/app"
	conf "github.com/chnxq/xkitpkg/conf/v1"
	ssetransport "github.com/chnxq/xkitpkg/transport/sse"
)

var (
	sharedSSEServerMu sync.RWMutex
	sharedSSEServer   *ssetransport.Server
)

func setSharedSSEServer(srv *ssetransport.Server) {
	sharedSSEServerMu.Lock()
	defer sharedSSEServerMu.Unlock()
	sharedSSEServer = srv
}

func SharedSSEServer() *ssetransport.Server {
	sharedSSEServerMu.RLock()
	defer sharedSSEServerMu.RUnlock()
	return sharedSSEServer
}

func NewSSEServer(appCtx *app.AppCtx) (*ssetransport.Server, error) {
	if appCtx == nil {
		setSharedSSEServer(nil)
		return nil, nil
	}
	cfg := appCtx.GetConfig()
	if cfg == nil || cfg.Server == nil || cfg.Server.Sse == nil {
		setSharedSSEServer(nil)
		return nil, nil
	}

	srv, err := newConfiguredSSEServer(appCtx, cfg.Server.Sse)
	if err != nil {
		setSharedSSEServer(nil)
		return nil, err
	}
	if srv == nil {
		setSharedSSEServer(nil)
		return nil, nil
	}
	setSharedSSEServer(srv)
	//if err := RegisterManualSSEHandlers(srv, appCtx); err != nil {
	//	return nil, err
	//}
	return srv, nil
}

func newConfiguredSSEServer(appCtx *app.AppCtx, cfg *conf.Server_SSE) (*ssetransport.Server, error) {
	if cfg == nil {
		return nil, nil
	}
	opts, err := SSEServerOptions(appCtx, cfg)
	if err != nil {
		return nil, err
	}
	return ssetransport.NewServer(opts...), nil
}
