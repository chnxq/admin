// Code generated from: xkit-template.
// generated at        2026-05-25 16:32:33 CST.

package server

import (
	"fmt"

	"github.com/chnxq/xkitpkg/app"
	conf "github.com/chnxq/xkitpkg/conf/v1"
	ssetransport "github.com/chnxq/xkitpkg/transport/sse"
)

func SSEServerOptions(appCtx *app.AppCtx, cfg *conf.Server_SSE) ([]ssetransport.ServerOption, error) {
	opts, err := sseConfigOptions(cfg)
	if err != nil {
		return nil, err
	}
	if appCtx != nil {
		logger := appCtx.NewLoggerHelper("sse-server")
		opts = append(opts,
			ssetransport.WithSubscriberFunction(func(streamID ssetransport.StreamID, _ *ssetransport.Subscriber) {
				logger.Infof("subscriber [%s] connected", streamID)
			}),
			ssetransport.WithUnSubscriberFunction(func(streamID ssetransport.StreamID, _ *ssetransport.Subscriber) {
				logger.Infof("subscriber [%s] disconnected", streamID)
			}),
		)
	}
	return opts, nil
}

func sseConfigOptions(cfg *conf.Server_SSE) ([]ssetransport.ServerOption, error) {
	if cfg == nil {
		return nil, nil
	}
	opts := make([]ssetransport.ServerOption, 0)
	if cfg.GetNetwork() != "" {
		opts = append(opts, ssetransport.WithNetwork(cfg.GetNetwork()))
	}
	if cfg.GetAddr() != "" {
		opts = append(opts, ssetransport.WithAddress(cfg.GetAddr()))
	}
	if cfg.GetPath() != "" {
		opts = append(opts, ssetransport.WithPath(cfg.GetPath()))
	}
	if cfg.GetCodec() != "" {
		opts = append(opts, ssetransport.WithCodec(cfg.GetCodec()))
	}
	if cfg.GetTimeout() != nil {
		opts = append(opts, ssetransport.WithTimeout(cfg.GetTimeout().AsDuration()))
	}
	if cfg.GetEventTtl() != nil {
		opts = append(opts, ssetransport.WithEventTTL(cfg.GetEventTtl().AsDuration()))
	}
	if cfg.GetTls() != nil {
		tlsConfig, err := loadServerTLSConfig(cfg.GetTls())
		if err != nil {
			return nil, fmt.Errorf("load sse tls config: %w", err)
		}
		if tlsConfig != nil {
			opts = append(opts, ssetransport.WithTLSConfig(tlsConfig))
		}
	}
	opts = append(opts,
		ssetransport.WithAutoStream(cfg.GetAutoStream()),
		ssetransport.WithAutoReply(cfg.GetAutoReply()),
		ssetransport.WithSplitData(cfg.GetSplitData()),
		ssetransport.WithEncodeBase64(cfg.GetEncodeBase64()),
	)
	return opts, nil
}
