#!/bin/bash
# Wrapper script for uninstall functionality
# Handles both interactive mode and --path argument for non-interactive uninstall

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

# Source common functions
source "$ROOT_DIR/lib/core/common.sh"
source "$ROOT_DIR/lib/uninstall/batch.sh"

# Parse arguments
app_path=""
force_rescan=false
debug_mode=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --path)
            app_path="$2"
            shift 2
            ;;
        --force-rescan)
            force_rescan=true
            shift
            ;;
        --debug)
            debug_mode=true
            shift
            ;;
        *)
            shift
            ;;
    esac
done

# If --path was provided, do non-interactive uninstall
if [[ -n "$app_path" ]]; then
    # Validate the path exists
    if [[ ! -d "$app_path" ]]; then
        log_error "Application not found: $app_path"
        exit 1
    fi

    # Extract app info
    app_name=$(basename "$app_path" .app)
    bundle_id="unknown"
    if [[ -e "$app_path/Contents/Info.plist" ]]; then
        bundle_id=$(defaults read "$app_path/Contents/Info.plist" CFBundleIdentifier 2>/dev/null || echo "unknown")
    fi

    # Create selected_apps array in the format expected by batch_uninstall_applications
    # Format: index|path|name|bundle_id|size_kb|extra
    selected_apps=("1|$app_path|$app_name|$bundle_id|0|")

    # Run the batch uninstall
    batch_uninstall_applications
    exit $?
fi

# Interactive mode - source and run the main uninstall UI
source "$ROOT_DIR/lib/uninstall/main.sh"
