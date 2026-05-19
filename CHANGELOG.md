# Changelog

All notable changes to ShellSage are documented in this file.

## [2.0.0] - 2026-05-19

### ✨ Major Enhancements

#### Interactive Features
- **🎭 AI Personas** - 6 pre-built personas to tailor AI responses:
  - General Assistant (default)
  - Expert Developer
  - Creative Writer
  - Patient Teacher
  - Data Analyst
  - Debug Assistant
- **🌡️ Temperature Control** - Dynamically adjust creativity level (0.0-1.0)
- **🔀 Dynamic Model Selection** - Switch between available OpenRouter models
- **🔍 Conversation Search** - Find specific topics in your chat history

#### Persistence & Management
- **💾 Save Conversations** - Export chats to JSON format
- **📂 Load Conversations** - Resume previous conversations
- **📋 List Saved Chats** - View all saved conversation files
- **📊 Session Statistics** - Track duration, message count, tokens used

#### Real-time Monitoring
- **📈 Token Usage Display** - See token counts after each response
- **⏱️ Session Duration Tracking** - Monitor conversation length
- **🔢 Message Counter** - Keep track of message count
- **📜 Message History Viewer** - Review last 10 messages easily

#### UI/UX Improvements
- Enhanced command help system with `/help`
- Better error messages and user feedback
- Improved banner with current model display
- Color-coded token usage statistics
- Interactive menus for model and persona selection

### 🔧 Technical Improvements

#### API & Client
- New `ChatWithUsage()` method to track token consumption
- New `ChatWithOptions()` method for customizable parameters
- Support for system prompts in messages
- Better temperature parameter handling

#### Configuration
- Updated default model to `meta-llama/llama-2-70b-chat` (better than free tier)
- Config file improvements for extensibility

#### Architecture
- **New file: interactive.go** - Contains all interactive menu functions
- **Modular design** - Personas, commands, and utilities separated logically
- **Better state management** - `ConversationState` struct for organizing session data

### 📚 New Files
- `interactive.go` - Interactive menu and utility functions
- `ENHANCED_FEATURES.md` - Comprehensive guide to v2.0 features
- `CHANGELOG.md` - This file

### 🔄 Changed Commands

| Old | New | Change |
|-----|-----|--------|
| `/help` | `/help` | Now shows all 14+ commands with descriptions |
| `/clear` | `/clear` | Now resets token tracking and start time |
| N/A | `/model` | New command for model selection |
| N/A | `/persona` | New command for persona selection |
| N/A | `/temp` | New command for temperature adjustment |
| N/A | `/save` | New command for saving conversations |
| N/A | `/load` | New command for loading conversations |
| N/A | `/list` | New command for listing saved chats |
| N/A | `/search` | New command for searching history |
| N/A | `/stats` | New command for statistics |
| N/A | `/copy` | New command for copying responses |
| N/A | `/history` | New command for viewing message history |

### 🐛 Fixes
- Better error handling for API failures
- Improved input validation
- Fixed command parsing for multi-word arguments

### 📖 Documentation
- Updated README with new features section
- Created ENHANCED_FEATURES.md with detailed guides
- Added command reference table
- Updated roadmap showing completed features

---

## [1.0.0] - 2026-05-18

### Initial Release
- Basic CLI chat interface
- OpenRouter API integration
- Conversation history support
- Simple command system (/exit, /clear, /help)
- Configuration via .env file
- Multi-model support
- Colored output for better UX

---

## Future Roadmap

### v2.1
- Clipboard integration for `/copy` command
- Export to Markdown/PDF
- Custom persona creation

### v3.0
- TUI/Interactive menu system
- Docker containerization
- Web interface option

---

## Version History Summary

| Version | Date | Focus |
|---------|------|-------|
| 2.0.0 | 2026-05-19 | Interactive features, personas, persistence |
| 1.0.0 | 2026-05-18 | Core CLI chat functionality |

---

For more information, see:
- [README.md](README.md)
- [ENHANCED_FEATURES.md](ENHANCED_FEATURES.md)
- [GitHub Issues](https://github.com/mahmud-r-farhan/ShellSage/issues)
