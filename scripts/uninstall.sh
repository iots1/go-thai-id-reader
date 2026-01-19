#!/bin/bash

# Go Thai ID API - macOS Uninstaller

set -e

APP_NAME="go-thai-id-api"
INSTALL_DIR="/usr/local/bin"
LAUNCHD_PLIST="$HOME/Library/LaunchAgents/com.gothaiid.api.plist"
LOG_DIR="$HOME/Library/Logs/GoThaiIDAPI"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "============================================"
echo "  Go Thai ID API - macOS Uninstaller"
echo "============================================"
echo ""

echo -e "${YELLOW}[WARNING]${NC} This will remove Go Thai ID API from your system."
echo ""
read -p "Are you sure? (y/N): " CONFIRM
if [[ ! "$CONFIRM" =~ ^[Yy]$ ]]; then
    echo "Uninstall cancelled."
    exit 0
fi

echo ""

# Step 1: Stop service
echo "[STEP 1/4] Stopping service..."
if launchctl list | grep -q "com.gothaiid.api"; then
    launchctl unload "$LAUNCHD_PLIST" 2>/dev/null || true
    echo -e "${GREEN}[OK]${NC} Service stopped"
else
    echo -e "${GREEN}[OK]${NC} No service running"
fi

# Step 2: Kill any running processes
echo "[STEP 2/4] Killing running processes..."
pkill -f "$APP_NAME" 2>/dev/null || true
echo -e "${GREEN}[OK]${NC} Processes terminated"

# Step 3: Remove files
echo "[STEP 3/4] Removing files..."
if [ -f "$INSTALL_DIR/$APP_NAME" ]; then
    sudo rm -f "$INSTALL_DIR/$APP_NAME"
    echo -e "${GREEN}[OK]${NC} Binary removed"
else
    echo -e "${GREEN}[OK]${NC} Binary not found"
fi

# Step 4: Remove LaunchAgent and logs
echo "[STEP 4/4] Removing LaunchAgent and logs..."
if [ -f "$LAUNCHD_PLIST" ]; then
    rm -f "$LAUNCHD_PLIST"
    echo -e "${GREEN}[OK]${NC} LaunchAgent removed"
else
    echo -e "${GREEN}[OK]${NC} LaunchAgent not found"
fi

if [ -d "$LOG_DIR" ]; then
    rm -rf "$LOG_DIR"
    echo -e "${GREEN}[OK]${NC} Logs removed"
else
    echo -e "${GREEN}[OK]${NC} Logs not found"
fi

echo ""
echo "============================================"
echo "  Uninstallation Complete!"
echo "============================================"
echo ""
