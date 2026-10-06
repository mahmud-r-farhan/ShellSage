# Changelog

All notable changes to **ShellSage** are documented in this file.

## [4.1.0] - 2026-10-06

### 📦 Windows Installer & Desktop Integration
- **Inno Setup Windows Installer**: Added `installer/ShellSage-Setup.iss` for automated GUI installation wizard.
- **MIT License Agreement Screen**: Direct integration of MIT License (`LICENSE`) displayed during installation with acceptance gate.
- **Pre & Post Installation Guidance**: Added comprehensive information screens (`installer/installer_info.txt` and `installer/install_complete.txt`) covering features, requirements, and commands.
- **Desktop Shortcut Checkmark Option**: Checkmark option during setup (`desktopicon` task) allowing users to create a Desktop shortcut (`ShellSage.lnk`).
- **Instant CLI Launch**: Finishing installation or double-clicking the desktop shortcut directly launches the interactive ShellSage CLI.
- **Environment PATH Integration**: Optional checkmark to register ShellSage directly into the system or user `PATH`.
- **Custom Application Icon**: High-resolution multi-size Windows icon (`installer/shellsage.ico`).
- **PowerShell Automated Installer**: Added `scripts/install.ps1` for script-based installation with Desktop shortcut checkmark and PATH integration.
- **Updated Setup Script**: Enhanced `setup.bat` with installer references and build steps.

### 🚀 CI/CD & GitHub Release Pipeline
- **Automated Windows Installer Build**: Updated `.github/workflows/release.yml` with `build-installer` job on `windows-latest` runner using Inno Setup compiler (`ISCC.exe`).
- **Release Dry-Run Tool**: Added `scripts/release-dry-run.ps1` and `make release-dry-run` to simulate and validate the full release pipeline, cross-platform compilation, checksums, and git push dry-run before publishing.
- **Author & Attribution**: Documented author Mahmud Rahman, Lead Developer at The Bengal Bytes across all packages, license files, and release metadata.

## [4.0.0] - 2026-10-05

### 🌐 Provider Engine (single source of truth + 12 new providers)
- **Provider registry**: one descriptor table (`internal/provider/registry.go`) now drives config defaults, generic `<PREFIX>_API_KEY/_MODEL/_BASE_URL` env overrides, interactive menus, `.env.example`, and the factory. Adding a provider is one struct entry.
- **New providers**: Mistral, xAI (Grok), Together AI, Fireworks, Cerebras, NVIDIA NIM, GitHub Models (`GITHUB_TOKEN`/`GH_TOKEN`), Hugging Face Inference, Perplexity, DeepInfra, Cohere, **Azure OpenAI** (`api-key` header auth + deployment-as-model, `AZURE_OPENAI_ENDPOINT`) — **20 providers total**.
- Aliases (`claude`, `grok`, `gh`, `lmstudio`…) normalize everywhere.
- Model catalogs refreshed (GPT-4.1/o4-mini, Claude Sonnet/Opus 4, Gemini 2.5 Pro/Flash, current Groq line-up…).
- `shellsage models list --remote` & Ollama installed-model auto-detection via `/api/tags` & `/models`.

### ⚡ Resilience & transport
- Shared HTTP layer with **exponential backoff + jitter** on 408/425/429/5xx, honors `Retry-After`, request recreation per attempt; works for JSON and SSE.
- Actionable API error messages (hints for 401/404/429/5xx: which key, which env var, which command fixes it).
- Per-provider timeout & retry budget config (`request_timeout_sec`, `max_retries`).

### 🧩 CLI power-up: subcommands, pipes, JSON
- Subcommands: `ask | agent | plan | debug | doc | audit | sec | chat | config | provider | models | sessions | doctor | completion | version` (all legacy `--flag` forms keep working).
- **Pipes everywhere**: `go test 2>&1 | shellsage debug -`; `--file`; prompts from stdin.
- **`--json` envelope** (`ok, provider, model, content, usage, cost_usd, duration_ms, error`) on stdout only — decorations move to stderr; documented exit codes (0/1/2).
- `shellsage doctor`: config perms, key presence, endpoint ping + latency, model availability, clipboard health.
- `shellsage config show|set|path` (non-interactive setup), `provider list|use|show`, `models list|use`, `sessions list|delete|rename|load`.
- `completion bash|zsh|fish` generators; `NO_COLOR`/`--no-color` respected; TTY detection.
- Vision: `--image <file|url>` (repeatable) and `/image` in the REPL; OpenAI `image_url` & Anthropic base64 blocks.

### 🤖 Agent upgrades
- **Approval gate** for mutating tools (`write_file`, `edit_file`, `apply_patch`, `run_command`, non-GET `http_request`) with `y/n/a(ll)`; modes `ask` (default) / `yolo` / `read-only`; non-TTY runs refuse mutations unless `--yes`.
- **Multi-tool steps**: several tool calls per assistant message (dedup'd), one assistant turn per step, results batched; parser accepts ```tool_call / ```json / bare JSON / `Action: … Action Input: …`.
- **Context pruning** of old tool results + `--steps`, `--max-time`, `--log-file` JSONL audit trail.
- **New tools**: `apply_patch` (atomic multi-hunk + dry-run), `find_files` (glob), `git_log`, `git_branch`, `http_request` (REST client with SSRF guard; private ranges gated by `SHELLSAGE_ALLOW_LOCAL_NET`).
- Shell hardening: expanded catastrophic deny-list; `sudo` refused.

### 💬 Chat & sessions
- `/compress` LLM-folds long conversations into a summary node; auto context-budget trim (`max_context_tokens`, default 60k).
- Sessions now in `~/.shellsage/conversations` (0600; legacy `./conversations` still merged into listings); `/resume`, `/rename`, `/delete`, `/tools`, `/approve`, `/models`.
- Personas: added **Senior Code Reviewer** and **Test & QA Engineer**.

### 🛡️ Security & storage
- `config.json` saved **0600** (was 0644 with API keys inside); corrupt config no longer silently half-loads.
- SAST: + GitHub PAT/OAuth, Slack, Stripe live keys, `InsecureSkipVerify`, unsafe-deserialization & SSRF-fetch rules; generic secret scanning now entropy-checked so `your_api_key_here` placeholders stop false-firing.
- Scheduler: next-due timer (was 100 ms polling), persistence, `/schedule daily`, `cancel`; missed-while-offline jobs reported, never silently run.

### 🧪 Testing / CI
- New suites: registry integrity (uniqueness, defaults-in-catalog, factory round-trip for all 20 providers), alias normalization, retry-then-succeed transport, Azure `api-key` header, multimodal payload shape, tool-parser matrix, approval allow/deny, apply_patch atomicity, find_files, http_request + metadata-IP block, config perms/env matrix, sessions isolation, SAST new rules & FP suppression, CLI config-set parser, completion generators.
- CI additionally runs on `arena/**` & `feat/**` push branches + `workflow_dispatch`.

### ⚠️ Behavior changes (v3 → v4)
- Agent/one-shot runs no longer auto-execute mutating tools in interactive contexts (approval gate); use `--yes` or `approve_mode: yolo` for the old behavior.
- Non-TTY `agent`/`ask` runs exit 1 on API errors (was 0) and print usage stats to **stderr**.
- Session save location moved to `~/.shellsage/conversations` (legacy files still listed/loadable).

---


## [3.0.0] - 2026-08-30

### 🚀 Major Architectural Transformation & Next-Gen Capabilities
- **Modular Internal Architecture**: Refactored monolithic codebase into clean, decoupled internal packages (`internal/config`, `internal/provider`, `internal/persona`, `internal/branch`, `internal/clipboard`, `internal/export`, `internal/tools`, `internal/agent`, `internal/scheduler`, `internal/history`, `internal/tui`).
- **Universal Multi-Provider Engine**: Direct integration with OpenRouter, OpenAI, Anthropic Claude, Google Gemini, Groq Cloud, DeepSeek, Ollama (Local LLMs), and custom OpenAI-compatible endpoints with real-time token streaming.
- **Interactive Configuration & Setup Wizard**: Added `/config` and `--config` interactive terminal wizard for rapid API key configuration, model selection, and endpoint customization.
- **True System Clipboard Integration**: Cross-platform clipboard support using native OS bindings for `/copy`, `/copy code`, and `/copy all`.
- **Multi-Format Rich Conversation Exporters**: Export conversations with metadata and token usage breakdowns to GitHub-Flavored Markdown, clean PDF documents, dark-mode HTML, and JSON.
- **Conversation Tree & Branching**: Full conversation tree data structure allowing alternative response generation (`/retry`, `/alt`), branch switching (`/branch`), and visual ASCII tree rendering (`/tree`).
- **Custom Persona Studio**: Interactive custom persona builder (`/persona create`) with local persistence in `~/.shellsage/personas/`, plus 8 prebuilt engineering personas.
- **Autonomous Agent & Tool Calling**: ReAct autonomous execution engine equipped with safe tools:
  - `read_file`, `write_file`, `edit_file`, `list_dir`, `search_code` (grep)
  - `run_command` (safe shell execution with timeouts)
  - `web_search` (DuckDuckGo instant answers & web parsing)
  - `web_scrape` (clean HTML text extraction)
  - `git_status` & `git_diff`
- **Developer Modes & Direct CLI Flags**:
  - `/plan` / `--plan` for architectural planning
  - `/debug` / `--debug` for root-cause error analysis
  - `/doc` / `--doc` for code documentation generation
  - `/agent` / `--agent` for direct autonomous goal execution
- **Task Queue & Local Time-Based Scheduler**: Background task queue (`/queue`) and local machine clock scheduler (`/schedule at <HH:MM>`, `/schedule in <duration>`).
- **Persistent Analytics & Cost Tracking**: Persistent session storage, token usage tracking per model, and estimated API cost analytics (`/stats`, `/analytics`).
- **Docker Support**: Added multi-stage `Dockerfile` and `docker-compose.yml`.
- **GitHub Actions CI/CD**: Added `.github/workflows/ci.yml` (multi-OS matrix on Ubuntu, macOS, Windows) and `.github/workflows/release.yml` (automated binary releases).
- **100% Unit Test Coverage**: Automated test suites across all 10 internal packages.

---

## [2.0.0] - 2026-05-19
- Added prebuilt AI personas.
- Added temperature controls.
- Added session save/load.
- Added basic token count display.

## [1.0.0] - 2026-04-01
- Initial release with basic OpenRouter chat.
