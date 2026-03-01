// Package plugins provides discovery, validation, and execution of
// DevForge plugins. Plugins are standalone executables that communicate
// via JSON over stdin/stdout.
package plugins

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/chinmay/devforge/internal/logger"
)

const pluginPrefix = "devforge-plugin-"

// PluginInput is the JSON contract sent to plugins via stdin.
type PluginInput struct {
	ProjectPath string                 `json:"projectPath"`
	Config      map[string]interface{} `json:"config"`
	DryRun      bool                   `json:"dryRun"`
}

// PluginOutput is the expected JSON response from a plugin.
type PluginOutput struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// PluginInfo describes a discovered plugin.
type PluginInfo struct {
	Name string
	Path string
}

// Manager handles plugin discovery, validation, and execution.
type Manager struct {
	pluginDir string
	log       *logger.Logger
}

// NewManager creates a plugin Manager. Plugins are discovered in
// ~/.devforge/plugins/.
func NewManager(log *logger.Logger) (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to determine home directory: %w", err)
	}

	dir := filepath.Join(home, ".devforge", "plugins")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create plugins directory: %w", err)
	}

	return &Manager{
		pluginDir: dir,
		log:       log,
	}, nil
}

// Discover scans the plugin directory for executables matching the
// naming convention devforge-plugin-<name>.
func (m *Manager) Discover() ([]PluginInfo, error) {
	entries, err := os.ReadDir(m.pluginDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugins directory: %w", err)
	}

	var plugins []PluginInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasPrefix(name, pluginPrefix) {
			continue
		}

		fullPath := filepath.Join(m.pluginDir, name)
		info, err := entry.Info()
		if err != nil {
			m.log.Warn(fmt.Sprintf("skipping plugin %q: %v", name, err))
			continue
		}

		// Verify it's executable.
		if info.Mode()&0o111 == 0 {
			m.log.Warn(fmt.Sprintf("skipping plugin %q: not executable", name))
			continue
		}

		pluginName := strings.TrimPrefix(name, pluginPrefix)
		plugins = append(plugins, PluginInfo{
			Name: pluginName,
			Path: fullPath,
		})
	}

	return plugins, nil
}

// Run executes a plugin by name, sending the input contract via stdin
// and capturing the JSON response from stdout.
func (m *Manager) Run(name string, input PluginInput) (*PluginOutput, error) {
	plugins, err := m.Discover()
	if err != nil {
		return nil, err
	}

	var pluginPath string
	for _, p := range plugins {
		if p.Name == name {
			pluginPath = p.Path
			break
		}
	}

	if pluginPath == "" {
		return nil, fmt.Errorf("plugin %q not found in %s", name, m.pluginDir)
	}

	m.log.Info(fmt.Sprintf("running plugin %q...", name))

	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal plugin input: %w", err)
	}

	cmd := exec.Command(pluginPath)
	cmd.Stdin = bytes.NewReader(inputJSON)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("plugin %q failed: %w\nstderr: %s", name, err, stderr.String())
	}

	var output PluginOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		return nil, fmt.Errorf("plugin %q returned invalid JSON: %w\nraw output: %s", name, err, stdout.String())
	}

	if !output.Success {
		return &output, fmt.Errorf("plugin %q reported failure: %s", name, output.Message)
	}

	m.log.Info(fmt.Sprintf("plugin %q completed: %s", name, output.Message))
	return &output, nil
}
