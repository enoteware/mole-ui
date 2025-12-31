#!/bin/bash
# Build Mole Mac App Installer
# Creates Mole.app (native Swift with embedded WebKit) and packages it as a DMG

set -e

echo "🐭 Building Mole Installer..."
echo ""

# 1. Build the Go web server
echo "📦 Building web server..."
go build -o bin/web-go ./cmd/web/
echo "✅ Web server built ($(du -h bin/web-go | cut -f1))"
echo ""

# 2. Build the Swift native app (which includes the Go binary)
echo "🔨 Building Swift native app..."
./build-swift-app.sh
echo ""

# 3. Create DMG installer
echo "💿 Creating DMG installer..."

# Get version from VERSION file
VERSION=$(cat VERSION 2> /dev/null || echo "dev")
DMG_NAME="Mole-v${VERSION}-$(date +%Y%m%d-%H%M%S).dmg"

rm -rf dmg-build Mole-Installer*.dmg Mole.app
mkdir -p dmg-build
cp -R MoleSwift.app dmg-build/Mole.app
ln -s /Applications dmg-build/Applications

hdiutil create -volname "Mole Installer v${VERSION}" -srcfolder dmg-build -ov -format UDZO "$DMG_NAME" > /dev/null 2>&1
rm -rf dmg-build

echo "✅ DMG created: $DMG_NAME ($(du -h "$DMG_NAME" | cut -f1))"
echo ""

echo "🎉 Build complete!"
echo ""
echo "📦 Files created:"
echo "   MoleSwift.app (native Swift app with embedded WebKit - $(du -sh MoleSwift.app | cut -f1))"
echo "   $DMG_NAME (for distribution - $(du -h "$DMG_NAME" | cut -f1))"
echo ""
echo "🔐 DMG Verification:"
echo "   MD5: $(md5 -q "$DMG_NAME")"
echo "   SHA256: $(shasum -a 256 "$DMG_NAME" | cut -d' ' -f1)"
echo ""
echo "📤 Share $DMG_NAME with your family via:"
echo "   • AirDrop"
echo "   • iCloud Drive"
echo "   • Email"
echo "   • USB Drive"
echo ""
echo "🚀 Test locally: open MoleSwift.app"
echo ""
