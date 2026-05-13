package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	osexec "os/exec"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/chinmay/devforge/internal/config"
	"github.com/chinmay/devforge/internal/envgen"
	"github.com/chinmay/devforge/internal/errors"
	"github.com/chinmay/devforge/internal/executor"
	"github.com/chinmay/devforge/internal/installer"
	"github.com/chinmay/devforge/internal/logger"
	"github.com/chinmay/devforge/internal/osdetect"
	"github.com/chinmay/devforge/internal/registry"
	"github.com/chinmay/devforge/internal/rollback"
	"github.com/chinmay/devforge/internal/security"
	"github.com/chinmay/devforge/internal/semver"
	"github.com/chinmay/devforge/internal/template"
	"github.com/chinmay/devforge/internal/ux"
	"github.com/chinmay/devforge/internal/wizard"
)

var templateName string
var noInteractive bool
var saveConfig bool

var initCmd = &cobra.Command{
	Use:   "init <project-name>",
	Short: "Scaffold a new project",
	Long: `Initialize a new project by:

  1. Detecting your OS and package manager
  2. Loading configuration (or launching the interactive wizard)
  3. Installing required dependencies with version pinning
  4. Cloning the starter template into <project-name>/
  5. Resetting git history so you start with a clean slate
  6. Generating a .env file (if envFile: true in config)
  7. Running post-init hooks (npm install, go mod tidy, …)

Any step failure triggers automatic rollback of completed steps.`,
	Args: cobra.ExactArgs(1),
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringVarP(&templateName, "template", "t", "", "name of a template from the remote registry")
	initCmd.Flags().BoolVar(&noInteractive, "no-interactive", false, "fail instead of launching the wizard when no config is found")
	initCmd.Flags().BoolVar(&saveConfig, "save-config", false, "write the resolved config to devforge.yaml in the current directory")
	rootCmd.AddCommand(initCmd)
}

func runInit(_ *cobra.Command, args []string) error {
	start := time.Now()
	projectName := args[0]

	// ── Validate project name ──────────────────────────────────────
	if err := security.ValidateProjectName(projectName); err != nil {
		ux.Error(err)
		return nil
	}

	destDir, err := filepath.Abs(projectName)
	if err != nil {
		ux.Error(fmt.Errorf("failed to resolve project path: %v", err))
		return nil
	}

	// ── Guard against accidental overwrite ─────────────────────────
	if _, err := os.Stat(destDir); err == nil {
		if !force {
			ux.Error(errors.New(
				errors.CodePathExists,
				fmt.Sprintf("directory %q already exists", projectName),
				"use --force to remove and re-scaffold it",
			))
			return nil
		}
		ux.Warning("Removing existing directory %q (--force)", projectName)
		if err := os.RemoveAll(destDir); err != nil {
			return fmt.Errorf("failed to remove existing directory %q: %w", projectName, err)
		}
	}

	// ── Step 1: Detect OS ──────────────────────────────────────────
	ux.Banner(fmt.Sprintf("Scaffolding project %q", projectName))

	osInfo, err := osdetect.DetectFull()
	if err != nil {
		return fmt.Errorf("OS detection failed: %w", err)
	}
	ux.Success("OS detected: %s (%s/%s) — package manager: %s",
		osInfo.Name, osInfo.RawOS, osInfo.Arch, osInfo.PackageMgr)

	// ── Step 2: Load config ────────────────────────────────────────
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	ux.InfoMsg("Configuration: %d %s, template: %s",
		len(cfg.Dependencies),
		pluralise("dependency", "dependencies", len(cfg.Dependencies)),
		cfg.Template)

	// Persist wizard-built or registry-fetched config for future reuse.
	if saveConfig {
		if yamlBytes, yamlErr := cfg.ToYAML(); yamlErr != nil {
			ux.Warning("could not serialise config: %v", yamlErr)
		} else if writeErr := os.WriteFile("devforge.yaml", yamlBytes, 0o644); writeErr != nil {
			ux.Warning("could not write devforge.yaml: %v", writeErr)
		} else {
			ux.Success("Config saved to devforge.yaml")
		}
	}

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

	// ── Step 4: Initialize rollback ────────────────────────────────
	rb := rollback.NewManager(log)

	// ── Step 5: Install dependencies ───────────────────────────────
	exec := executor.New(log, dryRun)
	inst, err := installer.NewFromOS(log, exec, osInfo)
	if err != nil {
		ux.Error(fmt.Errorf("installer unavailable for this OS: %v", err))
		return nil
	}

	if len(cfg.Dependencies) > 0 {
		ux.Header("Installing Dependencies")
		results := installDeps(cfg.Dependencies, inst, log)
		ux.PrintInstallResults(results)

		// Fail fast if any install failed.
		for _, r := range results {
			if r.Err != nil {
				ux.Error(r.Err)
				rb.Execute()
				return nil
			}
		}
	}

	// ── Step 6: Clone template ─────────────────────────────────────
	ux.Header("Cloning Template")
	cloner := template.NewCloner(log, rb, dryRun)
	if err := cloner.Clone(cfg.Template, destDir); err != nil {
		log.Error(fmt.Sprintf("template cloning failed: %v", err))
		rb.Execute()
		return fmt.Errorf("template cloning failed: %w", err)
	}

	// ── Step 7: Generate .env file ─────────────────────────────────
	if cfg.EnvFile {
		ux.Step("Generating .env from template")
		gen := envgen.NewGenerator(log, rb, dryRun)
		if err := gen.Generate(destDir); err != nil {
			log.Error(fmt.Sprintf("env generation failed: %v", err))
			rb.Execute()
			return fmt.Errorf(".env generation failed: %w", err)
		}
		ux.Success("Environment file generated")
	}

	// ── Step 8: Post-init hooks ────────────────────────────────────
	if !dryRun && len(cfg.PostInit) > 0 {
		ux.Header("Post-Init Hooks")
		runPostInitHooks(cfg.PostInit, destDir)
	}

	// ── Done ───────────────────────────────────────────────────────
	printSummary(projectName, destDir, cfg, dryRun, time.Since(start))
	log.Info("DevForge init completed successfully")
	return nil
}

// loadConfig resolves configuration from: --template flag → config file → wizard.
func loadConfig() (*config.Config, error) {
	if templateName != "" {
		return loadConfigFromRegistry(templateName)
	}

	cfg, loadErr := config.Load(cfgFile)
	if loadErr == nil {
		return cfg, nil
	}

	if noInteractive {
		return nil, fmt.Errorf("configuration error: %w", loadErr)
	}

	ux.Warning("No config file found — launching interactive setup.")
	cfg, wizardErr := wizard.Run(defaultRegistryURL, verbose, jsonLogs)
	if wizardErr != nil {
		return nil, fmt.Errorf("interactive setup failed: %w", wizardErr)
	}
	return cfg, nil
}

// loadConfigFromRegistry fetches a named template from the registry and
// downloads its devforge.yaml to build a Config.
func loadConfigFromRegistry(name string) (*config.Config, error) {
	spin := ux.NewSpinner(fmt.Sprintf("Fetching template %q from registry", name))

	client, _, err := getRegistryClient()
	if err != nil {
		spin.Fail("registry unavailable")
		return nil, fmt.Errorf("registry client: %w", err)
	}
	reg, err := client.Fetch(false)
	if err != nil {
		spin.Fail("registry fetch failed")
		return nil, fmt.Errorf("fetch registry: %w", err)
	}

	var matched registry.Template
	found := false
	for _, t := range reg.ValidTemplates() {
		if strings.EqualFold(t.Name, name) {
			matched = t
			found = true
			break
		}
	}
	if !found {
		spin.Fail(fmt.Sprintf("template %q not found", name))
		return nil, fmt.Errorf("template %q not found in registry", name)
	}

	// Fetch the devforge.yaml from the template's main branch.
	rawURL := strings.Replace(matched.URL, "github.com", "raw.githubusercontent.com", 1)
	rawURL = strings.TrimSuffix(rawURL, ".git") + "/main/devforge.yaml"

	resp, err := http.Get(rawURL)
	if err != nil {
		spin.Fail("could not download template config")
		return nil, fmt.Errorf("fetch devforge.yaml: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		spin.Fail(fmt.Sprintf("template %q has no devforge.yaml on main branch", name))
		return nil, fmt.Errorf("template config returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		spin.Fail("could not read template config")
		return nil, fmt.Errorf("read devforge.yaml: %w", err)
	}

	cfg, err := config.LoadFromBytes(body)
	if err != nil {
		spin.Fail("invalid template config")
		return nil, err
	}

	cfg.Template = matched.URL
	spin.Stop(fmt.Sprintf("Template %q loaded from registry", name))
	return cfg, nil
}

// installDeps runs up to 3 installs concurrently and returns ordered results.
func installDeps(deps []config.Dependency, inst installer.Installer, log *logger.Logger) []ux.InstallResult {
	results := make([]ux.InstallResult, len(deps))
	var mu sync.Mutex
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sem := make(chan struct{}, 3) // limit to 3 concurrent installs

	for i, dep := range deps {
		wg.Add(1)
		go func(idx int, dep config.Dependency) {
			defer wg.Done()

			res := ux.InstallResult{Name: dep.Name, Wanted: dep.Version}

			select {
			case <-ctx.Done():
				res.Err = fmt.Errorf("cancelled")
				mu.Lock()
				results[idx] = res
				mu.Unlock()
				return
			case sem <- struct{}{}:
				defer func() { <-sem }()
			}

			already, _ := inst.IsInstalled(dep.Name)
			if already {
				ver, _ := inst.GetVersion(dep.Name)
				res.Skipped = true
				res.Version = ver

				// Check for major version mismatch.
				if dep.Version != "" && dep.Version != "latest" && ver != "" {
					desired, err1 := semver.Parse(dep.Version)
					current, err2 := semver.Parse(ver)
					if err1 == nil && err2 == nil && !desired.IsZero() && !desired.MajorMatches(current) {
						log.Warn(fmt.Sprintf("version mismatch for %q: installed=%s, wanted=%s",
							dep.Name, ver, dep.Version))
					}
				}
				mu.Lock()
				results[idx] = res
				mu.Unlock()
				return
			}

			if err := inst.Install(dep.Name, dep.Version); err != nil {
				res.Err = errors.New(
					errors.CodeExecutionFailed,
					fmt.Sprintf("failed to install %q", dep.Name),
					"check your network or package manager logs",
				)
				cancel()
			} else {
				ver, _ := inst.GetVersion(dep.Name)
				res.Version = ver
			}

			mu.Lock()
			results[idx] = res
			mu.Unlock()
		}(i, dep)
	}

	wg.Wait()
	return results
}

// runPostInitHooks executes post-init shell commands inside destDir.
// Each hook is run sequentially; failures are warnings, not fatal errors.
func runPostInitHooks(hooks []string, destDir string) {
	for _, hook := range hooks {
		parts := strings.Fields(strings.TrimSpace(hook))
		if len(parts) == 0 {
			continue
		}
		ux.Step("Running: %s", hook)
		cmd := osexec.Command(parts[0], parts[1:]...)
		cmd.Dir = destDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			ux.Warning("hook %q failed: %v — continuing", hook, err)
		} else {
			ux.Success("hook: %s", hook)
		}
	}
}

// printSummary shows the final success banner and next-steps.
func printSummary(projectName, destDir string, cfg *config.Config, isDryRun bool, elapsed time.Duration) {
	fmt.Println()
	ux.Divider()

	if isDryRun {
		ux.InfoMsg("DRY RUN COMPLETE — no changes were made to disk.")
	} else {
		ux.Success("Project %q created in %s!", projectName, fmtDuration(elapsed))
	}

	fmt.Println()
	fmt.Println("  Next steps:")
	fmt.Printf("    cd %s\n", destDir)

	// Suggest the most relevant commands based on installed deps.
	for _, dep := range cfg.Dependencies {
		switch dep.Name {
		case "node", "nodejs":
			fmt.Println("    npm install        # install Node.js dependencies")
		case "go":
			fmt.Println("    go mod tidy        # tidy Go module graph")
		case "python", "python3":
			fmt.Println("    pip install -r requirements.txt")
		case "rust", "rustc":
			fmt.Println("    cargo build        # compile the Rust project")
		case "ruby":
			fmt.Println("    bundle install     # install Ruby gems")
		}
	}

	fmt.Println()
	fmt.Println("    devforge doctor    # verify all tools are installed")
	fmt.Println()
}

// pluralise returns singular when n == 1, else plural.
func pluralise(singular, plural string, n int) string {
	if n == 1 {
		return singular
	}
	return plural
}

// fmtDuration formats a duration as a human-friendly string.
func fmtDuration(d time.Duration) string {
	d = d.Round(time.Second)
	if d < time.Second {
		return "< 1s"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	if s == 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dm %ds", m, s)
}
