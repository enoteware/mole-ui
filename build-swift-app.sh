#!/bin/bash
# Build Mole Native Swift App
set -e

echo "🐭 Building Mole Native macOS App..."
echo ""

# Bundle in /tmp to avoid network drive detritus issues during signing
TMP_APP="/tmp/MoleSwift.app"
rm -rf "$TMP_APP"

# Create app bundle structure
echo "📦 Creating app bundle in $TMP_APP..."
mkdir -p "$TMP_APP/Contents/"{MacOS,Resources,Frameworks}

# Create Info.plist
cat > "$TMP_APP/Contents/Info.plist" << 'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleDevelopmentRegion</key>
    <string>en</string>
    <key>CFBundleExecutable</key>
    <string>MoleSwift</string>
    <key>CFBundleIconFile</key>
    <string>AppIcon</string>
    <key>CFBundleIdentifier</key>
    <string>com.mole.swift</string>
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
    <string>12.0</string>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>NSPrincipalClass</key>
    <string>NSApplication</string>
    <key>NSAppleEventsUsageDescription</key>
    <string>Mole needs permission to control system events for authentication when uninstalling protected apps.</string>
</dict>
</plist>
PLIST

# Build the Swift code (exclude PrivilegedHelper which is compiled separately)
echo "🔨 Compiling Swift code..."
swiftc \
    -o "$TMP_APP/Contents/MacOS/MoleSwift" \
    -target arm64-apple-macos12.0 \
    -framework SwiftUI \
    -framework AppKit \
    -framework WebKit \
    -framework Foundation \
    MoleApp.swiftapp/Sources/MoleApp.swift \
    MoleApp.swiftapp/Sources/AppDelegate.swift \
    MoleApp.swiftapp/Sources/ContentView.swift \
    MoleApp.swiftapp/Sources/ServerManager.swift

# Build the privileged helper tool
echo "🔐 Compiling privileged helper..."
swiftc \
    -o "$TMP_APP/Contents/Resources/privileged-helper" \
    -target arm64-apple-macos12.0 \
    -framework Foundation \
    -framework Security \
    MoleApp.swiftapp/Sources/PrivilegedHelper.swift

# Copy the Go binary
echo "📋 Copying Go server binary..."
if [ ! -f "bin/web-go" ]; then
    echo "⚠️  Building Go binary first..."
    go build -o bin/web-go ./cmd/web/
fi
cp bin/web-go "$TMP_APP/Contents/MacOS/"

# Bundle CLI script and library for self-containment
echo "📦 Bundling CLI scripts and library..."
cp mole "$TMP_APP/Contents/Resources/"
cp -R lib "$TMP_APP/Contents/Resources/"
cp -R bin "$TMP_APP/Contents/Resources/"

# Copy app icon if it exists
if [ -f "MoleApp.swiftapp/Resources/AppIcon.icns" ]; then
    echo "🎨 Adding app icon..."
    cp MoleApp.swiftapp/Resources/AppIcon.icns "$TMP_APP/Contents/Resources/"
fi

# Sign the app (ad-hoc signature for local use)
echo "✍️  Signing app..."
find "$TMP_APP" -name ".DS_Store" -delete 2> /dev/null || true
find "$TMP_APP" -name "._*" -delete 2> /dev/null || true
xattr -rc "$TMP_APP"
codesign --force --deep --sign - "$TMP_APP"

# Copy back to repository for the user
echo "🚚 Copying back to repository..."
rm -rf MoleSwift.app
cp -R "$TMP_APP" .

echo "✅ Build complete: MoleSwift.app ($(du -sh MoleSwift.app | cut -f1))"
echo ""
echo "🚀 To run: open MoleSwift.app"
