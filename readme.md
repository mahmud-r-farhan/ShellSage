# 🚀 ShellSage - Autonomous AI Terminal Assistant & Developer Platform

[![Go Version](https://img.shields.io/badge/Go-1.24%2B-00ADD8?style=for-the-badge&logo=go)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=for-the-badge)](https://opensource.org/licenses/MIT)
[![Docker Support](https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker)](Dockerfile)
[![CI Matrix](https://img.shields.io/badge/CI-Passing-brightgreen?style=for-the-badge&logo=github-actions)](.github/workflows/ci.yml)

**ShellSage** is an enterprise-grade, lightning-fast, and open-source AI terminal companion built for software engineers, systems architects, debuggers, technical writers, and DevOps professionals.

It brings universal multi-provider LLM support, autonomous tool execution (filesystem, shell, web search, web scraping, git), conversation branching, custom persona studios, task queues, local machine time-based schedulers, native clipboard integration, and multi-format exports (Markdown, PDF, HTML, JSON) right to your command line.

---

## ✨ Key Capabilities

### 🌐 Universal Multi-Provider Engine — 20 Providers
Connect to any major LLM API, open-model host, or self-hosted server — all configured the
same way (`<PROVIDER>_API_KEY` / `_MODEL` / `_BASE_URL`) and driven by one registry, so
`shellsage provider list` always shows what actually exists:

| Tier | Providers |
| :--- | :--- |
| **Aggregators** | OpenRouter (300+ models) · GitHub Models (your `gh` token!) · Hugging Face Inference · Custom/OpenAI-compatible (vLLM, LM Studio, llama.cpp) |
| **Frontier APIs** | OpenAI (GPT-4.1, o4-mini) · Anthropic Claude (Sonnet/Opus 4) · Google Gemini (2.5 Pro/Flash) · Mistral (Codestral) · xAI Grok · Cohere (Command A) · DeepSeek (V3/R1) · Perplexity Sonar (live web) |
| **Speed/cost inference** | Groq · Cerebras (~1000 tok/s, free tier) · Together · Fireworks · DeepInfra · NVIDIA NIM · Azure OpenAI (api-key auth, deployments) |
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
- **Specialized Modes**:
  - `/plan <requirement>`: Architect detailed step-by-step implementation plans.
  - `/debug <error/log>`: Perform deep root-cause diagnostic and automated patch generation.
  - `/doc <file>`: Generate comprehensive markdown documentation & API specifications.

### 🌿 Conversation Tree & Branching
- **Alternative Responses** (`/retry`, `/alt`): Fork conversations at any turn and generate alternative completions.
- **Branch Navigator** (`/branch`): Switch between multiple conversation paths seamlessly.
- **Visual ASCII Tree** (`/tree`): Render the complete conversation structure in your terminal.
- **Context Compression** (`/compress`): LLM-fold long sessions into a summary node when you hit
  model context budgets (auto-trim keeps you under `max_context_tokens` in the meantime).
- **Session Library**: auto-save to `~/.shellsage/conversations` with `/resume`, `/rename`,
  `/delete` — your history no longer lives in whatever directory you happened to `cd` into.

### 🎭 Custom Persona Studio
- **Prebuilt Personas**: Senior Software Engineer, Enterprise Architect, Root-Cause Specialist, Technical Writer, DevOps & SRE, Security Auditor, Agile Planner, **Senior Code Reviewer** and **Test & QA Engineer**.
- **Custom Persona Builder** (`/persona create`): Interactively create and persist specialized system personas saved to `~/.shellsage/personas/`.

### ⏰ Task Queue & Local Time Scheduler
- **Background Task Queue** (`/queue add`, `/queue list`, `/queue run`): Queue multi-step engineering tasks.
- **Local Time Scheduler** (`/schedule at 15:30 <task>`, `/schedule in 30m <task>`, `/schedule daily 09:00 <task>`):
  timer-driven (no busy polling), jobs persist to `~/.shellsage/schedule.json` and are marked *missed* —
  never silently fired — if the process was down at trigger time.

### 📋 Clipboard & Rich Exporters
- **Native OS Clipboard** (`/copy`, `/copy code`, `/copy all`): True cross-platform clipboard support.
- **Multi-Format Export** (`/export md`, `/export pdf`, `/export html`, `/export json`): Export conversation transcripts with metadata and token usage breakdowns to clean Markdown, printable PDF documents, or dark-themed HTML.

### 📊 Token Analytics & Cost Tracker
- Real-time token monitoring (prompt, completion, total).
- Cost calculator across different model pricing tiers (updated for GPT-4.1, Claude 4, Gemini 2.5, Grok 3, Cerebras free tier…).
- Context-token budget with automatic trimming — long sessions can't blow up request size.
- Detailed session metrics via `/stats` and `/analytics`.

---

## 🚀 Quick Start

### 1. Installation

**Option A: Install via Go**
```bash
go install github.com/mahmud-r-farhan/ShellSage@latest
```

**Option B: Build from Source**
```bash
git clone https://github.com/mahmud-r-farhan/ShellSage.git
cd ShellSage
make setup
make build
./shellsage
```

**Option C: Docker**
```bash
docker-compose up --build
```

### 2. Configuration Setup Wizard

Launch the interactive setup wizard to configure your preferred provider and API keys:
```bash
./shellsage --config
# OR inside the chat session:
/config
```

Alternatively, copy `.env.example` to `.env`:
```bash
cp .env.example .env
```

---

## 💬 Command Reference

| Command | Description |
| :--- | :--- |
| `/help` | Display interactive command guide |
| `/config` | Launch interactive Configuration & Provider Setup Wizard |
| `/provider` | Switch active provider (OpenAI, Claude, Gemini, Groq, Ollama...) |
| `/model` | Switch model for current provider |
| `/persona` | Choose persona or launch Custom Persona Wizard (`/persona create`) |
| `/temp` | Adjust temperature / creativity (0.0 - 1.0) |
| `/copy` | Copy last assistant response to system clipboard |
| `/copy code` | Copy only code snippets from last response |
| `/copy all` | Copy entire conversation transcript |
| `/export <md\|pdf\|html\|json>` | Export conversation to Markdown, PDF, HTML, or JSON |
| `/retry` / `/alt` | Generate alternative response for previous user turn |
| `/branch` | List and switch between conversation branches |
| `/tree` | Render visual conversation tree |
| `/agent <goal>` | Execute goal autonomously with local tools & web search |
| `/plan <task>` | Generate detailed architectural implementation plan |
| `/debug <error>` | Diagnose root cause and generate code fix |
| `/doc <file>` | Generate comprehensive documentation for a code file |
| `/queue <add\|list\|run>` | Manage background task queue |
| `/schedule <at\|in\|list>` | Schedule automated work at local machine time |
| `/stats` / `/analytics` | Display session duration, token consumption, and estimated cost |
| `/save` / `/load` / `/list` | Persist and retrieve conversation sessions |
| `/clear` | Reset conversation memory |
| `/exit` / `/quit` | Save state and exit ShellSage |

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

# Architecture planning / error diagnosis / doc generator
shellsage plan "Design a WebSocket chat server in Go"
shellsage debug "panic: invalid memory address" --file crash.txt
shellsage doc internal/branch/tree.go > docs/BRANCH.md

# Manage providers & models without leaving the terminal
shellsage provider list                     # all 20 providers + which keys are set
shellsage provider use anthropic            # persisted to ~/.shellsage/config.json (0600)
shellsage models list --remote              # live catalog from the provider API
shellsage config set groq.api_key gsk_...   # non-interactive setup (great for Docker/CI)

# Diagnostics & shell ergonomics
shellsage doctor                            # config, keys, endpoint ping, clipboard — with exit codes
shellsage completion bash > /etc/bash_completion.d/shellsage
shellsage sessions list | delete chat_x.json

# Legacy v3 flags keep working
shellsage --agent "…goal…" --plan "…requirement…" --debug "…error…" --doc file.go --audit path
```

**Global flags** (all subcommands): `--provider --model --temp --max-tokens --stream auto|on|off
--json --quiet --yes --no-color --timeout 120s --image shot.png --system "…" --config-file path`

**Scripting contract:** `--json` prints `{ok, provider, model, mode, content, usage, cost_usd,
duration_ms, error}`; exit codes are 0/1/2 (ok / provider error / usage error) so failures break
pipelines correctly. Decorative output never pollutes stdout (usage lines go to stderr), and
`NO_COLOR` is respected.

## 🌍 Real-World Recipes

```bash
# 1) CI quality gate: block merges when the AI flags new critical secrets
git diff origin/main | shellsage sec sast . --json | jq -e '.critical_count == 0'

# 2) Nightly repo health note (schedule works inside the REPL)
#    /schedule daily 09:00 Scan CI logs in .github and summarize flaky tests

# 3) Postmortem in 30 seconds
kubectl logs deploy/payments --previous > crash.log
shellsage debug - --provider claude < crash.log | tee postmortem.md

# 4) Onboard onto an unfamiliar codebase for free (no key needed)
ollama serve && ollama pull qwen2.5-coder
shellsage --provider ollama agent "map the architecture of ./internal into ARCHITECTURE.md"

# 5) Live-docs-aware answers while pairing
shellsage ask "latest Chi router middleware example" --provider perplexity

# 6) Paste a screenshot of a bug report (vision-capable models)
shellsage ask "write a Go failing test for this bug" --image report.png
```

---

## 🐳 Docker Support

Run ShellSage completely isolated inside a minimal container:

```bash
# Build and run with docker-compose
docker-compose run --rm shellsage

# Or build standalone Docker image
docker build -t shellsage:latest .
docker run -it --rm -v $(pwd)/conversations:/root/conversations shellsage:latest
```

---

## 🧪 Testing & CI/CD

ShellSage includes comprehensive unit tests with 100% test pass rates across all packages.

```bash
# Run all tests
make test

# Run tests with race detection
make test-race

# Run code linter
make lint
```

Cross-platform GitHub Actions workflows are configured in `.github/workflows/ci.yml` (Linux, macOS, Windows) and `.github/workflows/release.yml` for automated release binary distribution.

---

## 📄 License

Distributed under the MIT License. See [LICENSE](LICENSE) for details.

Made with ❤️ for the global developer community.
