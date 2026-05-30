// Code generated from: xkit-template.
// generated at        2026-05-25 16:32:33 CST.

package server

import (
	"context"
	"sync"

	algsbreaker "github.com/chnxq/xkitmod/algs/circuitbreaker"
	"github.com/chnxq/xkitmod/algs/circuitbreaker/sre"
	"github.com/chnxq/xkitmod/errors"
	"github.com/chnxq/xkitpkg/middleware"
	midbreaker "github.com/chnxq/xkitpkg/middleware/circuitbreaker"
	"github.com/chnxq/xkitpkg/transport"
)

func serverCircuitBreakerMiddleware() middleware.Middleware {
	breakers := &sync.Map{}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			breaker := loadServerCircuitBreaker(breakers, serverCircuitBreakerKey(ctx))
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

func loadServerCircuitBreaker(breakers *sync.Map, key string) algsbreaker.CircuitBreaker {
	if breaker, ok := breakers.Load(key); ok {
		return breaker.(algsbreaker.CircuitBreaker)
	}

	breaker, _ := breakers.LoadOrStore(key, sre.NewBreaker())
	return breaker.(algsbreaker.CircuitBreaker)
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
	return err != nil &&
		(errors.IsInternalServer(err) ||
			errors.IsServiceUnavailable(err) ||
			errors.IsGatewayTimeout(err))
}
