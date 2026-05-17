@echo off
REM ShellSage Setup Script for Windows

echo.
echo 🚀 ShellSage Setup
echo ==================
echo.

REM Check if Go is installed
go version >nul 2>&1
if errorlevel 1 (
    echo ❌ Go is not installed. Please install Go 1.24+ from https://golang.org
    pause
    exit /b 1
)

echo ✅ Go is installed:
go version
echo.

REM Check if .env file exists
if not exist .env (
    echo 📝 Creating .env file from .env.example...
    copy .env.example .env
    echo ✅ .env file created!
    echo.
    echo ⚠️  Please edit .env and add your OpenRouter API key
    echo    Get one for free at: https://openrouter.ai/keys
    echo.
    pause
)

REM Download dependencies
echo.
echo 📦 Downloading dependencies...
call go mod download
call go mod tidy
echo ✅ Dependencies installed!

REM Build the project
echo.
echo 🔨 Building ShellSage...
call go build -o shellsage.exe
echo ✅ Build complete!

echo.
echo 🎉 Setup finished!
echo.
echo 📖 Next steps:
echo    1. Make sure OPENROUTER_API_KEY is set in your .env file
echo    2. Run: shellsage.exe
echo    3. Type /help for available commands
echo.
echo Happy chatting! 💬
echo.
pause
