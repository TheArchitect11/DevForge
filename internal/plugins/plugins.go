// Package plugins provides discovery, validation, and execution of
// DevForge plugins. Plugins are standalone executables that communicate
// via JSON over stdin/stdout. Each plugin run is bounded by a 30-second
// timeout to prevent hangs.
package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/chinmay/devforge/internal/logger"
)

const (
	pluginPrefix  = "devforge-plugin-"
	pluginTimeout = 30 * time.Second
)

// PluginInput is the JSON contract sent to plugins via stdin.
type PluginInput struct {
	ProjectPath string                 `json:"projectPath"`
	Config      map[string]interface{} `json:"config"`
	DryRun      bool                   `json:"dryRun"`
}

// PluginOutput is the expected JSON response from a plugin via stdout.
type PluginOutput struct {
	Success   bool            `json:"success"`
	Message   string          `json:"message"`
	Artifacts []string        `json:"artifacts,omitempty"` // files created/modified by the plugin
	Data      json.RawMessage `json:"data,omitempty"`
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

// NewManager creates a Manager. Plugins are discovered in ~/.devforge/plugins/.
func NewManager(log *logger.Logger) (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to determine home directory: %w", err)
	}

	dir := filepath.Join(home, ".devforge", "plugins")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create plugins directory %s: %w", dir, err)
	}

	return &Manager{pluginDir: dir, log: log}, nil
}

// Discover scans the plugin directory for executables named devforge-plugin-<name>.
func (m *Manager) Discover() ([]PluginInfo, error) {
	entries, err := os.ReadDir(m.pluginDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugins directory %s: %w", m.pluginDir, err)
	}

	var found []PluginInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasPrefix(name, pluginPrefix) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			m.log.Warn(fmt.Sprintf("skipping plugin %q: stat failed: %v", name, err))
			continue
		}

		if info.Mode()&0o111 == 0 {
			m.log.Warn(fmt.Sprintf("skipping plugin %q: not executable (chmod +x to fix)", name))
			continue
		}

		found = append(found, PluginInfo{
			Name: strings.TrimPrefix(name, pluginPrefix),
			Path: filepath.Join(m.pluginDir, name),
		})
	}

	return found, nil
}

// Run executes a plugin by name, sending PluginInput as JSON on stdin
// and parsing the JSON response from stdout. The call is bounded by
// pluginTimeout (30 s) to prevent indefinite hangs.
func (m *Manager) Run(name string, input PluginInput) (*PluginOutput, error) {
	plugins, err := m.Discover()
	if err != nil {
		return nil, err
	}

	var pluginPath string
	for _, p := range plugins {
		if strings.EqualFold(p.Name, name) {
			pluginPath = p.Path
			break
		}
	}

	if pluginPath == "" {
		return nil, fmt.Errorf("plugin %q not found in %s\n  Tip: run 'devforge plugin list' to see installed plugins", name, m.pluginDir)
	}

	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal plugin input: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), pluginTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, pluginPath)
	cmd.Stdin = bytes.NewReader(inputJSON)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	m.log.Info(fmt.Sprintf("running plugin %q (timeout: %s)", name, pluginTimeout))

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("plugin %q timed out after %s", name, pluginTimeout)
		}
		errDetail := strings.TrimSpace(stderr.String())
		if errDetail == "" {
			errDetail = err.Error()
		}
		return nil, fmt.Errorf("plugin %q failed: %s", name, errDetail)
	}

	var output PluginOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		preview := stdout.String()
		if len(preview) > 200 {
			preview = preview[:197] + "…"
		}
		return nil, fmt.Errorf("plugin %q returned invalid JSON: %w\n  raw output: %s", name, err, preview)
	}

	if !output.Success {
		return &output, fmt.Errorf("plugin %q reported failure: %s", name, output.Message)
	}

	m.log.Info(fmt.Sprintf("plugin %q completed: %s", name, output.Message))
	return &output, nil
}
