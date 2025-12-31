#!/bin/bash
# Build Mole Native Swift App
set -e

echo "🐭 Building Mole Native macOS App..."
echo ""

# Clean previous build
rm -rf MoleSwift.app

# Create app bundle structure
echo "📦 Creating app bundle..."
mkdir -p MoleSwift.app/Contents/{MacOS,Resources,Frameworks}

# Create Info.plist
cat > MoleSwift.app/Contents/Info.plist << 'PLIST'
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
</dict>
</plist>
PLIST

# Build the Swift code
echo "🔨 Compiling Swift code..."
swiftc \
    -o MoleSwift.app/Contents/MacOS/MoleSwift \
    -target arm64-apple-macos12.0 \
    -framework SwiftUI \
    -framework AppKit \
    -framework WebKit \
    -framework Foundation \
    MoleApp.swiftapp/Sources/*.swift

# Copy the Go binary
echo "📋 Copying Go server binary..."
if [ ! -f "bin/web-go" ]; then
    echo "⚠️  Building Go binary first..."
    go build -o bin/web-go ./cmd/web/
fi
cp bin/web-go MoleSwift.app/Contents/MacOS/


# Copy app icon if it exists
if [ -f "MoleApp.swiftapp/Resources/AppIcon.icns" ]; then
    echo "🎨 Adding app icon..."
    cp MoleApp.swiftapp/Resources/AppIcon.icns MoleSwift.app/Contents/Resources/
fi
# Sign the app (ad-hoc signature for local use)
echo "✍️  Signing app..."
codesign --force --deep --sign - MoleSwift.app

echo "✅ Build complete: MoleSwift.app ($(du -sh MoleSwift.app | cut -f1))"
echo ""
echo "🚀 To run: open MoleSwift.app"
