package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// toolCheck holds the result of checking a single tool.
type toolCheck struct {
	Name      string
	Installed bool
	Version   string
	Error     string
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system readiness and dependency health",
	Long: `Run a series of checks against your development environment to
verify that all required tools are installed and functional.

Checks: Homebrew, Node.js, Git, Docker.`,
	RunE: runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(_ *cobra.Command, _ []string) error {
	fmt.Println()
	fmt.Println("  DevForge Doctor — System Readiness Report")
	fmt.Println("  ══════════════════════════════════════════")
	fmt.Printf("  DevForge version: %s\n", Version)
	fmt.Println()

	checks := []struct {
		name       string
		binary     string
		versionArg string
	}{
		{"Homebrew", "brew", "--version"},
		{"Node.js", "node", "--version"},
		{"Git", "git", "--version"},
		{"Docker", "docker", "--version"},
	}

	results := make([]toolCheck, 0, len(checks))
	allOK := true

	for _, c := range checks {
		tc := checkTool(c.name, c.binary, c.versionArg)
		results = append(results, tc)
		if !tc.Installed {
			allOK = false
		}
	}

	// Print results as a formatted table.
	fmt.Printf("  %-12s %-12s %s\n", "Tool", "Status", "Version")
	fmt.Printf("  %-12s %-12s %s\n", "────", "──────", "───────")
	for _, r := range results {
		status := "✓ installed"
		version := r.Version
		if !r.Installed {
			status = "✗ missing"
			version = r.Error
		}
		fmt.Printf("  %-12s %-12s %s\n", r.Name, status, version)
	}

	fmt.Println()
	if allOK {
		fmt.Println("  ✅ All checks passed — system is ready!")
	} else {
		fmt.Println("  ⚠️  Some tools are missing. Install them before running 'devforge init'.")
	}
	fmt.Println()

	return nil
}

// checkTool looks up a binary and tries to retrieve its version string.
func checkTool(name, binary, versionArg string) toolCheck {
	tc := toolCheck{Name: name}

	path, err := exec.LookPath(binary)
	if err != nil {
		tc.Installed = false
		tc.Error = "not found in PATH"
		return tc
	}

	out, err := exec.Command(path, versionArg).Output()
	if err != nil {
		tc.Installed = true
		tc.Version = "(installed, version unknown)"
		return tc
	}

	version := strings.TrimSpace(strings.Split(string(out), "\n")[0])
	tc.Installed = true
	tc.Version = version
	return tc
}
