// Code generated from: xkit-template.
// generated at        2026-06-20 12:52:15 CST.

package modulehost

import (
	"github.com/chnxq/xkitpkg/app"
	httptransport "github.com/chnxq/xkitpkg/transport/http"
	"google.golang.org/grpc"
)

type Module interface {
	Name() string
	RegisterData(*app.AppCtx) (any, func(), error)
	RegisterServices(*app.AppCtx, any) any
	RegisterHTTP(*httptransport.Server, any)
	RegisterGRPC(grpc.ServiceRegistrar, any)
}
