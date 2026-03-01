package installer

import (
	"fmt"
	"strings"

	"github.com/chinmay/devforge/internal/executor"
	"github.com/chinmay/devforge/internal/logger"
)

// BrewInstaller manages dependencies via Homebrew on macOS.
type BrewInstaller struct {
	baseInstaller
}

// NewBrewInstaller creates a BrewInstaller after verifying that the
// `brew` command is available on the system.
func NewBrewInstaller(log *logger.Logger, exec *executor.Executor) (*BrewInstaller, error) {
	bi := &BrewInstaller{
		baseInstaller: baseInstaller{log: log, exec: exec},
	}

	if err := bi.checkBrew(); err != nil {
		return nil, err
	}

	return bi, nil
}

// checkBrew ensures the brew binary exists on the system.
func (b *BrewInstaller) checkBrew() error {
	result, err := b.exec.Run("brew", "--version")
	if err != nil {
		return fmt.Errorf("homebrew is not installed or not in PATH: %w", err)
	}
	if result.DryRun {
		b.log.Info("[dry-run] would verify Homebrew installation")
		return nil
	}
	b.log.Debug("homebrew detected", map[string]interface{}{
		"version": strings.Split(result.Stdout, "\n")[0],
	})
	return nil
}

// IsInstalled checks if a formula is installed via Homebrew.
func (b *BrewInstaller) IsInstalled(name string) (bool, error) {
	b.log.Debug(fmt.Sprintf("checking if %q is installed via brew", name))
	result, err := b.exec.Run("brew", "list", "--formula", name)
	if err != nil {
		return false, nil
	}
	if result.DryRun {
		return false, nil
	}
	return true, nil
}

// Install installs a formula using Homebrew. If a specific version is
// provided, it attempts to install name@version.
func (b *BrewInstaller) Install(name string, version string) error {
	formula := name
	if !isLatest(version) {
		formula = fmt.Sprintf("%s@%s", name, version)
	}

	b.log.Info(fmt.Sprintf("installing %q via Homebrew...", formula))
	_, err := b.exec.Run("brew", "install", formula)
	if err != nil {
		return fmt.Errorf("failed to install %q via Homebrew: %w", formula, err)
	}
	b.log.Info(fmt.Sprintf("successfully installed %q", formula))
	return nil
}

// GetVersion returns the installed version of a Homebrew formula.
func (b *BrewInstaller) GetVersion(name string) (string, error) {
	result, err := b.exec.Run("brew", "list", "--versions", name)
	if err != nil {
		return "", fmt.Errorf("failed to get version for %q: %w", name, err)
	}
	if result.DryRun {
		return "dry-run", nil
	}

	parts := strings.Fields(result.Stdout)
	if len(parts) < 2 {
		return "", fmt.Errorf("unexpected version output for %q: %s", name, result.Stdout)
	}
	return parts[1], nil
}
