package installer

import (
	"fmt"
	"strings"

	"github.com/chinmay/devforge/internal/executor"
	"github.com/chinmay/devforge/internal/logger"
)

// ChocoInstaller manages dependencies via Chocolatey on Windows.
type ChocoInstaller struct {
	baseInstaller
}

// NewChocoInstaller creates a ChocoInstaller after verifying that
// choco is available.
func NewChocoInstaller(log *logger.Logger, exec *executor.Executor) (*ChocoInstaller, error) {
	ci := &ChocoInstaller{
		baseInstaller: baseInstaller{log: log, exec: exec},
	}

	if err := ci.checkChoco(); err != nil {
		return nil, err
	}

	return ci, nil
}

func (c *ChocoInstaller) checkChoco() error {
	_, err := c.exec.Run("choco", "--version")
	if err != nil {
		return fmt.Errorf("chocolatey is not installed or not in PATH: %w", err)
	}
	return nil
}

// IsInstalled checks if a package is installed via Chocolatey.
func (c *ChocoInstaller) IsInstalled(name string) (bool, error) {
	c.log.Debug(fmt.Sprintf("checking if %q is installed via choco", name))
	result, err := c.exec.Run("choco", "list", "--local-only", "--exact", name)
	if err != nil {
		return false, nil
	}
	if result.DryRun {
		return false, nil
	}
	// choco list output contains "X packages installed" at the end.
	// If the package is found, its name appears in the output.
	return strings.Contains(strings.ToLower(result.Stdout), strings.ToLower(name)), nil
}

// Install installs a package using Chocolatey. If a specific version
// is provided, it uses the --version flag.
func (c *ChocoInstaller) Install(name string, version string) error {
	c.log.Info(fmt.Sprintf("installing %q via Chocolatey...", name))

	args := []string{"install", name, "-y"}
	if !isLatest(version) {
		args = append(args, "--version", version)
	}

	_, err := c.exec.Run("choco", args...)
	if err != nil {
		return fmt.Errorf("failed to install %q via Chocolatey: %w", name, err)
	}
	c.log.Info(fmt.Sprintf("successfully installed %q", name))
	return nil
}

// GetVersion returns the installed version of a Chocolatey package.
func (c *ChocoInstaller) GetVersion(name string) (string, error) {
	result, err := c.exec.Run("choco", "list", "--local-only", "--exact", name)
	if err != nil {
		return "", fmt.Errorf("failed to get version for %q: %w", name, err)
	}
	if result.DryRun {
		return "dry-run", nil
	}

	// Parse "name version" lines from choco output.
	for _, line := range strings.Split(result.Stdout, "\n") {
		parts := strings.Fields(line)
		if len(parts) >= 2 && strings.EqualFold(parts[0], name) {
			return parts[1], nil
		}
	}
	return "", fmt.Errorf("could not determine version for %q", name)
}
