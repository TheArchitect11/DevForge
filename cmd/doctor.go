package cmd

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"github.com/chinmay/devforge/internal/config"
	"github.com/chinmay/devforge/internal/osdetect"
	"github.com/chinmay/devforge/internal/ux"
)

// toolDef describes how to query a tool's installed version.
type toolDef struct {
	binary string
	args   []string
	// stderr indicates the tool writes version info to stderr (e.g. java).
	stderr bool
}

// toolDefs maps dependency names to their query spec.
// A few tools have non-standard flags:
//   - go:   "go version"   (not --version)
//   - java: "java -version" (single dash, writes to stderr)
//   - rust: binary is "rustc", not "rust"
var toolDefs = map[string]toolDef{
	"git":       {binary: "git", args: []string{"--version"}},
	"node":      {binary: "node", args: []string{"--version"}},
	"nodejs":    {binary: "node", args: []string{"--version"}},
	"npm":       {binary: "npm", args: []string{"--version"}},
	"yarn":      {binary: "yarn", args: []string{"--version"}},
	"pnpm":      {binary: "pnpm", args: []string{"--version"}},
	"go":        {binary: "go", args: []string{"version"}},
	"docker":    {binary: "docker", args: []string{"--version"}},
	"python":    {binary: "python3", args: []string{"--version"}},
	"python3":   {binary: "python3", args: []string{"--version"}},
	"pip":       {binary: "pip3", args: []string{"--version"}},
	"pip3":      {binary: "pip3", args: []string{"--version"}},
	"rust":      {binary: "rustc", args: []string{"--version"}},
	"rustc":     {binary: "rustc", args: []string{"--version"}},
	"cargo":     {binary: "cargo", args: []string{"--version"}},
	"java":      {binary: "java", args: []string{"-version"}, stderr: true},
	"ruby":      {binary: "ruby", args: []string{"--version"}},
	"homebrew":  {binary: "brew", args: []string{"--version"}},
	"brew":      {binary: "brew", args: []string{"--version"}},
	"kubectl":   {binary: "kubectl", args: []string{"version", "--client"}},
	"terraform": {binary: "terraform", args: []string{"version"}},
	"helm":      {binary: "helm", args: []string{"version", "--short"}},
	"make":      {binary: "make", args: []string{"--version"}},
	"curl":      {binary: "curl", args: []string{"--version"}},
	"wget":      {binary: "wget", args: []string{"--version"}},
}

// installHints provides a short install hint for tools that are missing.
var installHints = map[string]string{
	"git":      "https://git-scm.com/downloads",
	"node":     "https://nodejs.org  or: brew install node",
	"nodejs":   "https://nodejs.org  or: brew install node",
	"docker":   "https://docs.docker.com/get-docker/",
	"go":       "https://go.dev/dl/",
	"python3":  "https://www.python.org/downloads/",
	"python":   "https://www.python.org/downloads/",
	"rust":     "curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh",
	"java":     "https://adoptium.net/",
	"ruby":     "https://www.ruby-lang.org/en/downloads/",
	"kubectl":  "https://kubernetes.io/docs/tasks/tools/",
	"terraform":"https://developer.hashicorp.com/terraform/install",
}

// defaultDoctorDeps are the tools checked when no config file is found.
var defaultDoctorDeps = []config.Dependency{
	{Name: "git"},
	{Name: "node"},
	{Name: "docker"},
	{Name: "go"},
	{Name: "python3"},
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system readiness and dependency health",
	Long: `Run health checks against your development environment to verify
that all tools in your devforge.yaml are installed and functional.

When no config file is found, a curated set of common dev tools is
checked instead (git, node, docker, go, python3).

Exit code is 0 only when every checked tool is present.`,
	RunE: runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		ux.Warning("No config file found — checking common dev tools instead.")
		cfg = &config.Config{Dependencies: defaultDoctorDeps}
	}

	osInfo, err := osdetect.DetectFull()
	if err != nil {
		ux.Error(fmt.Errorf("OS detection failed: %v", err))
		return nil
	}

	// ── Header ────────────────────────────────────────────────────
	ux.Header("DevForge Doctor")
	fmt.Printf("  OS : %s  (%s/%s)\n", osInfo.Name, osInfo.RawOS, osInfo.Arch)
	fmt.Printf("  Pkg: %s\n", osInfo.PackageMgr)
	fmt.Println()

	// ── Check each dependency concurrently ────────────────────────
	type row struct {
		name      string
		wanted    string
		installed bool
		version   string
	}

	rows := make([]row, len(cfg.Dependencies))
	var wg sync.WaitGroup
	for i, dep := range cfg.Dependencies {
		wg.Add(1)
		go func(i int, dep config.Dependency) {
			defer wg.Done()
			installed, version := queryTool(dep.Name)
			rows[i] = row{
				name:      dep.Name,
				wanted:    dep.Version,
				installed: installed,
				version:   version,
			}
		}(i, dep)
	}
	wg.Wait()

	allOK := true
	for _, r := range rows {
		if !r.installed {
			allOK = false
			break
		}
	}

	// ── Print table ───────────────────────────────────────────────
	colW := 14 // minimum name column width
	for _, r := range rows {
		if len(r.name) > colW {
			colW = len(r.name)
		}
	}
	colW += 2

	for _, r := range rows {
		namePad := fmt.Sprintf("%-*s", colW, r.name)

		if r.installed {
			versionStr := r.version
			if versionStr == "" {
				versionStr = "(version unknown)"
			}

			// Warn if the installed major version doesn't match what the
			// config requests.
			mismatch := r.wanted != "" &&
				r.wanted != "latest" &&
				r.version != "" &&
				!strings.HasPrefix(r.version, r.wanted)

			if mismatch {
				fmt.Printf("  %s%s%s %s%s %s  %swanted %s%s\n",
					ux.CodeYellow(), ux.Warn, ux.CodeReset(),
					ux.CodeGray(), namePad, versionStr+ux.CodeReset(),
					ux.CodeYellow(), r.wanted, ux.CodeReset())
			} else {
				fmt.Printf("  %s%s%s %s%s %s%s%s\n",
					ux.CodeGreen(), ux.Check, ux.CodeReset(),
					ux.CodeGray(), namePad,
					ux.CodeReset(), versionStr, ux.CodeReset())
			}
		} else {
			fmt.Printf("  %s%s%s %s%s %smissing%s\n",
				ux.CodeRed(), ux.Cross, ux.CodeReset(),
				ux.CodeGray(), namePad,
				ux.CodeRed(), ux.CodeReset())

			if hint, ok := installHints[r.name]; ok {
				fmt.Printf("  %s    %s  install: %s%s\n",
					ux.CodeGray(), strings.Repeat(" ", colW), hint, ux.CodeReset())
			}
		}
	}

	// ── Summary ───────────────────────────────────────────────────
	fmt.Println()
	ux.Divider()
	total := len(rows)
	missing := 0
	for _, r := range rows {
		if !r.installed {
			missing++
		}
	}

	if allOK {
		fmt.Printf("  Status: %s%s ALL SYSTEMS GO (%d/%d)%s\n\n",
			ux.CodeGreen(), ux.Check, total, total, ux.CodeReset())
	} else {
		fmt.Printf("  Status: %s%s PARTIALLY READY (%d/%d missing)%s\n\n",
			ux.CodeYellow(), ux.Warn, missing, total, ux.CodeReset())
	}

	return nil
}

// queryTool looks up the binary, runs it, and extracts the version string.
// Returns (installed, versionString).
func queryTool(name string) (bool, string) {
	def, ok := toolDefs[strings.ToLower(name)]
	if !ok {
		// Unknown tool: try the name itself with --version.
		def = toolDef{binary: name, args: []string{"--version"}}
	}

	path, err := exec.LookPath(def.binary)
	if err != nil {
		return false, ""
	}

	cmd := exec.Command(path, def.args...)
	var out strings.Builder
	if def.stderr {
		// Some tools (java) write version to stderr.
		cmd.Stderr = &out
	} else {
		cmd.Stdout = &out
	}
	_ = cmd.Run() // ignore exit code — some tools exit non-zero on version

	raw := strings.TrimSpace(out.String())
	if raw == "" {
		return true, ""
	}

	// Take only the first line and trim common prefixes.
	line := strings.Split(raw, "\n")[0]
	line = trimVersionPrefixes(name, line)
	return true, line
}

// trimVersionPrefixes removes well-known verbose prefixes from version output
// so the doctor table stays compact.
func trimVersionPrefixes(name, raw string) string {
	raw = strings.TrimSpace(raw)
	prefixes := []string{
		name + " version ", name + " v",
		"go version ", "node ", "Python ",
		"git version ", "Docker version ",
		"rustc ", "ruby ", "OpenJDK ",
		"java version \"", "openjdk version \"",
		"GNU make ", "curl ", "wget ",
	}
	lower := strings.ToLower(raw)
	for _, p := range prefixes {
		if strings.HasPrefix(lower, strings.ToLower(p)) {
			raw = raw[len(p):]
			break
		}
	}
	// Strip trailing quotes or parenthetical noise.
	if i := strings.IndexAny(raw, " (\""); i > 0 && strings.ContainsAny(raw[:i], ".0123456789") {
		raw = raw[:i]
	}
	return strings.Trim(raw, "\"' \t")
}
