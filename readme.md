# 🚀 ShellSage - Autonomous AI Terminal Assistant & Developer Platform

[![Go Version](https://img.shields.io/badge/Go-1.24%2B-00ADD8?style=for-the-badge&logo=go)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=for-the-badge)](https://opensource.org/licenses/MIT)
[![Docker Support](https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker)](Dockerfile)
[![CI Matrix](https://img.shields.io/badge/CI-Passing-brightgreen?style=for-the-badge&logo=github-actions)](.github/workflows/ci.yml)

**ShellSage** is an enterprise-grade, lightning-fast, and open-source AI terminal companion built for software engineers, systems architects, debuggers, technical writers, and DevOps professionals.

It brings universal multi-provider LLM support, autonomous tool execution (filesystem, shell, web search, web scraping, git), conversation branching, custom persona studios, task queues, local machine time-based schedulers, native clipboard integration, and multi-format exports (Markdown, PDF, HTML, JSON) right to your command line.

---

## ✨ Key Capabilities

### 🌐 Universal Multi-Provider Engine
Connect directly to any major LLM provider or self-hosted local model:
- **OpenRouter** (Aggregator of 200+ models)
- **OpenAI** (GPT-4o, GPT-4o-mini, o1, o3-mini)
- **Anthropic Claude** (Claude 3.5 & 3.7 Sonnet, Claude 3.5 Haiku)
- **Google Gemini** (Gemini 2.0 Flash, Gemini 1.5 Pro)
- **Groq Cloud** (Ultra-fast 500+ tok/s inference on Llama 3.3 & Mixtral)
- **DeepSeek** (DeepSeek-V3, DeepSeek-R1)
- **Ollama** (Local offline privacy-first LLMs)
- **Custom / Self-Hosted** (vLLM, LM Studio, LocalAI)

### 🤖 Autonomous Agent & Developer Tools
ShellSage features a built-in ReAct autonomous execution engine equipped with safe developer tools:
- **Filesystem**: Safe read, write, edit (find & replace), directory tree, and fast codebase grep search.
- **Shell Runner**: Execute shell commands, test suites, and build scripts with timeout protection.
- **Web Search & Scraping**: Search the live web (DuckDuckGo API/HTML) and extract clean readable text from online documentation.
- **Git Integration**: Inspect git status and analyze unstaged diffs.
- **Specialized Modes**:
  - `/plan <requirement>`: Architect detailed step-by-step implementation plans.
  - `/debug <error/log>`: Perform deep root-cause diagnostic and automated patch generation.
  - `/doc <file>`: Generate comprehensive markdown documentation & API specifications.

### 🌿 Conversation Tree & Branching
- **Alternative Responses** (`/retry`, `/alt`): Fork conversations at any turn and generate alternative completions.
- **Branch Navigator** (`/branch`): Switch between multiple conversation paths seamlessly.
- **Visual ASCII Tree** (`/tree`): Render the complete conversation structure in your terminal.

### 🎭 Custom Persona Studio
- **Prebuilt Personas**: Senior Software Engineer, Enterprise Architect, Root-Cause Specialist, Technical Documentation Writer, DevOps & SRE Engineer, Security Auditor, and Agile Planner.
- **Custom Persona Builder** (`/persona create`): Interactively create and persist specialized system personas saved to `~/.shellsage/personas/`.

### ⏰ Task Queue & Local Time Scheduler
- **Background Task Queue** (`/queue add`, `/queue list`, `/queue run`): Queue multi-step engineering tasks.
- **Local Time Scheduler** (`/schedule at 15:30 <task>`, `/schedule in 30m <task>`): Schedule automated agent tasks to trigger at specific local clock times or after relative durations.

### 📋 Clipboard & Rich Exporters
- **Native OS Clipboard** (`/copy`, `/copy code`, `/copy all`): True cross-platform clipboard support.
- **Multi-Format Export** (`/export md`, `/export pdf`, `/export html`, `/export json`): Export conversation transcripts with metadata and token usage breakdowns to clean Markdown, printable PDF documents, or dark-themed HTML.

### 📊 Token Analytics & Cost Tracker
- Real-time token monitoring (prompt, completion, total).
- Cost calculator across different model pricing tiers.
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

## 🛠️ CLI Flags & Subcommands

Run non-interactive tasks directly from the terminal or in CI/CD scripts:

```bash
# Autonomous Agent Mode
shellsage --agent "Find all TODO comments in internal/ and create a summary report"

# Architectural Planning
shellsage --plan "Design a high-throughput WebSocket chat server in Go"

# Error Diagnostics
shellsage --debug "panic: runtime error: invalid memory address or nil pointer dereference"

# Code Documentation Generation
shellsage --doc "internal/branch/tree.go"

# Launch Configuration Wizard
shellsage --config
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
