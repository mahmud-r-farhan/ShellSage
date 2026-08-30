# Changelog

All notable changes to **ShellSage** are documented in this file.

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
