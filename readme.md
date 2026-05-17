# 🚀 ShellSage - AI Chat CLI

A beautiful, fast, and production-ready CLI chat application powered by **OpenRouter API** and cutting-edge LLMs. Built with Go following best practices.

![Go](https://img.shields.io/badge/Go-1.24-blue?style=flat-square&logo=go)
![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)

## ✨ Features

- 🎨 **Beautiful CLI Interface** - Stunning colored output and intuitive design
- ⚡ **Fast & Responsive** - Built with Go for speed and efficiency
- 🔄 **Conversation History** - Maintains context across multiple messages
- 🎯 **Multi-Model Support** - Use any LLM available on OpenRouter
- 🛡️ **Error Handling** - Robust error handling and helpful messages
- 📦 **Easy Setup** - Simple configuration with `.env` files
- 🔧 **Best Practices** - Clean code, modular structure, well-documented

## 📋 Requirements

- **Go 1.24+** or higher
- **OpenRouter API Key** (free tier available at https://openrouter.ai)

## 🚀 Quick Start

### 1. Clone the Repository

```bash
git clone https://github.com/mahmud-r-farhan/ShellSage
cd shellsage
```

### 2. Set Up Environment Variables

Create a `.env` file in the project root (copy from `.env.example`):

```bash
cp .env.example .env
```

Edit `.env` and add your OpenRouter API key:

```env
OPENROUTER_API_KEY=your_api_key_here
OPENROUTER_MODEL=
```

Get your free API key from: https://openrouter.ai/keys

### 3. Download Dependencies

```bash
go mod download
go mod tidy
```

### 4. Build and Run

**Option A: Direct Run**
```bash
go run .
```

**Option B: Build Binary**
```bash
go build -o shellsage
./shellsage
```

**Option C: Install Globally**
```bash
go install
shellsage
```

## 💬 Usage

Once the CLI starts, you'll see a welcome banner. Start typing your questions or messages!

### Available Commands

- `/exit` or `/quit` - Exit the chat
- `/clear` - Clear conversation history
- `/help` - Show help message
- Type normally - Send a message to the LLM

### Example Interaction

```
👤 You: Hello! What is Go?
🤖 Assistant: Go is a statically typed, compiled programming language created by Google...

👤 You: Tell me more about its concurrency model
🤖 Assistant: Go's concurrency model is based on goroutines and channels...

👤 You: /exit
✨ Goodbye! Thanks for chatting.
```

## 🎯 Supported Models

ShellSage works with all models available on OpenRouter. Popular choices:


View full list: https://openrouter.ai/docs/models

To use a different model, update `.env`:
```env
OPENROUTER_MODEL=xai/grok-4.1
```

## 📁 Project Structure

```
shellsage/
├── main.go           # CLI entry point and main loop
├── client.go         # OpenRouter API client
├── config.go         # Configuration management
├── go.mod            # Go module definition
├── go.sum            # Dependency checksums
├── .env.example      # Example environment configuration
├── README.md         # This file
└── .gitignore        # Git ignore rules
```

## 🔧 Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `OPENROUTER_API_KEY` | Required | Your OpenRouter API key |
| `OPENROUTER_MODEL` | `xai/grok-4.1` | LLM model to use |
| `OPENROUTER_BASE_URL` | `https://openrouter.ai/api/v1` | OpenRouter API endpoint |

## 🛠️ Development

### Building from Source

```bash
# Clone the repository
git clone https://github.com/mahmud-r-farhan/ShellSage
cd shellsage

# Download dependencies
go mod download

# Build
go build -o shellsage

# Run
./shellsage
```

### Code Quality

The codebase follows:
- Go best practices and conventions
- Proper error handling
- Clear code structure and comments
- Modular design for maintainability

## 🐛 Troubleshooting

### "OPENROUTER_API_KEY not found"
- Make sure you created a `.env` file
- Check that the API key is correctly set
- Ensure the `.env` file is in the same directory as the binary

### "API error (status 401)"
- Your API key is invalid or expired
- Generate a new key from https://openrouter.ai/keys

### "API error (status 429)"
- You've exceeded your rate limit
- Wait a few moments before making another request

### Connection issues
- Check your internet connection
- Verify OpenRouter API is accessible
- Try a different model or check OpenRouter status page

## 🔐 Security Best Practices

1. **Never commit `.env`** - Always use `.env.example` as a template
2. **Use strong API keys** - Keep your OpenRouter key private
3. **Environment variables** - For production, use secure secret management
4. **Input validation** - The app validates and sanitizes all inputs

Example `.gitignore`:
```
.env
.env.local
*.log
*.db
```

## 📝 License

MIT License - see LICENSE file for details

## 🤝 Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit changes (`git commit -m 'Add AmazingFeature'`)
4. Push to branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📧 Support

- Open issues on GitHub for bug reports
- Check existing issues before creating new ones
- Provide detailed error messages and steps to reproduce

## 🙏 Acknowledgments

- [OpenRouter](https://openrouter.ai) - For providing unified LLM API
- [Fatih Akin](https://github.com/fatih/color) - For color package
- [Joho](https://github.com/joho/godotenv) - For .env file support

## 🚀 Roadmap

- [ ] Streaming responses for real-time output
- [ ] Custom system prompts
- [ ] Conversation persistence to file
- [ ] Multi-conversation support
- [ ] Token usage tracking and display
- [ ] Interactive model selection menu
- [ ] Configuration UI
- [ ] Docker support

---

**Made with ❤️ for the open source community**

Happy chatting! 🎉
