package plugins

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/chinmay/devforge/internal/logger"
)

func newTestLogger(t *testing.T) *logger.Logger {
	t.Helper()
	log, err := logger.New(false)
	if err != nil {
		t.Fatalf("logger: %v", err)
	}
	t.Cleanup(func() { log.Close() })
	return log
}

func TestDiscover_EmptyDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin executability check differs on Windows")
	}
	log := newTestLogger(t)
	m := &Manager{pluginDir: t.TempDir(), log: log}
	found, err := m.Discover()
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}
	if len(found) != 0 {
		t.Errorf("expected 0 plugins in empty dir, got %d", len(found))
	}
}

func TestDiscover_FindsExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("executable bit not applicable on Windows")
	}

	dir := t.TempDir()
	log := newTestLogger(t)

	// Create a valid plugin binary (just a shell script stub).
	pluginPath := filepath.Join(dir, "devforge-plugin-hello")
	if err := os.WriteFile(pluginPath, []byte("#!/bin/sh\necho '{}'"), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	// Create a non-executable file that should be ignored.
	ignoredPath := filepath.Join(dir, "devforge-plugin-ignored")
	if err := os.WriteFile(ignoredPath, []byte("#!/bin/sh"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	// Create a file without the plugin prefix — should be ignored.
	if err := os.WriteFile(filepath.Join(dir, "other-tool"), []byte("x"), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	m := &Manager{pluginDir: dir, log: log}
	found, err := m.Discover()
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}
	if len(found) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(found))
	}
	if found[0].Name != "hello" {
		t.Errorf("expected plugin name %q, got %q", "hello", found[0].Name)
	}
}

func TestRun_MissingPlugin(t *testing.T) {
	log := newTestLogger(t)
	m := &Manager{pluginDir: t.TempDir(), log: log}
	_, err := m.Run("nonexistent", PluginInput{})
	if err == nil {
		t.Error("expected error for missing plugin, got nil")
	}
}

func TestRun_ValidPlugin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell scripts not supported on Windows")
	}

	dir := t.TempDir()
	log := newTestLogger(t)

	// Write a plugin that echoes a valid success response.
	out, _ := json.Marshal(PluginOutput{
		Success:   true,
		Message:   "hello from plugin",
		Artifacts: []string{"README.md"},
	})
	script := "#!/bin/sh\necho '" + string(out) + "'"
	pluginPath := filepath.Join(dir, "devforge-plugin-greet")
	if err := os.WriteFile(pluginPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	m := &Manager{pluginDir: dir, log: log}
	result, err := m.Run("greet", PluginInput{ProjectPath: ".", DryRun: false})
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected Success=true, got false")
	}
	if result.Message != "hello from plugin" {
		t.Errorf("unexpected message: %q", result.Message)
	}
	if len(result.Artifacts) != 1 || result.Artifacts[0] != "README.md" {
		t.Errorf("unexpected artifacts: %v", result.Artifacts)
	}
}

func TestRun_PluginFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell scripts not supported on Windows")
	}

	dir := t.TempDir()
	log := newTestLogger(t)

	out, _ := json.Marshal(PluginOutput{Success: false, Message: "something broke"})
	script := "#!/bin/sh\necho '" + string(out) + "'"
	pluginPath := filepath.Join(dir, "devforge-plugin-badplugin")
	os.WriteFile(pluginPath, []byte(script), 0o755)

	m := &Manager{pluginDir: dir, log: log}
	_, err := m.Run("badplugin", PluginInput{})
	if err == nil {
		t.Error("expected error from failing plugin, got nil")
	}
}
