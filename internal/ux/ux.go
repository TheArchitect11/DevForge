// Package ux provides terminal styling, spinner, and UX utilities for
// the DevForge CLI.
//
// Color output is automatically disabled when:
//   - The NO_COLOR environment variable is set (https://no-color.org/)
//   - TERM=dumb
//   - stdout is not a TTY (pipes, CI, file redirects)
//   - SetColorEnabled(false) is called (e.g. --no-color flag)
//
// All exported helpers respect this setting. Use the CodeXxx() functions
// when you need raw ANSI codes in format strings; they return "" when
// color is disabled.
package ux

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// ── Color detection ────────────────────────────────────────────────────

var (
	colorOnce    sync.Once
	colorEnabled bool
)

func detectColor() {
	colorOnce.Do(func() {
		if os.Getenv("NO_COLOR") != "" {
			colorEnabled = false
			return
		}
		if strings.ToLower(os.Getenv("TERM")) == "dumb" {
			colorEnabled = false
			return
		}
		fi, err := os.Stdout.Stat()
		colorEnabled = err == nil && fi.Mode()&os.ModeCharDevice != 0
	})
}

// SetColorEnabled overrides automatic detection. Must be called before
// any output (e.g. from the --no-color flag handler).
func SetColorEnabled(v bool) {
	colorOnce.Do(func() {}) // mark as done so the auto-detect doesn't override
	colorEnabled = v
}

func colored() bool {
	detectColor()
	return colorEnabled
}

// ── Raw ANSI codes (unexported) ────────────────────────────────────────

const (
	resetCode  = "\033[0m"
	redCode    = "\033[31m"
	greenCode  = "\033[32m"
	yellowCode = "\033[33m"
	blueCode   = "\033[34m"
	cyanCode   = "\033[36m"
	grayCode   = "\033[90m"
	boldCode   = "\033[1m"
)

// ── Exported color-aware code getters ─────────────────────────────────
// These return the ANSI escape code when color is enabled, or "" otherwise.
// Use them in format strings when you need inline color control:
//
//	fmt.Printf("%stext%s\n", ux.CodeGreen(), ux.CodeReset())

func CodeReset() string  { if colored() { return resetCode }; return "" }
func CodeRed() string    { if colored() { return redCode }; return "" }
func CodeGreen() string  { if colored() { return greenCode }; return "" }
func CodeYellow() string { if colored() { return yellowCode }; return "" }
func CodeBlue() string   { if colored() { return blueCode }; return "" }
func CodeCyan() string   { if colored() { return cyanCode }; return "" }
func CodeGray() string   { if colored() { return grayCode }; return "" }
func CodeBold() string   { if colored() { return boldCode }; return "" }

// col is the package-internal helper that wraps s with an ANSI code.
func col(code, s string) string {
	if !colored() {
		return s
	}
	return code + s + resetCode
}

// ── Symbol constants ───────────────────────────────────────────────────

const (
	Check = "✔"
	Cross = "✖"
	Info  = "ℹ"
	Warn  = "⚠"
	Spin  = "⟳"
	Arrow = "→"
	Dot   = "•"
)

// ── Output helpers ─────────────────────────────────────────────────────

// Success prints a green ✔ line.
func Success(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	if colored() {
		fmt.Printf("%s%s%s %s\n", greenCode, Check, resetCode, msg)
	} else {
		fmt.Printf("[ok] %s\n", msg)
	}
}

// Error prints a red ✖ line.
func Error(err error) {
	if colored() {
		fmt.Printf("%s%s%s Error: %s\n", redCode, Cross, resetCode, err.Error())
	} else {
		fmt.Printf("[error] %s\n", err.Error())
	}
}

// Warning prints a yellow ⚠ line.
func Warning(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	if colored() {
		fmt.Printf("%s%s%s %s\n", yellowCode, Warn, resetCode, msg)
	} else {
		fmt.Printf("[warn] %s\n", msg)
	}
}

// InfoMsg prints a blue ℹ line.
func InfoMsg(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	if colored() {
		fmt.Printf("%s%s%s %s\n", blueCode, Info, resetCode, msg)
	} else {
		fmt.Printf("[info] %s\n", msg)
	}
}

// Step prints a ⟳ progress line (action in progress).
func Step(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	if colored() {
		fmt.Printf("%s%s%s %s...\n", cyanCode, Spin, resetCode, msg)
	} else {
		fmt.Printf("[ ] %s...\n", msg)
	}
}

// Header prints a prominent section title with a rule beneath it.
func Header(title string) {
	w := len(title)
	if w < 40 {
		w = 40
	}
	bar := strings.Repeat("─", w)
	if colored() {
		fmt.Printf("\n%s%s%s\n%s%s%s\n", boldCode+blueCode, title, resetCode, grayCode, bar, resetCode)
	} else {
		fmt.Printf("\n%s\n%s\n", title, bar)
	}
}

// Banner prints the DevForge logo banner.
func Banner(subtitle string) {
	if colored() {
		fmt.Printf("\n  %s%s⚒  DevForge%s\n", boldCode, blueCode, resetCode)
		if subtitle != "" {
			fmt.Printf("  %s%s%s\n", grayCode, subtitle, resetCode)
		}
		fmt.Println()
	} else {
		fmt.Printf("\n  ⚒  DevForge\n")
		if subtitle != "" {
			fmt.Printf("  %s\n", subtitle)
		}
		fmt.Println()
	}
}

// Divider prints a horizontal rule.
func Divider() {
	if colored() {
		fmt.Printf("%s%s%s\n", grayCode, strings.Repeat("─", 50), resetCode)
	} else {
		fmt.Println(strings.Repeat("-", 50))
	}
}

// Printf and Println are pass-throughs for convenience.
func Printf(format string, a ...interface{}) { fmt.Printf(format, a...) }
func Println(a ...interface{})               { fmt.Println(a...) }

// ExitWithMessage prints to stderr and calls os.Exit.
func ExitWithMessage(code int, format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	if colored() {
		fmt.Fprintf(os.Stderr, "%s%s%s %s\n", redCode, Cross, resetCode, msg)
	} else {
		fmt.Fprintf(os.Stderr, "[error] %s\n", msg)
	}
	os.Exit(code)
}

// ── Spinner ────────────────────────────────────────────────────────────

var spinFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner shows an animated braille spinner for long sequential operations.
// In non-TTY mode it prints a plain "[ ] msg..." line instead.
type Spinner struct {
	msg    string
	active bool
	stop   chan struct{}
	done   chan struct{}
}

// NewSpinner creates and immediately starts a Spinner.
func NewSpinner(msg string) *Spinner {
	s := &Spinner{
		msg:  msg,
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}
	if colored() {
		s.active = true
		go s.run()
	} else {
		fmt.Printf("[..] %s...\n", msg)
	}
	return s
}

func (s *Spinner) run() {
	defer close(s.done)
	i := 0
	for {
		select {
		case <-s.stop:
			fmt.Printf("\r\033[K")
			return
		case <-time.After(80 * time.Millisecond):
			fmt.Printf("\r%s%s%s %s", cyanCode, spinFrames[i%len(spinFrames)], resetCode, s.msg)
			i++
		}
	}
}

// Stop halts the spinner. If successMsg is non-empty a success line is printed.
func (s *Spinner) Stop(successMsg string) {
	if s.active {
		close(s.stop)
		<-s.done
	}
	if successMsg != "" {
		Success(successMsg)
	}
}

// Fail halts the spinner and prints a failure line.
func (s *Spinner) Fail(msg string) {
	if s.active {
		close(s.stop)
		<-s.done
	}
	if msg != "" {
		if colored() {
			fmt.Printf("%s%s%s %s\n", redCode, Cross, resetCode, msg)
		} else {
			fmt.Printf("[fail] %s\n", msg)
		}
	}
}

// ── Install result table ───────────────────────────────────────────────

// InstallResult holds the outcome of one dependency install.
type InstallResult struct {
	Name    string
	Version string // installed version (may be empty)
	Wanted  string // version from config
	Skipped bool   // was already installed
	Err     error
}

// PrintInstallResults prints a formatted summary of install outcomes.
func PrintInstallResults(results []InstallResult) {
	for _, r := range results {
		if r.Err != nil {
			if colored() {
				fmt.Printf("  %s%s%s %-14s %s%s%s\n",
					redCode, Cross, resetCode,
					r.Name,
					redCode, r.Err.Error(), resetCode)
			} else {
				fmt.Printf("  [fail] %-14s %s\n", r.Name, r.Err.Error())
			}
			continue
		}

		label := r.Name
		if r.Version != "" {
			label = fmt.Sprintf("%s (%s)", r.Name, r.Version)
		}

		tag := "installed"
		if r.Skipped {
			tag = "already installed"
		}

		if colored() {
			tagColored := col(greenCode, tag)
			if r.Skipped {
				tagColored = col(grayCode, tag)
			}
			fmt.Printf("  %s%s%s %-28s %s\n",
				greenCode, Check, resetCode, label, tagColored)
		} else {
			fmt.Printf("  [ok]   %-28s %s\n", label, tag)
		}

		// Warn on major version mismatch.
		if r.Wanted != "" && r.Wanted != "latest" && r.Version != "" {
			if !strings.HasPrefix(r.Version, r.Wanted) {
				Warning("version mismatch for %s: installed=%s, wanted=%s",
					r.Name, r.Version, r.Wanted)
			}
		}
	}
}
