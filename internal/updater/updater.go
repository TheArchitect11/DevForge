// Package updater provides auto-update functionality for the DevForge
// binary by checking GitHub releases, downloading new versions, and
// safely replacing the current executable.
package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/chinmay/devforge/internal/logger"
)

const (
	releasesURL = "https://api.github.com/repos/ChinmayyK/DevForge/releases/latest"
	httpTimeout = 15 * time.Second
)

// GitHubRelease represents a subset of the GitHub release API response.
type GitHubRelease struct {
	TagName string         `json:"tag_name"`
	Name    string         `json:"name"`
	Body    string         `json:"body"`
	Assets  []ReleaseAsset `json:"assets"`
}

// ReleaseAsset represents a downloadable file attached to a release.
type ReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// Updater handles version checking and binary replacement.
type Updater struct {
	currentVersion string
	log            *logger.Logger
	httpClient     *http.Client
}

// New creates an Updater with the current build version.
func New(currentVersion string, log *logger.Logger) *Updater {
	return &Updater{
		currentVersion: currentVersion,
		log:            log,
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
	}
}

// CheckResult holds the outcome of a version check.
type CheckResult struct {
	CurrentVersion  string
	LatestVersion   string
	UpdateAvailable bool
	Changelog       string
	AssetURL        string
	AssetName       string
}

// Check queries the GitHub releases API to determine if a newer
// version is available.
func (u *Updater) Check() (*CheckResult, error) {
	u.log.Info("checking for updates...")

	resp, err := u.httpClient.Get(releasesURL)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("failed to read release data: %w", err)
	}

	var release GitHubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("failed to parse release data: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	currentClean := strings.TrimPrefix(u.currentVersion, "v")

	result := &CheckResult{
		CurrentVersion:  currentClean,
		LatestVersion:   latestVersion,
		UpdateAvailable: latestVersion != currentClean && currentClean != "dev",
		Changelog:       release.Body,
	}

	// Find the right binary for this OS/arch.
	assetName := expectedAssetName()
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			result.AssetURL = asset.BrowserDownloadURL
			result.AssetName = asset.Name
			break
		}
	}

	return result, nil
}

// Update downloads and installs the new binary. It backs up the
// current binary and performs a safe replacement.
func (u *Updater) Update(result *CheckResult) error {
	if result.AssetURL == "" {
		return fmt.Errorf("no binary available for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to determine current executable path: %w", err)
	}
	currentExe, err = filepath.EvalSymlinks(currentExe)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	u.log.Info(fmt.Sprintf("downloading %s...", result.AssetName))

	// Download to a temporary file.
	tmpFile, err := os.CreateTemp(filepath.Dir(currentExe), "devforge-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath) // Clean up on any failure path.

	resp, err := u.httpClient.Get(result.AssetURL)
	if err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		tmpFile.Close()
		return fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}

	hasher := sha256.New()
	writer := io.MultiWriter(tmpFile, hasher)

	if _, err := io.Copy(writer, resp.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write update: %w", err)
	}
	tmpFile.Close()

	checksum := hex.EncodeToString(hasher.Sum(nil))
	u.log.Info(fmt.Sprintf("download complete (sha256: %s)", checksum))

	// Make executable.
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		return fmt.Errorf("failed to set executable permissions: %w", err)
	}

	// Backup current binary.
	backupPath := currentExe + ".backup"
	if err := os.Rename(currentExe, backupPath); err != nil {
		return fmt.Errorf("failed to backup current binary: %w", err)
	}

	// Move new binary into place.
	if err := os.Rename(tmpPath, currentExe); err != nil {
		// Rollback: restore backup.
		if rbErr := os.Rename(backupPath, currentExe); rbErr != nil {
			u.log.Error(fmt.Sprintf("CRITICAL: failed to rollback update: %v", rbErr))
		}
		return fmt.Errorf("failed to install update: %w", err)
	}

	// Remove backup on success.
	os.Remove(backupPath)

	u.log.Info(fmt.Sprintf("updated from v%s to v%s", result.CurrentVersion, result.LatestVersion))
	return nil
}

// expectedAssetName returns the expected binary name for this platform.
func expectedAssetName() string {
	os := runtime.GOOS
	arch := runtime.GOARCH
	ext := ""
	if os == "windows" {
		ext = ".exe"
	}
	return fmt.Sprintf("devforge-%s-%s%s", os, arch, ext)
}
