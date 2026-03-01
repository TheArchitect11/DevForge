<p align="center">
  <h1 align="center">⚒️ DevForge</h1>
  <p align="center">Enterprise-grade distributed provisioning platform</p>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/go-1.22+-00ADD8?style=flat-square&logo=go" alt="Go Version" />
  <img src="https://img.shields.io/badge/platform-macOS%20|%20Linux%20|%20Windows-lightgrey?style=flat-square" alt="Platform" />
  <img src="https://img.shields.io/badge/architecture-distributed-blueviolet?style=flat-square" alt="Architecture" />
  <img src="https://img.shields.io/badge/license-MIT-green?style=flat-square" alt="License" />
</p>

---

## What is DevForge?

DevForge is a **distributed DevOps automation platform** for enterprise project provisioning. It automates project scaffolding, dependency management, and environment setup across teams and infrastructure — with policy enforcement, RBAC, audit logging, and remote execution.

**Three deployment modes:**
- **CLI** — Local project scaffolding with cross-platform support
- **Agent** — HTTPS server for remote provisioning execution
- **Server** — Central config and policy management (SaaS-ready)

---

## Architecture

```
                        ┌──────────────────────┐
                        │    devforge-server    │
                        │  (Central Config +    │
                        │   Policy + RBAC +     │
                        │   Agent Registry)     │
                        └──────┬───────────────┘
                               │ REST API
              ┌────────────────┼────────────────┐
              │                │                │
    ┌─────────▼──────┐ ┌──────▼───────┐ ┌──────▼───────┐
    │ devforge-agent  │ │ devforge-agent│ │ devforge-agent│
    │ (Host A, :8443) │ │ (Host B)     │ │ (Host C)     │
    │ • TLS           │ │ • Heartbeat  │ │ • Audit      │
    │ • Token Auth    │ │ • Policy     │ │ • RBAC       │
    │ • Remote Exec   │ └──────────────┘ └──────────────┘
    └─────────▲──────┘
              │ HTTPS
    ┌─────────┴──────┐
    │   devforge CLI  │
    │ --remote flag   │
    │ • Local or      │
    │   remote init   │
    └────────────────┘
```

```
┌─────────────────────────────────────────────────────────────┐
│                    Internal Packages                        │
├──────────────┬──────────────┬──────────────┬───────────────┤
│  osdetect    │   config     │   logger     │   executor    │
│  (multi-OS)  │  (Viper)     │ (logrus+JSON)│  (os/exec)    │
├──────────────┼──────────────┼──────────────┼───────────────┤
│  installer/  │  template/   │  envgen/     │  rollback/    │
│  brew|apt|   │  (go-git)    │  (.env)      │  (LIFO undo)  │
│  yum|choco   │              │              │               │
├──────────────┼──────────────┼──────────────┼───────────────┤
│  registry/   │  updater/    │  plugins/    │  semver/      │
│  (HTTP+cache)│ (GitHub)     │ (JSON stdio) │  (parse/cmp)  │
├──────────────┼──────────────┼──────────────┼───────────────┤
│  remote/     │  agent/      │  server/     │  security/    │
│  (protocol)  │ (HTTPS svr)  │ (REST API)   │  (validate)   │
├──────────────┼──────────────┼──────────────┼───────────────┤
│  rbac/       │  policy/     │  audit/      │  tls/         │
│  (roles+mw)  │ (engine)     │ (JSON logs)  │  (certs)      │
└──────────────┴──────────────┴──────────────┴───────────────┘
```

---

## Installation

### From Source

```bash
git clone https://github.com/ChinmayyK/DevForge.git
cd DevForge
make build          # builds all 3 binaries
make build-all      # cross-compile for all platforms
make release        # build-all + SHA-256 checksums
```

### Individual Binaries

```bash
make build-cli      # devforge
make build-agent    # devforge-agent
make build-server   # devforge-server
```

---

## CLI Usage

### Scaffold a Project

```bash
devforge init my-app
devforge init my-app --dry-run --verbose
devforge init my-app --config ./custom.yaml
devforge init my-app --json-logs
```

### Remote Execution

```bash
export DEVFORGE_TOKEN=<agent-token>
devforge init my-app --remote https://host:8443
```

### System Health Check

```bash
devforge doctor
```

### Template Registry

```bash
devforge templates list
devforge templates search react
devforge templates use next-app
```

### Plugins

```bash
devforge plugin list
devforge plugin run my-plugin
```

### Auto-Update

```bash
devforge update
```

---

## Agent

The DevForge Agent is an HTTPS server that executes provisioning requests remotely.

```bash
# Production (with TLS)
devforge-agent --port 8443 --token <secret>

# Development (no TLS)
devforge-agent --port 8443 --dev --token <secret>

# With server registration
devforge-agent --port 8443 --token <secret> --server https://central.devforge.dev
```

**Capabilities:**
- TLS with auto-generated self-signed certs (dev mode)
- Token-based authentication
- RBAC enforcement per-request
- Audit logging of all operations
- Graceful shutdown with signal handling
- Registration + heartbeat with central server

---

## Server

The DevForge Server is the central management plane for orgs, policies, and agents.

```bash
devforge-server --port 8080
```

**REST API:**

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/auth/token` | Generate API token |
| `POST` | `/org` | Create organization |
| `GET` | `/org/{id}/config` | Get org configuration |
| `POST` | `/org/{id}/policy` | Set org policy |
| `POST` | `/api/v1/agents/register` | Register agent |
| `POST` | `/api/v1/agents/heartbeat` | Agent heartbeat |
| `GET` | `/health` | Health check |

---

## RBAC

| Role | Permissions |
|------|-------------|
| `admin` | init, install, update, plugin-run, policy, audit-read, org-manage |
| `developer` | init, install, update, plugin-run |
| `viewer` | audit-read |

RBAC is enforced via middleware in both the Agent and Server.

---

## Policy Engine

Define org-level policies in YAML:

```yaml
allowed_dependencies:
  - node
  - git

blocked_templates:
  - unknown-repo

max_node_version: 20

allowed_plugins:
  - lint
  - format

require_tls: true
```

Policies are enforced before every:
- Dependency installation
- Template clone
- Plugin execution

---

## Security Model

| Layer | Protection |
|-------|-----------|
| Transport | TLS 1.2+ with ECDSA P-256 certs |
| Auth | Bearer token validation |
| RBAC | Role-based permission middleware |
| Input | URL, path, name validation + sanitization |
| Audit | JSON-structured audit trail for all operations |
| Exec | No shell concatenation — argument arrays only |
| Policy | Whitelist-based dependency & template control |

---

## Audit Logging

All critical operations are logged to `~/.devforge/audit/audit.log`:

```json
{"timestamp":"2026-03-01T11:15:00Z","user":"admin","action":"remote_execute","resource":"my-app","success":true,"machineId":"host-1","detail":"command=init project=my-app"}
```

Fields: `timestamp`, `user`, `action`, `resource`, `success`, `machineId`, `detail`, `sourceIp`

---

## Testing

```bash
go test ./... -v -race -count=1
```

Integration tests cover:
- Policy blocking (deps, templates, versions)
- RBAC enforcement (roles, middleware)
- Remote protocol serialization
- Server handlers (org CRUD, agent registration)

---

## Folder Structure

```
devforge/
├── .github/workflows/release.yml    # CI/CD: 3 binaries × 4 platforms
├── cmd/
│   ├── agent/main.go                # devforge-agent binary
│   ├── server/main.go               # devforge-server binary
│   ├── root.go                      # CLI root + global flags
│   ├── init.go                      # Local + remote init
│   ├── doctor.go / templates.go     # System checks, registry
│   ├── update.go / plugin.go        # Auto-update, plugins
├── internal/
│   ├── agent/                       # HTTPS agent server
│   ├── server/                      # Central REST API server
│   ├── remote/                      # Remote execution protocol
│   ├── rbac/                        # Roles + permissions + middleware
│   ├── policy/                      # Policy engine + rules
│   ├── audit/                       # Structured audit logging
│   ├── tls/                         # TLS config + cert generation
│   ├── installer/                   # brew|apt|yum|choco
│   ├── config/ logger/ osdetect/    # Core utilities
│   ├── executor/ rollback/          # Command exec + undo
│   ├── registry/ updater/ plugins/  # Registry, update, plugins
│   ├── security/ semver/            # Validation, versioning
│   ├── template/ envgen/            # Cloning, env generation
├── tests/integration_test.go        # Integration test suite
├── config/default.yaml              # Default configuration
├── Makefile                         # Build all 3 binaries
└── main.go                          # CLI entry point
```

---

## Roadmap

- [ ] Interactive TUI mode
- [ ] Database-backed server storage
- [ ] Agent fleet dashboard
- [ ] Webhook notifications
- [ ] Template marketplace
- [ ] Config drift detection
- [ ] Multi-tenancy

---

## License

MIT

---

<p align="center">Built with ❤️ in Go — enterprise infrastructure for developer platforms</p>
