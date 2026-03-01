package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/chinmay/devforge/internal/config"
	"github.com/chinmay/devforge/internal/envgen"
	"github.com/chinmay/devforge/internal/executor"
	"github.com/chinmay/devforge/internal/installer"
	"github.com/chinmay/devforge/internal/logger"
	"github.com/chinmay/devforge/internal/osdetect"
	"github.com/chinmay/devforge/internal/rollback"
	"github.com/chinmay/devforge/internal/security"
	"github.com/chinmay/devforge/internal/semver"
	"github.com/chinmay/devforge/internal/template"
)

var initCmd = &cobra.Command{
	Use:   "init <project-name>",
	Short: "Scaffold a new project",
	Long: `Initialize a new project by:
  1. Detecting your OS
  2. Loading configuration
  3. Installing required dependencies (with version pinning)
  4. Cloning the starter template
  5. Generating environment configuration

If any step fails, previously completed steps are automatically rolled back.`,
	Args: cobra.ExactArgs(1),
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	projectName := args[0]

	// Validate project name for safety.
	if err := security.ValidateName(projectName); err != nil {
		return fmt.Errorf("invalid project name: %w", err)
	}

	// ── Step 1: Detect OS ──────────────────────────────────────────
	osInfo, err := osdetect.DetectFull()
	if err != nil {
		return fmt.Errorf("OS detection failed: %w", err)
	}
	fmt.Printf("✓ OS detected: %s (%s/%s) — package manager: %s\n", osInfo.Name, osInfo.RawOS, osInfo.Arch, osInfo.PackageMgr)

	// ── Step 2: Load config ────────────────────────────────────────
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}
	fmt.Printf("✓ Configuration loaded (%d dependencies, template: %s)\n", len(cfg.Dependencies), cfg.Template)

	// ── Step 3: Initialize logger ──────────────────────────────────
	log, err := logger.New(verbose, jsonLogs)
	if err != nil {
		return fmt.Errorf("logger initialization failed: %w", err)
	}
	defer log.Close()

	log.Info("DevForge init started", map[string]interface{}{
		"project":    projectName,
		"dryRun":     dryRun,
		"os":         osInfo.Name,
		"packageMgr": osInfo.PackageMgr,
	})

	// ── Step 4: Initialize rollback manager ────────────────────────
	rb := rollback.NewManager(log)

	// ── Step 5: Install dependencies ───────────────────────────────
	exec := executor.New(log, dryRun)
	inst, err := installer.NewFromOS(log, exec, osInfo)
	if err != nil {
		return fmt.Errorf("installer initialization failed: %w", err)
	}

	for _, dep := range cfg.Dependencies {
		installed, checkErr := inst.IsInstalled(dep.Name)
		if checkErr != nil {
			log.Warn(fmt.Sprintf("could not check if %q is installed: %v", dep.Name, checkErr))
		}

		if installed {
			currentVersion, _ := inst.GetVersion(dep.Name)
			fmt.Printf("✓ %s already installed (v%s)", dep.Name, currentVersion)

			// Version mismatch check.
			if dep.Version != "" && dep.Version != "latest" && currentVersion != "" {
				desired, parseErr := semver.Parse(dep.Version)
				current, curParseErr := semver.Parse(currentVersion)
				if parseErr == nil && curParseErr == nil && !desired.IsZero() {
					if !desired.MajorMatches(current) {
						fmt.Printf(" ⚠ wanted v%s", dep.Version)
						log.Warn(fmt.Sprintf("version mismatch for %q: installed=%s, wanted=%s", dep.Name, currentVersion, dep.Version))
					}
				}
			}
			fmt.Println()
			log.Info(fmt.Sprintf("dependency %q already installed", dep.Name))
			continue
		}

		versionLabel := dep.Version
		if versionLabel == "" || versionLabel == "latest" {
			versionLabel = "latest"
		}
		fmt.Printf("⟳ Installing %s (v%s)...\n", dep.Name, versionLabel)
		if err := inst.Install(dep.Name, dep.Version); err != nil {
			log.Error(fmt.Sprintf("failed to install %q", dep.Name))
			rbErr := rb.Execute()
			if rbErr != nil {
				log.Error(fmt.Sprintf("rollback errors: %v", rbErr))
			}
			return fmt.Errorf("dependency installation failed for %q: %w", dep.Name, err)
		}
		fmt.Printf("✓ %s installed\n", dep.Name)
	}

	// ── Step 6: Clone template ─────────────────────────────────────
	destDir, err := filepath.Abs(projectName)
	if err != nil {
		return fmt.Errorf("failed to resolve project path: %w", err)
	}

	cloner := template.NewCloner(log, rb, dryRun)
	if err := cloner.Clone(cfg.Template, destDir); err != nil {
		log.Error(fmt.Sprintf("template cloning failed: %v", err))
		rbErr := rb.Execute()
		if rbErr != nil {
			log.Error(fmt.Sprintf("rollback errors: %v", rbErr))
		}
		return fmt.Errorf("template cloning failed: %w", err)
	}
	fmt.Printf("✓ Template cloned into %s\n", destDir)

	// ── Step 7: Generate env file ──────────────────────────────────
	if cfg.EnvFile {
		gen := envgen.NewGenerator(log, rb, dryRun)
		if err := gen.Generate(destDir); err != nil {
			log.Error(fmt.Sprintf("env generation failed: %v", err))
			rbErr := rb.Execute()
			if rbErr != nil {
				log.Error(fmt.Sprintf("rollback errors: %v", rbErr))
			}
			return fmt.Errorf("env file generation failed: %w", err)
		}
		fmt.Println("✓ Environment configuration complete")
	}

	// ── Step 8: Print success summary ──────────────────────────────
	printSummary(projectName, destDir, cfg, dryRun)
	log.Info("DevForge init completed successfully")

	return nil
}

// printSummary displays a clear success message with next steps.
func printSummary(name, dir string, cfg *config.Config, dryRun bool) {
	fmt.Println()
	if dryRun {
		fmt.Println("═══════════════════════════════════════════")
		fmt.Println("  DRY RUN COMPLETE — no changes were made")
		fmt.Println("═══════════════════════════════════════════")
	} else {
		fmt.Println("═══════════════════════════════════════════")
		fmt.Printf("  🚀 Project %q created successfully!\n", name)
		fmt.Println("═══════════════════════════════════════════")
	}
	fmt.Println()
	fmt.Println("  Next steps:")
	fmt.Printf("    cd %s\n", dir)
	if cfg.Linting {
		fmt.Println("    # Linting is enabled")
	}
	if cfg.GitHooks {
		fmt.Println("    # Git hooks are configured")
	}
	fmt.Println()
	fmt.Println("  Run 'devforge doctor' to verify system readiness.")
	fmt.Println()
}
