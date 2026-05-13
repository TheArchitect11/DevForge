// Package security provides input validation and sanitization for DevForge.
package security

import (
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

// validNamePattern accepts dependency / plugin names.
// Allows: alphanumeric, hyphens, underscores, dots, @, and / (for paths).
var validNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._@/-]*$`)

// validProjectNamePattern is stricter — safe for use as a directory name.
// Allows: alphanumeric, hyphens, underscores, dots. No slashes or @.
var validProjectNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// ValidateName checks a dependency or plugin name for unsafe characters.
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

// ValidateProjectName checks a project name for safe use as a directory.
// It is stricter than ValidateName: no slashes, no @ signs, max 100 chars,
// and the name may not consist solely of dots.
func ValidateProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("project name cannot be empty")
	}
	if len(name) > 100 {
		return fmt.Errorf("project name %q exceeds maximum length of 100 characters", name)
	}
	if !validProjectNamePattern.MatchString(name) {
		return fmt.Errorf(
			"project name %q contains invalid characters; use letters, digits, hyphens, underscores, or dots only",
			name,
		)
	}
	// Reject names that are purely dots (e.g. ".", "..").
	if strings.Trim(name, ".") == "" {
		return fmt.Errorf("project name %q is not a valid directory name", name)
	}
	return nil
}

// ValidateURL checks that a URL is well-formed and uses http or https.
func ValidateURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL %q: %w", rawURL, err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("URL %q uses unsupported scheme %q; only http and https are allowed", rawURL, parsed.Scheme)
	}

	if parsed.Host == "" {
		return fmt.Errorf("URL %q has no host", rawURL)
	}

	return nil
}

// ValidatePath checks a file or directory path for directory-traversal
// attempts, ensuring the resolved path stays within basePath.
func ValidatePath(basePath, targetPath string) error {
	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return fmt.Errorf("failed to resolve base path %q: %w", basePath, err)
	}

	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("failed to resolve target path %q: %w", targetPath, err)
	}

	if !strings.HasPrefix(absTarget, absBase+string(filepath.Separator)) && absTarget != absBase {
		return fmt.Errorf("path %q escapes base directory %q (directory traversal detected)", targetPath, basePath)
	}
	return nil
}

// SanitizeInput strips null bytes and trims surrounding whitespace.
func SanitizeInput(input string) string {
	return strings.TrimSpace(strings.ReplaceAll(input, "\x00", ""))
}
