#!/bin/bash
#
# ITSM Shared Shell Library
# Common functions used by all deployment scripts.
#
# Usage:
#   SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
#   source "${SCRIPT_DIR}/lib/common.sh"
#

# Guard against double-sourcing
[[ -n "${_ITSM_COMMON_LOADED:-}" ]] && return 0
_ITSM_COMMON_LOADED=1

# ============================================================
# Project paths (set after sourcing)
# ============================================================
: "${PROJECT_ROOT:="$(cd "$(dirname "${BASH_SOURCE[1]:-${BASH_SOURCE[0]}}")/.." && pwd)"}"

# ============================================================
# Color output (no-color support via NO_COLOR env)
# ============================================================
if [[ -n "${NO_COLOR:-}" ]] || [[ ! -t 1 ]]; then
    RED='' GREEN='' YELLOW='' BLUE='' CYAN='' BOLD='' NC=''
else
    RED='\033[0;31m' GREEN='\033[0;32m' YELLOW='\033[1;33m'
    BLUE='\033[0;34m' CYAN='\033[0;36m' BOLD='\033[1m' NC='\033[0m'
fi

log_info()    { echo -e "${BLUE}[INFO]${NC}  $*"; }
log_success() { echo -e "${GREEN}[OK]${NC}    $*"; }
log_warn()    { echo -e "${YELLOW}[WARN]${NC}  $*"; }
log_error()   { echo -e "${RED}[ERROR]${NC} $*"; }
log_step()    { echo -e "${CYAN}==>${NC}   ${BOLD}$*${NC}"; }
log_phase()   { echo -e "\n${BOLD}${BLUE}═══ $* ═══${NC}\n"; }

# ============================================================
# Version
# ============================================================
ITSM_SCRIPT_VERSION="3.0.0"

# ============================================================
# Timing helpers
# ============================================================

# Start a timer. Returns start epoch on stdout.
timer_start() { date +%s; }

# Print elapsed time since the given start epoch.
# Usage: timer_end $start "Description"
timer_end() {
    local start="$1"
    shift
    local description="${*:-}"
    local end
    end=$(date +%s)
    local elapsed=$((end - start))
    local mins=$((elapsed / 60))
    local secs=$((elapsed % 60))
    if [ "$mins" -gt 0 ]; then
        log_info "${description} took ${mins}m ${secs}s"
    else
        log_info "${description} took ${secs}s"
    fi
}

# Run a command and print its elapsed time.
# Usage: timed "Building backend" command arg1 arg2
timed() {
    local description="$1"
    shift
    local start
    start=$(timer_start)
    "$@"
    local rc=$?
    if [ $rc -eq 0 ]; then
        timer_end "$start" "$description"
    fi
    return $rc
}

# ============================================================
# HTTP / health check
# ============================================================

# Wait for an HTTP endpoint to respond with 2xx/3xx.
# Usage: wait_for_http <url> <description> [max_wait_seconds]
wait_for_http() {
    local url="$1"
    local description="$2"
    local max_wait="${3:-60}"
    local interval="${4:-2}"
    local elapsed=0

    while [ $elapsed -lt $max_wait ]; do
        local http_code
        http_code=$(curl -sf -o /dev/null -w '%{http_code}' "$url" 2>/dev/null || echo "000")
        case "$http_code" in
            2??|3??) return 0 ;;
        esac
        sleep "$interval"
        elapsed=$((elapsed + interval))
    done
    return 1
}

# ============================================================
# Docker helpers
# ============================================================

# Check if a container is running by exact name.
container_running() {
    docker ps --format '{{.Names}}' 2>/dev/null | grep -q "^${1}$"
}

# Wait for a Docker container to reach "healthy" status.
# Usage: wait_for_container_healthy <container_name> [max_wait_seconds]
wait_for_container_healthy() {
    local name="$1"
    local max_wait="${2:-60}"
    local elapsed=0
    while [ $elapsed -lt $max_wait ]; do
        local health
        health=$(docker inspect --format='{{.State.Health.Status}}' "$name" 2>/dev/null || echo "missing")
        if [ "$health" = "healthy" ]; then
            return 0
        fi
        sleep 2
        elapsed=$((elapsed + 2))
    done
    return 1
}

# Check if a port is in use.
port_in_use() {
    local port="$1"
    lsof -iTCP:"$port" -sTCP:LISTEN &>/dev/null
}

# Run docker compose with BuildKit progress suppression.
# Quiet by default; set VERBOSE=true to see full output.
# Usage: dc <compose_args...>
dc() {
    if [[ "${VERBOSE:-false}" == "true" ]]; then
        docker compose "$@"
    else
        docker compose "$@" 2>&1 | {
            # Filter to show only named steps and errors
            local last_line=""
            while IFS= read -r line; do
                if echo "$line" | grep -qiE "^(#|ERROR|WARN|Container|Creating|Starting|Stopping|Built|=>)" \
                    || echo "$line" | grep -qiE "(DONE|CACHED|FAIL)" \
                    || [[ -z "$last_line" ]]; then
                    echo "$line"
                fi
                last_line="$line"
            done
        }
    fi
}

# ============================================================
# Deploy lock (production only)
# ============================================================

DEPLOY_LOCK_FILE=""

# Acquire a deploy lock file. Exits if another deploy is running.
# Usage: deploy_lock_acquire <lock_file_path>
deploy_lock_acquire() {
    DEPLOY_LOCK_FILE="$1"
    if [ -f "$DEPLOY_LOCK_FILE" ]; then
        local lock_pid
        lock_pid=$(cat "$DEPLOY_LOCK_FILE" 2>/dev/null || echo "unknown")
        if kill -0 "$lock_pid" 2>/dev/null; then
            log_error "Another deploy is already running (PID: $lock_pid)"
            log_error "Remove $DEPLOY_LOCK_FILE if this is stale."
            exit 1
        fi
        log_warn "Stale lock file found (PID $lock_pid no longer exists). Removing."
        rm -f "$DEPLOY_LOCK_FILE"
    fi
    echo $$ > "$DEPLOY_LOCK_FILE"
}

# Release the deploy lock.
deploy_lock_release() {
    if [[ -n "${DEPLOY_LOCK_FILE:-}" ]] && [[ -f "$DEPLOY_LOCK_FILE" ]]; then
        rm -f "$DEPLOY_LOCK_FILE"
    fi
}

# ============================================================
# Trap / cleanup helpers
# ============================================================

_CLEANUP_HOOKS=()

# Register a function to run on exit (even on SIGINT/SIGTERM).
# Usage: on_exit "rm -f /tmp/file"  OR  on_exit my_cleanup_func
on_exit() {
    _CLEANUP_HOOKS+=("$1")
    if [[ ${#_CLEANUP_HOOKS[@]} -eq 1 ]]; then
        trap '_run_exit_hooks' EXIT INT TERM
    fi
}

_run_exit_hooks() {
    local rc=$?
    for hook in "${_CLEANUP_HOOKS[@]}"; do
        eval "$hook" 2>/dev/null || true
    done
    exit $rc
}

# ============================================================
# Environment validation
# ============================================================

# Validate that required env vars are set. Returns 1 if any missing.
# Usage: require_env VAR1 VAR2 VAR3
require_env() {
    local missing=0
    for var in "$@"; do
        if [[ -z "${!var:-}" ]]; then
            log_error "Required environment variable $var is not set"
            missing=$((missing + 1))
        fi
    done
    return "$missing"
}

# Source an env file safely (set -a/+a to export).
# Usage: source_env_file <path>
source_env_file() {
    local file="$1"
    if [[ ! -f "$file" ]]; then
        log_error "Environment file not found: $file"
        return 1
    fi
    set -a
    # shellcheck disable=SC1090
    source "$file"
    set +a
}

# Check for insecure default values in production.
# Usage: check_prod_secrets
# Returns 1 if any default/dev values found.
check_prod_secrets() {
    local errors=0
    local dev_secrets=(
        "JWT_SECRET=dev-secret-key-placeholder-change-in-production-min-32-chars"
        "DB_PASSWORD=itsm_password_2026"
        "DB_PASSWORD=dev123"
    )
    for pair in "${dev_secrets[@]}"; do
        local var="${pair%%=*}"
        local bad_val="${pair#*=}"
        if [[ "${!var:-}" == "$bad_val" ]]; then
            log_error "$var is still set to a default/dev value"
            errors=$((errors + 1))
        fi
    done
    if [[ -z "${REDIS_PASSWORD:-}" ]]; then
        log_error "REDIS_PASSWORD is empty in production"
        errors=$((errors + 1))
    fi
    return "$errors"
}

# ============================================================
# Disk space check
# ============================================================

# Check minimum free disk space in GB.
# Usage: require_disk_space <min_gb>
require_disk_space() {
    local min_gb="$1"
    local available_kb
    available_kb=$(df -k "$PROJECT_ROOT" 2>/dev/null | awk 'NR==2 {print $4}')
    local available_gb=$((available_kb / 1024 / 1024))
    if [[ "$available_gb" -lt "$min_gb" ]]; then
        log_error "Insufficient disk space: ${available_gb}GB free (need >= ${min_gb}GB)"
        return 1
    fi
    log_success "Disk space: ${available_gb}GB free"
    return 0
}

# ============================================================
# Status display helpers
# ============================================================

# Print a colored status row.
# Usage: status_row "Backend" "running" "http://localhost:8090"
status_row() {
    local name="$1" status="$2" endpoint="$3"
    case "$status" in
        running)  echo -e "  ${GREEN}●${NC} ${name}: ${GREEN}running${NC}  ${CYAN}${endpoint}${NC}" ;;
        starting) echo -e "  ${YELLOW}●${NC} ${name}: ${YELLOW}starting${NC}  ${CYAN}${endpoint}${NC}" ;;
        *)        echo -e "  ${RED}●${NC} ${name}: ${RED}stopped${NC}   ${CYAN}${endpoint}${NC}" ;;
    esac
}

# Print a summary banner.
# Usage: print_banner "Title" [extra_lines...]
print_banner() {
    local title="$1"; shift
    echo ""
    echo -e "${BOLD}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${BOLD}  ${title}${NC}"
    echo -e "${BOLD}═══════════════════════════════════════════════════════════${NC}"
    for line in "$@"; do
        echo -e "  $line"
    done
    echo -e "${BOLD}═══════════════════════════════════════════════════════════${NC}"
    echo ""
}
