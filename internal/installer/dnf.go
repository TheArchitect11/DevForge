package installer

import (
	"fmt"
	"strings"

	"github.com/chinmay/devforge/internal/executor"
	"github.com/chinmay/devforge/internal/logger"
)

// DnfInstaller manages dependencies via DNF on modern Fedora/RHEL/Rocky/Alma systems.
// DNF is the successor to YUM and is the default package manager on Fedora 22+,
// RHEL 8+, Rocky Linux, and AlmaLinux.
type DnfInstaller struct {
	baseInstaller
}

// NewDnfInstaller creates a DnfInstaller after verifying that dnf is available.
func NewDnfInstaller(log *logger.Logger, exec *executor.Executor) (*DnfInstaller, error) {
	di := &DnfInstaller{
		baseInstaller: baseInstaller{log: log, exec: exec},
	}

	if err := di.checkDnf(); err != nil {
		return nil, err
	}

	return di, nil
}

func (d *DnfInstaller) checkDnf() error {
	_, err := d.exec.Run("dnf", "--version")
	if err != nil {
		return fmt.Errorf("dnf is not installed or not in PATH: %w", err)
	}
	return nil
}

// IsInstalled checks if a package is installed via rpm.
func (d *DnfInstaller) IsInstalled(name string) (bool, error) {
	d.log.Debug(fmt.Sprintf("checking if %q is installed via rpm", name))
	result, err := d.exec.Run("rpm", "-q", name)
	if err != nil {
		return false, nil
	}
	if result.DryRun {
		return false, nil
	}
	return !strings.Contains(result.Stdout, "not installed"), nil
}

// Install installs a package using dnf. If a specific version is provided,
// it passes name-version to dnf.
func (d *DnfInstaller) Install(name string, version string) error {
	pkg := name
	if !isLatest(version) {
		pkg = fmt.Sprintf("%s-%s", name, version)
	}

	d.log.Info(fmt.Sprintf("installing %q via dnf...", pkg))
	_, err := d.exec.Run("sudo", "dnf", "install", "-y", pkg)
	if err != nil {
		return fmt.Errorf("failed to install %q via dnf: %w", pkg, err)
	}
	d.log.Info(fmt.Sprintf("successfully installed %q", pkg))
	return nil
}

// GetVersion returns the installed version of a DNF/RPM package.
func (d *DnfInstaller) GetVersion(name string) (string, error) {
	result, err := d.exec.Run("rpm", "-q", "--queryformat", "%{VERSION}", name)
	if err != nil {
		return "", fmt.Errorf("failed to get version for %q: %w", name, err)
	}
	if result.DryRun {
		return "dry-run", nil
	}
	return strings.TrimSpace(result.Stdout), nil
}


// Upgrade upgrades an installed package to the given version (or latest).
func (d *DnfInstaller) Upgrade(name, version string) error {
	pkg := name
	if !isLatest(version) {
		pkg = fmt.Sprintf("%s-%s", name, version)
	}
	d.log.Info(fmt.Sprintf("upgrading %q via dnf", pkg))
	if _, err := d.exec.Run("sudo", "dnf", "update", "-y", pkg); err != nil {
		return fmt.Errorf("dnf update %q failed: %w", pkg, err)
	}
	return nil
}
