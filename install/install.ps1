# Ametie Installation Script for Windows (PowerShell)

Write-Host "Ametie Installation Script" -ForegroundColor Green
Write-Host "===========================" -ForegroundColor Green
Write-Host ""

# Check if running as administrator
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    Write-Host "Please run as Administrator" -ForegroundColor Red
    exit 1
}

# Get installation directory
$installDir = "$env:ProgramFiles\Ametie"
New-Item -ItemType Directory -Force -Path $installDir | Out-Null

# Prompt for configuration
Write-Host "Configuration:" -ForegroundColor Yellow
$apiKey = Read-Host "API Key"
$serverUrl = Read-Host "Server URL"
$nodeName = Read-Host "Node Name (default: $env:COMPUTERNAME)"
if ([string]::IsNullOrEmpty($nodeName)) {
    $nodeName = $env:COMPUTERNAME
}

# Build binaries
Write-Host ""
Write-Host "Building binaries..." -ForegroundColor Yellow
$scriptPath = Split-Path -Parent $MyInvocation.MyCommand.Path
$projectRoot = Split-Path -Parent $scriptPath
Set-Location $projectRoot

go build -o "$installDir\ametie.exe" .\cmd\ametie
go build -o "$installDir\ametie-client.exe" .\cmd\ametie-client

# Configure
Write-Host "Configuring..." -ForegroundColor Yellow
& "$installDir\ametie.exe" install --api-key $apiKey --server-url $serverUrl --node-name $nodeName

# Install service using NSSM
Write-Host "Installing Windows service..." -ForegroundColor Yellow

# Check if NSSM is available
$nssmPath = "$installDir\nssm.exe"
if (-not (Test-Path $nssmPath)) {
    Write-Host "NSSM not found. Please install NSSM first:" -ForegroundColor Yellow
    Write-Host "  Download from: https://nssm.cc/download" -ForegroundColor Yellow
    Write-Host "  Or use: choco install nssm" -ForegroundColor Yellow
    exit 1
}

# Install service
& $nssmPath install Ametie "$installDir\ametie-client.exe"
& $nssmPath set Ametie AppDirectory $installDir
& $nssmPath set Ametie Description "Ametie C2 Reverse Tunnel Service"
& $nssmPath set Ametie Start SERVICE_AUTO_START

# Start service
Start-Service Ametie

Write-Host ""
Write-Host "Installation complete!" -ForegroundColor Green
Write-Host "Use 'ametie' command to manage the service" -ForegroundColor Green

