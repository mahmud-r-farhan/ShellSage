# 🚀 ShellSage - Autonomous AI Terminal Assistant & Developer Platform

[![Go Version](https://img.shields.io/badge/Go-1.24%2B-00ADD8?style=for-the-badge&logo=go)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=for-the-badge)](https://opensource.org/licenses/MIT)
[![Docker Support](https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker)](Dockerfile)
[![CI Matrix](https://img.shields.io/badge/CI-Passing-brightgreen?style=for-the-badge&logo=github-actions)](.github/workflows/ci.yml)

**ShellSage** is an enterprise-grade, lightning-fast, and open-source AI terminal companion built for software engineers, systems architects, debuggers, technical writers, and DevOps professionals.

It brings universal multi-provider LLM support, autonomous tool execution (filesystem, shell, web search, web scraping, git), conversation branching, custom persona studios, task queues, local machine time-based schedulers, native clipboard integration, and multi-format exports (Markdown, PDF, HTML, JSON) right to your command line.

---

## 🏗️ Clean & Modular Architecture

ShellSage follows Go best practices with a clean, modular folder layout:

```text
ShellSage/
├── cmd/
│   └── shellsage/         # Binary entrypoint (main package)
├── internal/
│   ├── agent/             # Autonomous ReAct loop & prompt parser
│   ├── app/               # App orchestration & interactive REPL
│   ├── branch/            # Conversation tree & branching engine
│   ├── cli/               # CLI subcommands, oneshot modes, & security
│   ├── clipboard/         # OS clipboard integration
│   ├── config/            # Secure configuration manager (0600 permissions)
│   ├── export/            # Exporters (Markdown, PDF, HTML, JSON)
│   ├── history/           # Session persistence & token analytics
│   ├── persona/           # Custom persona studio & system prompts
│   ├── provider/          # Single-descriptor registry & 20 LLM providers
│   ├── scheduler/         # Task queue & local machine time scheduler
│   ├── security/          # SAST, SSL, Port & Headers security auditors
│   ├── tools/             # Agent tools (FS, Git, Shell, Web, Patching)
│   └── tui/               # Terminal UI banners, menus & wizards
├── installer/             # Inno Setup Windows installer & assets (.iss, .ico)
├── scripts/
│   ├── ShellSage.ps1      # PowerShell helper module & autocompletion
│   ├── install.ps1        # PowerShell automated installer with desktop shortcut
│   └── release-dry-run.ps1 # GitHub Release build & dry-run simulation
├── Makefile               # Modular build, test, lint & cross-compile targets
└── .github/workflows/     # CI, Release & Linting GitHub Actions
```

---

## ✨ Key Capabilities

### 🌐 Universal Multi-Provider Engine — 20 Providers (Groq, OpenRouter, Claude...)
Connect to any major LLM API, open-model host, or self-hosted server — all configured the
same way (`<PROVIDER>_API_KEY` / `_MODEL` / `_BASE_URL`) and driven by one registry, so
`shellsage provider list` always shows what actually exists:

| Tier | Providers |
| :--- | :--- |
| **Aggregators** | OpenRouter (300+ models) · GitHub Models (your `gh` token!) · Hugging Face Inference · Custom/OpenAI-compatible (vLLM, LM Studio, llama.cpp) |
| **Frontier APIs** | OpenAI (GPT-4.1, o4-mini) · Anthropic Claude (Sonnet/Opus 4) · Google Gemini (2.5 Pro/Flash) · Mistral (Codestral) · xAI Grok · Cohere (Command A) · DeepSeek (V3/R1) · Perplexity Sonar (live web) |
| **Speed/cost inference** | Groq (LPU ultra-fast) · Cerebras (~1000 tok/s, free tier) · Together · Fireworks · DeepInfra · NVIDIA NIM · Azure OpenAI (api-key auth, deployments) |
| **Local & private** | Ollama (installed-models auto-detected) |

Model catalogs live in one table (`internal/provider/registry.go`) — adding a provider is a
**single struct entry** that auto-wires config defaults, env overrides, menus and the factory.

### 🤖 Autonomous Agent & Developer Tools
ShellSage features a built-in ReAct autonomous execution engine equipped with safe developer tools:
- **Filesystem**: Safe read, write, edit, glob `find_files`, fast codebase grep, and atomic
  multi-hunk `apply_patch` (with `dry_run` previews — all-or-nothing writes).
- **Shell Runner**: Commands/tests/builds with timeouts and a catastrophic-command deny-list.
- **Web**: Live search (DuckDuckGo), doc scraping, and a full `http_request` tool for poking REST
  APIs — with SSRF guard (link-local/metadata IPs always blocked).
- **Git Integration**: `git_status`, `git_diff`, `git_log`, `git_branch`.
- **Approval Gate**: every state-changing tool (write/exec/net) prompts — `y` / `n` / `a(ll)` —
  unless `--yes` (CI) or `approve_mode: read-only` (hard lockdown).

### ⚡ PowerShell & Windows Integration
Include native PowerShell scripts and completions:
```powershell
Import-Module .\scripts\ShellSage.ps1
Set-ShellSageProvider -Provider "groq" -ApiKey "gsk_..."
Invoke-ShellSageAsk -Prompt "Explain Go channels"
Test-ShellSageEnvironment
```

---

## 🚀 Quick Start

### 1. Installation

**Option A: Windows Setup Installer (Recommended for Windows)**
Download `ShellSage-Setup.exe` from the latest [GitHub Release](https://github.com/mahmud-r-farhan/ShellSage/releases):
- 🛡️ Built-in **MIT License** agreement screen
- ℹ️ Detailed installation overview and system capability inspection
- 🖥️ **Desktop shortcut checkmark option** for instant 1-click CLI launch
- 🌐 Automatic PATH environment variable integration
- 🚀 Instant CLI launcher upon finish

Or install via PowerShell:
```powershell
powershell -ExecutionPolicy Bypass -File scripts/install.ps1
```

**Option B: Install via Go**
```bash
go install github.com/mahmud-r-farhan/ShellSage/cmd/shellsage@latest
```

**Option C: Build from Source**
```bash
git clone https://github.com/mahmud-r-farhan/ShellSage.git
cd ShellSage
make setup
make build
./shellsage
```

**Option D: Build All Cross-Platform Binaries & Installer**
```bash
make build-all
make installer
```

---

## 🛠️ CLI Subcommands — built for scripts, pipes & CI

```bash
# One-shot chat; reads pipes automatically, JSON out for scripting
shellsage ask "Explain this failure" < build.log
go test ./... 2>&1 | shellsage ask - --provider groq --json | jq -r .content
cat stack.log | shellsage debug -              # paste any error stream into debug

# Autonomous agent with approvals (CI: --yes) and an audit trail
shellsage agent "find all TODO comments in internal/ and open a summary doc"
shellsage agent --yes --steps 20 --log-file agent.jsonl "fix the failing tests"

# Manage providers & models without leaving the terminal
shellsage provider list                     # all 20 providers + which keys are set
shellsage provider use groq                 # set Groq as active provider
shellsage models list --remote              # live catalog from the provider API
shellsage config set groq.api_key gsk_...   # non-interactive setup (great for Docker/CI)

# Shell completions (Bash, Zsh, Fish, PowerShell)
shellsage completion powershell
```

---

## 🧪 Testing & CI/CD

```bash
# Run all unit tests
make test

# Run tests with race detection
make test-race

# Run test coverage summary
make test-coverage

# Run code linter & format verification
make lint
```

Cross-platform GitHub Actions workflows are configured in `.github/workflows/ci.yml`, `.github/workflows/lint.yml`, and `.github/workflows/release.yml`.

---

## 👥 Author & Maintainer

Developed and maintained by **Mahmud Rahman**, Lead Developer at **The Bengal Bytes**.

- **GitHub**: [@mahmud-r-farhan](https://github.com/mahmud-r-farhan)
- **Organization**: The Bengal Bytes

---

## 📄 License

Distributed under the MIT License. Copyright (c) 2026 Mahmud Rahman, Lead Developer at The Bengal Bytes. See [LICENSE](LICENSE) for details.
