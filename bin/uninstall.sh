#!/bin/bash
# Wrapper script for uninstall functionality
# This script delegates to the actual implementation in lib/uninstall/batch.sh

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

# Source common functions
source "$ROOT_DIR/lib/core/common.sh"

# Execute the actual uninstall script
exec "$ROOT_DIR/lib/uninstall/batch.sh" "$@"
