package executor

import (
	"strings"
	"testing"

	"github.com/chinmay/devforge/internal/logger"
)

func newTestLogger(t *testing.T) *logger.Logger {
	t.Helper()
	log, err := logger.New(false, false)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	t.Cleanup(func() { log.Close() })
	return log
}

func TestSanitize_Safe(t *testing.T) {
	safeArgs := []string{
		"install",
		"node",
		"node@20",
		"my-package",
		"v1.2.3",
		"--version",
		"20.0.0",
	}

	for _, arg := range safeArgs {
		if err := sanitize(arg); err != nil {
			t.Errorf("sanitize(%q) returned unexpected error: %v", arg, err)
		}
	}
}

func TestSanitize_Dangerous(t *testing.T) {
	dangerousArgs := []string{
		"node; rm -rf /",
		"pkg | cat /etc/passwd",
		"cmd `whoami`",
		"$(evil)",
		"arg > /tmp/out",
		"arg < /etc/passwd",
		"&& true",
		"{}",
	}

	for _, arg := range dangerousArgs {
		if err := sanitize(arg); err == nil {
			t.Errorf("sanitize(%q) expected error, got nil", arg)
		}
	}
}

func TestExecutor_DryRun(t *testing.T) {
	log := newTestLogger(t)
	exec := New(log, true) // dry-run mode

	result, err := exec.Run("echo", "hello")
	if err != nil {
		t.Fatalf("Run() unexpected error in dry-run: %v", err)
	}
	if !result.DryRun {
		t.Error("expected DryRun=true in result")
	}
	if result.Stdout != "" {
		t.Errorf("expected empty stdout in dry-run, got %q", result.Stdout)
	}
}

func TestExecutor_RealRun(t *testing.T) {
	log := newTestLogger(t)
	exec := New(log, false)

	result, err := exec.Run("echo", "devforge")
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}
	if result.DryRun {
		t.Error("expected DryRun=false in result")
	}
	if !strings.Contains(result.Stdout, "devforge") {
		t.Errorf("expected stdout to contain %q, got %q", "devforge", result.Stdout)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
}

func TestExecutor_CommandFailure(t *testing.T) {
	log := newTestLogger(t)
	exec := New(log, false)

	_, err := exec.Run("false") // `false` always exits with code 1
	if err == nil {
		t.Error("expected error from command with exit code 1, got nil")
	}
}

func TestExecutor_SanitizationGuard(t *testing.T) {
	log := newTestLogger(t)
	exec := New(log, false)

	_, err := exec.Run("echo", "hello; rm -rf /")
	if err == nil {
		t.Error("expected sanitization error, got nil")
	}
}
