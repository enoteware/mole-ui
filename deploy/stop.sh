#!/bin/bash
# Stop Mole Web UI server

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PID_FILE="$SCRIPT_DIR/mole-web.pid"

if [[ ! -f "$PID_FILE" ]]; then
    echo "No PID file found. Server may not be running."
    exit 0
fi

PID=$(cat "$PID_FILE")

if kill -0 "$PID" 2> /dev/null; then
    echo "Stopping Mole Web UI (PID: $PID)..."
    kill "$PID"
    rm -f "$PID_FILE"
    echo "✅ Server stopped"
else
    echo "Process $PID not running. Cleaning up PID file."
    rm -f "$PID_FILE"
fi
