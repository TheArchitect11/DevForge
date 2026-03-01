// Package cmd implements the CLI commands for DevForge using Cobra.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version is set via ldflags at build time.
var Version = "dev"

// Flag variables shared across commands.
var (
	cfgFile  string
	dryRun   bool
	verbose  bool
	jsonLogs bool
)

// rootCmd is the base command for the DevForge CLI.
var rootCmd = &cobra.Command{
	Use:   "devforge",
	Short: "DevForge — production-grade project scaffolding tool",
	Long: `DevForge is a cross-platform CLI tool that automates project setup:

  • Detects your operating system and architecture
  • Installs missing dependencies via your platform's package manager
  • Clones starter template repositories
  • Generates .env configuration files
  • Provides system health checks via the doctor command
  • Manages templates from a remote registry
  • Supports plugins for extensibility
  • Auto-updates from GitHub releases

Built for developers who value automation and consistency.`,
	Version: Version,
}

// Execute runs the root command. This is the entry point called from main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// SetVersion allows main.go to inject the build-time version.
func SetVersion(v string) {
	Version = v
	rootCmd.Version = v
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "path to config file (default: config/default.yaml)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "simulate all operations without making changes")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "enable debug-level logging output")
	rootCmd.PersistentFlags().BoolVar(&jsonLogs, "json-logs", false, "output logs in structured JSON format")
}
