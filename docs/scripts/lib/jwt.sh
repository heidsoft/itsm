#!/bin/bash
#
# JWT Token Utility Script
# 用于获取 ITSM API 访问令牌
#

set -uo pipefail

API_BASE="${API_BASE:-http://localhost:8090}"

usage() {
    cat <<EOF
Usage: $0 <username> [password]

获取 ITSM API JWT 令牌

参数:
  username  用户名 (必填)
  password  密码 (可选，默认使用预设密码)

示例:
  $0 admin admin123
  $0 user1 user123
EOF
    exit 1
}

# 默认密码映射
declare -A DEFAULT_PASSWORDS=(
    [admin]=admin123
    [user1]=user123
    [security1]=security123
    [engineer1]=eng123
    [manager1]=mgr123
    [tenant1admin]=ta123
)

get_token() {
    local user="$1"
    local pass="$2"

    local response
    response=$(curl -s -X POST "${API_BASE}/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$user\",\"password\":\"$pass\"}")

    # 检查响应
    local code
    code=$(echo "$response" | jq -r '.code // empty')

    if [[ "$code" != "0" ]]; then
        echo "Error: Login failed - $(echo "$response" | jq -r '.message // "unknown error"')" >&2
        return 1
    fi

    local token
    token=$(echo "$response" | jq -r '.data.access_token // empty')

    if [[ -z "$token" ]]; then
        echo "Error: No token in response" >&2
        return 1
    fi

    echo "$token"
}

# 主逻辑
main() {
    if [[ $# -lt 1 ]]; then
        usage
    fi

    local user="$1"
    local pass="${2:-${DEFAULT_PASSWORDS[$user]:-}}"

    if [[ -z "$pass" ]]; then
        echo "Error: No password provided and no default for user '$user'" >&2
        usage
    fi

    get_token "$user" "$pass"
}

main "$@"
