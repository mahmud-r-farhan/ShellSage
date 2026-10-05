# 🚀 ShellSage Quick Start Guide

Get started with **ShellSage** in less than 2 minutes!

---

## 📥 1. Build and Run

```bash
# Clone the repository
git clone https://github.com/mahmud-r-farhan/ShellSage.git
cd ShellSage

# Build binary using standard Go layout
make build

# Run ShellSage
./shellsage
```

---

## ⚙️ 2. Configure Your Provider

When you launch ShellSage for the first time, run the interactive setup wizard or CLI command:
```bash
shellsage config wizard
# OR inside the chat session:
/config
```
1. Select your preferred provider (Groq, OpenRouter, OpenAI, Claude, Gemini, DeepSeek, Ollama, etc.).
2. Enter your API key (if required).
3. Select your model.
4. Settings are automatically saved to `~/.shellsage/config.json` with secure `0600` permissions.

---

## 💡 3. Powerful Workflows to Try

### 1. Fast Groq Inference / One-Shot Chat
```bash
shellsage ask "Explain Go 1.24 range over func" --provider groq
```

### 2. Architectural Planning Mode
```
/plan Build a resilient distributed caching layer in Go with Redis fallback
```

### 3. Autonomous Agent (Search & Coding)
```
/agent Search for latest Go 1.24 features and summarize top 5 enhancements
```

### 4. Diagnose an Error Log
```bash
cat build.log | shellsage debug - --provider groq
```

### 5. Copy Code to Clipboard & Export
```
/copy code
/export pdf my_chat_summary.pdf
```

### 6. Schedule Automated Work at Local Time
```
/schedule at 16:30 Run go test ./... and summarize results
```

---

## ⚡ 60-Second ShellSage Tour

```bash
shellsage doctor                      # environment diagnostics
shellsage provider list               # 20 providers & key env settings
shellsage provider use groq           # set active provider
shellsage models list --remote        # live model catalog
shellsage ask "one-line question"     # pipe in, JSON out
shellsage agent "small task"          # autonomous agent with approval gate
```

## ❓ Need Help?
Type `/help` inside the CLI or run `shellsage help` at any time!
