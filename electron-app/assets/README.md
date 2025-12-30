# Mole App Icons

This directory should contain the following icon files:

## Required Icons

1. **icon.icns** - macOS app icon (512x512 or larger)
   - Used for the .app bundle and DMG installer
   - Can be created from a PNG using: `iconutil -c icns icon.iconset`

2. **icon.png** - Window icon (256x256 PNG)
   - Used for the app window

3. **tray-icon.png** - Menu bar icon (22x22 PNG with transparency)
   - Should be a simple, monochrome design
   - Works best with black/white for light/dark mode compatibility

4. **dmg-background.png** - DMG installer background (540x380 PNG)
   - Optional: Background image for the DMG installer window

## Creating Icons from a Logo

If you have a logo/icon image, you can create all required formats:

```bash
# Create macOS iconset
mkdir icon.iconset
sips -z 16 16     logo.png --out icon.iconset/icon_16x16.png
sips -z 32 32     logo.png --out icon.iconset/icon_16x16@2x.png
sips -z 32 32     logo.png --out icon.iconset/icon_32x32.png
sips -z 64 64     logo.png --out icon.iconset/icon_32x32@2x.png
sips -z 128 128   logo.png --out icon.iconset/icon_128x128.png
sips -z 256 256   logo.png --out icon.iconset/icon_128x128@2x.png
sips -z 256 256   logo.png --out icon.iconset/icon_256x256.png
sips -z 512 512   logo.png --out icon.iconset/icon_256x256@2x.png
sips -z 512 512   logo.png --out icon.iconset/icon_512x512.png
sips -z 1024 1024 logo.png --out icon.iconset/icon_512x512@2x.png

# Convert to .icns
iconutil -c icns icon.iconset

# Create regular icon
sips -z 256 256 logo.png --out icon.png

# Create tray icon (needs to be simple/monochrome)
sips -z 22 22 logo-simple.png --out tray-icon.png
```

## Temporary Solution

For testing, you can use simple colored squares or leave them out. The app will still work, just without pretty icons.
