// Package updater handles self-update for the DevForge binary by
// comparing the current version to the latest GitHub release,
// downloading the new binary, verifying its SHA-256 checksum against
// the release's checksums.txt, and atomically replacing the running
// executable with a backup-and-rollback strategy.
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

// GitHubRelease is a subset of the GitHub Releases API response.
type GitHubRelease struct {
	TagName string         `json:"tag_name"`
	Name    string         `json:"name"`
	Body    string         `json:"body"`
	Assets  []ReleaseAsset `json:"assets"`
}

// ReleaseAsset is a single file attached to a GitHub release.
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

// New creates an Updater for the given build version.
func New(currentVersion string, log *logger.Logger) *Updater {
	return &Updater{
		currentVersion: currentVersion,
		log:            log,
		httpClient:     &http.Client{Timeout: httpTimeout},
	}
}

// CheckResult holds everything needed to decide whether to update and
// to carry out the download.
type CheckResult struct {
	CurrentVersion  string
	LatestVersion   string
	UpdateAvailable bool
	Changelog       string
	AssetURL        string
	AssetName       string
	// ChecksumURL is the download URL for checksums.txt in the release.
	// Empty when the release has no checksums file (verification is skipped).
	ChecksumURL string
}

// Check queries the GitHub releases API and returns a populated CheckResult.
func (u *Updater) Check() (*CheckResult, error) {
	u.log.Info("checking for updates", map[string]interface{}{"current": u.currentVersion})

	resp, err := u.httpClient.Get(releasesURL)
	if err != nil {
		return nil, fmt.Errorf("failed to reach GitHub releases API: %w", err)
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
		return nil, fmt.Errorf("failed to parse release JSON: %w", err)
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(u.currentVersion, "v")

	result := &CheckResult{
		CurrentVersion:  current,
		LatestVersion:   latest,
		UpdateAvailable: latest != current && current != "dev",
		Changelog:       strings.TrimSpace(release.Body),
	}

	// Locate the platform binary and the checksums file.
	assetName := expectedAssetName()
	for _, asset := range release.Assets {
		switch asset.Name {
		case assetName:
			result.AssetURL = asset.BrowserDownloadURL
			result.AssetName = asset.Name
		case "checksums.txt":
			result.ChecksumURL = asset.BrowserDownloadURL
		}
	}

	return result, nil
}

// Update downloads the new binary, verifies its SHA-256 checksum, and
// atomically replaces the running executable. A .backup copy is kept
// until the operation succeeds, then removed. On failure the backup is
// restored automatically.
func (u *Updater) Update(result *CheckResult) error {
	if result.AssetURL == "" {
		return fmt.Errorf("no pre-built binary available for %s/%s — update manually", runtime.GOOS, runtime.GOARCH)
	}

	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine current executable path: %w", err)
	}
	currentExe, err = filepath.EvalSymlinks(currentExe)
	if err != nil {
		return fmt.Errorf("cannot resolve executable symlinks: %w", err)
	}

	// Download into a temp file in the same directory so os.Rename is
	// atomic (same filesystem, no cross-device move).
	tmpFile, err := os.CreateTemp(filepath.Dir(currentExe), ".devforge-update-*")
	if err != nil {
		return fmt.Errorf("cannot create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath) // cleaned up whether we succeed or fail

	u.log.Info(fmt.Sprintf("downloading %s", result.AssetName))

	resp, err := u.httpClient.Get(result.AssetURL)
	if err != nil {
		tmpFile.Close()
		return fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		tmpFile.Close()
		return fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}

	// Simultaneously write and hash the response body.
	hasher := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmpFile, hasher), resp.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write downloaded binary: %w", err)
	}
	tmpFile.Close()

	localHash := hex.EncodeToString(hasher.Sum(nil))
	u.log.Info(fmt.Sprintf("download complete — sha256: %s", localHash))

	// Verify checksum against the release's checksums.txt.
	if err := u.verifyChecksum(result.AssetName, tmpPath, localHash, result.ChecksumURL); err != nil {
		return err
	}

	if err := os.Chmod(tmpPath, 0o755); err != nil {
		return fmt.Errorf("cannot mark new binary executable: %w", err)
	}

	// Backup the current binary before overwriting.
	backupPath := currentExe + ".backup"
	if err := os.Rename(currentExe, backupPath); err != nil {
		return fmt.Errorf("cannot back up current binary: %w", err)
	}

	if err := os.Rename(tmpPath, currentExe); err != nil {
		// Restore backup so the CLI is still functional.
		if rbErr := os.Rename(backupPath, currentExe); rbErr != nil {
			u.log.Error(fmt.Sprintf("CRITICAL: failed to restore backup after failed update: %v", rbErr))
		}
		return fmt.Errorf("cannot install new binary: %w", err)
	}

	os.Remove(backupPath)
	u.log.Info(fmt.Sprintf("updated to v%s", result.LatestVersion))
	return nil
}

// verifyChecksum fetches checksums.txt from the release and confirms
// that localHash matches the expected hash for assetName.
// A missing checksums file or missing entry is treated as a warning,
// not a fatal error (older releases may not have this file).
func (u *Updater) verifyChecksum(assetName, tmpPath, localHash, checksumURL string) error {
	if checksumURL == "" {
		u.log.Warn("no checksums.txt in release — skipping integrity verification")
		return nil
	}

	resp, err := u.httpClient.Get(checksumURL)
	if err != nil {
		u.log.Warn(fmt.Sprintf("could not fetch checksums.txt: %v — skipping", err))
		return nil // non-fatal; older or partial releases
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1*1024*1024))
	if err != nil {
		u.log.Warn(fmt.Sprintf("could not read checksums.txt: %v — skipping", err))
		return nil
	}

	// checksums.txt line format: "<sha256>  <filename>"
	expectedHash := ""
	for _, line := range strings.Split(string(body), "\n") {
		parts := strings.Fields(line)
		if len(parts) >= 2 && strings.EqualFold(parts[1], assetName) {
			expectedHash = strings.ToLower(parts[0])
			break
		}
	}

	if expectedHash == "" {
		u.log.Warn(fmt.Sprintf("no entry for %q in checksums.txt — skipping", assetName))
		return nil
	}

	if localHash != expectedHash {
		// Re-read tmpPath to produce a fresh hash in the error message
		// in case memory corruption is suspected.
		return fmt.Errorf(
			"SHA-256 mismatch for %s:\n  expected: %s\n  got:      %s\n  The download may be corrupt or tampered with.",
			assetName, expectedHash, localHash,
		)
	}

	u.log.Info(fmt.Sprintf("checksum verified for %s", assetName))
	return nil
}

// expectedAssetName returns the binary filename for the current platform,
// matching the filenames produced by the release workflow.
func expectedAssetName() string {
	goos := runtime.GOOS
	arch := runtime.GOARCH
	ext := ""
	if goos == "windows" {
		ext = ".exe"
	}
	return fmt.Sprintf("devforge-%s-%s%s", goos, arch, ext)
}
