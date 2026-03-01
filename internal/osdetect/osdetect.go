// Package osdetect provides operating system detection and validation
// for DevForge. Supports macOS, Linux (Debian/Ubuntu, RHEL/CentOS/Fedora),
// and Windows.
package osdetect

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strings"
)

// supportedOS maps GOOS values to human-friendly names.
var supportedOS = map[string]string{
	"darwin":  "macOS",
	"linux":   "Linux",
	"windows": "Windows",
}

// Result holds the outcome of a basic OS detection check.
// Retained for backward compatibility with Phase 1.
type Result struct {
	OS          string
	DisplayName string
	Arch        string
	Supported   bool
}

// OSInfo provides detailed operating system information including the
// distro family and recommended package manager.
type OSInfo struct {
	// Name is the human-readable OS name (e.g. "macOS", "Ubuntu", "Windows").
	Name string
	// Family is the OS family used for package manager selection
	// (e.g. "darwin", "debian", "rhel", "windows").
	Family string
	// PackageMgr is the recommended package manager for this OS
	// (e.g. "brew", "apt", "yum", "choco").
	PackageMgr string
	// Arch is the CPU architecture (e.g. "arm64", "amd64").
	Arch string
	// RawOS is the raw GOOS value.
	RawOS string
}

// Detect performs a basic OS detection and returns a Result.
// Retained for backward compatibility with Phase 1 code.
func Detect() (Result, error) {
	goos := runtime.GOOS
	arch := runtime.GOARCH

	displayName, ok := supportedOS[goos]
	if !ok {
		return Result{
			OS:        goos,
			Arch:      arch,
			Supported: false,
		}, fmt.Errorf("unsupported operating system: %q (arch: %s). DevForge supports: macOS, Linux, Windows", goos, arch)
	}

	return Result{
		OS:          goos,
		DisplayName: displayName,
		Arch:        arch,
		Supported:   true,
	}, nil
}

// DetectFull performs comprehensive OS detection including distro family
// and package manager determination.
func DetectFull() (OSInfo, error) {
	goos := runtime.GOOS
	arch := runtime.GOARCH

	switch goos {
	case "darwin":
		return OSInfo{
			Name:       "macOS",
			Family:     "darwin",
			PackageMgr: "brew",
			Arch:       arch,
			RawOS:      goos,
		}, nil

	case "linux":
		return detectLinux(arch)

	case "windows":
		return OSInfo{
			Name:       "Windows",
			Family:     "windows",
			PackageMgr: "choco",
			Arch:       arch,
			RawOS:      goos,
		}, nil

	default:
		return OSInfo{}, fmt.Errorf("unsupported operating system: %q (arch: %s). DevForge supports: macOS, Linux, Windows", goos, arch)
	}
}

// detectLinux parses /etc/os-release to determine the Linux distribution
// family and appropriate package manager.
func detectLinux(arch string) (OSInfo, error) {
	info := OSInfo{
		Name:  "Linux",
		Arch:  arch,
		RawOS: "linux",
	}

	release, err := parseOSRelease()
	if err != nil {
		// If we can't read os-release, default to generic Linux with apt.
		info.Family = "debian"
		info.PackageMgr = "apt"
		return info, nil
	}

	prettyName, ok := release["PRETTY_NAME"]
	if ok {
		info.Name = prettyName
	}

	idLike := strings.ToLower(release["ID_LIKE"])
	id := strings.ToLower(release["ID"])

	switch {
	case strings.Contains(id, "ubuntu") || strings.Contains(id, "debian") ||
		strings.Contains(idLike, "ubuntu") || strings.Contains(idLike, "debian"):
		info.Family = "debian"
		info.PackageMgr = "apt"

	case strings.Contains(id, "rhel") || strings.Contains(id, "centos") ||
		strings.Contains(id, "fedora") || strings.Contains(id, "rocky") ||
		strings.Contains(idLike, "rhel") || strings.Contains(idLike, "fedora"):
		info.Family = "rhel"
		info.PackageMgr = "yum"

	default:
		// Default to apt for unknown distros.
		info.Family = "debian"
		info.PackageMgr = "apt"
	}

	return info, nil
}

// parseOSRelease reads and parses /etc/os-release into a key-value map.
func parseOSRelease() (map[string]string, error) {
	file, err := os.Open("/etc/os-release")
	if err != nil {
		return nil, fmt.Errorf("failed to open /etc/os-release: %w", err)
	}
	defer file.Close()

	result := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		value := strings.Trim(parts[1], "\"'")
		result[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading /etc/os-release: %w", err)
	}

	return result, nil
}
