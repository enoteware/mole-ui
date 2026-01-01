#!/bin/bash
# Wrapper script for optimize functionality
# This script delegates to the actual implementation in lib/optimize/

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

# Source common functions
source "$ROOT_DIR/lib/core/common.sh"

# Execute the actual optimize script
exec "$ROOT_DIR/lib/optimize/batch.sh" "$@"
