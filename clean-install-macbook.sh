#!/bin/bash
# Clean installation script for MacBook Pro
# Run this BEFORE dragging the app from the DMG

set -e

echo "🧹 Mole Clean Installation Script"
echo "=================================="
echo ""

# Check if running as root
if [ "$EUID" -eq 0 ]; then
   echo "❌ Do NOT run this script with sudo"
   exit 1
fi

echo "Step 1: Killing all Mole processes..."
pkill -9 Mole 2>/dev/null || true
pkill -9 web-go 2>/dev/null || true
sleep 2
echo "✅ Processes killed"
echo ""

echo "Step 2: Removing old application..."
if [ -d "/Applications/Mole.app" ]; then
    echo "Found Mole.app in /Applications"
    sudo rm -rf /Applications/Mole.app
    echo "✅ Removed /Applications/Mole.app"
else
    echo "No app in /Applications (good)"
fi

if [ -d "$HOME/Applications/Mole.app" ]; then
    echo "Found Mole.app in ~/Applications"
    rm -rf "$HOME/Applications/Mole.app"
    echo "✅ Removed ~/Applications/Mole.app"
fi
echo ""

echo "Step 3: Clearing all Mole caches..."
rm -rf ~/Library/Caches/com.enoteware.mole* 2>/dev/null || true
rm -rf ~/Library/WebKit/com.enoteware.mole* 2>/dev/null || true
rm -rf ~/Library/Saved\ Application\ State/com.enoteware.mole* 2>/dev/null || true
echo "✅ Caches cleared"
echo ""

echo "Step 4: Clearing WebKit global caches (may take a moment)..."
rm -rf ~/Library/Caches/com.apple.WebKit.Networking 2>/dev/null || true
rm -rf ~/Library/Caches/com.apple.WebKit.WebContent 2>/dev/null || true
echo "✅ WebKit caches cleared"
echo ""

echo "Step 5: Resetting Launch Services database..."
/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister -kill -r -domain local -domain system -domain user
echo "✅ Launch Services reset"
echo ""

echo "🎉 Cleanup complete!"
echo ""
echo "📥 Next steps:"
echo "   1. Mount the DMG: open Mole-v1.0.0-20251230-161343.dmg"
echo "   2. Drag Mole.app to the Applications folder"
echo "   3. If macOS asks to replace, click 'Replace'"
echo "   4. Launch: open /Applications/Mole.app"
echo "   5. Verify window title shows: 'Mole v1.0.0 - System Cleaner (NEW UI)'"
echo ""
echo "⚠️  If you still see old UI after this:"
echo "   - Restart your MacBook Pro"
echo "   - Run this script again"
echo "   - Reinstall from DMG"
echo ""
