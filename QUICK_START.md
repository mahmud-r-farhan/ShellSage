# Quick Reference - ShellSage

## Installation (One-Time Setup)

### Windows
```powershell
# 1. Copy config template
copy .env.example .env

# 2. Edit .env - add your OpenRouter API key
# Get free key: https://openrouter.ai/keys

# 3. Run setup script
.\setup.bat

# 4. Start chatting
.\shellsage.exe
```

### Linux / macOS
```bash
# 1. Copy config template
cp .env.example .env

# 2. Edit .env - add your OpenRouter API key
# Get free key: https://openrouter.ai/keys

# 3. Run setup script
chmod +x setup.sh
./setup.sh

# 4. Start chatting
./shellsage
```

## Daily Usage

### Start the Application
```bash
./shellsage          # Linux/Mac
./shellsage.exe      # Windows
```

### Ask Questions
```
👤 You: Any question here
🤖 Assistant: Smart response from LLM
```

### Available Commands
| Command | What it does |
|---------|-------------|
| `/help` | Show all commands |
| `/clear` | Start fresh conversation |
| `/exit` or `/quit` | Exit app |

### Change LLM Model
Edit `.env`:
```
OPENROUTER_MODEL=openai/gpt-4
```

## Development

### Build Project
```bash
make build          # Using Makefile
go build -o shellsage   # Direct build
```

### Run Project
```bash
make run            # Build and run
./shellsage         # Just run
```

### Code Quality
```bash
make lint           # Check code
make fmt            # Format code
make test           # Run tests
make check          # All checks
```

### Clean Up
```bash
make clean          # Remove build files
```

## Troubleshooting

| Problem | Solution |
|---------|----------|
| "API key not found" | Edit .env with your OpenRouter key |
| "401 error" | Check API key is valid |
| "429 error (rate limit)" | Wait a moment, try again |
| Connection timeout | Check internet, try different model |
| Build fails | Run `go mod tidy` then `go build` |

## File Structure

```
ShellSage/
├── main.go          <- CLI code
├── client.go        <- API client
├── config.go        <- Configuration
├── .env.example     <- Config template
├── README.md        <- Full documentation
├── EXAMPLES.md      <- Usage examples
└── Makefile         <- Dev commands
```


## Environment Variables

| Variable | Example | Default |
|----------|---------|---------|
| `OPENROUTER_API_KEY` | `sk-xxx...` | **Required** |
| `OPENROUTER_MODEL` | `` | `openrouter/free` |
| `OPENROUTER_BASE_URL` | `https://...` | Official endpoint |

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Ctrl+C` | Exit application |
| `Enter` | Send message |
| `Ctrl+L` | Clear screen (depends on terminal) |

## Tips for Better Answers

✅ **Be specific**: "How do I parse JSON in Go?" not "How do JSON?"
✅ **Ask follow-ups**: Conversation history is maintained
✅ **Be clear**: Rephrase if the AI doesn't understand
✅ **Request formats**: Ask for "simple example", "code", "pseudocode", etc.

## Common Questions

**Q: How do I save conversations?**
A: Use `/clear` to start fresh. For persistence, save chat manually or add feature!

**Q: Can I use multiple models?**
A: Change model in `.env` and restart the app.

**Q: Is my data private?**
A: Check OpenRouter's privacy policy: https://openrouter.ai/privacy

**Q: What's the cost?**
A: Free tier available! View pricing: https://openrouter.ai/pricing

## Getting Help

- 📖 Full docs: See README.md
- 💡 Examples: See EXAMPLES.md
- 🐛 Issues: GitHub issues
- 🤝 Contribute: See CONTRIBUTING.md

---

**Start chatting now!** 🚀

```bash
./shellsage
```
