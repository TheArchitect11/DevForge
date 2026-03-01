<p align="center">
  <h1 align="center">⚒️ DevForge</h1>
  <p align="center">Production-grade cross-platform CLI for automated project scaffolding</p>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/go-1.26+-00ADD8?style=flat-square&logo=go" alt="Go Version" />
  <img src="https://img.shields.io/badge/platform-macOS%20|%20Linux%20|%20Windows-lightgrey?style=flat-square" alt="Platform" />
  <img src="https://img.shields.io/badge/license-MIT-green?style=flat-square" alt="License" />
</p>

---

## What is DevForge?

DevForge is an extensible CLI tool that eliminates the tedium of project setup. It detects your OS, installs dependencies with version pinning, clones templates from a remote registry, generates environment configuration, and rolls back automatically on failure.

**Key capabilities:**
- 🌍 **Cross-platform** — macOS (Homebrew), Linux (APT/YUM), Windows (Chocolatey)
- 📌 **Version pinning** — install specific dependency versions with semver comparison
- 📦 **Template registry** — browse, search, and use templates from a remote registry
- 🔌 **Plugin system** — extend DevForge with executable plugins via JSON stdin/stdout
- 🔄 **Auto-update** — check and install updates from GitHub releases with checksum verification
- 🛡️ **Security hardened** — URL validation, path traversal prevention, input sanitization
- 📊 **Structured logging** — text or JSON output with persistent file logs
- ⏪ **Rollback engine** — LIFO undo stack for all critical operations
- 🏗️ **CI/CD ready** — GitHub Actions pipeline for cross-compilation and release

---

## Installation

### From Source

```bash
git clone https://github.com/ChinmayyK/DevForge.git
cd DevForge
make build
sudo mv devforge /usr/local/bin/
```

### Cross-compile All Platforms

```bash
make build-all    # darwin/amd64, darwin/arm64, linux/amd64, windows/amd64
make release      # build-all + SHA-256 checksums
```

### Verify

```bash
devforge --version
devforge doctor
```

---

## Usage

### Scaffold a New Project

```bash
devforge init my-app
```

### Dry Run

```bash
devforge init my-app --dry-run
```

### Custom Config

```bash
devforge init my-app --config ./custom.yaml
```

### JSON Logging

```bash
devforge init my-app --json-logs --verbose
```

### System Health Check

```bash
devforge doctor
```

```
  DevForge Doctor — System Readiness Report
  ══════════════════════════════════════════
  DevForge version: 1.0.0

  Tool         Status       Version
  ────         ──────       ───────
  Homebrew     ✓ installed  Homebrew 5.0.15
  Node.js      ✓ installed  v25.3.0
  Git          ✓ installed  git version 2.50.1
  Docker       ✓ installed  Docker version 29.1.3

  ✅ All checks passed — system is ready!
```

### Template Registry

```bash
devforge templates list          # list all
devforge templates search react  # search by keyword
devforge templates use next-app  # show template details
```

### Auto-Update

```bash
devforge update
```

### Plugins

```bash
devforge plugin list             # list installed plugins
devforge plugin run my-plugin    # execute a plugin
```

---

## Configuration

```yaml
dependencies:
  - name: node
    version: "18"
  - name: git
    version: "latest"
  - name: docker
    version: "latest"

template: "https://github.com/some-org/node-template"
registryUrl: "https://registry.devforge.dev/templates.json"

linting: true
gitHooks: true
envFile: true
```

---

## Architecture

```
┌─────────────┐    ┌──────────────────────────────────────────┐
│   main.go   │───→│  cmd/ (Cobra CLI Layer)                  │
│  (ldflags)  │    │  root · init · doctor · templates        │
└─────────────┘    │  update · plugin                         │
                   └──────┬───────────────────────────────────┘
                          │
          ┌───────────────┼───────────────────┐
          │               │                   │
  ┌───────▼──────┐ ┌─────▼──────┐  ┌────────▼────────┐
  │  osdetect    │ │   config   │  │    logger        │
  │  (multi-OS)  │ │  (Viper)   │  │ (logrus+JSON)   │
  └──────────────┘ └────────────┘  └─────────────────┘
          │
  ┌───────▼──────────────────────────────────────────┐
  │  installer/ (Unified Interface)                   │
  │  factory → brew | apt | yum | choco               │
  └──────────────────────────────────────────────────┘
          │
  ┌───────▼──────┐ ┌────────────┐ ┌────────────────┐
  │  template/   │ │  envgen/   │ │  rollback/     │
  │  (go-git)    │ │  (.env)    │ │  (LIFO undo)   │
  └──────────────┘ └────────────┘ └────────────────┘
          │
  ┌───────▼──────┐ ┌────────────┐ ┌────────────────┐
  │  registry/   │ │  updater/  │ │  plugins/      │
  │  (HTTP+cache)│ │ (GitHub)   │ │ (JSON stdio)   │
  └──────────────┘ └────────────┘ └────────────────┘
          │
  ┌───────▼──────┐ ┌────────────┐
  │  executor/   │ │  security/ │
  │  (os/exec)   │ │ (validate) │
  └──────────────┘ └────────────┘
```

---

## Folder Structure

```
devforge/
├── .github/workflows/
│   └── release.yml          # CI/CD: test → cross-compile → release
├── cmd/
│   ├── root.go              # Root command + global flags
│   ├── init.go              # Project scaffolding with rollback
│   ├── doctor.go            # System readiness checks
│   ├── templates.go         # Registry: list/search/use
│   ├── update.go            # Auto-update from GitHub
│   └── plugin.go            # Plugin list/run
├── internal/
│   ├── config/config.go     # Viper YAML + version pinning
│   ├── logger/logger.go     # logrus with JSON option
│   ├── osdetect/osdetect.go # Multi-OS detection
│   ├── executor/executor.go # Safe command execution
│   ├── rollback/rollback.go # LIFO rollback engine
│   ├── security/security.go # Input validation + sanitization
│   ├── semver/semver.go     # Version parsing + comparison
│   ├── installer/
│   │   ├── installer.go     # Installer interface
│   │   ├── factory.go       # OS-based factory
│   │   ├── brew.go          # Homebrew (macOS)
│   │   ├── apt.go           # APT (Debian/Ubuntu)
│   │   ├── yum.go           # YUM (RHEL/CentOS)
│   │   └── choco.go         # Chocolatey (Windows)
│   ├── template/clone.go    # go-git template cloner
│   ├── envgen/envgen.go     # .env generator
│   ├── registry/
│   │   ├── schema.go        # Template types
│   │   ├── client.go        # HTTPS client + search
│   │   └── cache.go         # Offline cache
│   ├── updater/updater.go   # GitHub release updater
│   └── plugins/plugins.go   # Plugin discovery + execution
├── config/default.yaml      # Default configuration
├── Makefile                 # build / build-all / release / test
├── main.go                  # Entry point with ldflags
├── go.mod
└── go.sum
```

---

## Plugin System

Plugins are standalone executables in `~/.devforge/plugins/` named `devforge-plugin-<name>`.

**Contract:** Plugins receive JSON via stdin and return JSON via stdout:

```json
// Input (stdin)
{ "projectPath": "/path/to/project", "config": {}, "dryRun": false }

// Output (stdout)
{ "success": true, "message": "Plugin completed" }
```

---

## Security Model

- All command arguments sanitized against shell metacharacters
- URLs validated (scheme + host)
- Paths checked for directory traversal
- Dependency names validated against allowlist pattern
- YAML config strictly validated
- No shell string concatenation — argument arrays only
- File permissions properly restricted

---

## CI/CD Pipeline

On tag push (`v*`):
1. **Test** — `go test`, `go vet`
2. **Build** — matrix: darwin/amd64, darwin/arm64, linux/amd64, windows/amd64
3. **Release** — SHA-256 checksums + GitHub Release with all binaries

---

## Roadmap

- [ ] Interactive TUI mode (bubbletea)
- [ ] Monorepo support
- [ ] Custom post-scaffold hooks
- [ ] Template marketplace UI
- [ ] Config validation command
- [ ] Dependency auto-upgrade command
- [ ] Plugin marketplace

---

## License

MIT

---

<p align="center">Built with ❤️ in Go</p>
