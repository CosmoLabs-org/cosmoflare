# R2Go2 Windows Installation Script
# Copyright © 2025 CosmoLabs (https://cosmolabs.org)
# License: MIT

param(
    [switch]$Help,
    [switch]$Version
)

# Configuration
$Repo = "CosmoLabs-org/cosmoflare"
$BinaryName = "cosmoflare"
$InstallDir = "$env:USERPROFILE\.local\bin"
$ChecksumsFile = "checksums-sha256.txt"

# Emoji for beautiful output
$Rocket = "🚀"
$Check = "✅"
$Warning = "⚠️"
$Error = "❌"
$Info = "ℹ️"
$Gear = "⚙️"
$Download = "📥"
$Install = "📦"

# Color functions
# Uses Write-Host (display stream) instead of Write-Output so status
# messages never pollute function return values.
function Write-ColorOutput($ForegroundColor) {
    $fc = $host.UI.RawUI.ForegroundColor
    $host.UI.RawUI.ForegroundColor = $ForegroundColor
    if ($args) {
        Write-Host $args
    }
    $host.UI.RawUI.ForegroundColor = $fc
}

# Print functions
function Write-Header {
    Write-ColorOutput Cyan "$Rocket R2Go2 Windows Installation Script"
    Write-ColorOutput Cyan "======================================"
    Write-Output ""
}

function Write-Success($message) {
    Write-ColorOutput Green "$Check $message"
}

function Write-Error($message) {
    Write-ColorOutput Red "$Error $message"
}

function Write-Warning($message) {
    Write-ColorOutput Yellow "$Warning $message"
}

function Write-Info($message) {
    Write-ColorOutput Blue "$Info $message"
}

function Write-Step($message) {
    Write-ColorOutput Magenta "$Gear $message"
}

# Detect Windows architecture
function Get-WindowsArch {
    $arch = $env:PROCESSOR_ARCHITECTURE.ToLower()
    switch ($arch) {
        "amd64" { return "amd64" }
        "arm64" { return "arm64" }
        default {
            Write-Error "Unsupported architecture: $arch"
            exit 1
        }
    }
}

# Get latest release version
function Get-LatestVersion {
    Write-Step "Fetching latest release version..."

    try {
        $response = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
        $version = $response.tag_name -replace '^v', ''
        Write-Success "Latest version: $version"
        return $version
    }
    catch {
        Write-Error "Failed to fetch latest version: $_"
        exit 1
    }
}

# Download the checksums file published alongside the release assets
# (fail-closed: abort when it is unavailable — never install unverified)
function Get-ChecksumsFile($version) {
    Write-Step "Downloading checksums file..."

    $checksumsUrl = "https://github.com/$Repo/releases/download/v$version/$ChecksumsFile"

    try {
        Invoke-WebRequest -Uri $checksumsUrl -OutFile $ChecksumsFile
    }
    catch {
        Write-Error "Checksums file unavailable: $checksumsUrl"
        Write-Warning "Refusing to install an unverified binary (fail-closed policy)."
        exit 1
    }

    if (!(Test-Path $ChecksumsFile) -or (Get-Item $ChecksumsFile).Length -eq 0) {
        Write-Error "Checksums file is missing or empty: $checksumsUrl"
        Write-Warning "Refusing to install an unverified binary (fail-closed policy)."
        Remove-Item $ChecksumsFile -Force -ErrorAction SilentlyContinue
        exit 1
    }

    Write-Success "Downloaded: $ChecksumsFile"
}

# Verify the SHA256 of the downloaded binary against the published
# checksums (fail-closed). Deletes the binary and exits 1 on a missing
# entry or a hash mismatch. Must run before the binary is installed
# or executed.
function Test-BinaryChecksum($filename, $binaryPath) {
    Write-Step "Verifying SHA256 checksum..."

    $expected = $null
    foreach ($line in @(Get-Content $ChecksumsFile)) {
        # sha256sum format: "<hash>  <filename>", optional "*" binary marker
        if ($line -match '^\s*([0-9A-Fa-f]{64})\s+\*?(.+?)\s*$' -and $Matches[2] -eq $filename) {
            $expected = $Matches[1]
            break
        }
    }

    if (-not $expected) {
        Write-Error "No checksum entry for '$filename' in $ChecksumsFile"
        Write-Warning "Refusing to install an unverified binary (fail-closed policy)."
        Remove-Item $binaryPath -Force -ErrorAction SilentlyContinue
        exit 1
    }

    $actual = (Get-FileHash -Path $binaryPath -Algorithm SHA256).Hash

    # String -eq/-ne comparisons are case-insensitive in PowerShell, so
    # upper/lowercase hash spellings compare equal.
    if ($actual -ne $expected) {
        Write-Error "Checksum mismatch for '$filename' - deleting unverified binary"
        Write-Info "Expected: $expected"
        Write-Info "Actual:   $actual"
        Remove-Item $binaryPath -Force -ErrorAction SilentlyContinue
        exit 1
    }

    Write-Success "Checksum verified: $filename"
}

# Download binary
function Download-Binary($version, $arch) {
    Write-Step "$Download Downloading R2Go2 binary..."

    $filename = "$BinaryName-windows-$arch.exe"
    $downloadUrl = "https://github.com/$Repo/releases/download/v$version/$filename"
    $outputPath = "$BinaryName.exe"

    try {
        Invoke-WebRequest -Uri $downloadUrl -OutFile $outputPath
        if (Test-Path $outputPath) {
            Write-Success "Downloaded: $filename"
            # Verify integrity before the binary is installed or executed
            Test-BinaryChecksum -filename $filename -binaryPath $outputPath
            return $outputPath
        } else {
            Write-Error "Failed to download binary"
            exit 1
        }
    }
    catch {
        Write-Error "Download failed: $_"
        exit 1
    }
}

# Install binary
function Install-Binary($binaryPath) {
    Write-Step "$Install Installing R2Go2..."

    # Create install directory if it doesn't exist
    if (!(Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }

    # Move binary to install directory
    $installPath = Join-Path $InstallDir "$BinaryName.exe"
    Move-Item -Path $binaryPath -Destination $installPath -Force

    Write-Success "Installed to: $installPath"

    # Add to PATH if not already there
    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($currentPath -notlike "*$InstallDir*") {
        [Environment]::SetEnvironmentVariable("Path", $currentPath + ";$InstallDir", "User")
        Write-Info "Added $InstallDir to user PATH"
        Write-Warning "Please restart PowerShell to use the updated PATH"
    }
}

# Verify installation
function Verify-Installation {
    Write-Step "Verifying installation..."

    try {
        # Try to run the binary
        $installPath = Join-Path $InstallDir "$BinaryName.exe"
        if (Test-Path $installPath) {
            $versionOutput = & $installPath --version 2>$null
            if ($LASTEXITCODE -eq 0) {
                Write-Success "Installation verified!"
                Write-Info "Installed version: $versionOutput"
            } else {
                Write-Warning "Binary installed but verification failed"
            }
        } else {
            Write-Warning "Binary not found at expected location: $installPath"
        }
    }
    catch {
        Write-Warning "Verification failed: $_"
    }
}

# Show next steps
function Show-NextSteps {
    Write-Output ""
    Write-ColorOutput Cyan "🎉 Installation Complete!"
    Write-ColorOutput Cyan "=========================="
    Write-Output ""
    Write-Info "Next steps to get started:"
    Write-Output ""
    Write-ColorOutput Blue "1. Restart PowerShell to update PATH:"
    Write-Output "   Close and reopen PowerShell"
    Write-Output ""
    Write-ColorOutput Blue "2. Run the interactive setup:"
    Write-Output "   $BinaryName setup"
    Write-Output ""
    Write-ColorOutput Blue "3. Or set up environment variables:"
    Write-Output '   $env:CLOUDFLARE_API_TOKEN = "your_token_here"'
    Write-Output '   $env:CLOUDFLARE_ACCOUNT_ID = "your_account_id"'
    Write-Output ""
    Write-ColorOutput Blue "4. Try the TUI dashboard:"
    Write-Output "   $BinaryName dashboard"
    Write-Output ""
    Write-ColorOutput Blue "5. List your buckets:"
    Write-Output "   $BinaryName list"
    Write-Output ""
    Write-Info "📚 Documentation: https://github.com/$Repo"
    Write-Info "💬 Support: https://github.com/$Repo/issues"
    Write-Output ""
}

# Cleanup function
function Cleanup {
    $binaryPath = "$BinaryName.exe"
    if (Test-Path $binaryPath) {
        Remove-Item $binaryPath -Force -ErrorAction SilentlyContinue
    }
    if (Test-Path $ChecksumsFile) {
        Remove-Item $ChecksumsFile -Force -ErrorAction SilentlyContinue
    }
}

# Main installation function
function Main {
    Write-Header

    Write-Info "Starting R2Go2 Windows installation..."
    Write-Output ""

    # Check if running as administrator (optional)
    # $isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
    # if ($isAdmin) {
    #     Write-Info "Running with administrator privileges"
    # }

    try {
        # Detect architecture
        $arch = Get-WindowsArch
        Write-Info "Detected platform: windows-$arch"

        # Get latest version
        $version = Get-LatestVersion

        # Download the checksums file for verification (fail-closed)
        Get-ChecksumsFile -version $version

        # Download binary (verifies its checksum before returning)
        $binaryPath = Download-Binary -version $version -arch $arch

        # Install binary
        Install-Binary -binaryPath $binaryPath

        # Verify installation
        Verify-Installation

        # Show next steps
        Show-NextSteps
    }
    catch {
        Write-Error "Installation failed: $_"
        exit 1
    }
    finally {
        Cleanup
    }
}

# Handle script parameters
if ($Help) {
    Write-Output "R2Go2 Windows Installation Script"
    Write-Output ""
    Write-Output "Usage: .\install.ps1 [options]"
    Write-Output ""
    Write-Output "Options:"
    Write-Output "  --Help         Show this help message"
    Write-Output "  --Version      Show script version"
    Write-Output ""
    Write-Output "This script installs the latest version of R2Go2 on Windows."
    Write-Output "It will download the binary and install it to ~/.local/bin"
    Write-Output ""
    exit 0
}

if ($Version) {
    Write-Output "R2Go2 Windows Installation Script v1.0.0"
    exit 0
}

# Check execution policy
if ((Get-ExecutionPolicy) -eq "Restricted") {
    Write-Warning "PowerShell execution policy is Restricted."
    Write-Info "To run this script, change the execution policy:"
    Write-Output "Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser"
    exit 1
}

# Run main installation
Main