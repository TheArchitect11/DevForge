package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/chinmay/devforge/internal/logger"
)

// Cache provides local filesystem caching for registry data.
type Cache struct {
	log *logger.Logger
}

// NewCache creates a Cache instance.
func NewCache(log *logger.Logger) *Cache {
	return &Cache{log: log}
}

// cacheDir returns the path to the DevForge cache directory.
func cacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine home directory: %w", err)
	}
	return filepath.Join(home, ".devforge", "cache"), nil
}

// cachePath returns the full path to the templates cache file.
func cachePath() (string, error) {
	dir, err := cacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "templates.json"), nil
}

// Save writes registry data to the local cache file.
func (c *Cache) Save(reg *Registry) error {
	path, err := cachePath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry for cache: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	c.log.Debug("registry cached", map[string]interface{}{
		"path": path,
	})
	return nil
}

// Load reads registry data from the local cache file.
func (c *Cache) Load() (*Registry, error) {
	path, err := cachePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("no cached registry available: %w", err)
	}

	var reg Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("failed to parse cached registry: %w", err)
	}

	c.log.Info("loaded registry from cache (offline mode)")
	return &reg, nil
}
