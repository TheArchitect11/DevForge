// Package installer defines the Installer interface and provides
// implementations for Homebrew, APT, YUM, DNF, and Chocolatey.
package installer

import (
	"github.com/chinmay/devforge/internal/executor"
	"github.com/chinmay/devforge/internal/logger"
)

// Installer is the interface every platform package manager must implement.
type Installer interface {
	// IsInstalled returns true if the named tool is present on the system.
	IsInstalled(name string) (bool, error)

	// Install installs the tool at the given version.
	// Pass "" or "latest" to install the newest available version.
	Install(name string, version string) error

	// Upgrade upgrades an already-installed tool to the given version.
	// If the tool is not installed, behaviour is implementation-defined
	// (most package managers will install it).
	Upgrade(name string, version string) error

	// GetVersion returns the installed version string, or "" when not installed.
	GetVersion(name string) (string, error)
}

// baseInstaller holds the shared logger and executor used by all
// concrete installer implementations.
type baseInstaller struct {
	log  *logger.Logger
	exec *executor.Executor
}

// isLatest returns true when the version string means "use the newest available".
func isLatest(version string) bool {
	return version == "" || version == "latest"
}
