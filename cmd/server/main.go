// Code generated from: xkit-template.
// generated at        2026-06-20 12:52:15 CST.

package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	Name        = "Admin"
	CommandName = "admin"
	Version     = "dev"
	BuildTime   = "unknown"
	GitCommit   = "unknown"
)

func main() {
	serverCmd := flag.NewFlagSet("server", flag.ExitOnError)
	serverCmd.StringVar(&ConfigPath, "config_path", "./configs", "config directory or file")
	serverCmd.StringVar(&ConfigPath, "c", "./configs", "config directory or file")
	serverCmd.BoolVar(&ForceStartupSync, "fs", false, "force startup resource sync and default data seed even when domain data already exists")

	if len(os.Args) < 2 {
		fmt.Printf("Welcome to %s %s\n", Name, Version)
		fmt.Printf("Usage: %s server [-config_path ./configs] [-fs]\n", CommandName)
		return
	}

	switch os.Args[1] {
	case "server":
		_ = serverCmd.Parse(os.Args[2:])
		if err := runServer(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Printf("unknown command %q\n", os.Args[1])
		os.Exit(1)
	}
}
