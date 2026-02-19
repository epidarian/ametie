@echo off
REM Ametie Installation Script for Windows (Batch)

echo Ametie Installation Script
echo ===========================
echo.

REM Check for admin privileges
net session >nul 2>&1
if %errorLevel% neq 0 (
    echo Please run as Administrator
    exit /b 1
)

REM Get installation directory
set INSTALL_DIR=%ProgramFiles%\Ametie
if not exist "%INSTALL_DIR%" mkdir "%INSTALL_DIR%"

REM Prompt for configuration
echo Configuration:
set /p API_KEY=API Key: 
set /p SERVER_URL=Server URL: 
set /p NODE_NAME=Node Name (default: %COMPUTERNAME%): 
if "%NODE_NAME%"=="" set NODE_NAME=%COMPUTERNAME%

REM Build binaries
echo.
echo Building binaries...
cd /d "%~dp0\.."
go build -o "%INSTALL_DIR%\ametie.exe" .\cmd\ametie
go build -o "%INSTALL_DIR%\ametie-client.exe" .\cmd\ametie-client

REM Configure
echo Configuring...
"%INSTALL_DIR%\ametie.exe" install --api-key "%API_KEY%" --server-url "%SERVER_URL%" --node-name "%NODE_NAME%"

REM Install service
echo Installing Windows service...
sc create Ametie binPath= "%INSTALL_DIR%\ametie-client.exe" start= auto
sc description Ametie "Ametie C2 Reverse Tunnel Service"
sc start Ametie

echo.
echo Installation complete!
echo Use 'ametie' command to manage the service

