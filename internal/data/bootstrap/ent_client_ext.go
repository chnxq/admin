package bootstrap

import (
	entCrud "github.com/chnxq/x-crud/entgo"
	"github.com/chnxq/xkitpkg/app"

	"admin/internal/data/ent"
)

// afterEntSchemaCreate is the handwritten hook that runs after schema migration.
// This file is created once and is never overwritten by xkit.
func afterEntSchemaCreate(ctx *app.AppCtx, entClient *entCrud.EntClient[*ent.Client]) error {
	return ensureDefaultData(ctx, entClient)
}
