# ⚡ Quick Reference Card

## 🎯 Most Used Commands

### Start Interactive Setup
```
/persona    # Choose your AI personality
/temp       # Set creativity level
/model      # Pick your LLM
```

### Chat & Manage
```
/save       # Save this conversation
/stats      # See token usage & time
/clear      # Start fresh
```

### Review & Organize
```
/history    # Show last 10 messages
/search foo # Find "foo" in history
/list       # See all saved chats
/load chat  # Restore old conversation
```

---

## 🎭 Persona Quick Guide

| Persona | Best For | Temp |
|---------|----------|------|
| **Developer** | Code & technical | 0.3 |
| **Writer** | Stories & ideas | 0.8 |
| **Teacher** | Learning | 0.5 |
| **Analyst** | Data & insights | 0.4 |
| **Debug** | Fixing issues | 0.2 |

---

## 🌡️ Temperature Quick Guide

| Temp | Usage | Example |
|------|-------|---------|
| 0.0 | Deterministic | `What is 2+2?` |
| 0.3 | Technical | Code reviews |
| 0.5 | Balanced | General questions |
| 0.8 | Creative | Brainstorming |
| 1.0 | Very random | Storytelling |

---

## 📊 Reading Token Display

```
[Tokens: +234 prompt, +512 completion | Total: 746]
         ↑              ↑                 ↑
    Input tokens    Output tokens   Session total
```

---

## 💡 Pro Tips

**Tip 1: Debug Mode**
```
/persona              # Select "Debug Assistant"
/temp                 # Set to 0.2
Paste your error...   # Get focused help
```

**Tip 2: Brainstorm Sessions**
```
/persona              # Select "Creative Writer"
/temp                 # Set to 0.8+
Generate ideas...     # More creative responses
```

**Tip 3: Work Smart**
```
/model                # Try different models
/search keyword       # Find previous solutions
/stats                # Monitor token usage
/save                 # Archive completed work
```

**Tip 4: Resume Work**
```
/list                 # Find old session
/load chat_name       # Bring it back
Continue where you left off...
```

---

## ❌ Common Issues

**Q: I don't see token counts**
A: Make sure you're on v2.0+. Check with `/stats`

**Q: How do I get cheaper responses?**
A: Use `/model` and select `openrouter/free`

**Q: Responses too random?**
A: Use `/temp` and set a lower value (0.2-0.4)

**Q: Can't find old chat?**
A: Use `/list` to see all saved conversations

---

## 🚀 Getting Started in 30 Seconds

```bash
# 1. Set up (first time only)
export OPENROUTER_API_KEY="your-key"

# 2. Start
go run .

# 3. Try this:
/persona           # Pick Developer
/temp              # Set to 0.5
What is Go?        # Ask something
/save              # Save the chat
/exit              # Done!
```

---

**ShellSage v2.0** - Made for productive AI conversations! 🚀
