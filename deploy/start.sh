#!/bin/bash
# Mole Web UI - Native Mac Mini Server Deployment
#
# This runs the Mole web UI directly on macOS for full system access.
# Recommended for Mac Mini servers.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
BINARY="$ROOT_DIR/bin/web-go"
PID_FILE="$ROOT_DIR/deploy/mole-web.pid"
LOG_FILE="$ROOT_DIR/deploy/mole-web.log"

# Load environment if exists
if [[ -f "$SCRIPT_DIR/.env" ]]; then
    export $(grep -v '^#' "$SCRIPT_DIR/.env" | xargs)
fi

# Default config
export MOLE_PORT="${MOLE_PORT:-8080}"
export MOLE_HOST="${MOLE_HOST:-0.0.0.0}"
export MOLE_NO_OPEN="${MOLE_NO_OPEN:-1}"
export MOLE_DIR="$ROOT_DIR"

# Check for auth (required for network access)
if [[ -z "$MOLE_AUTH_USER" ]] || [[ -z "$MOLE_AUTH_PASS" ]]; then
    echo "⚠️  Warning: No authentication configured!"
    echo "   Set MOLE_AUTH_USER and MOLE_AUTH_PASS in deploy/.env"
    echo ""
fi

# Build if needed
if [[ ! -f "$BINARY" ]] || [[ "$ROOT_DIR/cmd/web/main.go" -nt "$BINARY" ]]; then
    echo "Building web server..."
    if ! command -v go &> /dev/null; then
        echo "Error: Go is not installed. Install with: brew install go"
        exit 1
    fi
    cd "$ROOT_DIR"
    go build -o "$BINARY" ./cmd/web/
    echo "Build complete!"
fi

# Stop existing process if running
if [[ -f "$PID_FILE" ]]; then
    OLD_PID=$(cat "$PID_FILE")
    if kill -0 "$OLD_PID" 2>/dev/null; then
        echo "Stopping existing server (PID: $OLD_PID)..."
        kill "$OLD_PID"
        sleep 1
    fi
    rm -f "$PID_FILE"
fi

# Start server
echo ""
echo "🐭 Starting Mole Web UI..."
echo "   Port: $MOLE_PORT"
echo "   Host: $MOLE_HOST"
echo "   Logs: $LOG_FILE"
echo ""

nohup "$BINARY" >> "$LOG_FILE" 2>&1 &
echo $! > "$PID_FILE"

sleep 1

if kill -0 $(cat "$PID_FILE") 2>/dev/null; then
    echo "✅ Server started (PID: $(cat "$PID_FILE"))"

    # Get local IP
    LOCAL_IP=$(ipconfig getifaddr en0 2>/dev/null || echo "localhost")
    echo ""
    echo "   Access the dashboard at:"
    echo "   http://${LOCAL_IP}:${MOLE_PORT}"
    echo ""
else
    echo "❌ Failed to start server. Check $LOG_FILE for errors."
    exit 1
fi
