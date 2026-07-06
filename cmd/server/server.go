// Code generated from: xkit-template.
// generated at        2026-06-20 12:52:15 CST.

package main

import (
	"context"
	"fmt"

	"admin/internal/bootstrap"
)

var ConfigPath string
var ForceStartupSync bool

func runServer() error {
	app, cleanup, err := bootstrap.Initialize(context.Background(), bootstrap.Options{
		Name:             Name,
		Version:          Version,
		BuildTime:        BuildTime,
		GitCommit:        GitCommit,
		ConfigPath:       ConfigPath,
		ForceStartupSync: ForceStartupSync,
	})
	if err != nil {
		return err
	}
	defer cleanup()

	fmt.Printf("starting %s %s\n", Name, Version)
	return app.Run()
}
