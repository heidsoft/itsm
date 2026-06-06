#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
source "${SCRIPT_DIR}/lib/common.sh"

BASE_URL="${ITSM_BACKEND_URL:-http://127.0.0.1:8090}"
API_BASE="${BASE_URL}/api/v1"

TMP_DIR="$(mktemp -d)"
on_exit "rm -rf '${TMP_DIR}'"

FAILED=0
PASSED=0

log_result() {
    local status="$1"
    local message="$2"
    if [[ "$status" == "PASS" ]]; then
        PASSED=$((PASSED + 1))
        log_success "$message"
    else
        FAILED=$((FAILED + 1))
        log_error "$message"
    fi
}

require_tools() {
    command -v curl >/dev/null 2>&1 || { log_error "curl is required"; exit 1; }
    command -v jq >/dev/null 2>&1 || { log_error "jq is required"; exit 1; }
}

login() {
    local username="$1"
    local password="$2"
    local body_file="${TMP_DIR}/login-${username}.json"
    local http_code

    http_code=$(curl -sS -o "${body_file}" -w '%{http_code}' \
        -X POST "${API_BASE}/auth/login" \
        -H 'Content-Type: application/json' \
        -d "$(jq -nc --arg username "${username}" --arg password "${password}" '{username:$username,password:$password}')" || true)

    if [[ "${http_code}" != "200" ]]; then
        log_error "login failed for ${username} (HTTP ${http_code})"
        cat "${body_file}" >&2 || true
        exit 1
    fi

    jq -r '.data.access_token' "${body_file}"
}

api_call() {
    local method="$1"
    local path="$2"
    local token="$3"
    local outfile="$4"
    local payload="${5:-}"

    if [[ -n "${payload}" ]]; then
        curl -sS -o "${outfile}" -w '%{http_code}' \
            -X "${method}" "${API_BASE}${path}" \
            -H "Authorization: Bearer ${token}" \
            -H 'Content-Type: application/json' \
            -d "${payload}" || true
    else
        curl -sS -o "${outfile}" -w '%{http_code}' \
            -X "${method}" "${API_BASE}${path}" \
            -H "Authorization: Bearer ${token}" || true
    fi
}

assert_users_endpoint() {
    local username="$1"
    local token="$2"
    local expected_http="$3"
    local outfile="${TMP_DIR}/${username}-users.json"
    local http_code

    http_code=$(api_call GET "/users" "${token}" "${outfile}")
    if [[ "${http_code}" != "${expected_http}" ]]; then
        log_result FAIL "${username} /users HTTP=${http_code}, expected ${expected_http}"
        return
    fi

    if [[ "${expected_http}" == "200" ]]; then
        local code
        code=$(jq -r '.code // "null"' "${outfile}")
        if [[ "${code}" == "0" ]]; then
            log_result PASS "${username} can access /users as expected"
        else
            log_result FAIL "${username} /users returned code=${code}, expected code=0"
        fi
    else
        log_result PASS "${username} is blocked from /users as expected"
    fi
}

assert_menu_visibility() {
    local username="$1"
    local token="$2"
    shift 2
    local restricted=("$@")
    local outfile="${TMP_DIR}/${username}-menus.json"
    local http_code

    http_code=$(api_call GET "/auth/menus" "${token}" "${outfile}")
    if [[ "${http_code}" != "200" ]]; then
        log_result FAIL "${username} /auth/menus HTTP=${http_code}"
        return
    fi

    local names
    names="$(jq -r '.. | .name? // empty' "${outfile}")"
    local leaked=()
    local item
    for item in "${restricted[@]}"; do
        if grep -Fxq "${item}" <<< "${names}"; then
            leaked+=("${item}")
        fi
    done

    if [[ ${#leaked[@]} -eq 0 ]]; then
        log_result PASS "${username} menu no longer exposes restricted admin entries"
    else
        log_result FAIL "${username} menu leaked: ${leaked[*]}"
    fi
}

main() {
    require_tools
    log_phase "RBAC and Menu Regression"

    local admin_token user_token security_token
    admin_token="$(login admin admin123)"
    user_token="$(login user1 user123)"
    security_token="$(login security1 security123)"

    assert_users_endpoint "admin" "${admin_token}" "200"
    assert_users_endpoint "user1" "${user_token}" "403"
    assert_users_endpoint "security1" "${security_token}" "403"

    local restricted_labels=(
        "用户管理"
        "部门管理"
        "团队管理"
        "SLA配置"
        "工单分类"
        "工作流"
    )

    assert_menu_visibility "user1" "${user_token}" "${restricted_labels[@]}"
    assert_menu_visibility "security1" "${security_token}" "${restricted_labels[@]}"

    log_phase "Summary"
    echo "PASSED=${PASSED}"
    echo "FAILED=${FAILED}"

    if [[ "${FAILED}" -gt 0 ]]; then
        exit 1
    fi
}

main "$@"
