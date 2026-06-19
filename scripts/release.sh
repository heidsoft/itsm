#!/bin/bash
#
# ITSM Release Script
#
# Generate release artifacts locally (equivalent to GitHub Actions release workflow)
# Usage:
#   ./scripts/release.sh [version]    # e.g., ./scripts/release.sh v1.0.0
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
RELEASE_DIR="$PROJECT_ROOT/release"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m' # No Color

log_info() { echo -e "${BLUE}[INFO]${NC} $*"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $*"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }

print_banner() {
    echo -e "${BLUE}"
    echo "═══════════════════════════════════════════════════════════"
    echo "  ITSM Release Script"
    echo "═══════════════════════════════════════════════════════════${NC}"
}

usage() {
    print_banner
    cat <<EOF

${BOLD}Usage:${NC}
  $(basename "$0") [version]

${BOLD}Examples:${NC}
  $(basename "$0") v1.0.0          # Release version 1.0.0
  $(basename "$0") v1.0.0-rc.1    # Release candidate

${BOLD}This script generates:${NC}
  - Backend binaries (linux/darwin, amd64/arm64)
  - Frontend build
  - Docker Compose production files
  - SHA256 checksums

${BOLD}Output directory:${NC}
  $RELEASE_DIR/

EOF
}

# Parse arguments
VERSION="${1:-}"
if [[ -z "$VERSION" ]]; then
    usage
    exit 1
fi

# Validate version format
if ! [[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9]+)?$ ]]; then
    log_error "Invalid version format: $VERSION"
    log_error "Expected format: v1.0.0 or v1.0.0-rc.1"
    exit 1
fi

# Remove 'v' prefix for internal use
VERSION_NOV="${VERSION#v}"

print_banner
echo ""
echo "  Version: ${GREEN}$VERSION${NC}"
echo "  Output:  ${GREEN}$RELEASE_DIR/${VERSION}${NC}"
echo ""
echo "═══════════════════════════════════════════════════════════"
echo ""

# Check prerequisites
log_info "Checking prerequisites..."

# Docker
if ! command -v docker &>/dev/null; then
    log_error "Docker is not installed"
    exit 1
fi

# Node.js
if ! command -v node &>/dev/null; then
    log_error "Node.js is not installed"
    exit 1
fi

# Go
if ! command -v go &>/dev/null; then
    log_error "Go is not installed"
    exit 1
fi

log_success "Prerequisites check passed"

# Create release directory
RELEASE_VERSION_DIR="$RELEASE_DIR/$VERSION"
rm -rf "$RELEASE_VERSION_DIR"
mkdir -p "$RELEASE_VERSION_DIR"/{backends,frontend,docker}

# ============================================================
# Build Backends
# ============================================================
log_info "Building backend binaries..."

BUILD_PLATFORMS=(
    "linux:amd64"
    "linux:arm64"
    "darwin:amd64"
    "darwin:arm64"
    "windows:amd64"
)

for PLATFORM in "${BUILD_PLATFORMS[@]}"; do
    IFS=':' read -r GOOS GOARCH <<< "$PLATFORM"
    OUTPUT_NAME="itsm-${GOOS}-${GOARCH}"

    if [[ "$GOOS" == "windows" ]]; then
        OUTPUT_NAME="$OUTPUT_NAME.exe"
    fi

    log_info "  Building for $GOOS/$GOARCH..."

    (
        cd "$PROJECT_ROOT/itsm-backend"
        CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build \
            -ldflags="-s -w" \
            -o "$RELEASE_VERSION_DIR/backends/$OUTPUT_NAME" \
            ./main.go
    )

    # Create checksum
    if [[ "$GOOS" == "windows" ]]; then
        shasum -a 256 "$RELEASE_VERSION_DIR/backends/$OUTPUT_NAME" > "$RELEASE_VERSION_DIR/backends/$OUTPUT_NAME.sha256"
    else
        sha256sum "$RELEASE_VERSION_DIR/backends/$OUTPUT_NAME" > "$RELEASE_VERSION_DIR/backends/$OUTPUT_NAME.sha256"
    fi
done

log_success "Backend binaries built"

# ============================================================
# Build Frontend
# ============================================================
log_info "Building frontend..."

cd "$PROJECT_ROOT/itsm-frontend"

# Install dependencies if needed
if [[ ! -d "node_modules" ]]; then
    log_info "  Installing frontend dependencies..."
    npm ci --legacy-peer-deps
fi

# Build
NEXT_PUBLIC_API_URL=http://localhost:8090 \
NEXT_PUBLIC_ENABLE_AI=true \
npm run build

# Prepare standalone build
mkdir -p "$RELEASE_VERSION_DIR/frontend"
cp -r .next/standalone/* "$RELEASE_VERSION_DIR/frontend/" 2>/dev/null || {
    # Fallback if standalone export not configured
    log_warn "Standalone export not found, using standard .next build"
    cp -r .next "$RELEASE_VERSION_DIR/frontend/.next"
    cp -r public "$RELEASE_VERSION_DIR/frontend/public"
}

# Copy package.json for reference
cp package.json "$RELEASE_VERSION_DIR/frontend/"

log_success "Frontend built"

# ============================================================
# Package Docker Compose Files
# ============================================================
log_info "Preparing Docker Compose files..."

cp "$PROJECT_ROOT/docker-compose.prod.yml" "$RELEASE_VERSION_DIR/docker/"
cp "$PROJECT_ROOT/.env.prod.example" "$RELEASE_VERSION_DIR/docker/.env.prod.example"

# Create docker build script
cat > "$RELEASE_VERSION_DIR/docker/build-images.sh" << 'DOCKER_SCRIPT'
#!/bin/bash
# Build production Docker images
set -e

VERSION="${1:-latest}"
REGISTRY="${2:-}"

echo "Building Docker images for version $VERSION..."

# Build backend
echo "Building backend..."
docker build -t "${REGISTRY}itsm-backend:${VERSION}" \
    --build-arg BUILDKIT_INLINE_CACHE=1 \
    -f Dockerfile.prod ../itsm-backend

# Build frontend
echo "Building frontend..."
docker build -t "${REGISTRY}itsm-frontend:${VERSION}" \
    --build-arg NODE_ENV=production \
    -f Dockerfile ../itsm-frontend

echo "Done! Images created:"
echo "  ${REGISTRY}itsm-backend:${VERSION}"
echo "  ${REGISTRY}itsm-frontend:${VERSION}"
DOCKER_SCRIPT
chmod +x "$RELEASE_VERSION_DIR/docker/build-images.sh"

log_success "Docker files prepared"

# ============================================================
# Create Archives
# ============================================================
log_info "Creating release archives..."

cd "$RELEASE_VERSION_DIR"

# Backend binaries archive
tar -czf "itsm-${VERSION_NOV}-backends.tar.gz" backends/
sha256sum "itsm-${VERSION_NOV}-backends.tar.gz" > "itsm-${VERSION_NOV}-backends.tar.gz.sha256"

# Frontend archive
tar -czf "itsm-${VERSION_NOV}-frontend.tar.gz" frontend/
sha256sum "itsm-${VERSION_NOV}-frontend.tar.gz" > "itsm-${VERSION_NOV}-frontend.tar.gz.sha256"

# Docker files archive
tar -czf "itsm-${VERSION_NOV}-docker.tar.gz" docker/
sha256sum "itsm-${VERSION_NOV}-docker.tar.gz" > "itsm-${VERSION_NOV}-docker.tar.gz.sha256"

# Full release archive
tar -czf "itsm-${VERSION_NOV}-full.tar.gz" ./
sha256sum "itsm-${VERSION_NOV}-full.tar.gz" > "itsm-${VERSION_NOV}-full.tar.gz.sha256"

# Remove original directories (keep only archives)
rm -rf backends frontend docker

log_success "Archives created"

# ============================================================
# Summary
# ============================================================
echo ""
echo "═══════════════════════════════════════════════════════════"
log_success "Release $VERSION generated successfully!"
echo "═══════════════════════════════════════════════════════════"
echo ""
echo "  Output directory: ${BLUE}$RELEASE_VERSION_DIR/${NC}"
echo ""
echo "  Archives:"
ls -lh "$RELEASE_VERSION_DIR"/*.tar.gz 2>/dev/null | awk '{print "    " $9 " (" $5 ")"}'
echo ""
echo "  To create a GitHub release:"
echo "    1. Go to: https://github.com/heidsoft/itsm/releases/new"
echo "    2. Tag version: $VERSION"
echo "    3. Upload files from: $RELEASE_VERSION_DIR"
echo ""
echo "  Or push tag to trigger GitHub Actions:"
echo "    git tag $VERSION"
echo "    git push origin $VERSION"
echo ""
echo "═══════════════════════════════════════════════════════════"
