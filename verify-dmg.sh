#!/bin/bash
# Verify Mole DMG integrity on MacBook Pro

DMG_FILE="$1"

if [ -z "$DMG_FILE" ]; then
    echo "Usage: ./verify-dmg.sh <path-to-dmg>"
    echo ""
    echo "Example: ./verify-dmg.sh ~/Downloads/Mole-v1.0.0-20251230-161343.dmg"
    exit 1
fi

if [ ! -f "$DMG_FILE" ]; then
    echo "❌ File not found: $DMG_FILE"
    exit 1
fi

echo "🔍 Verifying DMG: $(basename "$DMG_FILE")"
echo ""

# Check MD5
EXPECTED_MD5="ce014d0c4b43f23c76a6ef51e8239fcf"
ACTUAL_MD5=$(md5 -q "$DMG_FILE")

echo "MD5 Hash:"
echo "  Expected: $EXPECTED_MD5"
echo "  Actual:   $ACTUAL_MD5"

if [ "$ACTUAL_MD5" = "$EXPECTED_MD5" ]; then
    echo "  ✅ MD5 MATCH - This is the correct file!"
else
    echo "  ❌ MD5 MISMATCH - This is NOT the correct file!"
    exit 1
fi

echo ""

# Check SHA256
EXPECTED_SHA256="ea9364f056a3d0d156a3c7bba0d8bac0600c572c23c8ae3a47e8b2b9d6294cc7"
ACTUAL_SHA256=$(shasum -a 256 "$DMG_FILE" | cut -d' ' -f1)

echo "SHA256 Hash:"
echo "  Expected: $EXPECTED_SHA256"
echo "  Actual:   $ACTUAL_SHA256"

if [ "$ACTUAL_SHA256" = "$EXPECTED_SHA256" ]; then
    echo "  ✅ SHA256 MATCH - File integrity verified!"
else
    echo "  ❌ SHA256 MISMATCH - File may be corrupted!"
    exit 1
fi

echo ""
echo "🎉 All checks passed! This is the correct DMG file."
echo ""
echo "Next steps:"
echo "  1. Remove old Mole app: sudo rm -rf /Applications/Mole.app"
echo "  2. Clear caches: rm -rf ~/Library/Caches/com.enoteware.mole* ~/Library/WebKit/com.enoteware.mole*"
echo "  3. Install: open '$DMG_FILE'"
echo "  4. Verify window title shows: 'Mole v1.0.0 - System Cleaner (NEW UI)'"
