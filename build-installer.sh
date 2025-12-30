#!/bin/bash
# Build Mole Mac App Installer
# Creates Mole.app and packages it as a DMG for distribution

set -e

echo "🐭 Building Mole Installer..."
echo ""

# 1. Build the Go web server
echo "📦 Building web server..."
go build -o bin/web-go ./cmd/web/
echo "✅ Web server built ($(du -h bin/web-go | cut -f1))"
echo ""

# 2. Create .app bundle structure
echo "🏗️  Creating app bundle..."
rm -rf Mole.app
mkdir -p Mole.app/Contents/{MacOS,Resources}

# Copy Info.plist (should already exist, but recreate if needed)
if [[ ! -f "Mole.app/Contents/Info.plist" ]]; then
    cat > Mole.app/Contents/Info.plist << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleDevelopmentRegion</key>
    <string>en</string>
    <key>CFBundleExecutable</key>
    <string>Mole</string>
    <key>CFBundleIconFile</key>
    <string>AppIcon</string>
    <key>CFBundleIdentifier</key>
    <string>com.mole.app</string>
    <key>CFBundleInfoDictionaryVersion</key>
    <string>6.0</string>
    <key>CFBundleName</key>
    <string>Mole</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>1.0.0</string>
    <key>CFBundleVersion</key>
    <string>1</string>
    <key>LSMinimumSystemVersion</key>
    <string>11.0</string>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>LSUIElement</key>
    <false/>
</dict>
</plist>
EOF
fi

# Create launch script
cat > Mole.app/Contents/MacOS/Mole << 'EOF'
#!/bin/bash
# Mole Mac App Launcher
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
APP_DIR="$( dirname "$( dirname "$SCRIPT_DIR" )" )"
MACOS_DIR="$APP_DIR/Contents/MacOS"
WEB_SERVER="$MACOS_DIR/web-go"
PID_FILE="$HOME/Library/Application Support/Mole/server.pid"
LOG_FILE="$HOME/Library/Application Support/Mole/server.log"

mkdir -p "$HOME/Library/Application Support/Mole"

export MOLE_PORT=8081
export MOLE_HOST="127.0.0.1"
export MOLE_NO_OPEN=1

# Check if already running
if [[ -f "$PID_FILE" ]]; then
    OLD_PID=$(cat "$PID_FILE")
    if ps -p "$OLD_PID" > /dev/null 2>&1; then
        open "http://127.0.0.1:${MOLE_PORT}"
        exit 0
    else
        rm -f "$PID_FILE"
    fi
fi

# Start server
nohup "$WEB_SERVER" >> "$LOG_FILE" 2>&1 &
echo $! > "$PID_FILE"

# Wait for server to be ready
for i in {1..20}; do
    if curl -s "http://127.0.0.1:${MOLE_PORT}/health" > /dev/null 2>&1; then
        sleep 0.5
        open "http://127.0.0.1:${MOLE_PORT}"
        osascript -e 'display notification "Mole is running at http://127.0.0.1:'${MOLE_PORT}'" with title "Mole Started"'
        exit 0
    fi
    sleep 0.5
done

osascript -e 'display alert "Mole Failed to Start" message "Check the log file at:\n'$LOG_FILE'" as critical'
exit 1
EOF

# Copy binary and make executable
cp bin/web-go Mole.app/Contents/MacOS/
chmod +x Mole.app/Contents/MacOS/{Mole,web-go}

echo "✅ App bundle created ($(du -sh Mole.app | cut -f1))"
echo ""

# 3. Create DMG
echo "💿 Creating DMG installer..."
rm -rf dmg-build Mole-Installer.dmg
mkdir -p dmg-build
cp -R Mole.app dmg-build/
ln -s /Applications dmg-build/Applications

hdiutil create -volname "Mole Installer" -srcfolder dmg-build -ov -format UDZO Mole-Installer.dmg > /dev/null 2>&1
rm -rf dmg-build

echo "✅ DMG created ($(du -h Mole-Installer.dmg | cut -f1))"
echo ""

echo "🎉 Build complete!"
echo ""
echo "📦 Files created:"
echo "   Mole.app (for direct use)"
echo "   Mole-Installer.dmg (for distribution)"
echo ""
echo "📤 Share Mole-Installer.dmg with your family via:"
echo "   • AirDrop"
echo "   • iCloud Drive"
echo "   • Email"
echo "   • USB Drive"
echo ""
