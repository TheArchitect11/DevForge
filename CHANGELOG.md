# Changelog

All notable changes to DevForge are documented here.
Follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and [Semantic Versioning](https://semver.org/).

---

## [1.1.1] - 2026-05-13

### Added
- DNF installer support for modern Fedora (22+), Rocky Linux, and AlmaLinux
- `doctor` command now works without a config file — falls back to checking common tools (git, node, docker, go, python3)
- `templates use <name>` now shows the correct `--template` flag syntax

### Fixed
- `go.mod` toolchain version corrected to Go 1.22
- Executor shell-injection guard now also rejects `<` and `>` redirection characters
- `osdetect` correctly distinguishes DNF-first distros from YUM-based ones
- `config/default.yaml` template URL updated to a real repository

---

## [1.0.0] - 2024-12-01

### Added
- `init <project-name>` command: interactive wizard + config-driven scaffolding
- OS detection for macOS, Debian/Ubuntu, RHEL/CentOS/Fedora, and Windows
- Package manager installers: Homebrew, APT, YUM, Chocolatey
- Parallel dependency installation with 3-worker semaphore and fail-fast cancel
- LIFO rollback engine for safe cleanup on failure
- Template registry client with 24-hour local cache
- Interactive setup wizard (powered by [Charmbracelet Huh](https://github.com/charmbracelet/huh))
- `doctor` command: per-dependency health check with version reporting
- `templates list / search / use` commands for the remote registry
- `plugin list / run` commands for stdin/stdout JSON plugin contract
- `update` command: GitHub release check + safe binary self-replacement
- `completion` command: bash, zsh, fish, and PowerShell
- `version` command: binary version, OS, Go runtime
- Structured error types with actionable hints (`ERR_*` codes)
- JSON and text log modes, persistent log file at `~/.devforge/logs/devforge.log`
- Semantic version parsing and major-version pinning
- `.env` file generation from `.env.template`
- Shell-injection sanitization on all executor arguments
- `--dry-run`, `--verbose`, `--json-logs`, `--force`, `--config` global flags
- Cross-platform CI/CD via GitHub Actions (macOS amd64/arm64, Linux amd64/arm64, Windows amd64)
