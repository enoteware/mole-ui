# Mole App Icon Guide

## Option 1: Download Free Mole Icons (Recommended)

### Best Free Sources:

1. **[Flaticon - Mole Icons](https://www.flaticon.com/free-icons/mole-animal)** (128+ icons)
   - Free download in PNG, SVG formats
   - Various styles: cute, realistic, cartoon
   - Recommended: Download 1024x1024 PNG

2. **[IconScout - Mole Icon](https://iconscout.com/icon/mole-1496790)**
   - Available in SVG, PNG, ICNS formats
   - Direct ICNS download available!

3. **[Vecteezy - Mole Animal Vectors](https://www.vecteezy.com/free-vector/mole-animal)**
   - 1,232+ free mole vector graphics
   - High quality, commercial use allowed

4. **[Pixabay - Mole Vectors](https://pixabay.com/vectors/search/mole/)**
   - 24+ free mole vector art
   - No attribution required

---

## Option 2: Convert Any Image to .icns

### Method A: Online Converter (Easiest)

1. **[img2icns.com](https://img2icns.com/)**
   - Upload PNG/JPG (1024x1024 recommended)
   - Automatically generates all icon sizes
   - Download .icns file

2. **[ConvertICO](https://convertico.com/mac/png-to-icns/)**
   - Supports all Retina sizes
   - Free, no signup required

### Method B: Command Line (macOS)

If you have a PNG image (e.g., `mole.png` at 1024x1024):

```bash
# 1. Create iconset directory
mkdir MyIcon.iconset

# 2. Generate all required sizes using sips
sips -z 16 16     mole.png --out MyIcon.iconset/icon_16x16.png
sips -z 32 32     mole.png --out MyIcon.iconset/icon_16x16@2x.png
sips -z 32 32     mole.png --out MyIcon.iconset/icon_32x32.png
sips -z 64 64     mole.png --out MyIcon.iconset/icon_32x32@2x.png
sips -z 128 128   mole.png --out MyIcon.iconset/icon_128x128.png
sips -z 256 256   mole.png --out MyIcon.iconset/icon_128x128@2x.png
sips -z 256 256   mole.png --out MyIcon.iconset/icon_256x256.png
sips -z 512 512   mole.png --out MyIcon.iconset/icon_256x256@2x.png
sips -z 512 512   mole.png --out MyIcon.iconset/icon_512x512.png
sips -z 1024 1024 mole.png --out MyIcon.iconset/icon_512x512@2x.png

# 3. Convert to .icns
iconutil -c icns MyIcon.iconset

# 4. Result: MyIcon.icns
```

---

## Option 3: Use macOS SF Symbols (Quick & Easy)

SF Symbols is Apple's icon library. While it doesn't have a mole, here are close alternatives:

```bash
# Download SF Symbols app (free from Apple)
open https://developer.apple.com/sf-symbols/

# Browse for animal icons:
# - ant.circle.fill
# - hare.fill
# - tortoise.fill
```

---

## Installing Icon to Mole App

Once you have `AppIcon.icns`:

### For Swift App (Current Setup):

```bash
# 1. Check current icon location
ls -la MoleApp.swiftapp/

# If using Assets.xcassets:
cp AppIcon.icns MoleApp.swiftapp/Assets.xcassets/AppIcon.appiconset/

# If icon needs to be in Resources:
mkdir -p MoleApp.swiftapp/Resources
cp AppIcon.icns MoleApp.swiftapp/Resources/

# 2. Update Info.plist (if needed)
# Add CFBundleIconFile key pointing to icon

# 3. Rebuild app
./build-installer.sh
```

---

## Quick Recommendations

**Best Option**: Download from [IconScout](https://iconscout.com/icon/mole-1496790) (has direct ICNS!)

**Runner-up**: Use [img2icns.com](https://img2icns.com/) with any 1024x1024 mole PNG from [Flaticon](https://www.flaticon.com/free-icons/mole-animal)

**Fallback**: Use the 🐭 emoji (run `./create-icon.sh`)

---

## Icon Size Requirements

macOS app icons need these sizes:
- 16x16, 32x32, 64x64, 128x128, 256x256, 512x512, 1024x1024
- Each with @2x Retina versions

The .icns format bundles all sizes together.

---

## Testing the New Icon

After building with new icon:

```bash
# 1. Rebuild
./build-installer.sh

# 2. Check icon is embedded
ls -la MoleSwift.app/Contents/Resources/*.icns

# 3. Test app
open MoleSwift.app

# Icon should appear in:
# - Dock
# - App Switcher (Cmd+Tab)
# - Finder
# - About dialog
```
