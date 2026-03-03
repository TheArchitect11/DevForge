package config

import "testing"

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
