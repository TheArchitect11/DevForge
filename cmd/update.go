package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/chinmay/devforge/internal/logger"
	"github.com/chinmay/devforge/internal/updater"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for and install DevForge updates",
	Long: `Check the latest DevForge release on GitHub. If a newer version
is available, download and install it automatically.

The current binary is backed up before replacement, and rolled
back if the update fails.`,
	RunE: runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(_ *cobra.Command, _ []string) error {
	log, err := logger.New(verbose, jsonLogs)
	if err != nil {
		return fmt.Errorf("logger initialization failed: %w", err)
	}
	defer log.Close()

	u := updater.New(Version, log)

	result, err := u.Check()
	if err != nil {
		return fmt.Errorf("update check failed: %w", err)
	}

	fmt.Printf("  Current version: %s\n", result.CurrentVersion)
	fmt.Printf("  Latest version:  %s\n", result.LatestVersion)

	if !result.UpdateAvailable {
		fmt.Println("\n  ✅ You are running the latest version!")
		return nil
	}

	fmt.Println()
	if result.Changelog != "" {
		fmt.Println("  Changelog:")
		fmt.Println("  ──────────")
		fmt.Println(result.Changelog)
		fmt.Println()
	}

	if dryRun {
		fmt.Println("  [dry-run] Would download and install the update.")
		return nil
	}

	if result.AssetURL == "" {
		return fmt.Errorf("no binary available for your platform; please update manually")
	}

	fmt.Println("  ⟳ Downloading update...")
	if err := u.Update(result); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	fmt.Printf("\n  ✅ Updated to v%s. Restart DevForge to use the new version.\n", result.LatestVersion)
	return nil
}
