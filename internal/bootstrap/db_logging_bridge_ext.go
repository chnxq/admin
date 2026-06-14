package bootstrap

import (
	"admin/internal/dblog"
	"github.com/chnxq/xkitpkg/middleware"
)

func (data *GeneratedData) DatabaseLoggingMiddleware() middleware.Middleware {
	return dblog.NewDatabaseLoggingMiddleware(data)
}
