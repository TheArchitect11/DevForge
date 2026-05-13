// Package template handles cloning starter template repositories and
// resetting their git history so the new project starts clean.
package template

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/chinmay/devforge/internal/logger"
	"github.com/chinmay/devforge/internal/rollback"
	"github.com/chinmay/devforge/internal/ux"
)

// Cloner handles cloning Git repositories into local directories.
type Cloner struct {
	log    *logger.Logger
	rb     *rollback.Manager
	dryRun bool
}

// NewCloner creates a Cloner with the given logger and rollback manager.
func NewCloner(log *logger.Logger, rb *rollback.Manager, dryRun bool) *Cloner {
	return &Cloner{log: log, rb: rb, dryRun: dryRun}
}

// Clone clones repoURL into destDir, strips the template's git history,
// and re-initialises the directory as a brand-new repository with a
// single "Initial commit" so the user starts with a clean slate.
//
// On dry-run the operation is logged but nothing touches the filesystem.
// A rollback action is registered so that a later step failure removes
// the cloned directory automatically.
func (c *Cloner) Clone(repoURL, destDir string) error {
	// Guard: destination must not already exist.
	if _, err := os.Stat(destDir); err == nil {
		return fmt.Errorf("destination directory %q already exists", destDir)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check destination directory %q: %w", destDir, err)
	}

	if c.dryRun {
		c.log.Info(fmt.Sprintf("[dry-run] would clone %s → %s", repoURL, destDir))
		return nil
	}

	spin := ux.NewSpinner(fmt.Sprintf("Cloning template from %s", repoURL))

	_, err := git.PlainClone(destDir, false, &git.CloneOptions{
		URL:      repoURL,
		Progress: io.Discard, // suppress raw git output; we show our own spinner
		Depth:    1,          // shallow clone — faster, no full history needed
	})
	if err != nil {
		spin.Fail(fmt.Sprintf("clone failed: %v", err))
		return fmt.Errorf("failed to clone repository %q: %w", repoURL, err)
	}

	spin.Stop("Template cloned")

	// Register rollback before doing any further work so a later
	// failure will clean up the directory.
	c.rb.Register(fmt.Sprintf("remove cloned directory %s", destDir), func() error {
		return os.RemoveAll(destDir)
	})

	// ── Strip template history and reinitialise ───────────────────
	if err := c.freshGitInit(destDir); err != nil {
		// Non-fatal: log and continue — the files are all there.
		c.log.Warn(fmt.Sprintf("could not reinitialise git repository: %v", err))
		ux.Warning("git reinitialisation skipped: %v", err)
	}

	c.log.Info(fmt.Sprintf("template ready at %s", destDir))
	return nil
}

// freshGitInit removes the cloned .git directory and creates a new one
// with a single initial commit containing all template files.
func (c *Cloner) freshGitInit(dir string) error {
	gitDir := filepath.Join(dir, ".git")
	if err := os.RemoveAll(gitDir); err != nil {
		return fmt.Errorf("remove template .git: %w", err)
	}

	repo, err := git.PlainInit(dir, false)
	if err != nil {
		return fmt.Errorf("git init: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("open worktree: %w", err)
	}

	// Stage all template files.
	if _, err := wt.Add("."); err != nil {
		return fmt.Errorf("git add .: %w", err)
	}

	_, err = wt.Commit("Initial commit (scaffolded by DevForge)", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "DevForge",
			Email: "devforge@localhost",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("initial commit: %w", err)
	}

	ux.Success("Git repository initialised with clean history")
	return nil
}
