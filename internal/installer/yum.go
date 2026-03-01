package installer

import (
	"fmt"
	"strings"

	"github.com/chinmay/devforge/internal/executor"
	"github.com/chinmay/devforge/internal/logger"
)

// YumInstaller manages dependencies via YUM on RHEL/CentOS/Fedora systems.
type YumInstaller struct {
	baseInstaller
}

// NewYumInstaller creates a YumInstaller after verifying that yum
// is available.
func NewYumInstaller(log *logger.Logger, exec *executor.Executor) (*YumInstaller, error) {
	yi := &YumInstaller{
		baseInstaller: baseInstaller{log: log, exec: exec},
	}

	if err := yi.checkYum(); err != nil {
		return nil, err
	}

	return yi, nil
}

func (y *YumInstaller) checkYum() error {
	_, err := y.exec.Run("yum", "--version")
	if err != nil {
		return fmt.Errorf("yum is not installed or not in PATH: %w", err)
	}
	return nil
}

// IsInstalled checks if a package is installed via rpm.
func (y *YumInstaller) IsInstalled(name string) (bool, error) {
	y.log.Debug(fmt.Sprintf("checking if %q is installed via rpm", name))
	result, err := y.exec.Run("rpm", "-q", name)
	if err != nil {
		return false, nil
	}
	if result.DryRun {
		return false, nil
	}
	return !strings.Contains(result.Stdout, "not installed"), nil
}

// Install installs a package using yum. If a specific version is
// provided, it passes name-version to yum.
func (y *YumInstaller) Install(name string, version string) error {
	pkg := name
	if !isLatest(version) {
		pkg = fmt.Sprintf("%s-%s", name, version)
	}

	y.log.Info(fmt.Sprintf("installing %q via yum...", pkg))
	_, err := y.exec.Run("sudo", "yum", "install", "-y", pkg)
	if err != nil {
		return fmt.Errorf("failed to install %q via yum: %w", pkg, err)
	}
	y.log.Info(fmt.Sprintf("successfully installed %q", pkg))
	return nil
}

// GetVersion returns the installed version of a YUM/RPM package.
func (y *YumInstaller) GetVersion(name string) (string, error) {
	result, err := y.exec.Run("rpm", "-q", "--queryformat", "%{VERSION}", name)
	if err != nil {
		return "", fmt.Errorf("failed to get version for %q: %w", name, err)
	}
	if result.DryRun {
		return "dry-run", nil
	}
	return strings.TrimSpace(result.Stdout), nil
}
