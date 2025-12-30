#!/bin/bash
# Install Mole Web UI as a launchd service (auto-start on boot)
#
# Usage: ./install-launchd.sh [--uninstall]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
PLIST_NAME="com.mole.web"
PLIST_PATH="$HOME/Library/LaunchAgents/${PLIST_NAME}.plist"
BINARY="$ROOT_DIR/bin/web-go"

if [[ "$1" == "--uninstall" ]]; then
    echo "Uninstalling Mole Web UI service..."
    launchctl unload "$PLIST_PATH" 2>/dev/null || true
    rm -f "$PLIST_PATH"
    echo "✅ Service uninstalled"
    exit 0
fi

# Check for .env
if [[ ! -f "$SCRIPT_DIR/.env" ]]; then
    echo "⚠️  No .env file found!"
    echo "   Copy deploy/.env.example to deploy/.env and configure authentication."
    exit 1
fi

# Load env
source "$SCRIPT_DIR/.env"

# Build if needed
if [[ ! -f "$BINARY" ]]; then
    echo "Building web server first..."
    cd "$ROOT_DIR"
    go build -o "$BINARY" ./cmd/web/
fi

# Create LaunchAgent plist
cat > "$PLIST_PATH" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>${PLIST_NAME}</string>
    <key>ProgramArguments</key>
    <array>
        <string>${BINARY}</string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>MOLE_DIR</key>
        <string>${ROOT_DIR}</string>
        <key>MOLE_PORT</key>
        <string>${MOLE_PORT:-8080}</string>
        <key>MOLE_HOST</key>
        <string>${MOLE_HOST:-0.0.0.0}</string>
        <key>MOLE_NO_OPEN</key>
        <string>1</string>
        <key>MOLE_AUTH_USER</key>
        <string>${MOLE_AUTH_USER}</string>
        <key>MOLE_AUTH_PASS</key>
        <string>${MOLE_AUTH_PASS}</string>
    </dict>
    <key>WorkingDirectory</key>
    <string>${ROOT_DIR}</string>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>${SCRIPT_DIR}/mole-web.log</string>
    <key>StandardErrorPath</key>
    <string>${SCRIPT_DIR}/mole-web.log</string>
</dict>
</plist>
EOF

# Load the service
launchctl unload "$PLIST_PATH" 2>/dev/null || true
launchctl load "$PLIST_PATH"

echo ""
echo "✅ Mole Web UI installed as launchd service"
echo ""
echo "   The server will now:"
echo "   - Start automatically on login"
echo "   - Restart if it crashes"
echo ""
echo "   Commands:"
echo "   - View logs:    tail -f $SCRIPT_DIR/mole-web.log"
echo "   - Stop:         launchctl unload $PLIST_PATH"
echo "   - Start:        launchctl load $PLIST_PATH"
echo "   - Uninstall:    $0 --uninstall"
echo ""

# Get local IP
LOCAL_IP=$(ipconfig getifaddr en0 2>/dev/null || echo "localhost")
echo "   Access: http://${LOCAL_IP}:${MOLE_PORT:-8080}"
echo ""
