#!/bin/bash
# Build Mole Native Swift App with WidgetKit Extension
# Uses xcodebuild for proper widget bundling
set -e

echo "🐭 Building Mole Native macOS App with Widgets..."
echo ""

# Check for Xcode command line tools
if ! command -v xcodebuild &> /dev/null; then
    echo "❌ xcodebuild not found. Please install Xcode command line tools:"
    echo "   xcode-select --install"
    exit 1
fi

# Bundle in /tmp to avoid network drive issues during signing
BUILD_DIR="/tmp/MoleBuild"
TMP_APP="/tmp/MoleSwift.app"
rm -rf "$BUILD_DIR" "$TMP_APP"
mkdir -p "$BUILD_DIR"

# First, build the Go binary if needed
echo "📋 Checking Go server binary..."
if [ ! -f "bin/web-go" ]; then
    echo "⚠️  Building Go binary first..."
    go build -o bin/web-go ./cmd/web/
fi

# Build with xcodebuild
echo "🔨 Building with xcodebuild..."
xcodebuild \
    -project MoleApp.xcodeproj \
    -scheme MoleSwift \
    -configuration Release \
    -derivedDataPath "$BUILD_DIR" \
    -destination "platform=macOS" \
    CODE_SIGN_IDENTITY="-" \
    CODE_SIGNING_REQUIRED=NO \
    CODE_SIGNING_ALLOWED=NO \
    clean build 2>&1 | grep -E "(Building|Compiling|Linking|error:|warning:)" || true

# Find the built app
BUILT_APP=$(find "$BUILD_DIR" -name "MoleSwift.app" -type d | head -1)

if [ -z "$BUILT_APP" ] || [ ! -d "$BUILT_APP" ]; then
    echo "❌ Build failed - MoleSwift.app not found in $BUILD_DIR"
    echo "   Run 'xcodebuild -project MoleApp.xcodeproj -scheme MoleSwift -configuration Release' for details"
    exit 1
fi

echo "✅ Found built app: $BUILT_APP"

# Copy to temp location
cp -R "$BUILT_APP" "$TMP_APP"

# Copy the Go binary and resources
echo "📋 Copying Go server binary..."
cp bin/web-go "$TMP_APP/Contents/MacOS/"

# Bundle CLI script and library for self-containment
echo "📦 Bundling CLI scripts and library..."
cp mole "$TMP_APP/Contents/Resources/" 2>/dev/null || true
cp -R lib "$TMP_APP/Contents/Resources/" 2>/dev/null || true
cp -R bin "$TMP_APP/Contents/Resources/" 2>/dev/null || true

# Copy app icon if it exists and wasn't included
if [ -f "MoleApp.swiftapp/Resources/AppIcon.icns" ] && [ ! -f "$TMP_APP/Contents/Resources/AppIcon.icns" ]; then
    echo "🎨 Adding app icon..."
    cp MoleApp.swiftapp/Resources/AppIcon.icns "$TMP_APP/Contents/Resources/"
fi

# Verify widget extension was bundled
if [ -d "$TMP_APP/Contents/PlugIns/MoleWidgets.appex" ]; then
    echo "✅ Widget extension bundled successfully"
else
    echo "⚠️  Widget extension not found in bundle (build may have skipped it)"
fi

# Clean up and sign
echo "✍️  Signing app..."
find "$TMP_APP" -name ".DS_Store" -delete 2>/dev/null || true
find "$TMP_APP" -name "._*" -delete 2>/dev/null || true
xattr -rc "$TMP_APP" 2>/dev/null || true
codesign --force --deep --sign - "$TMP_APP"

# Copy back to repository
echo "🚚 Copying back to repository..."
rm -rf MoleSwift.app
cp -R "$TMP_APP" .

# Cleanup build directory
rm -rf "$BUILD_DIR"

echo "✅ Build complete: MoleSwift.app ($(du -sh MoleSwift.app | cut -f1))"
echo ""

# List what's in the bundle
echo "📦 Bundle contents:"
ls -la MoleSwift.app/Contents/
if [ -d "MoleSwift.app/Contents/PlugIns" ]; then
    echo ""
    echo "🧩 PlugIns (Extensions):"
    ls -la MoleSwift.app/Contents/PlugIns/
fi

echo ""
echo "🚀 To run: open MoleSwift.app"
echo "🧩 To add widget: Right-click desktop → Edit Widgets → Search 'Mole'"
