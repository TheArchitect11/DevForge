package installer

import (
	"fmt"

	"github.com/chinmay/devforge/internal/executor"
	"github.com/chinmay/devforge/internal/logger"
	"github.com/chinmay/devforge/internal/osdetect"
)

// NewFromOS returns the appropriate Installer based on the detected
// OS information. This replaces the old runtime.GOOS switch in favor
// of the richer OSInfo-based selection.
func NewFromOS(log *logger.Logger, exec *executor.Executor, osInfo osdetect.OSInfo) (Installer, error) {
	switch osInfo.PackageMgr {
	case "brew":
		return NewBrewInstaller(log, exec)
	case "apt":
		return NewAptInstaller(log, exec)
	case "yum":
		return NewYumInstaller(log, exec)
	case "choco":
		return NewChocoInstaller(log, exec)
	default:
		return nil, fmt.Errorf("no installer available for package manager %q (OS: %s)", osInfo.PackageMgr, osInfo.Name)
	}
}
