// DevForge is a production-grade CLI tool for automated project scaffolding.
// It detects your OS, installs dependencies, clones starter templates, and
// generates environment configuration — all with rollback support.
package main

import "github.com/chinmay/devforge/cmd"

// version is set via ldflags at build time:
//
//	go build -ldflags "-X main.version=1.0.0" -o devforge .
var version = "dev"

func main() {
	cmd.SetVersion(version)
	cmd.Execute()
}
