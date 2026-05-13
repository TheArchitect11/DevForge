package installer

import (
	"fmt"
	osexec "os/exec"

	"github.com/chinmay/devforge/internal/executor"
	"github.com/chinmay/devforge/internal/logger"
	"github.com/chinmay/devforge/internal/osdetect"
)

// NewFromOS returns the appropriate Installer based on the detected
// OS information. This replaces the old runtime.GOOS switch in favor
// of the richer OSInfo-based selection.
//
// For "yum" systems (RHEL/CentOS/Fedora family), DNF is preferred when
// available as it is the modern successor to YUM.
func NewFromOS(log *logger.Logger, exec *executor.Executor, osInfo osdetect.OSInfo) (Installer, error) {
	switch osInfo.PackageMgr {
	case "brew":
		return NewBrewInstaller(log, exec)
	case "apt":
		return NewAptInstaller(log, exec)
	case "dnf":
		return NewDnfInstaller(log, exec)
	case "yum":
		// Prefer DNF on modern RHEL-family systems (Fedora 22+, RHEL 8+, Rocky, Alma).
		// Fall back to YUM if DNF is not found.
		if _, err := osexec.LookPath("dnf"); err == nil {
			return NewDnfInstaller(log, exec)
		}
		return NewYumInstaller(log, exec)
	case "choco":
		return NewChocoInstaller(log, exec)
	default:
		return nil, fmt.Errorf("no installer available for package manager %q (OS: %s)", osInfo.PackageMgr, osInfo.Name)
	}
}
