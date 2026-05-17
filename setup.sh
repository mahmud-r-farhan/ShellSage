#!/bin/bash

# ShellSage Setup Script
# This script helps you set up ShellSage quickly

set -e

echo "🚀 ShellSage Setup"
echo "=================="
echo ""

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.24+ from https://golang.org"
    exit 1
fi

echo "✅ Go is installed: $(go version)"
echo ""

# Check if .env file exists
if [ ! -f .env ]; then
    echo "📝 Creating .env file from .env.example..."
    cp .env.example .env
    echo "✅ .env file created!"
    echo ""
    echo "⚠️  Please edit .env and add your OpenRouter API key"
    echo "   Get one for free at: https://openrouter.ai/keys"
    echo ""
    read -p "Press Enter after you've updated .env..."
fi

# Download dependencies
echo ""
echo "📦 Downloading dependencies..."
go mod download
go mod tidy
echo "✅ Dependencies installed!"

# Build the project
echo ""
echo "🔨 Building ShellSage..."
go build -o shellsage
echo "✅ Build complete!"

echo ""
echo "🎉 Setup finished!"
echo ""
echo "📖 Next steps:"
echo "   1. Make sure OPENROUTER_API_KEY is set in your .env file"
echo "   2. Run: ./shellsage"
echo "   3. Type /help for available commands"
echo ""
echo "Happy chatting! 💬"
