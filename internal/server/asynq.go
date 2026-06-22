// Code generated from: xkit-template.
// generated at        2026-06-20 12:52:15 CST.

package server

import (
	"fmt"

	"github.com/chnxq/xkitpkg/app"
	conf "github.com/chnxq/xkitpkg/conf/v1"
	asynqtransport "github.com/chnxq/xkitpkg/transport/asynq"
)

func NewAsynqServer(appCtx *app.AppCtx) (*asynqtransport.Server, error) {
	if appCtx == nil {
		return nil, nil
	}
	cfg := appCtx.GetConfig()
	if cfg == nil || cfg.Server == nil || cfg.Server.Asynq == nil {
		return nil, nil
	}

	srv, err := newConfiguredAsynqServer(appCtx, cfg.Server.Asynq)
	if err != nil {
		return nil, err
	}
	if srv == nil {
		return nil, nil
	}
	//if err := RegisterManualAsynqSubscribers(srv, appCtx); err != nil {
	//	return nil, err
	//}
	return srv, nil
}

func newConfiguredAsynqServer(appCtx *app.AppCtx, cfg *conf.Server_Asynq) (srv *asynqtransport.Server, err error) {
	if cfg == nil {
		return nil, nil
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			srv = nil
			err = fmt.Errorf("new asynq server: %v", recovered)
		}
	}()

	opts, err := AsynqServerOptions(appCtx, cfg)
	if err != nil {
		return nil, err
	}
	return asynqtransport.NewServer(opts...), nil
}
