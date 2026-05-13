package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/chinmay/devforge/internal/logger"
	"github.com/chinmay/devforge/internal/updater"
	"github.com/chinmay/devforge/internal/ux"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for and install DevForge updates",
	Long: `Check the latest DevForge release on GitHub. If a newer version
is available, download and install it automatically.

The current binary is backed up before replacement and automatically
restored if the update fails. The downloaded binary is verified against
the SHA-256 checksums published with the release.`,
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

	// ── Check for updates ──────────────────────────────────────────
	spin := ux.NewSpinner("Checking for updates")
	u := updater.New(Version, log)

	result, err := u.Check()
	if err != nil {
		spin.Fail("Update check failed")
		return fmt.Errorf("update check failed: %w", err)
	}
	spin.Stop("")

	// ── Version summary ────────────────────────────────────────────
	ux.Header("DevForge Update")
	fmt.Printf("  Current version : %s\n", result.CurrentVersion)
	fmt.Printf("  Latest version  : %s\n", result.LatestVersion)
	fmt.Println()

	if !result.UpdateAvailable {
		ux.Success("Already on the latest version (%s)", result.CurrentVersion)
		return nil
	}

	// ── Changelog ─────────────────────────────────────────────────
	if result.Changelog != "" {
		ux.InfoMsg("What's new in v%s:", result.LatestVersion)
		for _, line := range strings.Split(result.Changelog, "\n") {
			fmt.Printf("  %s\n", line)
		}
		fmt.Println()
	}

	if dryRun {
		ux.InfoMsg("[dry-run] Would download and install v%s.", result.LatestVersion)
		return nil
	}

	if result.AssetURL == "" {
		return fmt.Errorf("no pre-built binary for this platform — visit https://github.com/ChinmayyK/DevForge/releases")
	}

	// ── Download & install ─────────────────────────────────────────
	dlSpin := ux.NewSpinner(fmt.Sprintf("Downloading v%s", result.LatestVersion))

	if err := u.Update(result); err != nil {
		dlSpin.Fail(fmt.Sprintf("Update failed: %v", err))
		return err
	}

	dlSpin.Stop(fmt.Sprintf("Updated to v%s — restart DevForge to use the new version", result.LatestVersion))
	return nil
}
