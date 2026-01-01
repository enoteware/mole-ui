#!/bin/bash
# Wrapper script for clean functionality
# This script delegates to the actual implementation in lib/clean/

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

# Source common functions
source "$ROOT_DIR/lib/core/common.sh"

# Execute the actual clean script
exec "$ROOT_DIR/lib/clean/batch.sh" "$@"
