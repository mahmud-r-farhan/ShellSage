# 📖 ShellSage Quick Reference (Cheatsheet)

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

## 💻 CLI Flags (Non-Interactive)

```bash
shellsage --version               # Print version
shellsage --config                # Launch configuration wizard
shellsage --agent "<goal>"        # Autonomous agent mode
shellsage --plan "<requirement>"  # Plan mode
shellsage --debug "<error>"       # Debug mode
shellsage --doc "<path>"          # Documentation generator
```
