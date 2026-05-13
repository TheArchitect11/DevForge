package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/chinmay/devforge/internal/logger"
	"github.com/chinmay/devforge/internal/plugins"
	"github.com/chinmay/devforge/internal/ux"
)

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage and run DevForge plugins",
	Long: `Discover and execute DevForge plugins.

Plugins are executables in ~/.devforge/plugins/ following the naming
convention devforge-plugin-<name>. They communicate via JSON over stdin/stdout.

  devforge plugin list           # discover installed plugins
  devforge plugin run <name>     # execute a plugin`,
}

var pluginRunProjectPath string

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
	pluginRunCmd.Flags().StringVarP(&pluginRunProjectPath, "project", "p", ".", "path to the project directory passed to the plugin")
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
		ux.InfoMsg("No plugins installed.")
		fmt.Println()
		fmt.Println("  To add a plugin, place an executable binary at:")
		fmt.Println("    ~/.devforge/plugins/devforge-plugin-<name>")
		fmt.Println()
		return nil
	}

	ux.Header(fmt.Sprintf("Installed Plugins (%d)", len(discovered)))
	fmt.Printf("  %-22s %s\n", "Name", "Path")
	fmt.Printf("  %-22s %s\n", "────", "────")
	for _, p := range discovered {
		fmt.Printf("  %-22s %s\n", p.Name, p.Path)
	}
	fmt.Println()
	fmt.Println("  Run:  devforge plugin run <name>")
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

	// Resolve the project path to an absolute path.
	projectPath, err := os.Getwd()
	if err != nil {
		projectPath = "."
	}
	if pluginRunProjectPath != "." {
		projectPath = pluginRunProjectPath
	}

	ux.Step("Running plugin %q against %s", name, projectPath)

	output, err := mgr.Run(name, plugins.PluginInput{
		ProjectPath: projectPath,
		Config:      make(map[string]interface{}),
		DryRun:      dryRun,
	})
	if err != nil {
		return fmt.Errorf("plugin %q failed: %w", name, err)
	}

	if output.Success {
		ux.Success("Plugin %q: %s", name, output.Message)
	} else {
		ux.Warning("Plugin %q returned: %s", name, output.Message)
	}

	for _, artifact := range output.Artifacts {
		fmt.Printf("  %s %s\n", ux.Arrow, artifact)
	}

	return nil
}
