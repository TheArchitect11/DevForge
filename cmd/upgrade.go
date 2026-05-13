package cmd

import (
	"context"
	"fmt"
	"sync"

	"github.com/spf13/cobra"

	"github.com/chinmay/devforge/internal/config"
	"github.com/chinmay/devforge/internal/executor"
	"github.com/chinmay/devforge/internal/installer"
	"github.com/chinmay/devforge/internal/logger"
	"github.com/chinmay/devforge/internal/osdetect"
	"github.com/chinmay/devforge/internal/ux"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade all dependencies listed in devforge.yaml",
	Long: `Upgrade every dependency listed in your devforge.yaml to the version
specified (or to the latest version if none is pinned).

Upgrades run in parallel (up to 3 at a time) and respect the same
version-pinning rules as 'devforge init'.

Exit code is non-zero if any upgrade fails.`,
	RunE: runUpgrade,
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
}

func runUpgrade(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	log, err := logger.New(verbose, jsonLogs)
	if err != nil {
		return fmt.Errorf("logger initialization failed: %w", err)
	}
	defer log.Close()

	osInfo, err := osdetect.DetectFull()
	if err != nil {
		return fmt.Errorf("OS detection failed: %w", err)
	}

	exec := executor.New(log, dryRun)
	inst, err := installer.NewFromOS(log, exec, osInfo)
	if err != nil {
		return fmt.Errorf("installer unavailable for this OS: %w", err)
	}

	ux.Banner("Upgrading Dependencies")
	ux.InfoMsg("OS: %s · package manager: %s · %d dep(s)",
		osInfo.Name, osInfo.PackageMgr, len(cfg.Dependencies))

	results := upgradeDeps(cfg.Dependencies, inst, log)
	ux.PrintInstallResults(results)

	failed := 0
	for _, r := range results {
		if r.Err != nil {
			failed++
		}
	}

	fmt.Println()
	ux.Divider()
	if failed == 0 {
		ux.Success("All %d dep(s) up to date.", len(results))
	} else {
		ux.Warning("%d of %d dep(s) failed to upgrade.", failed, len(results))
		return fmt.Errorf("%d upgrade(s) failed", failed)
	}
	return nil
}

// upgradeDeps runs Upgrade concurrently (≤3 workers) and returns ordered results.
func upgradeDeps(deps []config.Dependency, inst installer.Installer, log *logger.Logger) []ux.InstallResult {
	results := make([]ux.InstallResult, len(deps))
	var mu sync.Mutex
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sem := make(chan struct{}, 3)

	for i, dep := range deps {
		wg.Add(1)
		go func(i int, dep config.Dependency) {
			defer wg.Done()
			res := ux.InstallResult{Name: dep.Name, Wanted: dep.Version}

			select {
			case <-ctx.Done():
				res.Err = fmt.Errorf("cancelled")
				mu.Lock()
				results[i] = res
				mu.Unlock()
				return
			case sem <- struct{}{}:
				defer func() { <-sem }()
			}

			before, _ := inst.GetVersion(dep.Name)
			if err := inst.Upgrade(dep.Name, dep.Version); err != nil {
				res.Err = err
				log.Error(fmt.Sprintf("upgrade failed for %q: %v", dep.Name, err))
			} else {
				after, _ := inst.GetVersion(dep.Name)
				res.Version = after
				if before == after {
					res.Skipped = true // already at target version
				}
			}

			mu.Lock()
			results[i] = res
			mu.Unlock()
		}(i, dep)
	}

	wg.Wait()
	return results
}
