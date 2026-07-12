#!/usr/bin/env bash
#
# Compatibility wrapper. The canonical image builder is now
# scripts/build-images.sh, which builds every ITSM service with BuildKit
# caching. This keeps the old command name working for anyone who relied on it.
#
# Usage:
#   ./scripts/build-frontend-image.sh [version] [registry/]
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

VERSION="${1:-latest}"
REGISTRY="${2:-}"

exec "$SCRIPT_DIR/build-images.sh" "$VERSION" "$REGISTRY" frontend
