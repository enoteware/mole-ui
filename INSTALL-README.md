# Installing Mole on MacBook Pro

## Quick Install (Easy Method)

1. **Download these 2 files to your MacBook Pro:**
   - `Mole-v1.0.0-20251230-161343.dmg`
   - `install-mole.sh`

2. **Put both files in the same folder** (e.g., Downloads)

3. **Open Terminal and run:**
   ```bash
   cd ~/Downloads  # or wherever you saved the files
   chmod +x install-mole.sh
   ./install-mole.sh
   ```

4. **Follow the prompts:**
   - Enter your password when asked (for removing old app)
   - Drag Mole.app to Applications when the DMG opens
   - Press ENTER

5. **Verify it worked:**
   - Check the window title says: **"Mole v1.0.0 - System Cleaner (NEW UI)"**
   - Look for version number "v1.0.0" in top-right corner
   - Click "Analyze" tab and verify "Select Drive to Analyze" section appears

## What the Script Does

The `install-mole.sh` script automatically:
- ✅ Verifies DMG integrity (MD5 hash check)
- ✅ Kills old Mole processes
- ✅ Removes old application completely
- ✅ Clears all caches (Mole + WebKit)
- ✅ Guides you through installation
- ✅ Verifies new UI is installed
- ✅ Launches the app

## Troubleshooting

### Still seeing old UI after installation?

**Option 1: Restart and retry**
```bash
sudo shutdown -r now
# After restart:
cd ~/Downloads
./install-mole.sh
```

**Option 2: Manual verification**
```bash
# Check if new UI is in the binary
strings /Applications/Mole.app/Contents/MacOS/web-go | grep "NEW UI"
# Should output: <title>Mole v1.0.0 - System Cleaner (NEW UI)</title>

# If above shows nothing, the installation failed - run script again
```

**Option 3: Nuclear option**
```bash
# Clear EVERYTHING
sudo pkill -9 Mole
sudo rm -rf /Applications/Mole.app
rm -rf ~/Library/Caches/com.* ~/Library/WebKit/*
sudo reboot
# After reboot, run ./install-mole.sh again
```

## Manual Install (If Script Fails)

If the script doesn't work, you can install manually:

1. **Cleanup:**
   ```bash
   sudo pkill -9 Mole
   sudo rm -rf /Applications/Mole.app
   rm -rf ~/Library/Caches/com.enoteware.mole*
   rm -rf ~/Library/WebKit/com.enoteware.mole*
   ```

2. **Verify DMG hash:**
   ```bash
   md5 Mole-v1.0.0-20251230-161343.dmg
   # Must show: ce014d0c4b43f23c76a6ef51e8239fcf
   ```

3. **Install:**
   ```bash
   open Mole-v1.0.0-20251230-161343.dmg
   # Drag to Applications, click Replace if prompted
   ```

4. **Launch:**
   ```bash
   open /Applications/Mole.app
   ```

## Support

If you continue having issues:
1. Take a screenshot of the app window (showing title bar)
2. Open Terminal and run: `strings /Applications/Mole.app/Contents/MacOS/web-go | grep "NEW UI"`
3. Share both with the developer

---

**Expected window title:** `Mole v1.0.0 - System Cleaner (NEW UI)`
**Expected DMG MD5:** `ce014d0c4b43f23c76a6ef51e8239fcf`
