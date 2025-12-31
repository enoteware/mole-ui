#!/bin/bash
# Add icon to Mole app

set -e

echo ""
echo "🎨 Mole App Icon Setup"
echo "======================"
echo ""

# Check if icon file exists
if [ ! -f "AppIcon.icns" ]; then
    echo "❌ AppIcon.icns not found"
    echo ""
    echo "Please download a mole icon and convert to .icns format:"
    echo ""
    echo "📥 Quick Steps:"
    echo "   1. Visit: https://www.flaticon.com/free-icons/mole-animal"
    echo "   2. Download a mole icon (PNG, 1024x1024 recommended)"
    echo "   3. Convert to .icns using: https://img2icns.com/"
    echo "   4. Save as: AppIcon.icns"
    echo "   5. Place in this directory: $(pwd)"
    echo "   6. Run this script again: ./add-icon.sh"
    echo ""
    echo "📖 Or read ICON-GUIDE.md for more options"
    echo ""
    exit 1
fi

echo "✅ Found AppIcon.icns"
echo ""

# Check file size (should be reasonable for an icon)
SIZE=$(ls -lh AppIcon.icns | awk '{print $5}')
echo "📦 Icon file size: $SIZE"
echo ""

# Copy icon to Swift app resources (source)
echo "📋 Adding icon to Swift app source..."
mkdir -p MoleApp.swiftapp/Resources
cp AppIcon.icns MoleApp.swiftapp/Resources/

echo "✅ Icon added to source"
echo ""

# Rebuild the app
echo "🔨 Rebuilding app with new icon..."
./build-installer.sh

echo ""
echo "═══════════════════════════════════════"
echo "  ✅ Icon Successfully Added!"
echo "═══════════════════════════════════════"
echo ""
echo "The new icon should now appear in:"
echo "  • Dock"
echo "  • App Switcher (Cmd+Tab)"
echo "  • Finder"
echo "  • DMG installer"
echo ""
echo "Test it: open MoleSwift.app"
echo ""
