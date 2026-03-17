#!/bin/bash

# AI Developer Agent Stop Script

PID_FILE="/tmp/ai-dev-agent.pid"

if [ ! -f "$PID_FILE" ]; then
    echo "PID file not found. Is the agent running?"
    exit 1
fi

PID=$(cat "$PID_FILE")

if ! ps -p "$PID" > /dev/null 2>&1; then
    echo "Process $PID is not running. Cleaning up stale PID file."
    rm -f "$PID_FILE"
    exit 1
fi

echo "Stopping AI Developer Agent (PID: $PID)..."
kill "$PID"

# Wait for graceful shutdown
for i in {1..10}; do
    if ! ps -p "$PID" > /dev/null 2>&1; then
        echo "Agent stopped successfully"
        rm -f "$PID_FILE"
        exit 0
    fi
    sleep 1
done

# Force kill if still running
echo "Graceful shutdown timed out, forcing..."
kill -9 "$PID" 2>/dev/null
rm -f "$PID_FILE"

echo "Agent stopped (forced)"
