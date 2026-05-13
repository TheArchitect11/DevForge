package installer

import (
	"fmt"
	"strings"

	"github.com/chinmay/devforge/internal/executor"
	"github.com/chinmay/devforge/internal/logger"
	"github.com/chinmay/devforge/internal/semver"
)

// BrewInstaller manages dependencies via Homebrew on macOS.
type BrewInstaller struct{ baseInstaller }

// NewBrewInstaller verifies that brew is available, then returns an installer.
func NewBrewInstaller(log *logger.Logger, exec *executor.Executor) (*BrewInstaller, error) {
	bi := &BrewInstaller{baseInstaller{log: log, exec: exec}}
	if _, err := bi.exec.Run("brew", "--version"); err != nil {
		return nil, fmt.Errorf("homebrew is not installed or not in PATH: %w", err)
	}
	return bi, nil
}

// IsInstalled checks if a formula is present in the local Homebrew prefix.
func (b *BrewInstaller) IsInstalled(name string) (bool, error) {
	b.log.Debug(fmt.Sprintf("checking if %q is installed via brew", name))
	_, err := b.exec.Run("brew", "list", "--formula", name)
	return err == nil, nil
}

// Install installs a Homebrew formula. Homebrew version pinning uses only
// the major version (e.g. node@20), so a full semver like "20.11.0" is
// automatically reduced to its major component.
func (b *BrewInstaller) Install(name, version string) error {
	formula := b.formulaName(name, version)
	b.log.Info(fmt.Sprintf("installing %q via Homebrew", formula))
	if _, err := b.exec.Run("brew", "install", formula); err != nil {
		return fmt.Errorf("brew install %q failed: %w", formula, err)
	}
	b.log.Info(fmt.Sprintf("installed %q", formula))
	return nil
}

// Upgrade upgrades an existing formula to the latest (or pinned major) version.
func (b *BrewInstaller) Upgrade(name, version string) error {
	formula := b.formulaName(name, version)
	b.log.Info(fmt.Sprintf("upgrading %q via Homebrew", formula))
	if _, err := b.exec.Run("brew", "upgrade", formula); err != nil {
		return fmt.Errorf("brew upgrade %q failed: %w", formula, err)
	}
	b.log.Info(fmt.Sprintf("upgraded %q", formula))
	return nil
}

// GetVersion returns the installed version, or "" if not installed.
func (b *BrewInstaller) GetVersion(name string) (string, error) {
	result, err := b.exec.Run("brew", "list", "--versions", name)
	if err != nil {
		return "", nil // not installed
	}
	if result.DryRun {
		return "dry-run", nil
	}
	// Output: "name version [version2 …]"
	parts := strings.Fields(result.Stdout)
	if len(parts) < 2 {
		return "", nil // installed but version indeterminate
	}
	return parts[1], nil
}

// formulaName converts a name + version into the correct brew formula identifier.
// Homebrew uses name@MAJOR (e.g. node@20) for version pinning.
func (b *BrewInstaller) formulaName(name, version string) string {
	if isLatest(version) {
		return name
	}
	v, err := semver.Parse(version)
	if err == nil && !v.IsZero() {
		return fmt.Sprintf("%s@%d", name, v.Major)
	}
	// Fallback: use version as-is (handles non-semver strings)
	return fmt.Sprintf("%s@%s", name, version)
}
