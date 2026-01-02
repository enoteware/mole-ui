#!/bin/bash
# Create dummy test apps for Mole uninstall testing

set -e

echo "Creating test apps..."

# Function to create a minimal .app bundle
create_app() {
    local name="$1"
    local location="$2"
    local owner="$3"
    local bundle_id="com.moletest.${name// /}"

    local app_path="$location/${name}.app"

    echo "  Creating: $app_path"

    # Create bundle structure
    mkdir -p "$app_path/Contents/MacOS"
    mkdir -p "$app_path/Contents/Resources"

    # Create Info.plist
    cat > "$app_path/Contents/Info.plist" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>${name}</string>
    <key>CFBundleIdentifier</key>
    <string>${bundle_id}</string>
    <key>CFBundleName</key>
    <string>${name}</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleVersion</key>
    <string>1.0</string>
</dict>
</plist>
EOF

    # Create dummy executable
    cat > "$app_path/Contents/MacOS/${name}" << 'EOF'
#!/bin/bash
echo "Test app"
EOF
    chmod +x "$app_path/Contents/MacOS/${name}"

    # Add some bulk (10MB of random data to make size visible)
    dd if=/dev/zero of="$app_path/Contents/Resources/data.bin" bs=1024 count=10240 2> /dev/null

    # Set ownership if root requested
    if [[ "$owner" == "root" ]]; then
        sudo chown -R root:wheel "$app_path"
    fi
}

# Create user-owned test apps (no sudo needed to delete)
create_app "MoleTest User App 1" "/Applications" "user"
create_app "MoleTest User App 2" "/Applications" "user"

# Create root-owned test apps (sudo needed to delete)
create_app "MoleTest System App 1" "/Applications" "root"
create_app "MoleTest System App 2" "/Applications" "root"
create_app "MoleTest System App 3" "/Applications" "root"

# Create some fake support files that the uninstaller should find
echo "Creating fake support files..."
mkdir -p ~/Library/Application\ Support/MoleTest\ User\ App\ 1
dd if=/dev/zero of=~/Library/Application\ Support/MoleTest\ User\ App\ 1/data.db bs=1024 count=1024 2> /dev/null

mkdir -p ~/Library/Caches/com.moletest.MoleTestUserApp2
dd if=/dev/zero of=~/Library/Caches/com.moletest.MoleTestUserApp2/cache.db bs=1024 count=512 2> /dev/null

mkdir -p ~/Library/Preferences
touch ~/Library/Preferences/com.moletest.MoleTestSystemApp1.plist

echo ""
echo "✅ Created 5 test apps:"
echo "   - MoleTest User App 1 (user-owned, ~10MB)"
echo "   - MoleTest User App 2 (user-owned, ~10MB)"
echo "   - MoleTest System App 1 (root-owned, ~10MB) - needs sudo"
echo "   - MoleTest System App 2 (root-owned, ~10MB) - needs sudo"
echo "   - MoleTest System App 3 (root-owned, ~10MB) - needs sudo"
echo ""
echo "Test by:"
echo "1. Restart Mole.app"
echo "2. Select all 'MoleTest' apps"
echo "3. Delete - should only prompt for password ONCE"
echo ""
echo "To clean up manually: sudo rm -rf /Applications/MoleTest*.app"
