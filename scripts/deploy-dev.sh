#!/bin/bash
#
# ITSM Development Environment Deployment Script v3.0
#
# A production-quality dev script following open-source best practices:
#   - Shared library for DRY code
#   - Clean build output (suppress Docker noise by default)
#   - Timed operations
#   - Trap-based cleanup on SIGINT/SIGTERM
#   - Idempotent (safe to re-run)
#   - First-time init mode
#   - Self-diagnostic (doctor) command
#
# Usage:
#   ./scripts/deploy-dev.sh [command] [options]
#
# Commands:
#   up          Start all services (default)
#   down        Stop all services
#   restart     Restart all services
#   status      Show service status
#   logs        Tail service logs
#   health      Run health checks
#   init        First-time setup (install tools + start)
#   doctor      Diagnose common issues
#   reset       Stop and remove all containers + volumes
#   help        Show help
#
# Options:
#   --local       Force local development mode
#   --docker      Force Docker Compose mode
#   --skip-deps   Skip dependency installation
#   --no-build    Skip Docker image rebuild
#   --verbose     Show full Docker build output
#   -h, --help    Show help
#

set -euo pipefail

# Enable BuildKit so the multi-stage Dockerfiles can use cache mounts +
# inline layer caching (faster, smaller rebuilds).
export DOCKER_BUILDKIT=1

# ============================================================
# Bootstrap: load shared library
# ============================================================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# ============================================================
# Project-specific constants
# ============================================================
BACKEND_DIR="$PROJECT_ROOT/itsm-backend"
FRONTEND_DIR="$PROJECT_ROOT/itsm-frontend"
COMPOSE_DEV="$PROJECT_ROOT/docker-compose.dev.yml"
COMPOSE_OOB="$PROJECT_ROOT/docker-compose.yml"

BACKEND_URL="http://localhost:8090"
FRONTEND_URL="http://localhost:3000"
HEALTH_PATH="/api/v1/health"

DEFAULT_ADMIN_USER="admin"
DEFAULT_ADMIN_PASS="admin123"

LOG_DIR="$PROJECT_ROOT/logs"
PID_DIR="$PROJECT_ROOT/.pids"

# ============================================================
# Parse arguments
# ============================================================
COMMAND="up"
FORCE_MODE=""
SKIP_DEPS=false
NO_BUILD=false
VERBOSE=false
LOG_SERVICE=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        up|down|restart|status|logs|health|init|doctor|reset|help)
            COMMAND="$1"; shift ;;
        --local)      FORCE_MODE="local";   shift ;;
        --docker)     FORCE_MODE="docker";  shift ;;
        --skip-deps)  SKIP_DEPS=true;       shift ;;
        --no-build)   NO_BUILD=true;        shift ;;
        --verbose)    VERBOSE=true;         shift ;;
        -h|--help)    COMMAND="help";       shift ;;
        *)
            if [[ "$COMMAND" == "logs" && -z "$LOG_SERVICE" ]]; then
                LOG_SERVICE="$1"
                shift
            else
                log_error "Unknown option: $1"
                COMMAND="help"
                shift
            fi
            ;;
    esac
done

# Export for dc() helper
export VERBOSE

# ============================================================
# Mode detection
# ============================================================
detect_mode() {
    if [[ -n "$FORCE_MODE" ]]; then
        echo "$FORCE_MODE"
        return
    fi
    if [[ -f "$COMPOSE_DEV" ]] && command -v docker &>/dev/null && docker info &>/dev/null 2>&1; then
        echo "docker"
    else
        echo "local"
    fi
}

# ============================================================
# Cleanup on exit
# ============================================================
mkdir -p "$LOG_DIR" "$PID_DIR"

cleanup() {
    # Remove stale PID files if we wrote them
    :
}
on_exit "cleanup"

# ============================================================
# Docker Compose mode
# ============================================================
ensure_dev_env() {

    # Create required directories with correct permissions
    mkdir -p "$PROJECT_ROOT/logs" "$PROJECT_ROOT/uploads"
    chmod -R 777 "$PROJECT_ROOT/logs" "$PROJECT_ROOT/uploads"

    if [[ -f "$PROJECT_ROOT/.env" ]]; then
        return 0
    fi

    if [[ -f "$PROJECT_ROOT/.env.dev.example" ]]; then
        cp "$PROJECT_ROOT/.env.dev.example" "$PROJECT_ROOT/.env"
        log_info "Created .env from .env.dev.example"
    elif [[ -f "$PROJECT_ROOT/.env.example" ]]; then
        cp "$PROJECT_ROOT/.env.example" "$PROJECT_ROOT/.env"
        log_info "Created .env from .env.example"
    fi
}

docker_up() {
    local build_flag=""
    $NO_BUILD || build_flag="--build"

    log_step "Starting development environment (Docker Compose)"

    ensure_dev_env
    log_info "Cleaning up stale ITSM containers..."
    docker rm -f $(docker ps -a | grep -E "itsm-(backend|frontend|postgres|redis|minio|init)-dev" | awk '{print $1}') 2>/dev/null || true
    



    # Phase 1: Infrastructure
    log_step "[1/3] Starting PostgreSQL + Redis + MinIO"
    local start; start=$(timer_start)
    dc -f "$COMPOSE_DEV" --profile dev up -d postgres redis minio
    local infra_ok=true
    for svc in itsm-postgres-dev itsm-redis-dev itsm-minio-dev; do
        if ! wait_for_container_healthy "$svc" 45; then
            log_error "$svc not healthy after 45s"
            infra_ok=false
        fi
    done
    if ! $infra_ok; then
        dc -f "$COMPOSE_DEV" --profile dev logs --tail=30 postgres redis minio
        return 1
    fi
    timer_end "$start" "Infrastructure"

    # Phase 2: Backend
    log_step "[2/3] Building and starting backend"
    start=$(timer_start)
    dc -f "$COMPOSE_DEV" --profile dev up -d $build_flag itsm-backend
    log_info "Waiting for backend health check..."
    if wait_for_http "${BACKEND_URL}${HEALTH_PATH}" "backend" 120; then
        timer_end "$start" "Backend build + start"
        log_success "Backend is healthy"
    else
        log_error "Backend failed to start. Run with --verbose for full logs."
        log_info "Quick check: docker compose -f docker-compose.dev.yml --profile dev logs --tail=30 itsm-backend"
        return 1
    fi

    # Phase 3: Frontend
    log_step "[3/3] Building and starting frontend"
    start=$(timer_start)
    dc -f "$COMPOSE_DEV" --profile dev up -d $build_flag itsm-frontend
    log_info "Waiting for frontend..."
    if wait_for_http "$FRONTEND_URL" "frontend" 90; then
        timer_end "$start" "Frontend build + start"
        log_success "Frontend is healthy"
    else
        log_warn "Frontend may need more time. Check: docker compose -f docker-compose.dev.yml --profile dev logs itsm-frontend"
    fi

    print_summary
}

docker_down() {
    log_step "Stopping development environment"
    dc -f "$COMPOSE_DEV" --profile dev down --remove-orphans
    log_success "All services stopped"
}

docker_reset() {
    log_step "Resetting development environment (removes volumes)"
    log_warn "This will DELETE all data in PostgreSQL and Redis"
    read -rp "Continue? [y/N] " confirm
    if [[ "$confirm" =~ ^[Yy]$ ]]; then
        dc -f "$COMPOSE_DEV" --profile dev down -v --remove-orphans
        # Force clean up all remaining ITSM containers and volumes
                docker rm -f $(docker ps -a | grep -E "itsm-" | awk '{print $1}') 2>/dev/null || true
                docker volume rm $(docker volume ls | grep -E "itsm-" | awk '{print $2}') 2>/dev/null || true

        log_success "Environment reset complete"
    else
        log_info "Reset cancelled"
    fi
}

docker_logs() {
    local service="${1:-}"
    if [[ -n "$service" ]]; then
        dc -f "$COMPOSE_DEV" --profile dev logs -f "$service"
    else
        dc -f "$COMPOSE_DEV" --profile dev logs -f
    fi
}

# ============================================================
# Local development mode
# ============================================================
local_up() {
    log_step "Starting development environment (Local)"
    local total_start; total_start=$(timer_start)

    check_prerequisites
    start_infrastructure
    start_backend_local
    start_frontend_local
    run_health_checks

    timer_end "$total_start" "Total startup"
    print_summary
}

check_prerequisites() {
    log_info "Checking prerequisites..."
    local missing=()

    command -v go &>/dev/null   || missing+=("go")
    command -v node &>/dev/null || missing+=("node")
    command -v pnpm &>/dev/null || missing+=("pnpm")
    command -v docker &>/dev/null || missing+=("docker")

    if [[ ${#missing[@]} -gt 0 ]]; then
        log_error "Missing prerequisites: ${missing[*]}"
        log_info "Install missing tools or use --docker mode"
        exit 1
    fi

    # Version checks
    local go_ver node_ver
    go_ver=$(go version 2>/dev/null | grep -oE 'go[0-9]+\.[0-9]+' | tr -d 'go' || echo "?")
    node_ver=$(node --version 2>/dev/null | tr -d 'v' || echo "?")
    log_success "Go ${go_ver}, Node ${node_ver}, pnpm $(pnpm --version 2>/dev/null || echo '?'), Docker $(docker --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1 || echo '?')"
}

start_infrastructure() {
    log_step "Starting infrastructure (PostgreSQL + Redis)"

    # PostgreSQL
    if container_running "itsm-postgres-dev"; then
        log_info "PostgreSQL already running"
    elif port_in_use 5432; then
        log_warn "Port 5432 in use (external PostgreSQL?)"
    else
        docker run -d --name itsm-postgres-dev \
            -e POSTGRES_DB=itsm \
            -e POSTGRES_USER=itsm_user \
            -e POSTGRES_PASSWORD=dev123 \
            -p 5432:5432 \
            --health-cmd="pg_isready -U itsm_user -d itsm" \
            --health-interval=5s \
            --health-timeout=3s \
            --health-retries=10 \
            --restart unless-stopped \
            pgvector/pgvector:pg17
        if wait_for_container_healthy "itsm-postgres-dev" 45; then
            log_success "PostgreSQL ready on port 5432"
        else
            log_error "PostgreSQL failed to start"
            return 1
        fi
    fi

    # Redis
    if container_running "itsm-redis-dev"; then
        log_info "Redis already running"
    elif port_in_use 6379; then
        log_warn "Port 6379 in use (external Redis?)"
    else
        docker run -d --name itsm-redis-dev \
            -p 6379:6379 \
            --health-cmd="redis-cli ping" \
            --health-interval=5s \
            --health-timeout=3s \
            --health-retries=10 \
            --restart unless-stopped \
            redis:7-alpine
        if wait_for_container_healthy "itsm-redis-dev" 30; then
            log_success "Redis ready on port 6379"
        else
            log_error "Redis failed to start"
            return 1
        fi
    fi
}

start_backend_local() {
    log_step "Starting backend service"

    if wait_for_http "${BACKEND_URL}${HEALTH_PATH}" "backend" 2; then
        log_info "Backend already running on port 8090"
        return 0
    fi

    if port_in_use 8090; then
        log_error "Port 8090 in use but health check failed. Kill stale process first."
        return 1
    fi

    cd "$BACKEND_DIR"

    # Environment defaults (only set if not already defined)
    export DB_HOST="${DB_HOST:-localhost}"
    export DB_PORT="${DB_PORT:-5432}"
    export DB_USER="${DB_USER:-itsm_user}"
    export DB_PASSWORD="${DB_PASSWORD:-dev123}"
    export DB_NAME="${DB_NAME:-itsm}"
    export DB_SSLMODE="${DB_SSLMODE:-disable}"
    export REDIS_HOST="${REDIS_HOST:-localhost}"
    export REDIS_PORT="${REDIS_PORT:-6379}"
    export REDIS_PASSWORD="${REDIS_PASSWORD:-}"
    export REDIS_DB="${REDIS_DB:-0}"
    export JWT_SECRET="${JWT_SECRET:-dev-secret-key-placeholder-change-in-production-min-32-chars}"
    export LOG_LEVEL="${LOG_LEVEL:-debug}"
    export GIN_MODE="${GIN_MODE:-debug}"
    export SERVER_ENV="${SERVER_ENV:-development}"
    export MINIO_ENDPOINT="${MINIO_ENDPOINT:-http://localhost:9000}"
    export MINIO_ROOT_USER="${MINIO_ROOT_USER:-minioadmin}"
    export MINIO_ROOT_PASSWORD="${MINIO_ROOT_PASSWORD:-minioadmin123}"
    export MINIO_BUCKET="${MINIO_BUCKET:-itsm-uploads}"

    local start; start=$(timer_start)
    local backend_bin="$PID_DIR/itsm-backend-dev"
    log_info "Building Go backend..."
    go build -o "$backend_bin" main.go

    log_info "Starting Go backend..."
    nohup "$backend_bin" > "$LOG_DIR/backend.log" 2>&1 &
    local pid=$!
    echo "$pid" > "$PID_DIR/backend.pid"
    log_info "Backend PID: $pid"

    if wait_for_http "${BACKEND_URL}${HEALTH_PATH}" "backend" 60; then
        timer_end "$start" "Backend startup"
        log_success "Backend is healthy"
    else
        log_error "Backend failed to start. Tail: $LOG_DIR/backend.log"
        tail -20 "$LOG_DIR/backend.log" 2>/dev/null || true
        return 1
    fi
}

start_frontend_local() {
    log_step "Starting frontend service"

    if wait_for_http "$FRONTEND_URL" "frontend" 2; then
        log_info "Frontend already running on port 3000"
        return 0
    fi

    if port_in_use 3000; then
        log_error "Port 3000 in use but frontend not responding"
        return 1
    fi

    cd "$FRONTEND_DIR"

    if [[ "$SKIP_DEPS" == "false" ]] && [[ ! -d "node_modules" ]]; then
        log_info "Installing frontend dependencies..."
        pnpm install --ignore-scripts
        # Approve builds for native modules (sharp, etc.) to avoid pnpm blocking on startup
        pnpm approve-builds --silent 2>/dev/null || true
    fi

    export NEXT_PUBLIC_API_URL="${NEXT_PUBLIC_API_URL:-http://localhost:8090}"
    export NODE_ENV="${NODE_ENV:-development}"

    local start; start=$(timer_start)
    log_info "Starting Next.js dev server..."
    # Use direct node_modules path to bypass pnpm wrapper overhead on each reload
    nohup ./node_modules/.bin/next dev --port 3000 > "$LOG_DIR/frontend.log" 2>&1 &
    local pid=$!
    echo "$pid" > "$PID_DIR/frontend.pid"
    log_info "Frontend PID: $pid"

    if wait_for_http "$FRONTEND_URL" "frontend" 90; then
        timer_end "$start" "Frontend startup"
        log_success "Frontend is healthy"
    else
        log_warn "Frontend may need more time. Check: $LOG_DIR/frontend.log"
    fi
}

local_down() {
    log_step "Stopping local services"

    # Backend
    if [[ -f "$PID_DIR/backend.pid" ]]; then
        local pid; pid=$(cat "$PID_DIR/backend.pid")
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid" 2>/dev/null || true
            log_success "Backend stopped (PID: $pid)"
        fi
        rm -f "$PID_DIR/backend.pid"
    else
        pkill -f "go run main.go" 2>/dev/null && log_success "Backend stopped" || true
    fi

    # Frontend
    if [[ -f "$PID_DIR/frontend.pid" ]]; then
        local pid; pid=$(cat "$PID_DIR/frontend.pid")
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid" 2>/dev/null || true
            log_success "Frontend stopped (PID: $pid)"
        fi
        rm -f "$PID_DIR/frontend.pid"
    else
        pkill -f "next dev --port 3000" 2>/dev/null && log_success "Frontend stopped" || true
        pkill -f "pnpm dev" 2>/dev/null && log_success "Frontend stopped" || true
    fi

    # Optionally clean infrastructure
    if [[ "${1:-}" == "--clean" ]]; then
        log_info "Stopping infrastructure containers..."
        docker stop itsm-postgres-dev itsm-redis-dev 2>/dev/null || true
        docker rm itsm-postgres-dev itsm-redis-dev 2>/dev/null || true
        log_success "Infrastructure containers removed"
    fi
}

# ============================================================
# Health checks
# ============================================================
run_health_checks() {
    log_step "Running health checks"
    local all_ok=true

    # Backend
    if wait_for_http "${BACKEND_URL}${HEALTH_PATH}" "backend" 3; then
        log_success "Backend: healthy"
    else
        log_error "Backend: unreachable"
        all_ok=false
    fi

    # Frontend
    if wait_for_http "$FRONTEND_URL" "frontend" 3; then
        log_success "Frontend: healthy"
    else
        log_error "Frontend: unreachable"
        all_ok=false
    fi

    # PostgreSQL
    if container_running "itsm-postgres-dev"; then
        if docker exec itsm-postgres-dev pg_isready -U itsm_user -d itsm &>/dev/null; then
            log_success "PostgreSQL: healthy"
        else
            log_error "PostgreSQL: not ready"
            all_ok=false
        fi
    elif port_in_use 5432; then
        log_success "PostgreSQL: external (port 5432 in use)"
    else
        log_warn "PostgreSQL: not running"
    fi

    # Redis
    if container_running "itsm-redis-dev"; then
        if docker exec itsm-redis-dev redis-cli ping &>/dev/null; then
            log_success "Redis: healthy"
        else
            log_error "Redis: not ready"
            all_ok=false
        fi
    elif port_in_use 6379; then
        log_success "Redis: external (port 6379 in use)"
    else
        log_warn "Redis: not running"
    fi

    $all_ok && log_success "All health checks passed" || log_warn "Some health checks failed"
}

# ============================================================
# Init (first-time setup)
# ============================================================
cmd_init() {
    log_phase "First-Time Setup"

    # 1. Check Docker
    if ! command -v docker &>/dev/null; then
        log_error "Docker is required. Install: https://docs.docker.com/get-docker/"
        exit 1
    fi
    log_success "Docker: $(docker --version | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)"

    # 2. Create .env
    ensure_dev_env

    [[ -f "$PROJECT_ROOT/.env" ]] && log_success ".env ready"

    # 3. Ensure log directory
    mkdir -p "$LOG_DIR"
    log_success "Log directory ready"

    # 4. Install frontend deps
    if [[ ! -d "$FRONTEND_DIR/node_modules" ]]; then
        log_info "Installing frontend dependencies..."
        cd "$FRONTEND_DIR"
        if ! command -v pnpm &>/dev/null; then
            npm install -g pnpm@9.0.0
        fi
        pnpm install --ignore-scripts
        log_success "Frontend dependencies installed"
    else
        log_info "Frontend dependencies already installed"
    fi

    # 5. Download Go modules
    if [[ ! -d "$BACKEND_DIR/vendor" ]]; then
        log_info "Downloading Go modules..."
        cd "$BACKEND_DIR"
        go mod download
        log_success "Go modules downloaded"
    fi

    # 6. Start services
    log_info "Starting services..."
    local mode; mode=$(detect_mode)
    if [[ "$mode" == "docker" ]]; then
        docker_up
    else
        local_up
    fi
}

# ============================================================
# Doctor (self-diagnostic)
# ============================================================
cmd_doctor() {
    log_phase "ITSM Development Environment Diagnostic"

    local issues=0

    # Docker
    echo -e "${BOLD}Docker${NC}"
    if command -v docker &>/dev/null; then
        if docker info &>/dev/null 2>&1; then
            log_success "Docker daemon running"
        else
            log_error "Docker installed but daemon not running"
            issues=$((issues + 1))
        fi
    else
        log_error "Docker not installed"
        issues=$((issues + 1))
    fi

    # Docker Compose
    echo ""
    echo -e "${BOLD}Docker Compose${NC}"
    if dc version &>/dev/null 2>&1; then
        log_success "Docker Compose available (v1/v2 auto-detected)"
    else
        log_error "Docker Compose not available (need docker-compose v1 or 'docker compose' v2 plugin)"
        issues=$((issues + 1))
    fi

    # Ports
    echo ""
    echo -e "${BOLD}Ports${NC}"
    for port_desc in "8090:Backend" "3000:Frontend" "5432:PostgreSQL" "6379:Redis"; do
        local port="${port_desc%%:*}"
        local name="${port_desc##*:}"
        if port_in_use "$port"; then
            log_warn "Port $port ($name) in use"
        else
            log_success "Port $port ($name) available"
        fi
    done

    # Containers
    echo ""
    echo -e "${BOLD}Containers${NC}"
    for container in itsm-postgres-dev itsm-redis-dev itsm-backend-dev itsm-frontend-dev; do
        if container_running "$container"; then
            log_success "$container: running"
        else
            log_info "$container: not running"
        fi
    done

    # Health endpoints
    echo ""
    echo -e "${BOLD}Health Endpoints${NC}"
    if wait_for_http "${BACKEND_URL}${HEALTH_PATH}" "backend" 3; then
        log_success "Backend: healthy"
    else
        log_warn "Backend: unreachable"
    fi
    if wait_for_http "$FRONTEND_URL" "frontend" 3; then
        log_success "Frontend: healthy"
    else
        log_warn "Frontend: unreachable"
    fi

    # Disk space
    echo ""
    echo -e "${BOLD}Disk Space${NC}"
    require_disk_space 2 || issues=$((issues + 1))

    # Summary
    echo ""
    if [[ $issues -eq 0 ]]; then
        log_success "No issues found"
    else
        log_error "$issues issue(s) found. Fix before proceeding."
    fi
    return "$issues"
}

# ============================================================
# Status and display
# ============================================================
print_summary() {
    local backend_status="stopped"
    local frontend_status="stopped"

    wait_for_http "${BACKEND_URL}${HEALTH_PATH}" "backend" 3 && backend_status="running"
    wait_for_http "$FRONTEND_URL" "frontend" 3 && frontend_status="running"

    print_banner "ITSM Development Environment" \
        "$(status_row "Backend" "$backend_status" "$BACKEND_URL")" \
        "$(status_row "Frontend" "$frontend_status" "$FRONTEND_URL")" \
        "" \
        "${CYAN}Login:${NC} ${DEFAULT_ADMIN_USER} / ${DEFAULT_ADMIN_PASS}" \
        "${CYAN}API Docs:${NC} ${BACKEND_URL}/swagger"
}

local_infra_status() {
    local container="$1"
    local port="$2"

    if container_running "$container"; then
        echo "running"
    elif port_in_use "$port"; then
        echo "external"
    else
        echo "stopped"
    fi
}

show_status() {
    local mode; mode=$(detect_mode)
    echo ""
    echo -e "${BOLD}ITSM Development Environment [${mode} mode]${NC}"
    echo "─────────────────────────────────────────"

    if [[ "$mode" == "docker" ]]; then
        dc -f "$COMPOSE_DEV" --profile dev ps 2>/dev/null || echo "  No containers running"
    else
        # Backend
        if wait_for_http "${BACKEND_URL}${HEALTH_PATH}" "backend" 3; then
            status_row "Backend" "running" "$BACKEND_URL"
        else
            status_row "Backend" "stopped" "$BACKEND_URL"
        fi
        # Frontend
        if wait_for_http "$FRONTEND_URL" "frontend" 3; then
            status_row "Frontend" "running" "$FRONTEND_URL"
        else
            status_row "Frontend" "stopped" "$FRONTEND_URL"
        fi
        # Infrastructure
        status_row "PostgreSQL" "$(local_infra_status "itsm-postgres-dev" 5432)" "localhost:5432"
        status_row "Redis" "$(local_infra_status "itsm-redis-dev" 6379)" "localhost:6379"
    fi
    echo ""
}

show_logs() {
    local mode; mode=$(detect_mode)
    local service="${1:-}"

    if [[ "$mode" == "docker" ]]; then
        docker_logs "$service"
    else
        echo "Tailing logs (Ctrl+C to exit)"
        echo "─────────────────────────────────────────"
        [[ -f "$LOG_DIR/backend.log" ]] && tail -f "$LOG_DIR/backend.log" &
        [[ -f "$LOG_DIR/frontend.log" ]] && tail -f "$LOG_DIR/frontend.log" &
        wait
    fi
}

# ============================================================
# Help
# ============================================================
show_help() {
    print_banner "ITSM Development Deployment Script v${ITSM_SCRIPT_VERSION}"
    cat <<EOF
Usage: $(basename "$0") [command] [options]

${BOLD}Commands:${NC}
  up          Start all services (default)
  down        Stop all services
  restart     Restart all services
  status      Show service status
  logs [svc]  Tail service logs (e.g. logs itsm-backend)
  health      Run health checks
  init        First-time setup (install deps + start)
  doctor      Diagnose common issues
  reset       Remove all containers and volumes
  help        Show this help

${BOLD}Options:${NC}
  --local       Force local development mode
  --docker      Force Docker Compose mode
  --skip-deps   Skip dependency installation
  --no-build    Skip Docker image rebuild (faster restart)
  --verbose     Show full Docker build output
  -h, --help    Show help

${BOLD}Quick Start:${NC}
  ./scripts/deploy-dev.sh init          # First time: install + start
  ./scripts/deploy-dev.sh up            # Start (auto-detect mode)
  ./scripts/deploy-dev.sh up --docker   # Force Docker mode
  ./scripts/deploy-dev.sh up --no-build # Restart without rebuild
  ./scripts/deploy-dev.sh doctor        # Diagnose issues
  ./scripts/deploy-dev.sh down          # Stop everything

${BOLD}Access:${NC}
  Frontend:  http://localhost:3000
  Backend:   http://localhost:8090
  API Docs:  http://localhost:8090/swagger
  Login:     admin / admin123
EOF
}

# ============================================================
# Main
# ============================================================
main() {
    local mode; mode=$(detect_mode)

    case "$COMMAND" in
        up)
            if [[ "$mode" == "docker" ]]; then docker_up
            else local_up; fi ;;
        down)
            if [[ "$mode" == "docker" ]]; then docker_down
            else local_down "$@"; fi ;;
        restart)
            if [[ "$mode" == "docker" ]]; then docker_down && docker_up
            else local_down && sleep 2 && local_up; fi ;;
        status)  show_status ;;
        logs)    show_logs "$LOG_SERVICE" ;;
        health)  run_health_checks ;;
        init)    cmd_init ;;
        doctor)  cmd_doctor ;;
        reset)
            if [[ "$mode" == "docker" ]]; then docker_reset
            else local_down "--clean"; fi ;;
        help)    show_help ;;
        *)
            log_error "Unknown command: $COMMAND"
            show_help
            exit 1
            ;;
    esac
}

main
