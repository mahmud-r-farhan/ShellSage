# =====================================================================
# ShellSage Windows PowerShell Installer Script
# Author: Mahmud Rahman, Lead Developer at The Bengal Bytes
# Project: ShellSage (Autonomous AI Terminal Assistant & Developer Platform)
# License: MIT License
# =====================================================================

[CmdletBinding()]
param(
    [string]$InstallDir = "$env:LOCALAPPDATA\Programs\ShellSage",
    [switch]$CreateDesktopShortcut,
    [switch]$NoPrompt,
    [switch]$AddToPath
)

$ErrorActionPreference = "Stop"

function Show-Header {
    Write-Host ""
    Write-Host "======================================================================" -ForegroundColor Cyan
    Write-Host " 🚀 ShellSage Installer — Autonomous AI Terminal Assistant & Platform " -ForegroundColor Cyan
    Write-Host "======================================================================" -ForegroundColor Cyan
    Write-Host " Author:  Mahmud Rahman" -ForegroundColor White
    Write-Host " Role:    Lead Developer at The Bengal Bytes" -ForegroundColor White
    Write-Host " License: MIT License" -ForegroundColor Yellow
    Write-Host " Source:  https://github.com/mahmud-r-farhan/ShellSage" -ForegroundColor Gray
    Write-Host "======================================================================" -ForegroundColor Cyan
    Write-Host ""
}

function Show-Information {
    Write-Host "📋 Overview & Capabilities:" -ForegroundColor Green
    Write-Host " • Universal Multi-Provider Engine: 20 AI providers (Groq, OpenRouter, OpenAI, Claude, Gemini, DeepSeek...)"
    Write-Host " • Autonomous ReAct Agent: File edits, shell commands, git operations, web tools with approval gates"
    Write-Host " • Defensive Security Suite: SAST secret scanner, SSL cert validator, HTTP header auditor"
    Write-Host " • Developer Modes: /plan (architecture), /debug (root-cause), /doc (generator)"
    Write-Host " • Productivity: Conversation trees (/branch), token compression (/compress), task queues (/queue)"
    Write-Host ""
    Write-Host "📄 License Terms:" -ForegroundColor Yellow
    Write-Host " Distributed under the MIT License."
    Write-Host " Copyright (c) 2026 Mahmud Rahman, Lead Developer at The Bengal Bytes."
    Write-Host ""
}

function Create-DesktopShortcut([string]$TargetExePath, [string]$IconPath) {
    try {
        $desktopPath = [Environment]::GetFolderPath([Environment+SpecialFolder]::Desktop)
        $shortcutPath = Join-Path $desktopPath "ShellSage.lnk"

        $wshShell = New-Object -ComObject WScript.Shell
        $shortcut = $wshShell.CreateShortcut($shortcutPath)
        $shortcut.TargetPath = $TargetExePath
        $shortcut.WorkingDirectory = [Environment]::GetFolderPath([Environment+SpecialFolder]::UserProfile)
        $shortcut.Description = "Launch ShellSage Autonomous AI Terminal Assistant"
        if ($IconPath -and (Test-Path $IconPath)) {
            $shortcut.IconLocation = "$IconPath,0"
        }
        $shortcut.Save()
        Write-Host "✅ Desktop shortcut created: $shortcutPath" -ForegroundColor Green
        return $shortcutPath
    } catch {
        Write-Warning "Failed to create desktop shortcut: $_"
        return $null
    }
}

function Add-DirectoryToUserPath([string]$DirPath) {
    try {
        $userPath = [Environment]::GetEnvironmentVariable("Path", [EnvironmentVariableTarget]::User)
        $paths = $userPath -split ";"
        if ($paths -notcontains $DirPath) {
            $newPath = ($paths + $DirPath) -join ";"
            [Environment]::SetEnvironmentVariable("Path", $newPath, [EnvironmentVariableTarget]::User)
            $env:Path = "$env:Path;$DirPath"
            Write-Host "✅ Added $DirPath to User PATH environment variable." -ForegroundColor Green
        } else {
            Write-Host "ℹ️  $DirPath is already in User PATH." -ForegroundColor Gray
        }
    } catch {
        Write-Warning "Could not update User PATH: $_"
    }
}

Show-Header
Show-Information

# 1. Locate source executable
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoDir = Split-Path -Parent $scriptDir

$sourceExe = $null
$possibleLocations = @(
    (Join-Path $repoDir "dist\shellsage-windows-amd64.exe"),
    (Join-Path $repoDir "shellsage.exe"),
    (Join-Path $repoDir "dist\shellsage.exe")
)

foreach ($loc in $possibleLocations) {
    if (Test-Path $loc) {
        $sourceExe = $loc
        break
    }
}

if (-not $sourceExe) {
    Write-Host "🔨 ShellSage binary not found in dist/. Compiling via 'go build'..." -ForegroundColor Cyan
    if (Get-Command go -ErrorAction SilentlyContinue) {
        Push-Location $repoDir
        try {
            & go build -ldflags="-s -w" -o (Join-Path $repoDir "shellsage.exe") .
            $sourceExe = Join-Path $repoDir "shellsage.exe"
            Write-Host "✅ Built binary: $sourceExe" -ForegroundColor Green
        } finally {
            Pop-Location
        }
    } else {
        Write-Error "Go compiler is not installed and pre-built shellsage.exe was not found."
        exit 1
    }
}

# 2. Checkmark option for desktop shortcut
$installShortcut = $CreateDesktopShortcut
if (-not $NoPrompt -and -not $PSBoundParameters.ContainsKey('CreateDesktopShortcut')) {
    $resp = Read-Host "Create a desktop shortcut? (Checkmark option) [Y/n]"
    if ($resp -eq "" -or $resp -match "^[yY]") {
        $installShortcut = $true
    } else {
        $installShortcut = $false
    }
}

# 3. Create destination directory
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

# 4. Copy files
$targetExe = Join-Path $InstallDir "shellsage.exe"
Copy-Item -Path $sourceExe -Destination $targetExe -Force
Write-Host "✅ Copied binary to $targetExe" -ForegroundColor Green

$iconSrc = Join-Path $repoDir "installer\shellsage.ico"
$iconDest = Join-Path $InstallDir "shellsage.ico"
if (Test-Path $iconSrc) {
    Copy-Item -Path $iconSrc -Destination $iconDest -Force
}

$licenseSrc = Join-Path $repoDir "LICENSE"
if (Test-Path $licenseSrc) {
    Copy-Item -Path $licenseSrc -Destination (Join-Path $InstallDir "LICENSE") -Force
}

# 5. Add to PATH
Add-DirectoryToUserPath $InstallDir

# 6. Create Desktop Shortcut if checked
if ($installShortcut) {
    Create-DesktopShortcut -TargetExePath $targetExe -IconPath $iconDest
}

Write-Host ""
Write-Host "🎉 Installation finished successfully!" -ForegroundColor Green
Write-Host " • Run 'shellsage' in any terminal window"
Write-Host " • Double-click the Desktop shortcut to launch the interactive CLI"
Write-Host " • Run 'shellsage doctor' to inspect your environment"
Write-Host ""

if (-not $NoPrompt) {
    $launch = Read-Host "Finish install and launch ShellSage CLI now? [Y/n]"
    if ($launch -eq "" -or $launch -match "^[yY]") {
        Write-Host "🚀 Launching ShellSage..." -ForegroundColor Cyan
        Start-Process $targetExe
    }
}
