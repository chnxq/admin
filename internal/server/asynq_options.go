// Code generated from: xkit-template.
// generated at        2026-05-25 16:32:33 CST.

package server

import (
	"fmt"

	"github.com/chnxq/xkitpkg/app"
	conf "github.com/chnxq/xkitpkg/conf/v1"
	serverutils "github.com/chnxq/xkitpkg/server_utils"
	asynqtransport "github.com/chnxq/xkitpkg/transport/asynq"
)

func AsynqServerOptions(appCtx *app.AppCtx, cfg *conf.Server_Asynq) ([]asynqtransport.ServerOption, error) {
	opts, err := asynqConfigOptions(cfg)
	if err != nil {
		return nil, err
	}
	return opts, nil
}

func asynqConfigOptions(cfg *conf.Server_Asynq) ([]asynqtransport.ServerOption, error) {
	if cfg == nil {
		return nil, nil
	}

	opts := make([]asynqtransport.ServerOption, 0)
	if cfg.GetCodec() != "" {
		opts = append(opts, asynqtransport.WithCodec(cfg.GetCodec()))
	}

	switch cfg.RedisClientOpts.(type) {
	case *conf.Server_Asynq_RedisOpt:
		opts = append(opts, asynqRedisOptions(cfg)...)
	case *conf.Server_Asynq_RedisClusterOpt:
		opts = append(opts, asynqRedisClusterOptions(cfg)...)
	case *conf.Server_Asynq_RedisFailoverOpt:
		opts = append(opts, asynqRedisFailoverOptions(cfg)...)
	case *conf.Server_Asynq_Uri:
		if cfg.GetUri() != "" {
			opts = append(opts, asynqtransport.WithRedisURI(cfg.GetUri()))
		}
	}

	if cfg.GetLocation() != "" {
		opts = append(opts, asynqtransport.WithLocation(cfg.GetLocation()))
	}
	if cfg.Concurrency != nil {
		opts = append(opts, asynqtransport.WithConcurrency(cfg.GetConcurrency()))
	}
	if cfg.GroupMaxSize != nil {
		opts = append(opts, asynqtransport.WithGroupMaxSize(cfg.GetGroupMaxSize()))
	}
	if len(cfg.GetQueues()) > 0 {
		opts = append(opts, asynqtransport.WithQueues(cfg.GetQueues()))
	}
	if cfg.EnableGracefullyShutdown != nil {
		opts = append(opts, asynqtransport.WithGracefullyShutdown(cfg.GetEnableGracefullyShutdown()))
	}
	if cfg.EnableStrictPriority != nil {
		opts = append(opts, asynqtransport.WithStrictPriority(cfg.GetEnableStrictPriority()))
	}
	if cfg.GetShutdownTimeout() != nil {
		opts = append(opts, asynqtransport.WithShutdownTimeout(cfg.GetShutdownTimeout().AsDuration()))
	}
	if cfg.GetTaskCheckInterval() != nil {
		opts = append(opts, asynqtransport.WithTaskCheckInterval(cfg.GetTaskCheckInterval().AsDuration()))
	}
	if cfg.GetHealthCheckInterval() != nil {
		opts = append(opts, asynqtransport.WithHealthCheckInterval(cfg.GetHealthCheckInterval().AsDuration()))
	}
	if cfg.GetDelayedTaskCheckInterval() != nil {
		opts = append(opts, asynqtransport.WithDelayedTaskCheckInterval(cfg.GetDelayedTaskCheckInterval().AsDuration()))
	}
	if cfg.GetGroupGracePeriod() != nil {
		opts = append(opts, asynqtransport.WithGroupGracePeriod(cfg.GetGroupGracePeriod().AsDuration()))
	}
	if cfg.GetGroupMaxDelay() != nil {
		opts = append(opts, asynqtransport.WithGroupMaxDelay(cfg.GetGroupMaxDelay().AsDuration()))
	}
	if cfg.GetJanitorInterval() != nil {
		opts = append(opts, asynqtransport.WithJanitorInterval(cfg.GetJanitorInterval().AsDuration()))
	}
	if cfg.JanitorBatchSize != nil {
		opts = append(opts, asynqtransport.WithJanitorBatchSize(cfg.GetJanitorBatchSize()))
	}
	if cfg.GetTls() != nil {
		tlsConfig, err := serverutils.LoadClientTLSConfig(cfg.GetTls())
		if err != nil {
			return nil, fmt.Errorf("load asynq tls config: %w", err)
		}
		if tlsConfig != nil {
			opts = append(opts, asynqtransport.WithTLSConfig(tlsConfig))
		}
	}
	return opts, nil
}

func asynqRedisOptions(cfg *conf.Server_Asynq) []asynqtransport.ServerOption {
	redisOpt := cfg.GetRedisOpt()
	if redisOpt == nil {
		return nil
	}
	opts := []asynqtransport.ServerOption{asynqtransport.WithRedisType(asynqtransport.RedisTypeSingle)}
	if redisOpt.GetNetwork() != "" {
		network := redisOpt.GetNetwork()
		opts = append(opts, asynqtransport.WithNetwork(&network))
	}
	if redisOpt.GetAddr() != "" {
		opts = append(opts, asynqtransport.WithRedisAddress(redisOpt.GetAddr()))
	}
	if redisOpt.GetUsername() != "" || redisOpt.GetPassword() != "" {
		opts = append(opts, asynqtransport.WithRedisAuth(redisOpt.GetUsername(), redisOpt.GetPassword()))
	}
	opts = append(opts, asynqtransport.WithRedisDB(redisOpt.GetDb()))
	if redisOpt.GetPoolSize() > 0 {
		opts = append(opts, asynqtransport.WithRedisPoolSize(redisOpt.GetPoolSize()))
	}
	if cfg.GetDialTimeout() != nil {
		opts = append(opts, asynqtransport.WithDialTimeout(cfg.GetDialTimeout().AsDuration()))
	}
	if cfg.GetReadTimeout() != nil {
		opts = append(opts, asynqtransport.WithReadTimeout(cfg.GetReadTimeout().AsDuration()))
	}
	if cfg.GetWriteTimeout() != nil {
		opts = append(opts, asynqtransport.WithWriteTimeout(cfg.GetWriteTimeout().AsDuration()))
	}
	return opts
}

func asynqRedisClusterOptions(cfg *conf.Server_Asynq) []asynqtransport.ServerOption {
	clusterOpt := cfg.GetRedisClusterOpt()
	if clusterOpt == nil {
		return nil
	}
	opts := []asynqtransport.ServerOption{asynqtransport.WithRedisType(asynqtransport.RedisTypeCluster)}
	if len(clusterOpt.GetAddrs()) > 0 {
		opts = append(opts, asynqtransport.WithRedisAddresses(clusterOpt.GetAddrs()))
	}
	if clusterOpt.MaxRedirects != nil {
		maxRedirects := clusterOpt.GetMaxRedirects()
		opts = append(opts, asynqtransport.WithMaxRedirects(&maxRedirects))
	}
	if clusterOpt.GetUsername() != "" || clusterOpt.GetPassword() != "" {
		opts = append(opts, asynqtransport.WithRedisAuth(clusterOpt.GetUsername(), clusterOpt.GetPassword()))
	}
	if cfg.GetDialTimeout() != nil {
		opts = append(opts, asynqtransport.WithDialTimeout(cfg.GetDialTimeout().AsDuration()))
	}
	if cfg.GetReadTimeout() != nil {
		opts = append(opts, asynqtransport.WithReadTimeout(cfg.GetReadTimeout().AsDuration()))
	}
	if cfg.GetWriteTimeout() != nil {
		opts = append(opts, asynqtransport.WithWriteTimeout(cfg.GetWriteTimeout().AsDuration()))
	}
	return opts
}

func asynqRedisFailoverOptions(cfg *conf.Server_Asynq) []asynqtransport.ServerOption {
	failoverOpt := cfg.GetRedisFailoverOpt()
	if failoverOpt == nil {
		return nil
	}
	opts := []asynqtransport.ServerOption{asynqtransport.WithRedisType(asynqtransport.RedisTypeSentinel)}
	if len(failoverOpt.GetSentinelAddrs()) > 0 {
		opts = append(opts, asynqtransport.WithRedisAddresses(failoverOpt.GetSentinelAddrs()))
	}
	if failoverOpt.GetMasterName() != "" {
		masterName := failoverOpt.GetMasterName()
		opts = append(opts, asynqtransport.WithMasterName(&masterName))
	}
	if failoverOpt.GetSentinelUsername() != "" || failoverOpt.GetSentinelPassword() != "" {
		sentinelUsername := failoverOpt.GetSentinelUsername()
		sentinelPassword := failoverOpt.GetSentinelPassword()
		opts = append(opts, asynqtransport.WithSentinelAuth(&sentinelUsername, &sentinelPassword))
	}
	if failoverOpt.GetUsername() != "" || failoverOpt.GetPassword() != "" {
		opts = append(opts, asynqtransport.WithRedisAuth(failoverOpt.GetUsername(), failoverOpt.GetPassword()))
	}
	if failoverOpt.GetPoolSize() > 0 {
		opts = append(opts, asynqtransport.WithRedisPoolSize(failoverOpt.GetPoolSize()))
	}
	if cfg.GetDialTimeout() != nil {
		opts = append(opts, asynqtransport.WithDialTimeout(cfg.GetDialTimeout().AsDuration()))
	}
	if cfg.GetReadTimeout() != nil {
		opts = append(opts, asynqtransport.WithReadTimeout(cfg.GetReadTimeout().AsDuration()))
	}
	if cfg.GetWriteTimeout() != nil {
		opts = append(opts, asynqtransport.WithWriteTimeout(cfg.GetWriteTimeout().AsDuration()))
	}
	return opts
}
