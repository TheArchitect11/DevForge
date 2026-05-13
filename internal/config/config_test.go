package config

import (
	"strings"
	"testing"
)

func TestLoadFromBytes_Valid(t *testing.T) {
	yaml := []byte(`
dependencies:
  - name: node
    version: "20"
  - name: git

template: https://github.com/user/template.git

linting: true
gitHooks: false
envFile: true
`)

	cfg, err := LoadFromBytes(yaml)
	if err != nil {
		t.Fatalf("LoadFromBytes() unexpected error: %v", err)
	}

	if len(cfg.Dependencies) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(cfg.Dependencies))
	}
	if cfg.Dependencies[0].Name != "node" || cfg.Dependencies[0].Version != "20" {
		t.Errorf("dependency[0] = %+v, want node@20", cfg.Dependencies[0])
	}
	if cfg.Dependencies[1].Name != "git" {
		t.Errorf("dependency[1].Name = %q, want %q", cfg.Dependencies[1].Name, "git")
	}
	if cfg.Template != "https://github.com/user/template.git" {
		t.Errorf("Template = %q, want correct URL", cfg.Template)
	}
	if !cfg.Linting {
		t.Error("Linting should be true")
	}
	if cfg.GitHooks {
		t.Error("GitHooks should be false")
	}
	if !cfg.EnvFile {
		t.Error("EnvFile should be true")
	}
}

func TestLoadFromBytes_NoDeps(t *testing.T) {
	yaml := []byte(`
template: https://github.com/user/template.git
`)
	_, err := LoadFromBytes(yaml)
	if err == nil {
		t.Error("LoadFromBytes() should error when no dependencies specified")
	}
}

func TestLoadFromBytes_NoTemplate(t *testing.T) {
	yaml := []byte(`
dependencies:
  - name: node
`)
	_, err := LoadFromBytes(yaml)
	if err == nil {
		t.Error("LoadFromBytes() should error when no template specified")
	}
}

func TestLoadFromBytes_InvalidDepName(t *testing.T) {
	yaml := []byte(`
dependencies:
  - name: "../evil"
template: https://github.com/user/template.git
`)
	_, err := LoadFromBytes(yaml)
	if err == nil {
		t.Error("LoadFromBytes() should reject dependencies with invalid names")
	}
}

func TestLoadFromBytes_InvalidTemplateURL(t *testing.T) {
	yaml := []byte(`
dependencies:
  - name: node
template: ftp://evil.com/hax
`)
	_, err := LoadFromBytes(yaml)
	if err == nil {
		t.Error("LoadFromBytes() should reject non-http(s) template URLs")
	}
}

func TestLoadFromBytes_EmptyDepName(t *testing.T) {
	yaml := []byte(`
dependencies:
  - name: ""
template: https://github.com/user/template.git
`)
	_, err := LoadFromBytes(yaml)
	if err == nil {
		t.Error("LoadFromBytes() should reject empty dependency names")
	}
}

func TestConfig_ToYAML_RoundTrip(t *testing.T) {
	cfg := &Config{
		Dependencies: []Dependency{
			{Name: "node", Version: "20"},
			{Name: "git"},
		},
		Template: "https://github.com/example/tmpl.git",
		EnvFile:  true,
		Linting:  false,
		GitHooks: true,
		PostInit: []string{"npm install", "go mod tidy"},
	}

	yamlBytes, err := cfg.ToYAML()
	if err != nil {
		t.Fatalf("ToYAML() error: %v", err)
	}
	if len(yamlBytes) == 0 {
		t.Fatal("ToYAML() returned empty bytes")
	}

	// Round-trip: load the produced YAML back and verify key fields.
	loaded, err := LoadFromBytes(yamlBytes)
	if err != nil {
		t.Fatalf("LoadFromBytes(ToYAML()) error: %v\nYAML:\n%s", err, yamlBytes)
	}

	if len(loaded.Dependencies) != 2 {
		t.Errorf("want 2 dependencies, got %d", len(loaded.Dependencies))
	}
	if loaded.Dependencies[0].Name != "node" {
		t.Errorf("want dep[0].Name=node, got %q", loaded.Dependencies[0].Name)
	}
	if loaded.Dependencies[0].Version != "20" {
		t.Errorf("want dep[0].Version=20, got %q", loaded.Dependencies[0].Version)
	}
	if loaded.Template != cfg.Template {
		t.Errorf("want Template=%q, got %q", cfg.Template, loaded.Template)
	}
	if loaded.GitHooks != cfg.GitHooks {
		t.Errorf("GitHooks mismatch: want %v, got %v", cfg.GitHooks, loaded.GitHooks)
	}
	if len(loaded.PostInit) != 2 {
		t.Errorf("want 2 PostInit hooks, got %d", len(loaded.PostInit))
	}
}

func TestConfig_ToYAML_NoPostInit(t *testing.T) {
	cfg := &Config{
		Dependencies: []Dependency{{Name: "git"}},
		Template:     "https://github.com/example/tmpl.git",
	}
	yamlBytes, err := cfg.ToYAML()
	if err != nil {
		t.Fatalf("ToYAML() error: %v", err)
	}
	// Should contain the commented-out postInit hint, not a real array.
	if !strings.Contains(string(yamlBytes), "postInit") {
		t.Errorf("expected postInit comment in YAML output")
	}
}
