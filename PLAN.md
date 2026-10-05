# 🧭 ShellSage v4.0 — Comprehensive Enhancement Plan

> **Goal:** evolve ShellSage from a solid multi-provider chat CLI into a genuinely
> real-world, scriptable, production-grade AI developer agent: more providers, a more
> powerful CLI surface, a safer & stronger agent, and hardening for daily-driver use.

This document is the engineering plan behind the `v4.0.0` release. Every phase below
is implemented in this branch (not just a wish-list).

---

## 1. Audit of the Current State (v3.0.0)

**Strengths**
- Clean layered architecture: `provider` (LLM adapters), `agent` (ReAct loop),
  `tools` (registry of local capabilities), `branch` (conversation tree),
  `persona`, `scheduler`, `export`, `security` (SAST/headers/SSL/ports), `tui`, `history`.
- Zero-heavy-dependencies philosophy (only 3 external modules).
- Multi-provider chat, conversation branching, export, personas, task queue/scheduler.

**Gaps found during the audit (each one is addressed in v4.0.0)**

| # | Gap | Impact in real-world use | Fix |
|---|-----|--------------------------|-----|
| G1 | Only 8 providers; per-provider boilerplate duplicated in 4 tables (`config.DefaultModels`, `config.DefaultBaseURLs`, `applyEnvOverrides`, `tui.SelectProviderMenu`) | Adding a provider is error-prone; drift between tables | Provider **registry** (single descriptor table) + 12 new providers |
| G2 | No one-shot / piping UX (`ask`) — everything requires the REPL | Can't use it in scripts, CI, or `cmd | shellsage` workflows | `shellsage ask`, stdin/-file input, `--json`, exit codes |
| G3 | No subcommands; only 7 global flags | Awkward UX (`shellsage --agent "…"` vs `shellsage agent …`) | Subcommand dispatcher (`ask/agent/plan/debug/doc/audit/sec/config/provider/models/sessions/doctor/completion`) with full flag set; legacy flags still work |
| G4 | Agent has no approval gate — `write_file`/`run_command` execute silently | Dangerous for autonomous mode on real machines | Approval callbacks + `--yes`, per-tool risk classification, non-TTY auto-policy |
| G5 | Agent parser only accepts one fenced ```tool_call format; assistant message duplicated per tool call | Fragile with many models; breaks context | Robust multi-call parser (fenced JSON, `Action/Action Input`, raw JSON line), single assistant turn per step |
| G6 | No retries/backoff on 429/5xx | Rate limits & flaky VPNs kill sessions | Shared HTTP layer with exponential backoff, `Retry-After` support |
| G7 | Long sessions blow the context window | API 400 errors, wasted $$ | Context budget (auto-trim) + `/compress` (LLM fold of old turns) |
| G8 | `config.json` written world-readable (0644) with **API keys** in it | Local-infosec leak | 0600 perms, `config show` masks keys, `config set` non-interactive |
| G9 | Scheduler busy-polls every 100 ms | Wasteful CPU/battery | Next-due timer (`time.Timer`) |
| G10 | Conversations saved relative to CWD (`./conversations`) | Lost between projects; surprising writes | `~/.shellsage/conversations` (legacy dir still listed), 0600 |
| G11 | No `doctor` / connectivity diagnostics; no way to see which keys are set | Slow onboarding, "is it my key or the API?" | `shellsage doctor` (config, keys, endpoint ping, clipboard, model availability), `models list --remote` |
| G12 | Vision/images unsupported | Can't paste screenshots/errors visually | `--image` (ask) + `/image` (REPL) → multimodal content parts for OpenAI-compatible & Anthropic providers |
| G13 | No shell completions, no help for `shellsage completion` | CLI ergonomics | `completion bash/zsh` generators |
| G14 | Limited toolset for coding agents (no glob, no multi-edit, no git log, no HTTP client tool) | Agent does 3 tool calls what should be 1 | New tools: `find_files`, `apply_patch` (atomic multi-edit + dry-run), `git_log`, `git_branch`, `http_request` |
| G15 | CI doesn't run on feature/agent branches like this one | Unverified pushes | CI triggers include `arena/**`; PRs already covered |

---

## 2. Provider Engine — Single Source of Truth + 12 New Providers

### 2.1 Registry design (`internal/provider/registry.go`)

```go
type Descriptor struct {
    ID, DisplayName, DocsURL string
    Format                   string // "openai" | "anthropic" | "ollama"
    EnvPrefix                string // e.g. "MISTRAL" -> MISTRAL_API_KEY/_MODEL/_BASE_URL
    DefaultBaseURL, DefaultModel string
    Models                   []string
    ExtraHeaders             map[string]string
    AuthHeader               string // "Authorization: Bearer" (default) or "api-key" (Azure)
    RequiresKey              bool   // false for Ollama/local
}
```

- `provider.All()` → ordered list. Everything derives from it:
  config defaults, env overrides (`<PREFIX>_API_KEY/_MODEL/_BASE_URL` generically),
  `/provider` menu, setup wizard, `shellsage provider list`, `.env.example` docs,
  pricing lookups (provider name for cost display).
- Adding a provider = **one struct**. No other file changes.
- Dependency direction fixed: `provider` no longer imports `config`
  (factory takes explicit args); `config` imports `provider` for defaults.

### 2.2 New providers (12)

| ID | Provider | Default model | Why it matters |
|----|----------|---------------|----------------|
| `mistral` | Mistral AI | `mistral-large-latest` | Codestral, EU alternative |
| `xai` |xAI (Grok)| `grok-3` | strong reasoning, long context |
| `together` | Together AI | `meta-llama/Llama-3.3-70B-Instruct-Turbo` | open-model hosting |
| `fireworks` | Fireworks AI | `accounts/fireworks/models/llama-v3p1-70b-instruct` | fast OSS inference |
| `cerebras` | Cerebras Inference | `llama3.1-70b` | ~1000 tok/s, free tier |
| `nvidia` | NVIDIA NIM | `deepseek-ai/deepseek-r1` | GPU-cloud models, free credits |
| `github` | GitHub Models | `gpt-4o` | use your `gh`/PAT — no new signup; CI-friendly |
| `huggingface` | HF Inference | `meta-llama/Llama-3.3-70B-Instruct` | huge OSS catalog |
| `perplexity` | Perplexity | `sonar` | built-in live web-search answers |
| `deepinfra` | DeepInfra | `deepseek-ai/DeepSeek-R1` | cheap OSS endpoint |
| `cohere` | Cohere (v2) | `command-a-03-2025` | enterprise RAG models |
| `azure` | Azure OpenAI | `gpt-4o` (`deployment`) | enterprise compliance; `api-key` auth header + deployment model |

Plus refresh existing model catalogs (o-series/gpt-4.1 for OpenAI, Claude Sonnet 4 /
Opus 4 for Anthropic, Gemini 2.5 Pro/Flash, current Groq line-up).

### 2.3 Robust transport (`internal/provider/transport.go`)
- `doWithRetry`: 429/5xx/network errors → backoff (250 ms → 8 s, jitter), honors
  `Retry-After`; configurable `MaxRetries` (default 3). Applied to both JSON and SSE.
- Clearer error messages: body snippet, provider display name, hint on 401/403/404
  (wrong key / model id / base URL).
- `ListRemoteModels(ctx)` on the base provider (GET `/models`) → `shellsage models
  list --remote` (works for OpenAI-compatible + Ollama `/api/tags`).

## 3. CLI Power-Ups (real-world scripting)

```
shellsage [subcommand] [flags] [args]
  ask "<prompt>"            one-shot chat (reads stdin/`-`/--file; pipes in, pipes out)
  agent "<goal>"            autonomous ReAct run (--yes, --steps, --max-time)
  plan|debug|doc|audit|sec  one-shot modes (same flags; debug/doc read files if piped)
  chat                      interactive REPL (default when no subcommand)
  config wizard|show|set k=v|path     show masks secrets; set is non-interactive
  provider list|use <id>|show
  models list [--remote]|use <id>
  sessions list|load|delete|rename
  doctor                    connectivity + config diagnostics (exit code reflects health)
  completion bash|zsh       shell completion script
```

- **Global flags:** `--provider`, `--model`, `--temp`, `--max-tokens`, `--stream=auto|on|off`,
  `--image <file>` (repeatable), `--json`, `--quiet`, `--yes`, `--no-color`, `--timeout`,
  `--config-file <path>` (config precedence: `--config-file` flag >
  `$SHELLSAGE_CONFIG` > `~/.shellsage/config.json`; project-local config files were deliberately
  skipped — secrets hygiene). Legacy `--agent/--plan/--debug/--doc/--audit/--config/--version`
  keep working (compat guarantee).
- **`--json` envelope:** `{provider, model, content, usage:{prompt,completion,total},
  cost_usd, duration_ms, error?}` → trivially consumable by `jq` in CI.
- **Exit codes:** 0 success, 1 provider/API error, 2 usage error → usable in pipelines.
- **Pipes everywhere:** `go test ./... | shellsage debug -`, `shellsage ask --image err.png "what broke?"`,
  `tail -n200 build.log | shellsage --provider groq ask "summarize failures"`.
- `NO_COLOR`/`--no-color` respected; TTY detection so colors/prompting never corrupt pipes.

## 4. Agent Upgrades

1. **Multi-tool parsing** per step; tolerates ```tool_call, ```json, bare JSON lines,
   and `Action:` / `Action Input:` formats. One assistant message per step (context correct).
2. **Approval gate:** tools classified `read | write | exec | net`. `run_command`,
   `write_file`, `edit_file`, `apply_patch`, `http_request` (non-GET) require approval
   unless `--yes`/non-TTY; interactive prompt `y/a(n)/n` with per-session "allow all".
3. **Context pruning:** tool results older than last 4 steps are truncated in-loop;
   global `MaxTokens` guard (est. chars/4) to stay under model budget.
4. **Budgets:** `--steps` (default 12), `--max-time` (default 5 m wall clock; scheduler-safe).
5. **Final-answer capture** also streams a compact transcript of tool actions (audit trail)
   on request (`--verbose`), exported to `--log-file` for CI records.
6. **New tools:** `find_files` (name patterns, skip node_modules/.git), `apply_patch`
   (multi-hunk edits, `dry_run` first, atomic all-or-nothing write), `git_log`,
   `git_branch`, `http_request` (method/headers/body, JSON pretty, 64 KB cap).

## 5. Chat & Conversation

- **Context budget & `/compress`:** auto-summarize older turns into a single
  `[CONTEXT SUMMARY]` assistant node when estimate exceeds `max_context_tokens`
  (config, default 60k); manual `/compress` anytime.
- **Vision:** `/image <path>` attaches to next message; multimodal parts encoded per
  provider format (OpenAI `image_url` data-URI parts; Anthropic `source.base64` blocks).
- **Sessions:** stored under `~/.shellsage/conversations` (0600); `/rename`,
  `/delete`, `/resume <name|last>`; legacy `./conversations` still readable/listed.
- **Scheduler:** next-due timer (no 100 ms tick); persisted to
  `~/.shellsage/schedule.json` so jobs scheduled from a live REPL survive restarts
  (missed jobs report as `MISSED`, they don't silently run).
- Personas: added **Code Reviewer** and **Test Engineer** built-ins.

## 6. Security & Hardening

- Config & saved sessions written 0600; `Save()` preserves existing key sets and
  masks on display.
- Shell tool: dangerous-command blocklist expanded (fork bomb, `dd of=/dev/*`,
  `mkfs`, `shutdown`, `curl … | sh` pipeline warn-then-deny, `sudo` needs approval).
- `http_request` tool: SSRF guard (blocks link-local/metadata IPs by default,
  `SHELLSAGE_ALLOW_LOCAL_NET=1` for development against local APIs; cloud metadata always blocked).
- SAST: new secret signatures (GitHub fine-grained PAT `github_pat_`, Slack `xox[baprs]-`,
  private key blocks, JWT `eyJ…` + issuer heuristic) — fewer false positives via
  entropy check on generic hex/base64 assignments.

## 7. Testing, CI/CD, Distribution

- New/updated unit tests: registry consistency (unique ids, env prefixes, factory
  round-trip), transport retry (httptest 429→200), tool parser matrix, approvals,
  apply_patch atomicity, scheduler timer, context trim, azure auth header,
  `--json` envelope shape, sessions dir fallback, completion output contains commands.
- CI: runs on `main`, PRs, **`arena/**` & `feat/**` pushes** (this branch included),
  Linux/macOS/Windows matrix, `go vet`, `go test -race`, cross-builds via release workflow (unchanged).
- `Makefile`: `make lint` = vet + gofmt check; `make cover` adds; Docker image now
  non-root user + `HOME` volume hint (real containers run as uid 1000).
- Release workflow: same cross-OS matrix; version injected via ldflags already in
  place — bump constant `Version = "4.0.0"`.

## 8. Documentation & DX

- README: new provider table (20 providers), subcommand reference, scripting recipes
  (`jq`, CI gate example with `--json`, pipe-to-debug), approval-mode security note.
- QUICK_REFERENCE.md, CHANGELOG.md (v4.0.0 entry), `.env.example` regenerated from the
  registry (all prefixes), CONTRIBUTING: "adding a provider in 5 lines" guide.

## 9. Rollout / Verification

1. `gofmt`+`go vet`+`go build`+`go test ./...` green locally (toolchain bootstrapped in CI regardless).
2. Push to `arena/…` branch → CI matrix green on Linux/macOS/Windows.
3. Manual smoke: `shellsage doctor`, `shellsage provider list`, `shellsage ask` (no key → clean
   error, exit 1), completion script sanity.

## 10. Out of Scope (v4.x backlog, recorded for transparency)

- Full-screen TUI (bubbletea) with syntax highlighting — needs new deps.
- MCP server mode / embeddings & RAG memory.
- Windows ConPTY raw-mode input (arrow-key history) — needs syscall layer.
- Provider-native function-calling APIs (tool_choice) — current JSON-block protocol is
  universal; native adapters are a registry flag (`Format: "tools"`) follow-up.

---

**Definition of done for this release:** everything in §2–§8 implemented, tested,
docs updated, pushed to `arena/01a10a74-shellsage`, CI green on all three OSes.
