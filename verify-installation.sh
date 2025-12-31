#!/bin/bash
# Verify Mole installation on MacBook Pro

echo "🔍 Mole Installation Verification"
echo "=================================="
echo ""

# Check if app exists
if [ ! -d "/Applications/Mole.app" ]; then
    echo "❌ Mole.app not found in /Applications"
    echo "   Please install from DMG first"
    exit 1
fi

echo "✅ Found /Applications/Mole.app"
echo ""

# Check app size
APP_SIZE=$(du -sh /Applications/Mole.app | cut -f1)
echo "📦 App size: $APP_SIZE"
if [[ "$APP_SIZE" != "13M" && "$APP_SIZE" != "13.3M" ]]; then
    echo "⚠️  Warning: Expected size around 13M"
fi
echo ""

# Check if web-go binary exists
if [ ! -f "/Applications/Mole.app/Contents/MacOS/web-go" ]; then
    echo "❌ web-go binary not found inside app bundle"
    exit 1
fi

WEB_GO_SIZE=$(du -sh /Applications/Mole.app/Contents/MacOS/web-go | cut -f1)
echo "🔧 web-go binary size: $WEB_GO_SIZE"
echo ""

# Check for new UI markers in the binary
echo "🔍 Checking for new UI markers in binary..."
echo ""

if strings /Applications/Mole.app/Contents/MacOS/web-go | grep -q "Select Drive to Analyze"; then
    echo "✅ Found 'Select Drive to Analyze' - NEW UI is present!"
else
    echo "❌ Missing 'Select Drive to Analyze' - OLD UI detected!"
    echo "   The installation did not work properly."
    exit 1
fi

if strings /Applications/Mole.app/Contents/MacOS/web-go | grep -q "NEW UI"; then
    echo "✅ Found 'NEW UI' in title - Latest version confirmed!"
else
    echo "❌ Missing 'NEW UI' in title - OLD version detected!"
    exit 1
fi

if strings /Applications/Mole.app/Contents/MacOS/web-go | grep -q "app-version"; then
    echo "✅ Found 'app-version' element - Version indicator present!"
else
    echo "❌ Missing 'app-version' element - OLD version detected!"
    exit 1
fi

echo ""
echo "🎉 Installation verification PASSED!"
echo ""
echo "The installed app contains the NEW UI."
echo "If you still see the old UI when running the app,"
echo "the issue is with WKWebView caching, not the installation."
echo ""
echo "Next steps:"
echo "  1. Launch: open /Applications/Mole.app"
echo "  2. Check window title shows: 'Mole v1.0.0 - System Cleaner (NEW UI)'"
echo "  3. If still showing old UI, restart your Mac and try again"
echo ""
