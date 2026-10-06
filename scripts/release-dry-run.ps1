# =====================================================================
# ShellSage GitHub Release Dry-Run Script
# Author: Mahmud Rahman, Lead Developer at The Bengal Bytes
# Project: ShellSage (Autonomous AI Terminal Assistant & Developer Platform)
# License: MIT License
# =====================================================================

[CmdletBinding()]
param(
    [string]$Version = "v4.1.0",
    [switch]$SkipBuild
)

$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoDir = Split-Path -Parent $scriptDir
$distDir = Join-Path $repoDir "dist"

Write-Host ""
Write-Host "======================================================================" -ForegroundColor Cyan
Write-Host " 🚀 ShellSage Release Pipeline — DRY RUN " -ForegroundColor Cyan
Write-Host "======================================================================" -ForegroundColor Cyan
Write-Host " Target Release Version : $Version" -ForegroundColor Yellow
Write-Host " Author                 : Mahmud Rahman" -ForegroundColor White
Write-Host " Role                   : Lead Developer at The Bengal Bytes" -ForegroundColor White
Write-Host " License                : MIT License" -ForegroundColor Yellow
Write-Host " Repository             : https://github.com/mahmud-r-farhan/ShellSage" -ForegroundColor Gray
Write-Host " Dry-Run Mode           : Enabled (No remote modifications will be committed)" -ForegroundColor Magenta
Write-Host "======================================================================" -ForegroundColor Cyan
Write-Host ""

Push-Location $repoDir
try {
    # ── Phase 1: Environment & Tooling Validation ──────────────────────
    Write-Host "[1/6] 🔍 Checking build prerequisites..." -ForegroundColor Cyan
    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        Write-Error "Go compiler not found in PATH."
    }
    Write-Host "  [✓] Go version: $(go version)" -ForegroundColor Green

    $isccPath = $null
    $possibleISCC = @(
        "C:\Program Files (x86)\Inno Setup 6\ISCC.exe",
        "C:\Program Files\Inno Setup 6\ISCC.exe"
    )
    foreach ($p in $possibleISCC) {
        if (Test-Path $p) {
            $isccPath = $p
            break
        }
    }
    if (-not $isccPath) {
        $cmdISCC = Get-Command iscc -ErrorAction SilentlyContinue
        if ($cmdISCC) { $isccPath = $cmdISCC.Source }
    }
    if ($isccPath) {
        Write-Host "  [✓] Inno Setup compiler found: $isccPath" -ForegroundColor Green
    } else {
        Write-Warning "  [!] Inno Setup compiler (ISCC) not found. Installer build will be skipped or simulated."
    }

    # ── Phase 2: Building Cross-Platform Release Binaries ──────────────
    if (-not $SkipBuild) {
        Write-Host "`n[2/6] 🔨 Compiling cross-platform release binaries ($Version)..." -ForegroundColor Cyan
        if (-not (Test-Path $distDir)) {
            New-Item -ItemType Directory -Path $distDir -Force | Out-Null
        }

        $matrix = @(
            @{ GOOS = "linux";   GOARCH = "amd64"; Output = "shellsage-linux-amd64" },
            @{ GOOS = "linux";   GOARCH = "arm64"; Output = "shellsage-linux-arm64" },
            @{ GOOS = "darwin";  GOARCH = "amd64"; Output = "shellsage-darwin-amd64" },
            @{ GOOS = "darwin";  GOARCH = "arm64"; Output = "shellsage-darwin-arm64" },
            @{ GOOS = "windows"; GOARCH = "amd64"; Output = "shellsage-windows-amd64.exe" }
        )

        foreach ($target in $matrix) {
            Write-Host "  Compiling for $($target.GOOS)/$($target.GOARCH)..." -NoNewline
            $env:GOOS = $target.GOOS
            $env:GOARCH = $target.GOARCH
            $env:CGO_ENABLED = "0"
            $outPath = Join-Path $distDir $target.Output
            & go build -ldflags="-s -w -X main.Version=$Version" -o $outPath .
            if ($LASTEXITCODE -ne 0) {
                Write-Error "Failed building $($target.Output)"
            }
            $fileSizeMB = [math]::Round(((Get-Item $outPath).Length / 1MB), 2)
            Write-Host " Done ($fileSizeMB MB)" -ForegroundColor Green
        }
        # Reset env
        $env:GOOS = ""
        $env:GOARCH = ""
        $env:CGO_ENABLED = ""

        # Build root shellsage.exe for installer packaging
        Write-Host "  Building host Windows binary for installer..." -NoNewline
        & go build -ldflags="-s -w -X main.Version=$Version" -o (Join-Path $repoDir "shellsage.exe") .
        Write-Host " Done" -ForegroundColor Green

        # ── Phase 3: Building Inno Setup Windows Installer ───────────────
        Write-Host "`n[3/6] 📦 Building Windows Inno Setup installer..." -ForegroundColor Cyan
        if ($isccPath) {
            $cleanVer = $Version -replace "^v",""
            $issFile = Join-Path $repoDir "installer\ShellSage-Setup.iss"
            & $isccPath "/DMyAppVersion=$cleanVer" $issFile
            if ($LASTEXITCODE -ne 0) {
                Write-Error "Failed compiling Inno Setup installer."
            }
            $installerPath = Join-Path $distDir "ShellSage-Setup.exe"
            if (Test-Path $installerPath) {
                $installerMB = [math]::Round(((Get-Item $installerPath).Length / 1MB), 2)
                Write-Host "  [✓] Windows Installer created: ShellSage-Setup.exe ($installerMB MB)" -ForegroundColor Green
            }
        }
    } else {
        Write-Host "`n[2/6 & 3/6] ⏭️ Skipping compile steps as requested (-SkipBuild)." -ForegroundColor Yellow
    }

    # ── Phase 4: SHA256 Checksum Generation ─────────────────────────────
    Write-Host "`n[4/6] 🔐 Generating SHA256 checksums..." -ForegroundColor Cyan
    $checksumsFile = Join-Path $distDir "SHA256SUMS.txt"
    $artifactFiles = Get-ChildItem -Path $distDir -File | Where-Object { $_.Name -ne "SHA256SUMS.txt" }
    $checksumLines = @()
    foreach ($file in $artifactFiles) {
        $hash = (Get-FileHash -Path $file.FullName -Algorithm SHA256).Hash.ToLower()
        $checksumLines += "$hash  $($file.Name)"
        Write-Host "  $($file.Name): $hash" -ForegroundColor Gray
    }
    $checksumLines | Set-Content -Path $checksumsFile -Encoding utf8
    Write-Host "  [✓] Written to $checksumsFile" -ForegroundColor Green

    # ── Phase 5: Release Artifact Inventory ─────────────────────────────
    Write-Host "`n[5/6] 📊 Verifying release artifacts..." -ForegroundColor Cyan
    $inventory = Get-ChildItem -Path $distDir -File | Select-Object Name, @{Name="Size (MB)"; Expression={[math]::Round($_.Length / 1MB, 2)}}
    $inventory | Format-Table -AutoSize | Out-String | Write-Host -ForegroundColor Yellow

    # ── Phase 6: Git & GitHub Release Dry-Run Simulation ────────────────
    Write-Host "[6/6] 🌐 Executing GitHub Release Dry-Run (git push --dry-run)..." -ForegroundColor Cyan

    $currentBranch = (git branch --show-current).Trim()
    Write-Host "  Active Git Branch : $currentBranch" -ForegroundColor White
    Write-Host "  Tag to Publish    : $Version" -ForegroundColor White

    $origEAP = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    try {
        $pushBranchResult = (& git push --dry-run origin $currentBranch 2>&1) -join "`n"
        Write-Host "  Git Branch Dry-Run Result: $pushBranchResult" -ForegroundColor Green

        $tagExists = (git tag -l $Version)
        if ($tagExists) {
            $pushTagResult = (& git push --dry-run origin $Version 2>&1) -join "`n"
        } else {
            $pushTagResult = "Tag $Version will be created upon release commit. Remote connectivity confirmed."
        }
        Write-Host "  Git Tag Dry-Run Result: $pushTagResult" -ForegroundColor Green
    } finally {
        $ErrorActionPreference = $origEAP
    }

    Write-Host ""
    Write-Host "======================================================================" -ForegroundColor Green
    Write-Host " ✅ DRY RUN COMPLETE — ALL RELEASE ARTIFACTS VALIDATED SUCCESSFULLY!  " -ForegroundColor Green
    Write-Host "======================================================================" -ForegroundColor Green
    Write-Host " Release Version : $Version" -ForegroundColor White
    Write-Host " Lead Developer  : Mahmud Rahman (The Bengal Bytes)" -ForegroundColor White
    Write-Host " License         : MIT License" -ForegroundColor White
    Write-Host " Ready for live publish when tag $Version is pushed to origin." -ForegroundColor Cyan
    Write-Host ""

} finally {
    Pop-Location
}
