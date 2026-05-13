// Package envgen generates .env files from .env.template files.
// It uses the same huh TUI as the wizard so the prompting style is
// consistent throughout the DevForge experience.
package envgen

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/chinmay/devforge/internal/logger"
	"github.com/chinmay/devforge/internal/rollback"
	"github.com/chinmay/devforge/internal/ux"
)

// Generator handles .env file creation from .env.template files.
type Generator struct {
	log    *logger.Logger
	rb     *rollback.Manager
	dryRun bool
}

// NewGenerator creates a Generator.
func NewGenerator(log *logger.Logger, rb *rollback.Manager, dryRun bool) *Generator {
	return &Generator{log: log, rb: rb, dryRun: dryRun}
}

// Generate reads projectDir/.env.template, prompts the user for each
// key using an interactive form, then writes projectDir/.env.
// If no .env.template exists the function returns nil silently.
// An existing .env is never overwritten.
func (g *Generator) Generate(projectDir string) error {
	templatePath := filepath.Join(projectDir, ".env.template")
	envPath := filepath.Join(projectDir, ".env")

	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		g.log.Info("no .env.template found, skipping env generation")
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to check for .env.template: %w", err)
	}

	if _, err := os.Stat(envPath); err == nil {
		g.log.Warn(".env already exists — skipping to avoid overwrite")
		ux.Warning(".env already exists in %s — skipping generation.", projectDir)
		return nil
	}

	keys, defaults, err := g.parseTemplate(templatePath)
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		g.log.Info(".env.template is empty, skipping")
		return nil
	}

	if g.dryRun {
		g.log.Info(fmt.Sprintf("[dry-run] would prompt for %d .env key(s): %s",
			len(keys), strings.Join(keys, ", ")))
		return nil
	}

	values, err := g.promptAllKeys(keys, defaults)
	if err != nil {
		return fmt.Errorf(".env setup cancelled or failed: %w", err)
	}

	if err := g.writeEnvFile(envPath, keys, values); err != nil {
		return err
	}

	g.rb.Register("remove generated .env file", func() error {
		return os.Remove(envPath)
	})

	ux.Success(".env written to %s", envPath)
	g.log.Info(fmt.Sprintf(".env generated at %s", envPath))
	return nil
}

// promptAllKeys shows all keys in a single huh form so the user can
// fill them in together, consistent with the wizard's batch style.
// Falls back to line-by-line stdin prompts when the terminal is not
// interactive (e.g. pipes, CI).
func (g *Generator) promptAllKeys(keys []string, defaults map[string]string) (map[string]string, error) {
	valuePtrs := make([]string, len(keys))
	for i, k := range keys {
		valuePtrs[i] = defaults[k] // pre-fill with template default
	}

	fields := make([]huh.Field, len(keys))
	for i, key := range keys {
		i, key := i, key
		defVal := defaults[key]
		desc := ""
		if defVal != "" {
			desc = fmt.Sprintf("default: %s", defVal)
		}
		fields[i] = huh.NewInput().
			Title(key).
			Description(desc).
			Placeholder(defVal).
			Value(&valuePtrs[i])
	}

	err := huh.NewForm(huh.NewGroup(fields...)).
		WithTheme(huh.ThemeBase()).
		Run()

	if err != nil {
		// huh may fail in non-interactive environments; fall back to stdin.
		g.log.Warn(fmt.Sprintf("TUI unavailable (%v) — falling back to line prompts", err))
		return g.promptFallback(keys, defaults)
	}

	values := make(map[string]string, len(keys))
	for i, key := range keys {
		v := strings.TrimSpace(valuePtrs[i])
		if v == "" {
			v = defaults[key]
		}
		values[key] = v
	}
	return values, nil
}

// promptFallback reads key values line-by-line from stdin.
// Used as a safety net in non-TTY environments.
func (g *Generator) promptFallback(keys []string, defaults map[string]string) (map[string]string, error) {
	scanner := bufio.NewScanner(os.Stdin)
	values := make(map[string]string, len(keys))

	fmt.Println("\n  Configure .env values (press Enter to accept default):")
	for _, key := range keys {
		def := defaults[key]
		if def != "" {
			fmt.Printf("  %s [%s]: ", key, def)
		} else {
			fmt.Printf("  %s: ", key)
		}

		if scanner.Scan() {
			v := strings.TrimSpace(scanner.Text())
			if v == "" {
				v = def
			}
			values[key] = v
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("stdin read error: %w", err)
	}
	return values, nil
}

// parseTemplate reads KEY=DEFAULT pairs from a .env.template file.
func (g *Generator) parseTemplate(path string) ([]string, map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open .env.template: %w", err)
	}
	defer file.Close()

	var keys []string
	defaults := make(map[string]string)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		key := strings.TrimSpace(parts[0])
		if key == "" {
			continue
		}
		keys = append(keys, key)
		if len(parts) == 2 {
			defaults[key] = strings.TrimSpace(parts[1])
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("error reading .env.template: %w", err)
	}
	return keys, defaults, nil
}

// writeEnvFile writes KEY=VALUE lines to path, preserving key order.
func (g *Generator) writeEnvFile(path string, keys []string, values map[string]string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create .env file: %w", err)
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	fmt.Fprintln(w, "# Generated by DevForge — do not commit this file")
	for _, key := range keys {
		if _, err := fmt.Fprintf(w, "%s=%s\n", key, values[key]); err != nil {
			return fmt.Errorf("failed to write key %q to .env: %w", key, err)
		}
	}
	return w.Flush()
}
