// Package config handles loading, parsing, validating, and serialising
// DevForge configuration files (YAML).
package config

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/spf13/viper"

	"github.com/chinmay/devforge/internal/security"
)

// Dependency represents a single tool that DevForge ensures is installed.
type Dependency struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
}

// Config is the top-level structure loaded from a devforge.yaml file.
type Config struct {
	Dependencies []Dependency `mapstructure:"dependencies"`
	Template     string       `mapstructure:"template"`
	RegistryURL  string       `mapstructure:"registryUrl"`
	Linting      bool         `mapstructure:"linting"`
	GitHooks     bool         `mapstructure:"gitHooks"`
	EnvFile      bool         `mapstructure:"envFile"`
	// PostInit commands run inside the new project directory after scaffolding.
	// Examples: "npm install", "go mod tidy", "pip install -r requirements.txt"
	PostInit []string `mapstructure:"postInit"`
}

// Load reads configuration from configPath (or default search paths when empty).
func Load(configPath string) (*Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("default")
		v.AddConfigPath("config")
		v.AddConfigPath(".")
	}
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}
	return &cfg, nil
}

// LoadFromBytes reads configuration from an in-memory YAML slice.
func LoadFromBytes(data []byte) (*Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(bytes.NewReader(data)); err != nil {
		return nil, fmt.Errorf("failed to read config from bytes: %w", err)
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config bytes: %w", err)
	}
	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}
	return &cfg, nil
}

// ToYAML serialises the Config back to a YAML byte slice with comments.
// This is used by the --save-config flag to persist wizard-built configs.
func (c *Config) ToYAML() ([]byte, error) {
	var sb strings.Builder

	sb.WriteString("# DevForge project configuration\n")
	sb.WriteString("# Run: devforge init <project-name>\n\n")

	sb.WriteString("dependencies:\n")
	for _, dep := range c.Dependencies {
		if dep.Version != "" && dep.Version != "latest" {
			sb.WriteString(fmt.Sprintf("  - name: %s\n    version: %q\n", dep.Name, dep.Version))
		} else {
			sb.WriteString(fmt.Sprintf("  - name: %s\n", dep.Name))
		}
	}

	sb.WriteString(fmt.Sprintf("\ntemplate: %q\n", c.Template))
	sb.WriteString(fmt.Sprintf("\nenvFile:  %v\n", c.EnvFile))
	sb.WriteString(fmt.Sprintf("linting:  %v\n", c.Linting))
	sb.WriteString(fmt.Sprintf("gitHooks: %v\n", c.GitHooks))

	if len(c.PostInit) > 0 {
		sb.WriteString("\npostInit:\n")
		for _, hook := range c.PostInit {
			sb.WriteString(fmt.Sprintf("  - %q\n", hook))
		}
	} else {
		sb.WriteString("\n# postInit: []  # commands to run inside the new project after scaffolding\n")
	}

	return []byte(sb.String()), nil
}

// validate checks required fields and validates their content.
func validate(cfg *Config) error {
	var errs []string

	if len(cfg.Dependencies) == 0 {
		errs = append(errs, "at least one dependency must be specified")
	}
	for i, dep := range cfg.Dependencies {
		if dep.Name == "" {
			errs = append(errs, fmt.Sprintf("dependency[%d] has an empty name", i))
		} else if err := security.ValidateName(dep.Name); err != nil {
			errs = append(errs, fmt.Sprintf("dependency[%d]: %v", i, err))
		}
	}

	if cfg.Template == "" {
		errs = append(errs, "template URL must be specified")
	} else if err := security.ValidateURL(cfg.Template); err != nil {
		errs = append(errs, fmt.Sprintf("template URL: %v", err))
	}

	if cfg.RegistryURL != "" {
		if err := security.ValidateURL(cfg.RegistryURL); err != nil {
			errs = append(errs, fmt.Sprintf("registry URL: %v", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("configuration errors:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}
