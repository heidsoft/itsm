#!/bin/bash
set -euo pipefail

# Compatibility wrapper. The canonical development entrypoint is deploy-dev.sh.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

case "${1:-}" in
    --volumes|--reset|reset)
        exec "$SCRIPT_DIR/deploy-dev.sh" reset "${@:2}"
        ;;
    *)
        exec "$SCRIPT_DIR/deploy-dev.sh" down "$@"
        ;;
esac
