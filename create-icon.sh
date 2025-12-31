#!/bin/bash
# Create Mole app icon from emoji

set -e

echo "🎨 Creating Mole App Icon..."
echo ""

# Create temporary directory
TEMP_DIR=$(mktemp -d)
ICONSET_DIR="$TEMP_DIR/AppIcon.iconset"
mkdir -p "$ICONSET_DIR"

# Function to create icon at specific size
create_icon_size() {
    SIZE=$1
    SCALE=$2
    OUTPUT_NAME="${SIZE}x${SIZE}"
    if [ "$SCALE" = "2" ]; then
        OUTPUT_NAME="${SIZE}x${SIZE}@2x"
    fi

    # Use Python to create PNG with emoji
    python3 - <<EOF
from PIL import Image, ImageDraw, ImageFont
import sys

size = $SIZE * $SCALE
img = Image.new('RGBA', (size, size), (0, 0, 0, 0))
draw = ImageDraw.Draw(img)

# Try to use system font for emoji
try:
    font = ImageFont.truetype('/System/Library/Fonts/Apple Color Emoji.ttc', int(size * 0.8))
except:
    font = ImageFont.truetype('/System/Library/Fonts/Supplemental/Arial Unicode.ttf', int(size * 0.8))

# Draw mole emoji
emoji = '🐭'
bbox = draw.textbbox((0, 0), emoji, font=font)
text_width = bbox[2] - bbox[0]
text_height = bbox[3] - bbox[1]
x = (size - text_width) // 2 - bbox[0]
y = (size - text_height) // 2 - bbox[1]

draw.text((x, y), emoji, font=font, embedded_color=True)
img.save('$ICONSET_DIR/icon_${OUTPUT_NAME}.png')
EOF
}

# Check if Python PIL is available
if ! python3 -c "import PIL" 2>/dev/null; then
    echo "❌ Python PIL library not found"
    echo ""
    echo "Installing pillow..."
    pip3 install pillow --break-system-packages 2>/dev/null || pip3 install pillow
fi

echo "Generating icon sizes..."

# Generate all required sizes
create_icon_size 16 1
create_icon_size 16 2
create_icon_size 32 1
create_icon_size 32 2
create_icon_size 128 1
create_icon_size 128 2
create_icon_size 256 1
create_icon_size 256 2
create_icon_size 512 1
create_icon_size 512 2

echo "✅ Generated all icon sizes"
echo ""

# Convert to icns
echo "Converting to .icns format..."
iconutil -c icns "$ICONSET_DIR" -o AppIcon.icns

# Cleanup
rm -rf "$TEMP_DIR"

echo "✅ Icon created: AppIcon.icns"
echo ""
echo "To use this icon:"
echo "  1. Copy to app: cp AppIcon.icns MoleApp.swiftapp/Assets.xcassets/AppIcon.appiconset/"
echo "  2. Rebuild app: ./build-installer.sh"
echo ""
