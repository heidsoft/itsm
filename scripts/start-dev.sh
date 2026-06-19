#!/bin/bash
set -euo pipefail

# Compatibility wrapper. The canonical development entrypoint is deploy-dev.sh.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec "$SCRIPT_DIR/deploy-dev.sh" up "$@"
