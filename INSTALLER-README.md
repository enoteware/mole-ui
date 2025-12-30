# Mole Mac App - Installation Guide

Mole is now packaged as a simple Mac app! No terminal commands needed.

## What's Included

- **Mole.app** - Native macOS application (7MB)
- **Mole-Installer.dmg** - Drag-and-drop installer (6.9MB)

## For Family Members: How to Install

1. **Download** the `Mole-Installer.dmg` file
2. **Double-click** the DMG file to open it
3. **Drag** the Mole app to the Applications folder
4. **Close** the installer window
5. **Open** Mole from your Applications folder

That's it! When you open Mole, it will:
- Start the Mole web server in the background
- Open your web browser to the Mole dashboard
- Show a notification that it's running

## Using Mole

- **First time**: Double-click Mole in Applications
- **Already running**: Just open http://127.0.0.1:8081 in your browser
- **Close**: Quit from your browser (the server keeps running in background)
- **Stop completely**: Run this command in Terminal: `pkill web-go`

## Features

- ✅ Clean system junk files
- ✅ Uninstall applications completely
- ✅ Analyze disk usage
- ✅ Optimize system performance
- ✅ Purge unnecessary files

## Technical Details

**How it works:**
- The .app is a lightweight wrapper (just 7MB!)
- Runs a local web server on port 8081
- Stores logs in `~/Library/Application Support/Mole/`
- Only accessible from your Mac (localhost only for security)
- No authentication required (safe since it's localhost only)

**System Requirements:**
- macOS 11.0 or later
- ARM64 or Intel processor
- ~20MB free disk space

## Distributing to Family

You can share the Mole-Installer.dmg file by:
- **AirDrop**: Right-click the DMG > Share > AirDrop
- **iCloud**: Upload to iCloud Drive and share the link
- **Email**: Attach the DMG (it's only 6.9MB)
- **USB Drive**: Copy the DMG to a USB stick

## For Developers: Building from Source

If you want to rebuild the app:

```bash
# 1. Build the Go binary
go build -o bin/web-go ./cmd/web/

# 2. Recreate the .app bundle
mkdir -p Mole.app/Contents/{MacOS,Resources}
cp bin/web-go Mole.app/Contents/MacOS/
cp Mole.app/Contents/MacOS/Mole ./
chmod +x Mole.app/Contents/MacOS/{Mole,web-go}

# 3. Create the DMG
mkdir -p dmg-build
cp -R Mole.app dmg-build/
ln -s /Applications dmg-build/Applications
hdiutil create -volname "Mole Installer" -srcfolder dmg-build -ov -format UDZO Mole-Installer.dmg
rm -rf dmg-build
```

## Security Note

The first time you open the app, macOS might show a security warning because it's not signed with an Apple Developer certificate. To open it:

1. Right-click (or Control-click) the app
2. Choose "Open"
3. Click "Open" in the dialog

You only need to do this once. After that, you can open it normally.

## Adding a Custom Icon

The app currently uses the default macOS app icon. To add a custom icon, see:
`Mole.app/Contents/Resources/README.md`

## Troubleshooting

**App won't open:**
- Right-click > Open (don't double-click the first time)
- Check `~/Library/Application Support/Mole/server.log` for errors

**Port already in use:**
- Another app might be using port 8081
- Stop other Mole instances: `pkill web-go`

**Browser doesn't open:**
- Manually visit: http://127.0.0.1:8081
- Check if server is running: `ps aux | grep web-go`

## Support

For issues or questions, check the logs at:
`~/Library/Application Support/Mole/server.log`

---

**File Locations:**
- App: `Mole.app` (7MB)
- Installer: `Mole-Installer.dmg` (6.9MB)
- Logs: `~/Library/Application Support/Mole/server.log`
- PID: `~/Library/Application Support/Mole/server.pid`
