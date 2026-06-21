// Code generated from: xkit-template.
// generated at        2026-06-20 12:52:15 CST.

package modulehost

import (
	"context"

	"github.com/chnxq/xkitpkg/app"
	httptransport "github.com/chnxq/xkitpkg/transport/http"
	"google.golang.org/grpc"
)

var modules = []Module{
	// Register host modules here.
	// Each module should implement shared/modulehost.Module and expose
	// its own module entry, so the host only maintains this module table.

	// New Module place here
}

type Module interface {
	Name() string
	RegisterData(*app.AppCtx) (any, func(), error)
	RegisterServices(*app.AppCtx, any) any
	RegisterHTTP(*httptransport.Server, any)
	RegisterGRPC(grpc.ServiceRegistrar, any)

	OpenAPIDocuments() []OpenAPIDocument
	SyncResources(context.Context, *app.AppCtx, any, ResourceSyncer) error
	SeedDefaultData(context.Context, *app.AppCtx, any) error
}

type OpenAPIDocument struct {
	Name     string
	FileName string
	Format   string
	Data     []byte
}

type MenuType string

const (
	MenuTypeCatalog  MenuType = "catalog"
	MenuTypeMenu     MenuType = "menu"
	MenuTypeEmbedded MenuType = "embedded"
	MenuTypeLink     MenuType = "link"
	MenuTypeButton   MenuType = "button"
)

type MenuMeta struct {
	Authority       []string
	Icon            *string
	Link            *string
	OpenInNewWindow *bool
	Title           *string
}

type MenuResource struct {
	Children  []MenuResource
	Component string
	Meta      MenuMeta
	Name      string
	Path      string
	Redirect  string
	Type      MenuType
}

type ResourceSyncer interface {
	UpsertMenus(context.Context, []MenuResource) error
}

func RegisterModule(module Module) {
	for _, m := range modules {
		if m.Name() == module.Name() {
			return
		}
	}
	modules = append(modules, module)
}

func GetRegisteredHostModules() []Module {
	return modules
}

func GetRegisteredHostModule(moduleName string) Module {
	for _, module := range modules {
		if module.Name() == moduleName {
			return module
		}
	}
	return nil
}
