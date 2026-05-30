// Code generated from: xkit-template.
// generated at        2026-05-25 16:32:33 CST.

package bootstrap

import "github.com/chnxq/xkitpkg/app"

type DataResources struct{}

func NewDataResources(appCtx *app.AppCtx) (*DataResources, func(), error) {
	_ = appCtx
	return &DataResources{}, func() {}, nil
}
