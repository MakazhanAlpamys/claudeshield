# ClaudeShield

**Secure sandbox for Claude Code agents.**

> Work with Claude Code at full speed — without risking your machine or secrets.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](https://go.dev)
[![Docker](https://img.shields.io/badge/Docker-required-2496ED?logo=docker)](https://docs.docker.com/get-docker/)

---

## Why ClaudeShield?

Claude Code is incredibly powerful — it can write, run, test code, work with git, Docker, and more. Many developers run it in "YOLO mode" (`--dangerously-skip-permissions`) to avoid clicking "approve" every 5 seconds.

But this creates real risks:
- 🔑 Agent accidentally reads/exposes your secrets (API keys, `.env`, SSH keys)
- 💀 Dangerous commands executed: `sudo`, `rm -rf`, network calls
- 📡 Potential data exfiltration through malicious repos
- 🔀 Parallel agents overwriting each other's files
- 📋 No audit trail of what the agent actually did
- ⏪ No easy rollback when things break

**ClaudeShield** fixes all of this. One command — and your agent runs in a hardened sandbox with full audit trail.

## Features

| Feature | Description |
|---------|-------------|
| 🔒 **Docker Isolation** | Agent runs in a locked-down container — `no-new-privileges`, all capabilities dropped, network disabled, 2GB memory limit |
| 🛡️ **Policy Engine** | Deny-by-default. Allow/block rules for commands. Blocks `sudo`, `rm -rf /`, `curl \| sh` out of the box |
| 🔑 **Secret Protection** | Secrets injected at runtime from ENV, 1Password, or HashiCorp Vault — never stored in plain text |
| 📋 **Audit Logging** | Full JSONL log of every command, file access, and policy decision |
| ⏪ **Rollback** | Docker layer checkpoints before risky actions — one-click restore |
| 🔀 **Multi-Agent** | Each parallel agent gets its own git worktree + container. Clean merge via git |
| 🖥️ **TUI Dashboard** | Terminal UI to monitor sessions, audit logs, and rules in real-time |

## Quick Start

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) installed and running
- [Git](https://git-scm.com/) (for multi-agent worktrees)
- [Go 1.25+](https://go.dev/dl/) (to build from source)

### Install from source

```bash
git clone https://github.com/MakazhanAlpamys/claudeshield
cd claudeshield
make build
make docker-sandbox
```

### Usage

```bash
# Initialize ClaudeShield in your project
claudeshield init

# Start a sandboxed session
claudeshield start

# Start with a named agent
claudeshield start --agent backend-dev

# Check active sessions
claudeshield status

# View audit logs
claudeshield audit
claudeshield audit --last 20
claudeshield audit --json

# Launch the TUI dashboard
claudeshield ui

# Stop a session
claudeshield stop
claudeshield stop --all

# Rollback to last checkpoint
claudeshield rollback --list
claudeshield rollback --latest

# Multi-agent workflow
claudeshield agent spawn frontend
claudeshield agent spawn backend
claudeshield agent list
claudeshield agent stop frontend --merge
```

## What happens under the hood

When you run `claudeshield start`:

1. Loads policy rules from `.claudeshield.yaml`
2. Loads secrets from the configured provider (ENV / 1Password / Vault)
3. Creates a **hardened Docker container** with:
   - `--security-opt no-new-privileges`
   - `--cap-drop ALL` (only CHOWN, FOWNER, SETGID, SETUID added)
   - `--network none` (no internet access by default)
   - `--memory 2g` limit
4. Mounts only your project directory as `/workspace`
5. Injects secrets as environment variables (never written to disk)
6. Starts logging every action to `.claudeshield/logs/`

Every command executed inside the sandbox goes through the **policy engine** first — blocked commands are denied and logged.

## Configuration

ClaudeShield uses `.claudeshield.yaml` in your project root:

```yaml
# Sandbox settings
sandbox:
  mount: ".:/workspace:rw"
  network: false            # Block network access
  use_gvisor: false         # Use gVisor for extra isolation (Linux)
  memory_limit: "2g"
  cpu_limit: 2.0

# Command policy rules
rules:
  allow:
    - pattern: "git *"
    - pattern: "npm *"
    - pattern: "python *"
    - pattern: "go *"

  block:
    - pattern: "sudo *"
      reason: "Privilege escalation not allowed"
    - pattern: "rm -rf /"
      reason: "Root filesystem deletion blocked"
    - pattern: "curl * | sh"
      reason: "Remote code execution blocked"

# Secret management
secrets:
  provider: "env"  # env | 1password | vault

# Audit logging
audit:
  enabled: true
  log_dir: ".claudeshield/logs"
```

## Architecture

```
┌─────────────────────────────────────────────┐
│                ClaudeShield CLI              │
│        (Cobra commands + Bubbletea TUI)     │
├──────────┬──────────┬──────────┬────────────┤
│  Policy  │  Secrets │  Audit   │  Rollback  │
│  Engine  │ Registry │  Logger  │  Manager   │
├──────────┴──────────┴──────────┴────────────┤
│              Sandbox Engine                  │
│         (Docker API + gVisor)               │
├─────────────────────────────────────────────┤
│           Orchestrator                       │
│     (Git worktrees + multi-agent)           │
└─────────────────────────────────────────────┘
```

## Tech Stack

- **Go** — single binary, fast, excellent Docker SDK
- **Cobra** — CLI framework (same as kubectl, docker, gh)
- **Bubble Tea** — TUI framework
- **Docker SDK** — container management
- **YAML** — user-facing config
- **JSONL** — machine-parseable audit logs

## Development

```bash
# Clone
git clone https://github.com/MakazhanAlpamys/claudeshield
cd claudeshield

# Install dependencies
make deps

# Build
make build

# Run tests
make test

# Build sandbox Docker image
make docker-sandbox

# Run TUI
make run
```

## Contributing

Contributions are welcome! Feel free to open issues and pull requests.

## License

[MIT](LICENSE)
