#!/bin/bash

# AI Developer Agent Start Script

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
AGENT_DIR="$PROJECT_ROOT/ai-dev-agent"
PID_FILE="/tmp/ai-dev-agent.pid"
LOG_FILE="/tmp/ai-dev-agent.log"

# Default config path
CONFIG_PATH="$AGENT_DIR/config.yaml"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -c|--config)
            CONFIG_PATH="$2"
            shift 2
            ;;
        -l|--log-level)
            LOG_LEVEL="$2"
            shift 2
            ;;
        -d|--daemon)
            DAEMON=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Check if config exists
if [ ! -f "$CONFIG_PATH" ]; then
    echo "Error: Config file not found: $CONFIG_PATH"
    exit 1
fi

# Check if already running
if [ -f "$PID_FILE" ]; then
    PID=$(cat "$PID_FILE")
    if ps -p "$PID" > /dev/null 2>&1; then
        echo "AI Developer Agent is already running (PID: $PID)"
        exit 1
    fi
fi

# Build arguments
ARGS="--config $CONFIG_PATH"
if [ -n "$LOG_LEVEL" ]; then
    ARGS="$ARGS --log-level $LOG_LEVEL"
fi

# Create log directory
mkdir -p "$(dirname "$LOG_FILE")"

# Change to agent directory
cd "$AGENT_DIR"

if [ "$DAEMON" = true ]; then
    echo "Starting AI Developer Agent in daemon mode..."
    go run main.go $ARGS >> "$LOG_FILE" 2>&1 &
    echo $! > "$PID_FILE"
    echo "AI Developer Agent started (PID: $(cat "$PID_FILE"))"
    echo "Log file: $LOG_FILE"
else
    echo "Starting AI Developer Agent..."
    exec go run main.go $ARGS
fi
