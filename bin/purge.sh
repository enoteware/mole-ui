#!/bin/bash
# Wrapper script for purge functionality
# This script delegates to the actual implementation in lib/manage/

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

# Source common functions
source "$ROOT_DIR/lib/core/common.sh"

# Execute the actual purge script
exec "$ROOT_DIR/lib/manage/purge.sh" "$@"

