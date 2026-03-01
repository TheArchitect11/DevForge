// Package installer defines the interface for platform-specific package
// installers and provides implementations for Homebrew, APT, YUM, and
// Chocolatey.
package installer

import (
	"github.com/chinmay/devforge/internal/executor"
	"github.com/chinmay/devforge/internal/logger"
)

// Installer is the interface that platform-specific package managers
// must implement.
type Installer interface {
	// IsInstalled checks whether a given dependency is already present
	// on the system.
	IsInstalled(name string) (bool, error)

	// Install installs the given dependency. If version is empty or
	// "latest", the latest version is installed. Otherwise it attempts
	// to install the specified version.
	Install(name string, version string) error

	// GetVersion returns the installed version string for a dependency,
	// or an error if the dependency is not installed.
	GetVersion(name string) (string, error)
}

// baseInstaller provides common fields and methods shared by all
// installer implementations.
type baseInstaller struct {
	log  *logger.Logger
	exec *executor.Executor
}

// isLatest returns true if the version string indicates "install latest".
func isLatest(version string) bool {
	return version == "" || version == "latest"
}
