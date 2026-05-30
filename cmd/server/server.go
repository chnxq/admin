// Code generated from: xkit-template.
// generated at        2026-05-25 16:32:33 CST.

package main

import (
	"context"
	"fmt"

	"admin/internal/bootstrap"
)

var ConfigPath string

func runServer() error {
	app, cleanup, err := bootstrap.Initialize(context.Background(), bootstrap.Options{
		Name:       Name,
		Version:    Version,
		BuildTime:  BuildTime,
		GitCommit:  GitCommit,
		ConfigPath: ConfigPath,
	})
	if err != nil {
		return err
	}
	defer cleanup()

	fmt.Printf("starting %s %s\n", Name, Version)
	return app.Run()
}
