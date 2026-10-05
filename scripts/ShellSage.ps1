# ShellSage PowerShell Helper Module
# Provides Windows / PowerShell environment validation, provider helper functions,
# completion setup, and CLI wrappers.

<#
.SYNOPSIS
    Initializes environment variables, alias, and PowerShell autocomplete for ShellSage.
.EXAMPLE
    Import-Module .\scripts\ShellSage.ps1
    Set-ShellSageProvider -Provider "groq"
    Invoke-ShellSageAsk "Explain Go concurrency"
#>

function Set-ShellSageProvider {
    [CmdletBinding()]
    param (
        [Parameter(Mandatory=$true)]
        [ValidateSet("groq", "openrouter", "openai", "anthropic", "gemini", "deepseek", "ollama", "github")]
        [string]$Provider,

        [string]$ApiKey,
        [string]$Model
    )

    $envVarMap = @{
        "groq"       = "GROQ_API_KEY"
        "openrouter" = "OPENROUTER_API_KEY"
        "openai"     = "OPENAI_API_KEY"
        "anthropic"  = "ANTHROPIC_API_KEY"
        "gemini"     = "GEMINI_API_KEY"
        "deepseek"   = "DEEPSEEK_API_KEY"
        "github"     = "GITHUB_TOKEN"
    }

    if ($ApiKey -and $envVarMap.ContainsKey($Provider)) {
        [Environment]::SetEnvironmentVariable($envVarMap[$Provider], $ApiKey, [System.EnvironmentVariableTarget]::Process)
        Write-Host "Set environment variable $($envVarMap[$Provider])" -ForegroundColor Green
    }

    if (Get-Command "shellsage" -ErrorAction SilentlyContinue) {
        if ($Model) {
            shellsage config set "$Provider.model=$Model" | Out-Null
        }
        shellsage provider use $Provider
    } else {
        Write-Host "shellsage binary not found in PATH. Ensure 'make build' or 'go build' was run." -ForegroundColor Yellow
    }
}

function Invoke-ShellSageAsk {
    [CmdletBinding()]
    param (
        [Parameter(Mandatory=$true, Position=0)]
        [string]$Prompt,

        [string]$Provider,
        [string]$Model,
        [switch]$Json
    )

    $argsList = @("ask", $Prompt)
    if ($Provider) { $argsList += @("--provider", $Provider) }
    if ($Model)    { $argsList += @("--model", $Model) }
    if ($Json)     { $argsList += "--json" }

    & shellsage @argsList
}

function Test-ShellSageEnvironment {
    [CmdletBinding()]
    param ()

    Write-Host "=== ShellSage PowerShell Environment Audit ===" -ForegroundColor Cyan

    $keys = @("GROQ_API_KEY", "OPENROUTER_API_KEY", "OPENAI_API_KEY", "ANTHROPIC_API_KEY", "GEMINI_API_KEY", "DEEPSEEK_API_KEY", "GITHUB_TOKEN")
    foreach ($k in $keys) {
        $val = [Environment]::GetEnvironmentVariable($k)
        if ($val) {
            $masked = $val.Substring(0, [System.Math]::Min(6, $val.Length)) + "..."
            Write-Host "  [✓] $k = $masked" -ForegroundColor Green
        } else {
            Write-Host "  [ ] $k is unset" -ForegroundColor DarkGray
        }
    }

    if (Get-Command "shellsage" -ErrorAction SilentlyContinue) {
        Write-Host "`nRunning 'shellsage doctor'..." -ForegroundColor Cyan
        shellsage doctor
    } else {
        Write-Host "`n[!] 'shellsage' binary is not currently in PATH." -ForegroundColor Yellow
    }
}

# Auto-register PowerShell completion for shellsage if binary is present
if (Get-Command "shellsage" -ErrorAction SilentlyContinue) {
    Register-ArgumentCompleter -Native -CommandName shellsage -ScriptBlock {
        param($wordToComplete, $commandAst, $cursorPosition)
        $subcommands = @("chat", "ask", "agent", "plan", "debug", "doc", "audit", "sec", "config", "provider", "models", "sessions", "doctor", "completion", "version", "help")
        $subcommands | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
        }
    }
}

Export-ModuleMember -Function Set-ShellSageProvider, Invoke-ShellSageAsk, Test-ShellSageEnvironment
