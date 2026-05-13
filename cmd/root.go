// Package cmd implements the CLI commands for DevForge using Cobra.
package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/chinmay/devforge/internal/ux"
)

// Version is set via ldflags at build time.
var Version = "dev"

// Flag variables shared across commands.
var (
	cfgFile  string
	dryRun   bool
	verbose  bool
	jsonLogs bool
	force    bool
	noColor  bool
)

var rootCmd = &cobra.Command{
	Use:   "devforge <command>",
	Short: "DevForge — Development Environment Automation CLI",
	Long: `DevForge — Development Environment Automation CLI

A fast, standalone tool for scaffolding and hardening development environments:
  ✔ Detects OS and package manager automatically
  ✔ Installs required toolchains with version pinning
  ✔ Clones starter templates and initialises a fresh git history
  ✔ Generates .env files from templates
  ✔ Supports post-init hooks (npm install, go mod tidy, …)
  ✔ Provides interactive wizard when no config file is found

Run "devforge init <project-name>" to get started.`,

	// Suppress cobra's default "Usage:" dump and double error printing.
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute is the entry point called from main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		ux.Error(err)
		os.Exit(1)
	}
}

// SetVersion lets main.go inject the build-time version string.
func SetVersion(v string) {
	Version = v
	rootCmd.Version = v
}

func init() {
	cobra.OnInitialize(func() {
		if noColor {
			ux.SetColorEnabled(false)
		}
	})

	rootCmd.SetHelpTemplate(`{{.Long}}

Usage:
  {{.UseLine}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}

Use "{{.CommandPath}} [command] --help" for more information about a command.
`)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "path to config file (default: config/default.yaml)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "simulate operations without making changes")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "enable debug logging")
	rootCmd.PersistentFlags().BoolVar(&jsonLogs, "json-logs", false, "structured JSON output for CI pipelines")
	rootCmd.PersistentFlags().BoolVar(&force, "force", false, "force overwrite of existing directories/files")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output (also honoured via NO_COLOR env var)")
}
