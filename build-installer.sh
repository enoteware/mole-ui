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
rm -rf dmg-build Mole-Installer.dmg Mole.app
mkdir -p dmg-build
cp -R MoleSwift.app dmg-build/Mole.app
ln -s /Applications dmg-build/Applications

hdiutil create -volname "Mole Installer" -srcfolder dmg-build -ov -format UDZO Mole-Installer.dmg > /dev/null 2>&1
rm -rf dmg-build

echo "✅ DMG created ($(du -h Mole-Installer.dmg | cut -f1))"
echo ""

echo "🎉 Build complete!"
echo ""
echo "📦 Files created:"
echo "   MoleSwift.app (native Swift app with embedded WebKit - $(du -sh MoleSwift.app | cut -f1))"
echo "   Mole-Installer.dmg (for distribution - $(du -h Mole-Installer.dmg | cut -f1))"
echo ""
echo "📤 Share Mole-Installer.dmg with your family via:"
echo "   • AirDrop"
echo "   • iCloud Drive"
echo "   • Email"
echo "   • USB Drive"
echo ""
echo "🚀 Test locally: open MoleSwift.app"
echo ""
