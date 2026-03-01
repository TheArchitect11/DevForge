package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		cache: NewCache(log),
		log:   log,
	}
}

// Fetch retrieves the template registry. It first tries the remote
// endpoint and falls back to the local cache if the network request
// fails.
func (c *Client) Fetch() (*Registry, error) {
	c.log.Debug("fetching template registry", map[string]interface{}{
		"url": c.registryURL,
	})

	reg, err := c.fetchRemote()
	if err != nil {
		c.log.Warn(fmt.Sprintf("failed to fetch remote registry: %v; trying cache", err))
		return c.cache.Load()
	}

	// Cache the successful response for offline use.
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

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // 10MB limit
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var reg Registry
	if err := json.Unmarshal(body, &reg); err != nil {
		return nil, fmt.Errorf("failed to parse registry JSON: %w", err)
	}

	return &reg, nil
}

// Search filters templates by keyword, matching against name,
// description, and tags.
func (c *Client) Search(reg *Registry, keyword string) []Template {
	if keyword == "" {
		return reg.ValidTemplates()
	}

	var results []Template
	for _, t := range reg.ValidTemplates() {
		if containsIgnoreCase(t.Name, keyword) ||
			containsIgnoreCase(t.Description, keyword) ||
			tagsContain(t.Tags, keyword) {
			results = append(results, t)
		}
	}
	return results
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(substr) == 0 ||
			findIgnoreCase(s, substr))
}

func findIgnoreCase(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalFoldSlice(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

func equalFoldSlice(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

func tagsContain(tags []string, keyword string) bool {
	for _, tag := range tags {
		if containsIgnoreCase(tag, keyword) {
			return true
		}
	}
	return false
}
