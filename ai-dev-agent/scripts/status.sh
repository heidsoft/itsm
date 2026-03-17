#!/bin/bash

# AI Developer Agent Status Script

PID_FILE="/tmp/ai-dev-agent.pid"
LOG_FILE="/tmp/ai-dev-agent.log"

echo "=== AI Developer Agent Status ==="

if [ ! -f "$PID_FILE" ]; then
    echo "Status: Not running (no PID file)"
    exit 1
fi

PID=$(cat "$PID_FILE")

if ! ps -p "$PID" > /dev/null 2>&1; then
    echo "Status: Not running (stale PID file)"
    echo "PID: $PID (process not found)"
    rm -f "$PID_FILE"
    exit 1
fi

echo "Status: Running"
echo "PID: $PID"

# Get process info
PROC_INFO=$(ps -p "$PID" -o pid,ppid,cmd,etime --no-headers 2>/dev/null)
if [ -n "$PROC_INFO" ]; then
    echo "Process Info: $PROC_INFO"
fi

# Show log tail if log file exists
if [ -f "$LOG_FILE" ]; then
    echo ""
    echo "=== Recent Logs ==="
    tail -20 "$LOG_FILE"
fi
