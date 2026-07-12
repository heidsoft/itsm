#!/usr/bin/env bash
#
# ITSM — Unified image builder
#
# Builds all (or selected) ITSM service images with BuildKit caching in one
# place, so the many one-off wrapper scripts can be removed. This is the
# single source of truth for "how do I build the Docker images".
#
# Usage:
#   ./scripts/build-images.sh [version] [registry/] [service ...]
#
# Examples:
#   ./scripts/build-images.sh                 # build :latest, all services
#   ./scripts/build-images.sh v1.2.0          # tag all images v1.2.0
#   ./scripts/build-images.sh latest "" backend frontend
#   REGISTRY=ghcr.io/heidsoft/ ./scripts/build-images.sh v1.2.0
#
# Environment overrides:
#   GOPROXY        Go module proxy   (default: https://goproxy.cn,direct)
#   NPM_REGISTRY   npm registry      (default: https://registry.npmjs.org)
#   TORCH_INDEX    torch wheel index (default: CPU wheels)
#   REGISTRY       image registry prefix (e.g. ghcr.io/heidsoft/)
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

# BuildKit is required for cache mounts + inline cache.
export DOCKER_BUILDKIT=1

VERSION="${1:-latest}"
REGISTRY="${2:-${REGISTRY:-}}"
shift 2 2>/dev/null || true
SELECTED=("$@")

# service -> "context|dockerfile|target|build-args..."
ALL_SERVICES=(
  "backend|itsm-backend|Dockerfile.prod||--build-arg GOPROXY=${GOPROXY:-https://goproxy.cn,direct}"
  "frontend|itsm-frontend|Dockerfile|production|--build-arg NPM_REGISTRY=${NPM_REGISTRY:-https://registry.npmjs.org} --build-arg NEXT_PUBLIC_ENABLE_AI=${NEXT_PUBLIC_ENABLE_AI:-true}"
  "ai-service|itsm-ai-service|Dockerfile||--build-arg TORCH_INDEX=${TORCH_INDEX:-https://download.pytorch.org/whl/cpu}"
  "guidance_sidecar|itsm-backend/guidance_sidecar|Dockerfile||--build-arg TORCH_INDEX=${TORCH_INDEX:-https://download.pytorch.org/whl/cpu}"
)

log_info()  { echo -e "\033[0;34m[INFO]\033[0m  $*"; }
log_success(){ echo -e "\033[0;32m[OK]\033[0m    $*"; }
log_error() { echo -e "\033[0;31m[ERROR]\033[0m $*"; }

build_one() {
  local svc="$1" ctx="$2" df="$3" target="$4" extra="$5"
  local tag="${REGISTRY}itsm-${svc}:${VERSION}"
  log_info "Building ${tag} (context=${ctx}, dockerfile=${df}${target:+, target=${target}})"
  # shellcheck disable=SC2086
  docker build \
    --platform="${BUILDPLATFORM:-linux/amd64}" \
    --build-arg BUILDKIT_INLINE_CACHE=1 \
    ${target:+--target "$target"} \
    -f "$ctx/$df" \
    -t "$tag" \
    $extra \
    "$ctx"
  log_success "Built ${tag}"
}

should_build() {
  local svc="$1"
  [[ ${#SELECTED[@]} -eq 0 ]] && return 0
  for s in "${SELECTED[@]}"; do
    [[ "$s" == "$svc" ]] && return 0
  done
  return 1
}

for entry in "${ALL_SERVICES[@]}"; do
  IFS='|' read -r svc ctx df target extra <<< "$entry"
  if should_build "$svc"; then
    build_one "$svc" "$ctx" "$df" "$target" "$extra"
  else
    log_info "Skipping ${svc} (not selected)"
  fi
done

log_success "Done. Images tagged with version '${VERSION}'."
