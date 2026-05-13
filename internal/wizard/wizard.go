// Package wizard provides an interactive TUI setup experience when
// DevForge is run without a configuration file. It walks the user
// through dependency selection, version pinning (all at once), template
// choice, and project options, then returns a populated *config.Config.
package wizard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/chinmay/devforge/internal/config"
	"github.com/chinmay/devforge/internal/logger"
	"github.com/chinmay/devforge/internal/registry"
	"github.com/chinmay/devforge/internal/ux"
)

// knownDependency is a curated tool available in the wizard picker.
type knownDependency struct {
	Name        string
	Description string
}

var availableDeps = []knownDependency{
	{Name: "node", Description: "Node.js runtime"},
	{Name: "git", Description: "Version control system"},
	{Name: "docker", Description: "Container runtime"},
	{Name: "python3", Description: "Python interpreter"},
	{Name: "go", Description: "Go programming language"},
	{Name: "rust", Description: "Rust toolchain (rustc + cargo)"},
	{Name: "java", Description: "Java Development Kit"},
	{Name: "ruby", Description: "Ruby interpreter"},
}

// Run launches the interactive wizard and returns a fully populated
// Config. registryURL is used to fetch templates; pass the default
// registry URL from the cmd package.
func Run(registryURL string, verbose, jsonLogs bool) (*config.Config, error) {
	// ── Banner ─────────────────────────────────────────────────────
	fmt.Println()
	ux.Banner("Interactive Setup  —  no devforge.yaml found")
	fmt.Print("  Answer a few questions to scaffold your project.\n\n")

	// ── Step 1: Select dependencies ────────────────────────────────
	depOptions := make([]huh.Option[string], len(availableDeps))
	for i, d := range availableDeps {
		depOptions[i] = huh.NewOption(
			fmt.Sprintf("%-10s  %s", d.Name, d.Description),
			d.Name,
		)
	}

	var selectedDeps []string
	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Which tools do you need?").
				Description("Space to toggle · Enter to confirm").
				Options(depOptions...).
				Value(&selectedDeps),
		),
	).Run(); err != nil {
		return nil, fmt.Errorf("dependency selection cancelled: %w", err)
	}

	if len(selectedDeps) == 0 {
		ux.Warning("No dependencies selected. You can add them later in devforge.yaml.")
	}

	// ── Step 2: Pin versions (all in one form) ─────────────────────
	// Build a slice of version pointers so we can bind each Input to a
	// separate string without sequential prompts.
	versionPtrs := make([]string, len(selectedDeps))
	deps := make([]config.Dependency, 0, len(selectedDeps))

	if len(selectedDeps) > 0 {
		fields := make([]huh.Field, len(selectedDeps))
		for i, name := range selectedDeps {
			i, name := i, name // capture loop vars
			fields[i] = huh.NewInput().
				Title(fmt.Sprintf("%s version", name)).
				Description("Leave blank to use latest").
				Placeholder("latest").
				Value(&versionPtrs[i])
		}

		if err := huh.NewForm(huh.NewGroup(fields...)).Run(); err != nil {
			return nil, fmt.Errorf("version input cancelled: %w", err)
		}

		for i, name := range selectedDeps {
			v := strings.TrimSpace(versionPtrs[i])
			if v == "" {
				v = "latest"
			}
			deps = append(deps, config.Dependency{Name: name, Version: v})
		}
	}

	// ── Step 3: Choose a starter template ──────────────────────────
	templateURL, err := selectTemplate(registryURL, verbose, jsonLogs)
	if err != nil {
		return nil, err
	}

	// ── Step 4: Project options ─────────────────────────────────────
	var envFile, linting, gitHooks bool

	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Generate a .env file?").
				Description("Creates a starter .env from .env.template in the project").
				Value(&envFile),
			huh.NewConfirm().
				Title("Enable linting?").
				Description("Configure linters (ESLint, golangci-lint, etc.)").
				Value(&linting),
			huh.NewConfirm().
				Title("Enable git hooks?").
				Description("Install pre-commit and lint-staged via husky / lefthook").
				Value(&gitHooks),
		),
	).Run(); err != nil {
		return nil, fmt.Errorf("options selection cancelled: %w", err)
	}

	// ── Step 5: Confirmation ────────────────────────────────────────
	fmt.Println()
	ux.Header("Configuration Summary")
	if len(deps) > 0 {
		parts := make([]string, len(deps))
		for i, d := range deps {
			if d.Version == "latest" || d.Version == "" {
				parts[i] = d.Name
			} else {
				parts[i] = fmt.Sprintf("%s@%s", d.Name, d.Version)
			}
		}
		fmt.Printf("  Dependencies : %s\n", strings.Join(parts, ", "))
	} else {
		fmt.Println("  Dependencies : (none)")
	}
	fmt.Printf("  Template     : %s\n", templateURL)
	fmt.Printf("  .env file    : %v\n", envFile)
	fmt.Printf("  Linting      : %v\n", linting)
	fmt.Printf("  Git hooks    : %v\n", gitHooks)
	fmt.Println()

	var confirmed bool
	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Proceed with this configuration?").
				Value(&confirmed),
		),
	).Run(); err != nil {
		return nil, fmt.Errorf("confirmation cancelled: %w", err)
	}

	if !confirmed {
		return nil, fmt.Errorf("setup cancelled by user")
	}

	return &config.Config{
		Dependencies: deps,
		Template:     templateURL,
		Linting:      linting,
		GitHooks:     gitHooks,
		EnvFile:      envFile,
	}, nil
}

// selectTemplate fetches the registry and shows a template picker.
// Falls back to a manual URL prompt if the registry is unreachable.
func selectTemplate(registryURL string, verbose, jsonLogs bool) (string, error) {
	log, err := logger.New(verbose, jsonLogs)
	if err != nil {
		return promptManualURL()
	}
	defer log.Close()

	client := registry.NewClient(registryURL, log)
	reg, err := client.Fetch(false)
	if err != nil {
		ux.Warning("Could not reach template registry — enter a URL manually.")
		return promptManualURL()
	}

	templates := reg.ValidTemplates()
	if len(templates) == 0 {
		ux.Warning("Registry has no templates — enter a URL manually.")
		return promptManualURL()
	}

	options := make([]huh.Option[string], 0, len(templates)+1)
	for _, t := range templates {
		label := fmt.Sprintf("%-22s %s", t.Name, t.Description)
		if len(label) > 70 {
			label = label[:67] + "…"
		}
		options = append(options, huh.NewOption(label, t.URL))
	}
	options = append(options, huh.NewOption("✎  Enter a custom Git URL…", "__custom__"))

	var selectedURL string
	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Choose a starter template").
				Description("Templates from the DevForge registry").
				Options(options...).
				Value(&selectedURL),
		),
	).Run(); err != nil {
		return "", fmt.Errorf("template selection cancelled: %w", err)
	}

	if selectedURL == "__custom__" {
		return promptManualURL()
	}
	return selectedURL, nil
}

// promptManualURL asks the user to type a Git repository URL.
func promptManualURL() (string, error) {
	var repoURL string
	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Template Git URL").
				Description("e.g. https://github.com/your-org/my-template.git").
				Placeholder("https://github.com/…").
				Value(&repoURL).
				Validate(func(s string) error {
					s = strings.TrimSpace(s)
					if s == "" {
						return fmt.Errorf("a template URL is required")
					}
					if !strings.HasPrefix(s, "https://") {
						return fmt.Errorf("URL must start with https://")
					}
					return nil
				}),
		),
	).Run(); err != nil {
		return "", fmt.Errorf("URL input cancelled: %w", err)
	}
	return strings.TrimSpace(repoURL), nil
}
