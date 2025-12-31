#!/bin/bash
# Mole Installation Script for MacBook Pro
# This script handles everything: verification, cleanup, installation

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Expected DMG details
EXPECTED_DMG="Mole-v1.0.0-20251230-161343.dmg"
EXPECTED_MD5="ce014d0c4b43f23c76a6ef51e8239fcf"

echo ""
echo "═══════════════════════════════════════════"
echo "   🐭 Mole Installation Script"
echo "═══════════════════════════════════════════"
echo ""

# Check if DMG exists in current directory
if [ ! -f "$EXPECTED_DMG" ]; then
    echo -e "${RED}❌ DMG file not found: $EXPECTED_DMG${NC}"
    echo ""
    echo "Please make sure the DMG is in the same directory as this script:"
    echo "  $(pwd)/$EXPECTED_DMG"
    echo ""
    exit 1
fi

echo -e "${BLUE}Step 1/5: Verifying DMG integrity...${NC}"
ACTUAL_MD5=$(md5 -q "$EXPECTED_DMG")
if [ "$ACTUAL_MD5" = "$EXPECTED_MD5" ]; then
    echo -e "${GREEN}✅ DMG verified - MD5 hash matches!${NC}"
else
    echo -e "${RED}❌ DMG verification failed!${NC}"
    echo "Expected: $EXPECTED_MD5"
    echo "Got:      $ACTUAL_MD5"
    echo ""
    echo "Please download the DMG again."
    exit 1
fi
echo ""

echo -e "${BLUE}Step 2/5: Killing Mole processes...${NC}"
pkill -9 Mole 2> /dev/null && echo -e "${GREEN}✅ Killed Mole processes${NC}" || echo "No Mole processes running"
pkill -9 web-go 2> /dev/null && echo -e "${GREEN}✅ Killed web-go processes${NC}" || echo "No web-go processes running"
sleep 1
echo ""

echo -e "${BLUE}Step 3/5: Removing old application...${NC}"
if [ -d "/Applications/Mole.app" ]; then
    echo "Removing /Applications/Mole.app (requires sudo password)..."
    sudo rm -rf /Applications/Mole.app
    echo -e "${GREEN}✅ Removed old application${NC}"
else
    echo "No existing application found (clean install)"
fi

if [ -d "$HOME/Applications/Mole.app" ]; then
    rm -rf "$HOME/Applications/Mole.app"
    echo -e "${GREEN}✅ Removed from ~/Applications${NC}"
fi
echo ""

echo -e "${BLUE}Step 4/5: Clearing all caches...${NC}"
rm -rf ~/Library/Caches/com.enoteware.mole* 2> /dev/null || true
rm -rf ~/Library/WebKit/com.enoteware.mole* 2> /dev/null || true
rm -rf ~/Library/Saved\ Application\ State/com.enoteware.mole* 2> /dev/null || true
rm -rf ~/Library/Caches/com.apple.WebKit.Networking 2> /dev/null || true
rm -rf ~/Library/Caches/com.apple.WebKit.WebContent 2> /dev/null || true
echo -e "${GREEN}✅ All caches cleared${NC}"
echo ""

echo -e "${BLUE}Step 5/5: Installing Mole...${NC}"
echo "Opening DMG and mounting..."
hdiutil attach "$EXPECTED_DMG" -quiet

echo ""
echo -e "${YELLOW}╔═══════════════════════════════════════════╗${NC}"
echo -e "${YELLOW}║  ACTION REQUIRED:                         ║${NC}"
echo -e "${YELLOW}║                                           ║${NC}"
echo -e "${YELLOW}║  1. Drag Mole.app to Applications folder ║${NC}"
echo -e "${YELLOW}║  2. If prompted, click 'Replace'          ║${NC}"
echo -e "${YELLOW}║  3. Wait for copy to complete             ║${NC}"
echo -e "${YELLOW}║  4. Press ENTER when done                 ║${NC}"
echo -e "${YELLOW}╚═══════════════════════════════════════════╝${NC}"
echo ""
read -p "Press ENTER after dragging app to Applications... "

echo ""
echo "Ejecting DMG..."
hdiutil detach /Volumes/Mole\ Installer\ v1.0.0 -quiet 2> /dev/null || true
sleep 1

# Verify installation
echo ""
echo -e "${BLUE}Verifying installation...${NC}"

if [ ! -d "/Applications/Mole.app" ]; then
    echo -e "${RED}❌ Installation failed - Mole.app not found in Applications${NC}"
    exit 1
fi

# Check for new UI markers
if strings /Applications/Mole.app/Contents/MacOS/web-go | grep -q "NEW UI"; then
    echo -e "${GREEN}✅ New UI confirmed in installed app!${NC}"
else
    echo -e "${RED}❌ Warning: Old UI detected in installed app${NC}"
    echo "The installation may have failed."
    exit 1
fi

echo ""
echo "═══════════════════════════════════════════"
echo -e "${GREEN}  🎉 Installation Complete!${NC}"
echo "═══════════════════════════════════════════"
echo ""
echo "Launching Mole..."
sleep 1
open /Applications/Mole.app

echo ""
echo -e "${YELLOW}⚠️  IMPORTANT VERIFICATION:${NC}"
echo ""
echo "Check the app window title bar."
echo "It should say: ${GREEN}\"Mole v1.0.0 - System Cleaner (NEW UI)\"${NC}"
echo ""
echo "If you see the old UI instead:"
echo "  1. Quit Mole"
echo "  2. Restart your Mac"
echo "  3. Run this script again"
echo ""
echo "The app is now running. Enjoy! 🐭"
echo ""
