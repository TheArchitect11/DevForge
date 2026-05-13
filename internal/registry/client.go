package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/chinmay/devforge/internal/logger"
)

const defaultTimeout = 10 * time.Second

// Client fetches template registry data from the remote endpoint.
type Client struct {
	registryURL string
	httpClient  *http.Client
	cache       *Cache
	log         *logger.Logger
}

// NewClient creates a registry client with the given URL and logger.
func NewClient(registryURL string, log *logger.Logger) *Client {
	return &Client{
		registryURL: registryURL,
		httpClient:  &http.Client{Timeout: defaultTimeout},
		cache:       NewCache(log),
		log:         log,
	}
}

// Fetch retrieves the template registry. It checks the local cache first
// unless forceRefresh is true or the cache is expired/missing.
func (c *Client) Fetch(forceRefresh bool) (*Registry, error) {
	if !forceRefresh {
		reg, err := c.cache.Load()
		if err == nil {
			return reg, nil
		}
		c.log.Debug(fmt.Sprintf("cache miss or expired: %v", err))
	}

	c.log.Debug("fetching template registry", map[string]interface{}{"url": c.registryURL})

	reg, err := c.fetchRemote()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch remote registry: %w", err)
	}

	if cacheErr := c.cache.Save(reg); cacheErr != nil {
		c.log.Warn(fmt.Sprintf("failed to cache registry: %v", cacheErr))
	}

	return reg, nil
}

// fetchRemote performs the HTTPS GET request and parses the response.
func (c *Client) fetchRemote() (*Registry, error) {
	resp, err := c.httpClient.Get(c.registryURL)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var reg Registry
	if err := json.Unmarshal(body, &reg); err != nil {
		return nil, fmt.Errorf("failed to parse registry JSON: %w", err)
	}

	return &reg, nil
}

// Search filters templates by keyword, matching against name, description,
// and tags (case-insensitive).
func (c *Client) Search(reg *Registry, keyword string) []Template {
	if keyword == "" {
		return reg.ValidTemplates()
	}

	kw := strings.ToLower(keyword)
	var results []Template
	for _, t := range reg.ValidTemplates() {
		if strings.Contains(strings.ToLower(t.Name), kw) ||
			strings.Contains(strings.ToLower(t.Description), kw) ||
			tagsContain(t.Tags, kw) {
			results = append(results, t)
		}
	}
	return results
}

func tagsContain(tags []string, kwLower string) bool {
	for _, tag := range tags {
		if strings.Contains(strings.ToLower(tag), kwLower) {
			return true
		}
	}
	return false
}

// ClearCache removes the local cache file. The next Fetch() will pull
// fresh data from the remote registry.
func (c *Client) ClearCache() error {
	return c.cache.Clear()
}
