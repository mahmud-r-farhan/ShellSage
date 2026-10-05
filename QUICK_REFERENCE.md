# 📖 ShellSage v4.0 Quick Reference (Cheatsheet)

## ⌨️ Command Cheatsheet

### Core Commands
- `/help` - Show command reference
- `/config` - Launch configuration & provider setup wizard
- `/provider` - Switch active provider (OpenRouter, OpenAI, Claude, Gemini, Groq, Ollama...)
- `/model` - Switch active model for current provider
- `/persona` - Switch active persona or create a new custom persona (`/persona create`)
- `/temp` - Adjust temperature / creativity level (0.0 - 1.0)
- `/clear` - Clear conversation memory & start fresh tree
- `/exit` or `/quit` - Save session and exit

### Clipboard & Export
- `/copy` - Copy last assistant response to clipboard
- `/copy code` - Extract and copy only code snippets from last response
- `/copy all` - Copy complete conversation transcript
- `/export md [filename]` - Export conversation to Markdown
- `/export pdf [filename]` - Export conversation to PDF document
- `/export html [filename]` - Export conversation to HTML
- `/export json [filename]` - Export conversation to JSON

### Branching & Conversation Tree
- `/retry` or `/alt` - Regenerate alternative response for last user turn
- `/branch` - List and switch between conversation branches
- `/tree` - Render visual ASCII conversation tree

### Autonomous Agent & Developer Tools
- `/agent <goal>` - Execute multi-step goal autonomously using tools
- `/plan <requirement>` - Generate structured architectural plan
- `/debug <error>` - Deep root-cause diagnostic & fix generation
- `/doc <file>` - Generate documentation / README for code
- `/search <text>` - Search in conversation history
- `/stats` or `/analytics` - View token usage and estimated API cost

### Task Queue & Local Time Scheduler
- `/queue add <task>` - Add task to background execution queue
- `/queue list` - List tasks in queue
- `/queue run` - Execute next task in queue
- `/schedule at <HH:MM> <goal>` - Schedule work at specific local machine time
- `/schedule in <duration> <goal>` - Schedule work after duration (e.g. 10m, 1h)
- `/schedule list` - List scheduled jobs

### Sessions
- `/save [filename]` - Save conversation session to JSON
- `/load [filename]` - Load previously saved session
- `/list` - List all saved sessions
- `/history [N]` - Show last N messages

---

## 💻 CLI Subcommands (Non-Interactive)

```bash
shellsage ask "<prompt>"           # one-shot chat; '-' or a pipe reads stdin
shellsage agent "<goal>"           # autonomous run (approval gate; --yes to auto-allow)
shellsage plan "<requirement>"     # architecture plan
shellsage debug "<error|stack>"    # root-cause diagnosis (pastes fine)
shellsage doc "<path>"             # markdown docs for a file
shellsage audit [path|url]        # combined security posture audit
shellsage sec <headers|ssl|ports|sast|audit> <target> [--json]
shellsage provider list|use <id>|show
shellsage models list [--remote] | use <id>
shellsage config show|path|set <provider>.<field>=<value>
shellsage sessions list|delete|rename|load
shellsage doctor                   # env health (exit code reflects problems)
shellsage completion <bash|zsh|fish>
# pipes & scripting
go test ./... 2>&1 | shellsage ask - --json | jq -r .content
shellsage agent --yes --log-file run.jsonl "bump CI actions to latest"
# legacy v3 flags still work: --agent/--plan/--debug/--doc/--audit/--config/--version
```

## 🔌 Providers (20) — all configured as `<PREFIX>_API_KEY`
openrouter · openai · anthropic · gemini · groq · deepseek · mistral · xai · together ·
fireworks · cerebras · nvidia · github · huggingface · perplexity · deepinfra · cohere ·
azure · ollama · custom — plus `<PREFIX>_MODEL` / `<PREFIX>_BASE_URL` overrides.

## 🆕 v4 REPL additions
- `/compress` — fold old turns into an AI summary (context budget: `max_context_tokens`)
- `/image <file|url>` — attach screenshot/diagram to your next message (vision)
- `/approve <ask|yolo|read-only>` — agent autonomy dial
- `/tools` — tool inventory with risk classes · `/models` — quick list
- `/resume` · `/rename` · `/delete` — sessions in `~/.shellsage/conversations`
- `/schedule daily 09:00 <goal>` · `/schedule cancel <id>` — persisted timer jobs
