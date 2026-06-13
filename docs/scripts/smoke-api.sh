#!/bin/bash
#
# ITSM API Smoke Test Script
# 测试矩阵: specs/001-role-based-testing/contracts/api-smoke-matrix.md
#
# Exit codes:
#   0 - all tests passed
#   1 - one or more tests failed
#   2 - login failed (critical)
#

set -uo pipefail

# 配置
API_BASE="${API_BASE:-http://localhost:8090}"
JWT_SECRET="${JWT_SECRET:-dev-secret}"
ADMIN_USER="${ADMIN_USER:-admin}"
ADMIN_PASS="${ADMIN_PASS:-admin123}"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 失败计数
fails=0

# 日志函数
log_info() { echo -e "${GREEN}[INFO]${NC} $*"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_fail() { echo -e "${RED}[FAIL]${NC} $*"; }

# 检查函数
check() {
    local name="$1"
    local expected="$2"
    local actual="$3"

    if [[ "$actual" == "$expected" ]]; then
        log_info "$name: PASS"
        return 0
    else
        log_fail "$name: expected '$expected', got '$actual'"
        ((fails++))
        return 1
    fi
}

check_contains() {
    local name="$1"
    local expected="$2"
    local actual="$3"

    if [[ "$actual" == *"$expected"* ]]; then
        log_info "$name: PASS"
        return 0
    else
        log_fail "$name: expected to contain '$expected', got '$actual'"
        ((fails++))
        return 1
    fi
}

# 登录获取 token
get_token() {
    local user="$1"
    local pass="$2"

    local response
    response=$(curl -s -X POST "${API_BASE}/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$user\",\"password\":\"$pass\"}")

    local token
    token=$(echo "$response" | jq -r '.data.access_token // empty')

    if [[ -z "$token" ]]; then
        log_fail "Login failed for $user: $response"
        echo "$response" >&2
        return 1
    fi

    echo "$token"
}

# API 调用封装
api_get() {
    local token="$1"
    local path="$2"

    curl -s -X GET "${API_BASE}${path}" \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json"
}

api_post() {
    local token="$1"
    local path="$2"
    local data="$3"

    curl -s -X POST "${API_BASE}${path}" \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json" \
        -d "$data"
}

# 主测试流程
main() {
    log_info "=== ITSM API Smoke Test ==="
    log_info "API Base: $API_BASE"

    # Step 1: 登录
    log_info "Step 1: Login..."
    local admin_token
    admin_token=$(get_token "$ADMIN_USER" "$ADMIN_PASS")
    if [[ -z "$admin_token" ]]; then
        log_fail "Critical: Cannot login, exiting..."
        exit 2
    fi
    log_info "Login successful"

    # Step 2: 健康检查
    log_info "Step 2: Health Check..."
    local health
    health=$(curl -s "${API_BASE}/api/v1/health")
    check_contains "health" "ok" "$health" || true

    # Step 3: GA Readiness
    log_info "Step 3: GA Readiness..."
    local ga_ready
    ga_ready=$(api_get "$admin_token" "/api/v1/readiness/ga")
    check_contains "ga_readiness" "ready" "$ga_ready" || true
    local modules_count
    modules_count=$(echo "$ga_ready" | jq -r '.data.modules | length' 2>/dev/null || echo "0")
    check "modules_count" "12" "$modules_count" || true

    # Step 4: Tenant
    log_info "Step 4: Tenant API..."
    local tenants
    tenants=$(api_get "$admin_token" "/api/v1/tenants")
    check_contains "tenants" "code" "$tenants" || true

    # Step 5: Users
    log_info "Step 5: User API..."
    local users
    users=$(api_get "$admin_token" "/api/v1/users")
    check_contains "users" "code" "$users" || true

    # Step 6: Roles
    log_info "Step 6: Roles API..."
    local roles
    roles=$(api_get "$admin_token" "/api/v1/roles")
    check_contains "roles" "code" "$roles" || true

    # Step 7: Menus
    log_info "Step 7: Menus API..."
    local menus
    menus=$(api_get "$admin_token" "/api/v1/auth/menus")
    check_contains "menus" "code" "$menus" || true

    # Step 8: Tickets CRUD
    log_info "Step 8: Ticket Create..."
    local ticket_resp
    ticket_resp=$(api_post "$admin_token" "/api/v1/tickets" '{"title":"Smoke Test Ticket","priority":"low","category":"general"}')
    check_contains "ticket_create" "code" "$ticket_resp" || true

    # Step 9: Ticket List
    log_info "Step 9: Ticket List..."
    local tickets
    tickets=$(api_get "$admin_token" "/api/v1/tickets")
    check_contains "ticket_list" "code" "$tickets" || true

    # Step 10: Service Catalog
    log_info "Step 10: Service Catalog..."
    local catalog
    catalog=$(api_get "$admin_token" "/api/v1/service-catalogs")
    check_contains "service_catalog" "code" "$catalog" || true

    # Step 11: SLA Definitions
    log_info "Step 11: SLA Definitions..."
    local sla
    sla=$(api_get "$admin_token" "/api/v1/sla/definitions")
    check_contains "sla_definitions" "code" "$sla" || true

    # Step 12: Approval Workflows
    log_info "Step 12: Approval Workflows..."
    local approvals
    approvals=$(api_get "$admin_token" "/api/v1/approval-workflows")
    check_contains "approval_workflows" "code" "$approvals" || true

    # Step 13: Configuration Items
    log_info "Step 13: Configuration Items..."
    local ci
    ci=$(api_get "$admin_token" "/api/v1/configuration-items")
    check_contains "configuration_items" "code" "$ci" || true

    # Step 14: Knowledge Articles
    log_info "Step 14: Knowledge Articles..."
    local knowledge
    knowledge=$(api_get "$admin_token" "/api/v1/knowledge/articles")
    check_contains "knowledge_articles" "code" "$knowledge" || true

    # Step 15: BPMN Process Instances
    log_info "Step 15: BPMN Process Instances..."
    local bpmn
    bpmn=$(api_get "$admin_token" "/api/v1/bpmn/process-instances")
    check_contains "bpmn_process" "code" "$bpmn" || true

    # Step 16: Workflow Instances
    log_info "Step 16: Workflow Instances..."
    local workflow
    workflow=$(api_get "$admin_token" "/api/v1/workflow/instances")
    check_contains "workflow_instances" "code" "$workflow" || true

    # Step 17: Audit Logs
    log_info "Step 17: Audit Logs..."
    local audit
    audit=$(api_get "$admin_token" "/api/v1/audit-logs")
    check_contains "audit_logs" "code" "$audit" || true

    # 总结
    log_info "=== Test Summary ==="
    if [[ $fails -eq 0 ]]; then
        log_info "All smoke tests PASSED"
        exit 0
    else
        log_fail "$fails test(s) FAILED"
        exit 1
    fi
}

main "$@"
