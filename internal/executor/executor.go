// Package executor provides a safe wrapper around os/exec for running
// shell commands. It supports dry-run mode, structured results, input
// sanitization, and logging.
package executor

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/chinmay/devforge/internal/logger"
)

// Result holds the outcome of a command execution.
type Result struct {
	// Command is the full command string that was executed.
	Command string
	// Stdout contains the standard output captured from the command.
	Stdout string
	// Stderr contains the standard error captured from the command.
	Stderr string
	// ExitCode is the exit code returned by the command.
	ExitCode int
	// DryRun indicates the command was not actually executed.
	DryRun bool
}

// Executor runs shell commands with logging, dry-run support, and
// input sanitization.
type Executor struct {
	log    *logger.Logger
	dryRun bool
}

// New creates an Executor. When dryRun is true, commands are logged
// but not actually executed.
func New(log *logger.Logger, dryRun bool) *Executor {
	return &Executor{
		log:    log,
		dryRun: dryRun,
	}
}

// dangerousPattern matches characters commonly used in shell injection.
var dangerousPattern = regexp.MustCompile(`[;&|` + "`" + `$(){}\\<>]`)

// sanitize validates that a command argument does not contain shell
// metacharacters that could lead to command injection.
func sanitize(arg string) error {
	if dangerousPattern.MatchString(arg) {
		return fmt.Errorf("potentially unsafe characters detected in argument: %q", arg)
	}
	return nil
}

// Run executes the given command with arguments. Each element in args
// is sanitized before execution. In dry-run mode the command is logged
// but not run, and a synthetic Result is returned.
func (e *Executor) Run(name string, args ...string) (*Result, error) {
	// Sanitize every argument.
	for _, arg := range args {
		if err := sanitize(arg); err != nil {
			return nil, fmt.Errorf("command sanitization failed: %w", err)
		}
	}

	cmdStr := name + " " + strings.Join(args, " ")
	e.log.Debug("executing command", map[string]interface{}{
		"command": cmdStr,
		"dryRun":  e.dryRun,
	})

	if e.dryRun {
		e.log.Info(fmt.Sprintf("[dry-run] would execute: %s", cmdStr))
		return &Result{
			Command: cmdStr,
			DryRun:  true,
		}, nil
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.Command(name, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := &Result{
		Command:  cmdStr,
		Stdout:   strings.TrimSpace(stdout.String()),
		Stderr:   strings.TrimSpace(stderr.String()),
		ExitCode: 0,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			return result, fmt.Errorf("failed to execute %q: %w", cmdStr, err)
		}
		e.log.Warn(fmt.Sprintf("command exited with code %d: %s", result.ExitCode, cmdStr), map[string]interface{}{
			"stderr": result.Stderr,
		})
		return result, fmt.Errorf("command %q exited with code %d: %s", cmdStr, result.ExitCode, result.Stderr)
	}

	e.log.Debug("command completed successfully", map[string]interface{}{
		"command": cmdStr,
		"output":  result.Stdout,
	})

	return result, nil
}

// RunIn is like Run but executes the command with the given working directory.
// Use this for commands that must run inside a specific folder (e.g. npm install
// inside the newly scaffolded project).
func (e *Executor) RunIn(dir, name string, args ...string) (*Result, error) {
	for _, arg := range args {
		if err := sanitize(arg); err != nil {
			return nil, fmt.Errorf("command sanitization failed: %w", err)
		}
	}

	cmdStr := name + " " + strings.Join(args, " ")
	e.log.Debug("executing command", map[string]interface{}{
		"command": cmdStr,
		"dir":     dir,
		"dryRun":  e.dryRun,
	})

	if e.dryRun {
		e.log.Info(fmt.Sprintf("[dry-run] would execute in %s: %s", dir, cmdStr))
		return &Result{Command: cmdStr, DryRun: true}, nil
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := &Result{
		Command:  cmdStr,
		Stdout:   strings.TrimSpace(stdout.String()),
		Stderr:   strings.TrimSpace(stderr.String()),
		ExitCode: 0,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			return result, fmt.Errorf("failed to execute %q in %s: %w", cmdStr, dir, err)
		}
		return result, fmt.Errorf("command %q exited with code %d: %s", cmdStr, result.ExitCode, result.Stderr)
	}

	return result, nil
}
