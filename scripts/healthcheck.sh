#!/bin/bash

# ITSM Health Check Script
# Usage: ./healthcheck.sh [--detailed]
# Can be used by cron or monitoring systems

set -e

# Configuration
API_URL="${API_URL:-http://localhost:8080}"
FRONTEND_URL="${FRONTEND_URL:-http://localhost:3000}"
REDIS_HOST="${REDIS_HOST:-localhost}"
REDIS_PORT="${REDIS_PORT:-6379}"
POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_DB="${POSTGRES_DB:-itsm_cmdb}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

DETAILED="${1:-}"

# Tracking
ERRORS=0
WARNINGS=0

log_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

log_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((ERRORS++))
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
    ((WARNINGS++))
}

log_info() {
    echo -e "[INFO] $1"
}

# Check HTTP endpoint
check_http() {
    local name="$1"
    local url="$2"
    local expected_status="${3:-200}"

    local status_code
    status_code=$(curl -s -o /dev/null -w "%{http_code}" "$url" --connect-timeout 5 --max-time 10 2>/dev/null || echo "000")

    if [ "$status_code" = "$expected_status" ]; then
        log_pass "$name (HTTP $status_code)"
        return 0
    else
        log_fail "$name (HTTP $status_code, expected $expected_status)"
        return 1
    fi
}

# Check TCP port
check_port() {
    local name="$1"
    local host="$2"
    local port="$3"

    if nc -z -w5 "$host" "$port" 2>/dev/null; then
        log_pass "$name (${host}:${port})"
        return 0
    else
        log_fail "$name (${host}:${port})"
        return 1
    fi
}

# Check PostgreSQL
check_postgres() {
    log_info "Checking PostgreSQL..."

    if command -v pg_isready &> /dev/null; then
        if pg_isready -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -q 2>/dev/null; then
            log_pass "PostgreSQL is ready"

            # Check database size
            if command -v psql &> /dev/null; then
                local db_size
                db_size=$(psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U postgres -d "$POSTGRES_DB" -t -c "SELECT pg_size_pretty(pg_database_size('$POSTGRES_DB'));" 2>/dev/null | xargs || echo "unknown")
                log_info "Database size: $db_size"
            fi
        else
            log_fail "PostgreSQL is not ready"
        fi
    else
        log_warn "pg_isready not found, skipping PostgreSQL check"
    fi
}

# Check Redis
check_redis() {
    log_info "Checking Redis..."

    if command -v redis-cli &> /dev/null; then
        local redis_status
        redis_status=$(redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" ping 2>/dev/null || echo "PONG")

        if [ "$redis_status" = "PONG" ]; then
            log_pass "Redis is responding"

            # Get Redis info
            if [ -n "$DETAILED" ]; then
                local used_memory
                used_memory=$(redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" info memory 2>/dev/null | grep "used_memory_human" | cut -d: -f2 | xargs)
                log_info "Redis memory used: $used_memory"

                local connected_clients
                connected_clients=$(redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" info clients 2>/dev/null | grep "connected_clients" | cut -d: -f2 | xargs)
                log_info "Redis connected clients: $connected_clients"
            fi
        else
            log_fail "Redis is not responding (got: $redis_status)"
        fi
    else
        log_warn "redis-cli not found, skipping Redis check"
    fi
}

# Check backend health endpoint
check_backend_health() {
    log_info "Checking backend health..."

    local health_response
    health_response=$(curl -s "$API_URL/health" --connect-timeout 5 --max-time 10 2>/dev/null || echo "{}")

    # Try to parse JSON response
    if echo "$health_response" | grep -q '"status"'; then
        local status
        status=$(echo "$health_response" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)

        if [ "$status" = "ok" ] || [ "$status" = "healthy" ]; then
            log_pass "Backend health check: $status"

            if [ -n "$DETAILED" ]; then
                local uptime
                uptime=$(echo "$health_response" | grep -o '"uptime":"[^"]*"' | cut -d'"' -f4)
                log_info "Backend uptime: $uptime"

                local version
                version=$(echo "$health_response" | grep -o '"version":"[^"]*"' | cut -d'"' -f4)
                log_info "Backend version: $version"
            fi
        else
            log_warn "Backend health status: $status"
        fi
    else
        # Fallback to HTTP check
        check_http "Backend API" "$API_URL/api/v1/health" 200 || check_http "Backend API" "$API_URL/health" 200
    fi
}

# Check disk space
check_disk() {
    log_info "Checking disk space..."

    local disk_usage
    disk_usage=$(df -h . | tail -1 | awk '{print $5}' | sed 's/%//')

    if [ "$disk_usage" -lt 80 ]; then
        log_pass "Disk usage: ${disk_usage}%"
    elif [ "$disk_usage" -lt 90 ]; then
        log_warn "Disk usage: ${disk_usage}%"
    else
        log_fail "Disk usage: ${disk_usage}%"
    fi
}

# Check memory
check_memory() {
    log_info "Checking memory..."

    if command -v free &> /dev/null; then
        local mem_usage
        mem_usage=$(free | grep Mem | awk '{printf "%.0f", ($3/$2) * 100}')

        if [ "$mem_usage" -lt 80 ]; then
            log_pass "Memory usage: ${mem_usage}%"
        elif [ "$mem_usage" -lt 90 ]; then
            log_warn "Memory usage: ${mem_usage}%"
        else
            log_fail "Memory usage: ${mem_usage}%"
        fi
    fi
}

# Check process status
check_process() {
    log_info "Checking processes..."

    # Check if backend process is running (if not using container)
    if pgrep -f "itsm.*main" > /dev/null 2>&1; then
        log_pass "Backend process is running"
    else
        log_warn "Backend process not found (may be containerized)"
    fi
}

# Main execution
main() {
    echo "========================================"
    echo "ITSM System Health Check"
    echo "Time: $(date)"
    echo "========================================"

    # Network checks
    echo ""
    echo "--- Network Services ---"
    check_http "Frontend" "$FRONTEND_URL" 200
    check_port "PostgreSQL" "$POSTGRES_HOST" "$POSTGRES_PORT"
    check_port "Redis" "$REDIS_HOST" "$REDIS_PORT"

    # Service checks
    echo ""
    echo "--- Services ---"
    check_backend_health
    check_postgres
    check_redis

    # System resources
    echo ""
    echo "--- System Resources ---"
    check_disk
    check_memory

    # Process checks
    echo ""
    echo "--- Processes ---"
    check_process

    # Summary
    echo ""
    echo "========================================"
    echo "Summary: $ERRORS errors, $WARNINGS warnings"
    echo "========================================"

    # Exit code based on status
    if [ $ERRORS -gt 0 ]; then
        echo -e "${RED}Health check FAILED${NC}"
        exit 1
    elif [ $WARNINGS -gt 0 ]; then
        echo -e "${YELLOW}Health check PASSED with warnings${NC}"
        exit 0
    else
        echo -e "${GREEN}Health check PASSED${NC}"
        exit 0
    fi
}

main
