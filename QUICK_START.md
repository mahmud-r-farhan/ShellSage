# 🚀 ShellSage Quick Start Guide

Get started with **ShellSage** in less than 2 minutes!

---

## 📥 1. Build and Run

```bash
# Clone the repository
git clone https://github.com/mahmud-r-farhan/ShellSage.git
cd ShellSage

# Build binary
make build

# Run ShellSage
./shellsage
```

---

## ⚙️ 2. Configure Your Provider

When you launch ShellSage for the first time, run the interactive setup wizard:
```bash
/config
```
1. Select your preferred provider (OpenRouter, OpenAI, Claude, Gemini, Groq, DeepSeek, Ollama, etc.).
2. Enter your API key (if required).
3. Select your model.
4. Settings are automatically saved to `~/.shellsage/config.json`.

---

## 💡 3. Five Powerful Workflows to Try

### 1. Architectural Planning Mode
```
/plan Build a resilient distributed caching layer in Go with Redis fallback
```

### 2. Autonomous Agent (Search & Coding)
```
/agent Search for latest Go 1.24 features and summarize top 5 enhancements
```

### 3. Diagnose an Error Log
```
/debug panic: runtime error: index out of range [3] with length 2 in main.go:45
```

### 4. Copy Code to Clipboard & Export
```
/copy code
/export pdf my_chat_summary.pdf
```

### 5. Schedule Automated Work at Local Time
```
/schedule at 16:30 Run go test ./... and summarize results
```

---

## ❓ Need Help?
Type `/help` inside the CLI at any time!

## ⚡ 60-Second v4 Tour

```bash
shellsage doctor                      # what's configured, what works, what's broken
shellsage provider list               # 20 providers, their key env vars, active pick
shellsage provider use groq           # or: config set groq.api_key gsk_...
shellsage models list --remote        # live model catalog from the provider
shellsage ask "one-line question"     # pipes in, --json out
shellsage agent "small task"          # mutating tools ask for approval (--yes in CI)
```
