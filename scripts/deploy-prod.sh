#!/bin/bash
#
# ITSM Production Deployment Script v3.0
#
# Production-grade deployment following open-source best practices:
#   - 5-phase pipeline: validate → backup → build → deploy → verify
#   - Deploy lock to prevent concurrent deploys
#   - Automatic rollback on failure
#   - Timed operations with clear progress
#   - Build output suppression (clean by default, --verbose for full)
#   - Trap-based cleanup on SIGINT/SIGTERM
#   - Pre-flight security checks (no default passwords)
#   - Smoke test after deploy
#   - Backup rotation (keep last 5)
#
# Usage:
#   ./scripts/deploy-prod.sh [command] [options]
#
# Commands:
#   deploy      Full deploy (validate → backup → build → deploy → verify)
#   rollback    Rollback to previous deployment
#   backup      Backup database only
#   status      Show production service status
#   health      Run production health checks
#   logs        Tail production logs
#   down        Stop production environment
#   help        Show help
#
# Options:
#   --skip-backup    Skip database backup before deploy
#   --skip-build     Use existing images (no rebuild)
#   --dry-run        Preview without executing
#   --verbose        Show full build output
#   -h, --help       Show help
#

set -euo pipefail

# ============================================================
# Bootstrap: load shared library
# ============================================================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# ============================================================
# Constants
# ============================================================
COMPOSE_PROD="$PROJECT_ROOT/docker-compose.prod.yml"
ENV_FILE="$PROJECT_ROOT/.env.prod"
BACKUP_DIR="$PROJECT_ROOT/backups"
LOG_DIR="$PROJECT_ROOT/logs"

BACKEND_URL="http://localhost:8090"
FRONTEND_URL="http://localhost:3000"
HEALTH_PATH="/api/v1/health"

DEPLOY_STATE_DIR="$PROJECT_ROOT/.deploy"
CURRENT_STATE="$DEPLOY_STATE_DIR/current"
PREVIOUS_STATE="$DEPLOY_STATE_DIR/previous"
DEPLOY_LOCK="$DEPLOY_STATE_DIR/deploy.lock"

# ============================================================
# Parse arguments
# ============================================================
COMMAND="deploy"
SKIP_BACKUP=false
SKIP_BUILD=false
DRY_RUN=false
VERBOSE=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        deploy|rollback|backup|status|health|logs|down|help)
            COMMAND="$1"; shift ;;
        --skip-backup) SKIP_BACKUP=true; shift ;;
        --skip-build)  SKIP_BUILD=true;  shift ;;
        --dry-run)     DRY_RUN=true;     shift ;;
        --verbose)     VERBOSE=true;     shift ;;
        -h|--help)     COMMAND="help";   shift ;;
        *)
            log_error "Unknown option: $1"
            COMMAND="help"
            shift
            ;;
    esac
done

export VERBOSE

# ============================================================
# Deploy lock + trap cleanup
# ============================================================
mkdir -p "$DEPLOY_STATE_DIR" "$BACKUP_DIR" "$LOG_DIR"

acquire_lock() {
    $DRY_RUN && return 0
    deploy_lock_acquire "$DEPLOY_LOCK"
    on_exit "deploy_lock_release"
}

# ============================================================
# Utility: run or dry-run
# ============================================================
run() {
    if $DRY_RUN; then
        echo -e "  ${YELLOW}[DRY-RUN]${NC} $*"
    else
        "$@"
    fi
}

# ============================================================
# Phase 1: Pre-flight validation
# ============================================================
preflight_checks() {
    log_phase "Phase 1/5: Pre-flight Validation"
    local errors=0

    # Docker
    if ! command -v docker &>/dev/null; then
        log_error "Docker is not installed"; errors=$((errors + 1))
    elif ! docker info &>/dev/null 2>&1; then
        log_error "Docker daemon is not running"; errors=$((errors + 1))
    else
        log_success "Docker available"
    fi

    # Docker Compose
    if ! docker compose version &>/dev/null 2>&1; then
        log_error "Docker Compose v2 not available"; errors=$((errors + 1))
    else
        log_success "Docker Compose v2 available"
    fi

    # Compose file
    if [[ ! -f "$COMPOSE_PROD" ]]; then
        log_error "docker-compose.prod.yml not found"; errors=$((errors + 1))
    else
        log_success "docker-compose.prod.yml found"
    fi

    # .env.prod
    if [[ ! -f "$ENV_FILE" ]]; then
        log_error ".env.prod not found. Create: cp .env.prod.example .env.prod"
        errors=$((errors + 1))
    else
        log_success ".env.prod found"
        source_env_file "$ENV_FILE"

        # Required variables
        local required=("DB_PASSWORD" "REDIS_PASSWORD" "JWT_SECRET" "MINIO_ACCESS_KEY" "MINIO_SECRET_KEY")
        for var in "${required[@]}"; do
            if [[ -z "${!var:-}" ]]; then
                log_error "$var not set in .env.prod"; errors=$((errors + 1))
            else
                log_success "$var is set"
            fi
        done

        # Security: no default/dev values
        if ! check_prod_secrets; then
            errors=$((errors + 1))
        fi
    fi

    # Disk space
    if ! require_disk_space 2; then
        errors=$((errors + 1))
    fi

    # Ports
    for port in 8090 3000; do
        if port_in_use "$port"; then
            log_warn "Port $port is in use (will fail if container can't bind)"
        fi
    done

    if [[ $errors -gt 0 ]]; then
        log_error "Pre-flight failed with $errors error(s)"
        return 1
    fi
    log_success "All pre-flight checks passed"
}

# ============================================================
# Phase 2: Backup
# ============================================================
backup_database() {
    log_phase "Phase 2/5: Database Backup"

    if $SKIP_BACKUP; then
        log_warn "Backup skipped (--skip-backup)"
        return 0
    fi

    local timestamp; timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_file="$BACKUP_DIR/itsm_prod_${timestamp}.sql.gz"

    if container_running "itsm-postgres-prod"; then
        log_info "Creating database backup..."
        if $DRY_RUN; then
            run echo "Would backup to $backup_file"
        else
            local start; start=$(timer_start)
            docker exec itsm-postgres-prod pg_dump -U "${DB_USER:-itsm}" -d "${DB_NAME:-itsm_prod}" \
                | gzip > "$backup_file"
            local size; size=$(du -h "$backup_file" | cut -f1)
            timer_end "$start" "Database backup"
            log_success "Backup: $backup_file ($size)"

            # Rotate: keep last 5
            ls -t "$BACKUP_DIR"/itsm_prod_*.sql.gz 2>/dev/null | tail -n +6 | xargs -r rm -f
            log_info "Rotated old backups (keeping last 5)"
        fi
    else
        log_warn "PostgreSQL container not running, skipping backup"
    fi
}

# ============================================================
# Phase 3: Build images
# ============================================================
build_images() {
    log_phase "Phase 3/5: Build Images"

    if $SKIP_BUILD; then
        log_warn "Build skipped (--skip-build)"
        return 0
    fi

    # Snapshot current state for rollback
    if ! $DRY_RUN && [[ -f "$CURRENT_STATE" ]]; then
        cp "$CURRENT_STATE" "$PREVIOUS_STATE"
    fi

    # Backend
    log_step "Building backend image"
    local start; start=$(timer_start)
    if $DRY_RUN; then
        run echo "Would build itsm-backend:prod"
    else
        if [[ "$VERBOSE" == "true" ]]; then
            docker compose --env-file "$ENV_FILE" -f "$COMPOSE_PROD" build itsm-backend
        else
            docker compose --env-file "$ENV_FILE" -f "$COMPOSE_PROD" build itsm-backend 2>&1 | tail -5
        fi
    fi
    timer_end "$start" "Backend image build"
    log_success "Backend image built"

    # Frontend
    log_step "Building frontend image"
    start=$(timer_start)
    if $DRY_RUN; then
        run echo "Would build itsm-frontend:prod"
    else
        if [[ "$VERBOSE" == "true" ]]; then
            docker compose --env-file "$ENV_FILE" -f "$COMPOSE_PROD" build itsm-frontend
        else
            docker compose --env-file "$ENV_FILE" -f "$COMPOSE_PROD" build itsm-frontend 2>&1 | tail -5
        fi
    fi
    timer_end "$start" "Frontend image build"
    log_success "Frontend image built"

    # Save deployment state
    if ! $DRY_RUN; then
        {
            echo "deploy_time=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
            echo "git_commit=$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
            echo "backend_digest=$(docker inspect --format='{{.Id}}' itsm-backend:prod 2>/dev/null || echo unknown)"
            echo "frontend_digest=$(docker inspect --format='{{.Id}}' itsm-frontend:prod 2>/dev/null || echo unknown)"
        } > "$CURRENT_STATE"
    fi
}

# ============================================================
# Phase 4: Deploy services
# ============================================================
deploy_services() {
    log_phase "Phase 4/5: Deploy Services"
    local start; start=$(timer_start)

    # Pull infrastructure images (with retry)
    log_step "Pulling infrastructure images"
    if ! $DRY_RUN; then
        local pull_retries=0
        while [ $pull_retries -lt 3 ]; do
            if dc --env-file "$ENV_FILE" -f "$COMPOSE_PROD" pull postgres redis minio 2>/dev/null; then
                break
            fi
            pull_retries=$((pull_retries + 1))
            log_warn "Pull attempt $pull_retries failed, retrying..."
            sleep 5
        done
    fi

    # Start infrastructure
    log_step "Starting infrastructure (PostgreSQL, Redis, MinIO)"
    run dc --env-file "$ENV_FILE" -f "$COMPOSE_PROD" up -d postgres redis minio

    log_info "Waiting for infrastructure..."
    if ! $DRY_RUN; then
        local infra_ok=true
        for svc in itsm-postgres-prod itsm-redis-prod itsm-minio-prod; do
            if ! wait_for_container_healthy "$svc" 45; then
                log_error "$svc not healthy after 45s"
                infra_ok=false
            fi
        done
        if ! $infra_ok; then
            dc --env-file "$ENV_FILE" -f "$COMPOSE_PROD" logs --tail=30 postgres redis minio
            return 1
        fi
        log_success "Infrastructure healthy"
    fi

    # Start backend
    log_step "Starting backend service"
    run dc --env-file "$ENV_FILE" -f "$COMPOSE_PROD" up -d itsm-backend

    log_info "Waiting for backend health check (up to 120s)..."
    if ! $DRY_RUN; then
        if wait_for_http "${BACKEND_URL}${HEALTH_PATH}" "backend" 120; then
            log_success "Backend is healthy"
        else
            log_error "Backend failed to start within 120s"
            dc --env-file "$ENV_FILE" -f "$COMPOSE_PROD" logs --tail=50 itsm-backend
            return 1
        fi
    fi

    # Start frontend
    log_step "Starting frontend service"
    run dc --env-file "$ENV_FILE" -f "$COMPOSE_PROD" up -d itsm-frontend

    log_info "Waiting for frontend..."
    if ! $DRY_RUN; then
        if wait_for_http "$FRONTEND_URL" "frontend" 60; then
            log_success "Frontend is healthy"
        else
            log_error "Frontend failed to start"
            dc --env-file "$ENV_FILE" -f "$COMPOSE_PROD" logs --tail=50 itsm-frontend
            return 1
        fi
    fi

    timer_end "$start" "Service deployment"
    log_success "All services deployed"
}

# ============================================================
# Phase 5: Verify deployment
# ============================================================
verify_deployment() {
    log_phase "Phase 5/5: Verify Deployment"
    local errors=0

    # Backend
    if wait_for_http "${BACKEND_URL}${HEALTH_PATH}" "backend" 5; then
        log_success "Backend: healthy"
    else
        log_error "Backend: unreachable"; errors=$((errors + 1))
    fi

    # Frontend
    if wait_for_http "$FRONTEND_URL" "frontend" 5; then
        log_success "Frontend: healthy"
    else
        log_error "Frontend: unreachable"; errors=$((errors + 1))
    fi

    # Containers
    for c in itsm-postgres-prod itsm-redis-prod itsm-minio-prod; do
        if container_running "$c"; then
            log_success "$c: running"
        else
            log_error "$c: not running"; errors=$((errors + 1))
        fi
    done

    # Smoke test: login
    log_info "Running API smoke test..."
    local login_rc
    login_rc=$(curl -sf -X POST "${BACKEND_URL}/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"admin\",\"password\":\"${ADMIN_PASSWORD:-admin123}\"}" 2>/dev/null || echo '{"code":-1}')
    if echo "$login_rc" | grep -q '"code":0'; then
        log_success "API login: passed"
    else
        log_warn "API login: failed (password may differ from default)"
    fi

    [[ $errors -gt 0 ]] && return 1
    log_success "All verification checks passed"
}

# ============================================================
# Rollback
# ============================================================
do_rollback() {
    log_phase "Rollback"

    if [[ ! -f "$PREVIOUS_STATE" ]]; then
        log_error "No previous deployment state found for rollback"
        return 1
    fi

    # Read previous state safely (key=value format)
    local prev_deploy_time prev_git_commit
    prev_deploy_time=$(grep '^deploy_time=' "$PREVIOUS_STATE" | cut -d= -f2- || echo "unknown")
    prev_git_commit=$(grep '^git_commit=' "$PREVIOUS_STATE" | cut -d= -f2- || echo "unknown")
    log_warn "Rolling back to deployment from ${prev_deploy_time} (commit: ${prev_git_commit})..."

    log_step "Stopping current services"
    run dc --env-file "$ENV_FILE" -f "$COMPOSE_PROD" down

    cp "$PREVIOUS_STATE" "$CURRENT_STATE"

    log_step "Starting previous deployment"
    run dc --env-file "$ENV_FILE" -f "$COMPOSE_PROD" up -d

    log_info "Waiting for backend health check..."
    sleep 10
    if wait_for_http "${BACKEND_URL}${HEALTH_PATH}" "backend" 90; then
        log_success "Rollback complete - backend is healthy"
    else
        log_error "Rollback failed - backend did not become healthy"
        return 1
    fi
}

# ============================================================
# Status / health / logs
# ============================================================
show_status() {
    echo ""
    if [[ -f "$CURRENT_STATE" ]]; then
        local _deploy_time _git_commit
        _deploy_time=$(grep '^deploy_time=' "$CURRENT_STATE" | cut -d= -f2-)
        _git_commit=$(grep '^git_commit=' "$CURRENT_STATE" | cut -d= -f2-)
        print_banner "ITSM Production Environment" \
            "Deploy: ${CYAN}${_deploy_time}${NC}  Commit: ${CYAN}${_git_commit}${NC}"
    else
        print_banner "ITSM Production Environment"
    fi
    dc --env-file "$ENV_FILE" -f "$COMPOSE_PROD" ps 2>/dev/null || echo "  No containers running"
    echo ""
}

show_health() {
    log_step "Production health checks"
    echo ""

    # HTTP endpoints
    if wait_for_http "${BACKEND_URL}${HEALTH_PATH}" "backend" 5; then
        log_success "Backend: healthy"
    else
        log_error "Backend: unreachable"
    fi

    local fe_code; fe_code=$(curl -sf -o /dev/null -w '%{http_code}' "$FRONTEND_URL" 2>/dev/null || echo "000")
    if [[ "$fe_code" == 2?? || "$fe_code" == 3?? ]]; then
        log_success "Frontend: HTTP $fe_code"
    else
        log_error "Frontend: HTTP $fe_code"
    fi

    # Containers
    for c in itsm-postgres-prod itsm-redis-prod itsm-minio-prod itsm-backend-prod itsm-frontend-prod; do
        if container_running "$c"; then
            local started; started=$(docker inspect --format='{{.State.StartedAt}}' "$c" 2>/dev/null | cut -d'.' -f1 || echo "?")
            log_success "$c: running since $started"
        else
            log_error "$c: not running"
        fi
    done

    # Disk
    log_info "Docker disk usage:"
    docker system df 2>/dev/null || true
}

show_logs() {
    local service="${1:-}"
    if [[ -n "$service" ]]; then
        dc --env-file "$ENV_FILE" -f "$COMPOSE_PROD" logs -f --tail=100 "$service"
    else
        dc --env-file "$ENV_FILE" -f "$COMPOSE_PROD" logs -f --tail=100
    fi
}

stop_production() {
    log_step "Stopping production environment"
    dc --env-file "$ENV_FILE" -f "$COMPOSE_PROD" down
    log_success "Production environment stopped"
}

# ============================================================
# Full deploy pipeline
# ============================================================
full_deploy() {
    local total_start; total_start=$(timer_start)

    print_banner "ITSM Production Deployment" \
        "Time: $(date -u +%Y-%m-%dT%H:%M:%SZ)" \
        "Commit: $(git rev-parse --short HEAD 2>/dev/null || echo unknown)"

    # Acquire lock
    acquire_lock

    # Phase 1
    if ! preflight_checks; then
        log_error "Deployment aborted: pre-flight checks failed"
        exit 1
    fi

    # Phase 2
    if ! backup_database; then
        log_error "Deployment aborted: backup failed"
        exit 1
    fi

    # Phase 3
    if ! build_images; then
        log_error "Deployment aborted: build failed"
        exit 1
    fi

    # Phase 4
    if ! deploy_services; then
        log_error "Deploy failed! Attempting rollback..."
        do_rollback || true
        exit 1
    fi

    # Phase 5
    if ! verify_deployment; then
        log_error "Verification failed! Attempting rollback..."
        do_rollback || true
        exit 1
    fi

    local duration; duration=$(( $(date +%s) - total_start ))
    local mins=$((duration / 60)) secs=$((duration % 60))

    print_banner "Deployment Successful!" \
        "$(status_row "Frontend" "running" "$FRONTEND_URL")" \
        "$(status_row "Backend" "running" "$BACKEND_URL")" \
        "$(status_row "API Docs" "running" "${BACKEND_URL}/swagger")" \
        "" \
        "Duration: ${CYAN}${mins}m ${secs}s${NC}" \
        "" \
        "${YELLOW}Configure DNS and SSL for public access${NC}"
}

# ============================================================
# Help
# ============================================================
show_help() {
    print_banner "ITSM Production Deployment Script v${ITSM_SCRIPT_VERSION}"
    cat <<EOF
Usage: $(basename "$0") [command] [options]

${BOLD}Commands:${NC}
  deploy      Full deploy (validate → backup → build → deploy → verify)
  rollback    Rollback to previous deployment
  backup      Backup database only
  status      Show production service status
  health      Run production health checks
  logs [svc]  Tail production logs
  down        Stop production environment
  help        Show this help

${BOLD}Options:${NC}
  --skip-backup    Skip database backup before deploy
  --skip-build     Use existing images (no rebuild)
  --dry-run        Preview without executing
  --verbose        Show full build output
  -h, --help       Show help

${BOLD}Prerequisites:${NC}
  1. cp .env.prod.example .env.prod
  2. Set all required secrets (DB_PASSWORD, REDIS_PASSWORD, JWT_SECRET, etc.)
  3. Ensure ports 3000 and 8090 are available

${BOLD}Examples:${NC}
  $(basename "$0") deploy                    # Full production deploy
  $(basename "$0") deploy --dry-run          # Preview without deploying
  $(basename "$0") deploy --skip-build       # Deploy with existing images
  $(basename "$0") deploy --verbose          # Full build output
  $(basename "$0") rollback                  # Rollback to previous version
  $(basename "$0") health                    # Check service health
  $(basename "$0") logs itsm-backend-prod    # View backend logs
  $(basename "$0") backup                    # Backup database only
EOF
}

# ============================================================
# Entry point
# ============================================================
case "$COMMAND" in
    deploy)   full_deploy ;;
    rollback) do_rollback ;;
    backup)   backup_database ;;
    status)   show_status ;;
    health)   show_health ;;
    logs)     show_logs ;;
    down)     stop_production ;;
    help)     show_help ;;
    *)
        log_error "Unknown command: $COMMAND"
        show_help
        exit 1
        ;;
esac
