#!/bin/bash

# Go Thai ID API - macOS Installer
# Supports Apple Silicon and Intel Macs

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
echo "  Go Thai ID API - macOS Installer"
echo "============================================"
echo ""

# Detect architecture
ARCH=$(uname -m)
if [ "$ARCH" = "arm64" ]; then
    echo -e "${GREEN}[INFO]${NC} Detected Apple Silicon (M1/M2/M3)"
    BINARY_SUFFIX="darwin-arm64"
else
    echo -e "${GREEN}[INFO]${NC} Detected Intel Mac"
    BINARY_SUFFIX="darwin-amd64"
fi

# Find the binary in current directory
BINARY_FILE=""
if [ -f "./${APP_NAME}-${BINARY_SUFFIX}" ]; then
    BINARY_FILE="./${APP_NAME}-${BINARY_SUFFIX}"
elif [ -f "./${APP_NAME}" ]; then
    BINARY_FILE="./${APP_NAME}"
fi

if [ -z "$BINARY_FILE" ]; then
    echo -e "${RED}[ERROR]${NC} Cannot find go-thai-id-api binary"
    echo "Please place this script in the same folder as the binary"
    exit 1
fi

echo -e "${GREEN}[INFO]${NC} Found binary: $BINARY_FILE"
echo -e "${GREEN}[INFO]${NC} Install directory: $INSTALL_DIR"
echo ""

# Step 1: Stop existing service
echo "[STEP 1/4] Stopping existing service..."
if launchctl list | grep -q "com.gothaiid.api"; then
    launchctl unload "$LAUNCHD_PLIST" 2>/dev/null || true
    echo -e "${GREEN}[OK]${NC} Existing service stopped"
else
    echo -e "${GREEN}[OK]${NC} No existing service found"
fi

# Step 2: Create directories
echo "[STEP 2/4] Creating directories..."
mkdir -p "$LOG_DIR"
mkdir -p "$(dirname "$LAUNCHD_PLIST")"
echo -e "${GREEN}[OK]${NC} Directories created"

# Step 3: Copy binary
echo "[STEP 3/4] Installing binary..."
sudo cp "$BINARY_FILE" "$INSTALL_DIR/$APP_NAME"
sudo chmod +x "$INSTALL_DIR/$APP_NAME"

# Remove quarantine attribute if exists (for downloaded files)
sudo xattr -rd com.apple.quarantine "$INSTALL_DIR/$APP_NAME" 2>/dev/null || true

echo -e "${GREEN}[OK]${NC} Binary installed to $INSTALL_DIR/$APP_NAME"

# Step 4: Create LaunchAgent plist
echo "[STEP 4/4] Creating LaunchAgent..."

cat > "$LAUNCHD_PLIST" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.gothaiid.api</string>
    <key>ProgramArguments</key>
    <array>
        <string>$INSTALL_DIR/$APP_NAME</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>$LOG_DIR/stdout.log</string>
    <key>StandardErrorPath</key>
    <string>$LOG_DIR/stderr.log</string>
    <key>WorkingDirectory</key>
    <string>$INSTALL_DIR</string>
</dict>
</plist>
EOF

# Load the service
launchctl load "$LAUNCHD_PLIST"

echo -e "${GREEN}[OK]${NC} LaunchAgent created and loaded"

echo ""
echo "============================================"
echo "  Installation Complete!"
echo "============================================"
echo ""
echo -e "API URL: ${GREEN}http://localhost:8080/api/read${NC}"
echo "Binary: $INSTALL_DIR/$APP_NAME"
echo "Logs: $LOG_DIR/"
echo ""
echo "The service will start automatically on login."
echo ""
echo "Commands:"
echo "  Start:   launchctl load $LAUNCHD_PLIST"
echo "  Stop:    launchctl unload $LAUNCHD_PLIST"
echo "  Status:  launchctl list | grep gothaiid"
echo ""
