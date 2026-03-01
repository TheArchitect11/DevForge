package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/chinmay/devforge/internal/logger"
	"github.com/chinmay/devforge/internal/plugins"
)

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage and run DevForge plugins",
	Long: `Discover and execute DevForge plugins.

Plugins are executable binaries located in ~/.devforge/plugins/
following the naming convention: devforge-plugin-<name>

They communicate via JSON over stdin/stdout.`,
}

var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all installed plugins",
	RunE:  runPluginList,
}

var pluginRunCmd = &cobra.Command{
	Use:   "run <name>",
	Short: "Run a plugin by name",
	Args:  cobra.ExactArgs(1),
	RunE:  runPluginRun,
}

func init() {
	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginRunCmd)
	rootCmd.AddCommand(pluginCmd)
}

func runPluginList(_ *cobra.Command, _ []string) error {
	log, err := logger.New(verbose, jsonLogs)
	if err != nil {
		return fmt.Errorf("logger initialization failed: %w", err)
	}
	defer log.Close()

	mgr, err := plugins.NewManager(log)
	if err != nil {
		return fmt.Errorf("plugin manager initialization failed: %w", err)
	}

	discovered, err := mgr.Discover()
	if err != nil {
		return fmt.Errorf("plugin discovery failed: %w", err)
	}

	if len(discovered) == 0 {
		fmt.Println()
		fmt.Println("  No plugins installed.")
		fmt.Println()
		fmt.Println("  To install a plugin, place an executable binary in:")
		fmt.Println("    ~/.devforge/plugins/devforge-plugin-<name>")
		fmt.Println()
		return nil
	}

	fmt.Println()
	fmt.Println("  Installed Plugins")
	fmt.Println("  ═════════════════")
	fmt.Println()
	fmt.Printf("  %-20s %s\n", "Name", "Path")
	fmt.Printf("  %-20s %s\n", "────", "────")
	for _, p := range discovered {
		fmt.Printf("  %-20s %s\n", p.Name, p.Path)
	}
	fmt.Println()

	return nil
}

func runPluginRun(_ *cobra.Command, args []string) error {
	name := args[0]

	log, err := logger.New(verbose, jsonLogs)
	if err != nil {
		return fmt.Errorf("logger initialization failed: %w", err)
	}
	defer log.Close()

	mgr, err := plugins.NewManager(log)
	if err != nil {
		return fmt.Errorf("plugin manager initialization failed: %w", err)
	}

	input := plugins.PluginInput{
		ProjectPath: ".",
		Config:      make(map[string]interface{}),
		DryRun:      dryRun,
	}

	output, err := mgr.Run(name, input)
	if err != nil {
		return fmt.Errorf("plugin execution failed: %w", err)
	}

	fmt.Printf("  Plugin %q completed: %s\n", name, output.Message)
	return nil
}
