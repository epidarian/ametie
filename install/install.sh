#!/bin/bash
# Ametie Installation Script for Linux/macOS

set -e

echo "Ametie Installation Script"
echo "==========================="
echo ""

# Detect OS
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    OS="linux"
    SERVICE_FILE="ametie.service"
elif [[ "$OSTYPE" == "darwin"* ]]; then
    OS="macos"
    SERVICE_FILE="com.ametie.plist"
else
    echo "Unsupported OS: $OSTYPE"
    exit 1
fi

# Get installation directory
INSTALL_DIR="/usr/local/bin"
if [ "$OS" == "macos" ]; then
    INSTALL_DIR="/usr/local/bin"
fi

# Check if running as root for system-wide installation
if [ "$EUID" -ne 0 ]; then
    echo "Please run as root (use sudo)"
    exit 1
fi

# Prompt for configuration
echo "Configuration:"
read -p "API Key: " API_KEY
read -p "Server URL: " SERVER_URL
read -p "Node Name (default: $(hostname)): " NODE_NAME
NODE_NAME=${NODE_NAME:-$(hostname)}

# Build binaries
echo ""
echo "Building binaries..."
cd "$(dirname "$0")/.."
go build -o bin/ametie ./cmd/ametie
go build -o bin/ametie-client ./cmd/ametie-client

# Install binaries
echo "Installing binaries..."
cp bin/ametie "$INSTALL_DIR/ametie"
cp bin/ametie-client "$INSTALL_DIR/ametie-client"
chmod +x "$INSTALL_DIR/ametie"
chmod +x "$INSTALL_DIR/ametie-client"

# Configure
echo "Configuring..."
"$INSTALL_DIR/ametie" install --api-key "$API_KEY" --server-url "$SERVER_URL" --node-name "$NODE_NAME"

# Install service
if [ "$OS" == "linux" ]; then
    echo "Installing systemd service..."
    cp "install/$SERVICE_FILE" /etc/systemd/system/
    systemctl daemon-reload
    systemctl enable ametie
    systemctl start ametie
    echo "Service installed and started"
elif [ "$OS" == "macos" ]; then
    echo "Installing launchd service..."
    PLIST_DIR="$HOME/Library/LaunchAgents"
    mkdir -p "$PLIST_DIR"
    cp "install/$SERVICE_FILE" "$PLIST_DIR/"
    launchctl load "$PLIST_DIR/$SERVICE_FILE"
    launchctl start com.ametie
    echo "Service installed and started"
fi

echo ""
echo "Installation complete!"
echo "Use 'ametie' command to manage the service"

