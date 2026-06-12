// Code generated from: xkit-template.
// generated at        2026-05-25 16:32:33 CST.

package server

import (
	"context"
	stderrors "errors"
	"sync"

	algsbreaker "github.com/chnxq/xkitmod/algs/circuitbreaker"
	"github.com/chnxq/xkitmod/algs/circuitbreaker/sre"
	errorsx "github.com/chnxq/xkitmod/errors"
	conf "github.com/chnxq/xkitpkg/conf/v1"
	"github.com/chnxq/xkitpkg/middleware"
	midbreaker "github.com/chnxq/xkitpkg/middleware/circuitbreaker"
	"github.com/chnxq/xkitpkg/transport"
)

func serverCircuitBreakerMiddleware(cfg *conf.Middleware_CircuitBreaker) middleware.Middleware {
	breakers := &sync.Map{}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			breaker := loadServerCircuitBreaker(breakers, serverCircuitBreakerKey(ctx), cfg)
			if err := breaker.Allow(); err != nil {
				breaker.MarkFailed()
				return nil, midbreaker.ErrNotAllowed
			}
			defer func() {
				if recovered := recover(); recovered != nil {
					breaker.MarkFailed()
					panic(recovered)
				}
			}()

			reply, err := handler(ctx, req)
			if serverCircuitBreakerFailure(err) {
				breaker.MarkFailed()
			} else {
				breaker.MarkSuccess()
			}
			return reply, err
		}
	}
}

func loadServerCircuitBreaker(breakers *sync.Map, key string, cfg *conf.Middleware_CircuitBreaker) algsbreaker.CircuitBreaker {
	if breaker, ok := breakers.Load(key); ok {
		return breaker.(algsbreaker.CircuitBreaker)
	}

	breaker, _ := breakers.LoadOrStore(key, newServerCircuitBreaker(cfg))
	return breaker.(algsbreaker.CircuitBreaker)
}

func newServerCircuitBreaker(cfg *conf.Middleware_CircuitBreaker) algsbreaker.CircuitBreaker {
	opts := make([]sre.Option, 0, 4)
	if cfg != nil {
		if d := cfg.GetWindow(); d != nil {
			opts = append(opts, sre.WithWindow(d.AsDuration()))
		}
		if v := cfg.GetRequest(); v > 0 {
			opts = append(opts, sre.WithRequest(v))
		}
		if v := cfg.GetBucket(); v > 0 {
			opts = append(opts, sre.WithBucket(int(v)))
		}
		if v := cfg.GetSuccess(); v > 0 {
			opts = append(opts, sre.WithSuccess(v))
		}
	}
	return sre.NewBreaker(opts...)
}

func serverCircuitBreakerKey(ctx context.Context) string {
	tr, ok := transport.FromServerContext(ctx)
	if !ok || tr == nil {
		return "server"
	}
	if op := tr.Operation(); op != "" {
		return tr.Kind().String() + ":" + op
	}
	return tr.Kind().String()
}

func serverCircuitBreakerFailure(err error) bool {
	if err == nil {
		return false
	}
	if stderrors.Is(err, context.Canceled) || stderrors.Is(err, context.DeadlineExceeded) {
		return true
	}
	return errorsx.IsInternalServer(err) ||
		errorsx.IsServiceUnavailable(err) ||
		errorsx.IsGatewayTimeout(err)
}
