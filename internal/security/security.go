// Package security provides input validation and sanitization utilities
// for DevForge to prevent injection attacks, directory traversal, and
// other security issues.
package security

import (
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

// validNamePattern matches safe dependency/plugin names: alphanumeric,
// hyphens, underscores, dots, and @ for version specifiers.
var validNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._@/-]*$`)

// ValidateName checks that a name string (dependency, plugin, project)
// contains only safe characters.
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if len(name) > 255 {
		return fmt.Errorf("name exceeds maximum length of 255 characters")
	}
	if !validNamePattern.MatchString(name) {
		return fmt.Errorf("name %q contains invalid characters; only alphanumeric, hyphens, underscores, dots, /, and @ are allowed", name)
	}
	return nil
}

// ValidateURL checks that a URL string is well-formed and uses a safe
// scheme (http or https).
func ValidateURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL %q: %w", rawURL, err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("URL %q has unsupported scheme %q; only http and https are allowed", rawURL, parsed.Scheme)
	}

	if parsed.Host == "" {
		return fmt.Errorf("URL %q has no host", rawURL)
	}

	return nil
}

// ValidatePath checks a file or directory path for directory traversal
// attempts. It ensures the resolved path stays within the expected base
// directory.
func ValidatePath(basePath, targetPath string) error {
	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return fmt.Errorf("failed to resolve base path %q: %w", basePath, err)
	}

	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("failed to resolve target path %q: %w", targetPath, err)
	}

	// Ensure the target is within the base directory.
	if !strings.HasPrefix(absTarget, absBase+string(filepath.Separator)) && absTarget != absBase {
		return fmt.Errorf("path %q escapes base directory %q (directory traversal detected)", targetPath, basePath)
	}

	return nil
}

// SanitizeInput strips null bytes and trims whitespace from user input.
func SanitizeInput(input string) string {
	// Remove null bytes.
	cleaned := strings.ReplaceAll(input, "\x00", "")
	// Trim leading/trailing whitespace.
	return strings.TrimSpace(cleaned)
}
