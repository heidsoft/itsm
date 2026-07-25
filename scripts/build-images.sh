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
#   BUILDPLATFORM  optional target platform (e.g. linux/amd64); native by default
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

# BuildKit is required for cache mounts + inline cache.
export DOCKER_BUILDKIT=1

VERSION="${1:-${VERSION:-latest}}"
if [[ $# -gt 0 ]]; then shift; fi
REGISTRY="${1:-${REGISTRY:-}}"
if [[ $# -gt 0 ]]; then shift; fi
SELECTED_COUNT=$#
SELECTED=("$@")

if [[ ! "$VERSION" =~ ^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$ ]]; then
  echo "Invalid image version/tag: $VERSION" >&2
  exit 2
fi
if [[ -n "$REGISTRY" ]]; then
  REGISTRY="${REGISTRY%/}/"
fi

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

if ! command -v docker >/dev/null 2>&1; then
  log_error "Docker is required to build images"
  exit 1
fi

if ! docker info >/dev/null 2>&1; then
  log_error "Docker daemon is not available"
  exit 1
fi

build_one() {
  local svc="$1" ctx="$2" df="$3" target="$4" extra="$5"
  local tag="${REGISTRY}itsm-${svc}:${VERSION}"
  local -a command=(docker build --build-arg BUILDKIT_INLINE_CACHE=1)
  local -a extra_args=()

  if [[ -n "${BUILDPLATFORM:-}" ]]; then
    command+=(--platform="$BUILDPLATFORM")
  fi
  if [[ -n "$target" ]]; then
    command+=(--target "$target")
  fi
  if [[ -n "$extra" ]]; then
    read -r -a extra_args <<< "$extra"
  fi

  command+=(-f "$ctx/$df" -t "$tag")
  command+=("${extra_args[@]}")
  command+=("$ctx")

  log_info "Building ${tag} (context=${ctx}, dockerfile=${df}${target:+, target=${target}})"
  "${command[@]}"
  log_success "Built ${tag}"
}

should_build() {
  local svc="$1"
  [[ $SELECTED_COUNT -eq 0 ]] && return 0
  for s in "${SELECTED[@]}"; do
    [[ "$s" == "$svc" ]] && return 0
  done
  return 1
}

if [[ $SELECTED_COUNT -gt 0 ]]; then
  for selected in "${SELECTED[@]}"; do
    known=false
    for entry in "${ALL_SERVICES[@]}"; do
      IFS='|' read -r svc _ <<< "$entry"
      if [[ "$selected" == "$svc" ]]; then
        known=true
        break
      fi
    done
    if [[ "$known" != "true" ]]; then
      log_error "Unknown service '$selected'. Valid services: backend frontend ai-service guidance_sidecar"
      exit 2
    fi
  done
fi

for entry in "${ALL_SERVICES[@]}"; do
  IFS='|' read -r svc ctx df target extra <<< "$entry"
  if should_build "$svc"; then
    build_one "$svc" "$ctx" "$df" "$target" "$extra"
  else
    log_info "Skipping ${svc} (not selected)"
  fi
done

log_success "Done. Images tagged with version '${VERSION}'."
