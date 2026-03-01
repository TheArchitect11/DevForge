package installer

import (
	"fmt"
	"strings"

	"github.com/chinmay/devforge/internal/executor"
	"github.com/chinmay/devforge/internal/logger"
)

// AptInstaller manages dependencies via APT on Debian/Ubuntu systems.
type AptInstaller struct {
	baseInstaller
}

// NewAptInstaller creates an AptInstaller after verifying that apt-get
// is available.
func NewAptInstaller(log *logger.Logger, exec *executor.Executor) (*AptInstaller, error) {
	ai := &AptInstaller{
		baseInstaller: baseInstaller{log: log, exec: exec},
	}

	if err := ai.checkApt(); err != nil {
		return nil, err
	}

	return ai, nil
}

func (a *AptInstaller) checkApt() error {
	_, err := a.exec.Run("apt-get", "--version")
	if err != nil {
		return fmt.Errorf("apt-get is not installed or not in PATH: %w", err)
	}
	return nil
}

// IsInstalled checks if a package is installed via dpkg.
func (a *AptInstaller) IsInstalled(name string) (bool, error) {
	a.log.Debug(fmt.Sprintf("checking if %q is installed via dpkg", name))
	result, err := a.exec.Run("dpkg", "-s", name)
	if err != nil {
		return false, nil
	}
	if result.DryRun {
		return false, nil
	}
	return strings.Contains(result.Stdout, "Status: install ok installed"), nil
}

// Install installs a package using apt-get. If a specific version is
// provided, it passes name=version to apt-get.
func (a *AptInstaller) Install(name string, version string) error {
	pkg := name
	if !isLatest(version) {
		pkg = fmt.Sprintf("%s=%s*", name, version)
	}

	a.log.Info(fmt.Sprintf("installing %q via apt-get...", pkg))
	_, err := a.exec.Run("sudo", "apt-get", "install", "-y", pkg)
	if err != nil {
		return fmt.Errorf("failed to install %q via apt-get: %w", pkg, err)
	}
	a.log.Info(fmt.Sprintf("successfully installed %q", pkg))
	return nil
}

// GetVersion returns the installed version of an APT package.
func (a *AptInstaller) GetVersion(name string) (string, error) {
	result, err := a.exec.Run("dpkg-query", "--showformat=${Version}", "--show", name)
	if err != nil {
		return "", fmt.Errorf("failed to get version for %q: %w", name, err)
	}
	if result.DryRun {
		return "dry-run", nil
	}
	return strings.TrimSpace(result.Stdout), nil
}
