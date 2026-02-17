# ClaudeShield

**Secure sandbox for Claude Code agents.**

> Work with Claude Code at full speed — without risking your machine or secrets.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go)](https://go.dev)

---

## The Problem

Claude Code is incredibly powerful — it can write, run, test code, work with git, Docker, and more. Many developers run it in "YOLO mode" (`--dangerously-skip-permissions`) to avoid clicking "approve" every 5 seconds.

But this creates real risks:
- 🔑 Agent accidentally reads/exposes your secrets (API keys, `.env`, SSH keys)
- 💀 Dangerous commands executed: `sudo`, `rm -rf`, network calls
- 📡 Potential data exfiltration through malicious repos
- 🔀 Parallel agents overwriting each other's files
- 📋 No audit trail of what the agent actually did
- ⏪ No easy rollback when things break

## The Solution

ClaudeShield wraps Claude Code in a **secure Docker sandbox** with:

| Feature | Description |
|---------|-------------|
| 🔒 **Isolation** | Agent runs in a locked-down Docker container — sees only your project |
| 🛡️ **Policy Engine** | Allow/block rules for commands. Blocks `sudo`, `rm -rf /`, `curl \| sh` by default |
| 🔑 **Secret Protection** | Secrets injected at runtime from ENV, 1Password, or Vault — never stored in plain text |
| 📋 **Audit Logging** | Full JSON log of every command, file access, and policy decision |
| ⏪ **Rollback** | Docker layer checkpoints before risky actions — one-click restore |
| 🔀 **Multi-Agent** | Each parallel agent gets its own git worktree + container. Clean merge via git |
| 🖥️ **TUI Dashboard** | Beautiful terminal UI to monitor everything in real-time |

## Quick Start

### Install

```bash
# macOS / Linux
brew install MakazhanAlpamys/tap/claudeshield

# Windows
scoop bucket add claudeshield https://github.com/MakazhanAlpamys/claudeshield
scoop install claudeshield

# From source
go install github.com/MakazhanAlpamys/claudeshield@latest
```

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) installed and running
- [Git](https://git-scm.com/) (for multi-agent worktrees)

### Usage

```bash
# Initialize ClaudeShield in your project
claudeshield init

# Start a sandboxed session
claudeshield start

# Start with a named agent
claudeshield start --agent backend-dev

# Launch the TUI dashboard
claudeshield ui

# View audit logs
claudeshield audit
claudeshield audit --last 20
claudeshield audit --json

# Check active sessions
claudeshield status

# Stop a session
claudeshield stop
claudeshield stop --all

# Rollback to last checkpoint
claudeshield rollback --latest

# Multi-agent workflow
claudeshield agent spawn frontend
claudeshield agent spawn backend
claudeshield agent list
claudeshield agent stop frontend --merge
```

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
│    (Cobra commands + Bubbletea TUI)         │
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

## License

[MIT](LICENSE)
