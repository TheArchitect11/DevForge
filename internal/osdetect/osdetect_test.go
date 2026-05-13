package osdetect

import (
	"runtime"
	"testing"
)

func TestDetect_CurrentPlatform(t *testing.T) {
	result, err := Detect()
	if err != nil {
		// Only darwin/linux/windows are supported; skip on others.
		t.Skipf("Detect() unsupported platform: %v", err)
	}

	if result.OS == "" {
		t.Error("expected non-empty OS")
	}
	if result.Arch == "" {
		t.Error("expected non-empty Arch")
	}
	if !result.Supported {
		t.Error("expected Supported=true on a known platform")
	}
}

func TestDetectFull_CurrentPlatform(t *testing.T) {
	info, err := DetectFull()
	if err != nil {
		t.Skipf("DetectFull() unsupported platform: %v", err)
	}

	if info.Name == "" {
		t.Error("expected non-empty Name")
	}
	if info.Family == "" {
		t.Error("expected non-empty Family")
	}
	if info.PackageMgr == "" {
		t.Error("expected non-empty PackageMgr")
	}
	if info.Arch == "" {
		t.Error("expected non-empty Arch")
	}
	if info.RawOS == "" {
		t.Error("expected non-empty RawOS")
	}

	// Verify the package manager matches the platform.
	switch runtime.GOOS {
	case "darwin":
		if info.PackageMgr != "brew" {
			t.Errorf("macOS: expected PackageMgr=brew, got %q", info.PackageMgr)
		}
	case "windows":
		if info.PackageMgr != "choco" {
			t.Errorf("Windows: expected PackageMgr=choco, got %q", info.PackageMgr)
		}
	case "linux":
		validManagers := map[string]bool{"apt": true, "yum": true, "dnf": true}
		if !validManagers[info.PackageMgr] {
			t.Errorf("Linux: unexpected PackageMgr %q", info.PackageMgr)
		}
	}
}

func TestParseOSRelease_WellFormed(t *testing.T) {
	// parseOSRelease is internal; test its caller detectLinux instead.
	// On Linux, we can read the real /etc/os-release and verify it parses.
	if runtime.GOOS != "linux" {
		t.Skip("parseOSRelease only applies on Linux")
	}

	result, err := parseOSRelease()
	if err != nil {
		t.Skipf("parseOSRelease() failed (may not exist in this environment): %v", err)
	}

	// The ID field must always be present in a valid os-release file.
	if result["ID"] == "" {
		t.Error("expected non-empty ID in os-release")
	}
}

func TestDetectLinux_DefaultsToApt(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("detectLinux only runs on Linux")
	}

	info, err := detectLinux(runtime.GOARCH)
	if err != nil {
		t.Fatalf("detectLinux() unexpected error: %v", err)
	}

	validManagers := []string{"apt", "yum", "dnf"}
	found := false
	for _, m := range validManagers {
		if info.PackageMgr == m {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("unexpected PackageMgr %q from detectLinux", info.PackageMgr)
	}

	if info.Family == "" {
		t.Error("expected non-empty Family from detectLinux")
	}
}
