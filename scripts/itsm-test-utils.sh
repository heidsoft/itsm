#!/bin/bash
# ITSM 功能测试 - 通用工具
# 加载方法: source scripts/itsm-test-utils.sh

set -e

BASE="${ITSM_BASE:-http://localhost:8090/api/v1}"
TOKEN_FILE="/tmp/itsm_test_token.env"
IDS_FILE="/Users/heidsoft/Downloads/research/itsm/output/functional-test/ids_index.json"
OUT_DIR="/Users/heidsoft/Downloads/research/itsm/output/functional-test"

# Token 加载
if [ -f "$TOKEN_FILE" ]; then
    source "$TOKEN_FILE"
    AUTH="Authorization: Bearer $TOKEN"
else
    echo "ERROR: $TOKEN_FILE 不存在, 请先执行阶段0登录" >&2
    exit 1
fi

CURL=/usr/bin/curl
PYTHON3=/Library/Frameworks/Python.framework/Versions/3.8/bin/python3

# 通用 API 调用
api_get() {
    local path="$1"
    $CURL -s -X GET "$BASE$path" -H "$AUTH" -H "Content-Type: application/json"
}

api_post() {
    local path="$1"
    local data="$2"
    $CURL -s -X POST "$BASE$path" -H "$AUTH" -H "Content-Type: application/json" ${data:+-d "$data"}
}

api_put() {
    local path="$1"
    local data="$2"
    $CURL -s -X PUT "$BASE$path" -H "$AUTH" -H "Content-Type: application/json" ${data:+-d "$data"}
}

api_delete() {
    local path="$1"
    $CURL -s -X DELETE "$BASE$path" -H "$AUTH" -H "Content-Type: application/json"
}

# 获取响应 code 字段
code() {
    echo "$1" | $PYTHON3 -c "import sys, json; d=json.load(sys.stdin); print(d.get('code', 'NO_CODE'))" 2>/dev/null || echo "PARSE_ERR"
}

# 获取响应 message 字段
msg() {
    echo "$1" | $PYTHON3 -c "import sys, json; d=json.load(sys.stdin); print(d.get('message', 'NO_MSG'))" 2>/dev/null || echo "PARSE_ERR"
}

# 获取 data 字段（raw）
data() {
    echo "$1" | $PYTHON3 -c "import sys, json; d=json.load(sys.stdin); print(json.dumps(d.get('data', None), ensure_ascii=False) if d.get('data') is not None else 'null')" 2>/dev/null || echo "PARSE_ERR"
}

# HTTP 状态码
http_code() {
    local method="$1"
    local path="$2"
    local data="$3"
    if [ -n "$data" ]; then
        $CURL -s -o /dev/null -w "%{http_code}" -X "$method" "$BASE$path" -H "$AUTH" -H "Content-Type: application/json" -d "$data"
    else
        $CURL -s -o /dev/null -w "%{http_code}" -X "$method" "$BASE$path" -H "$AUTH" -H "Content-Type: application/json"
    fi
}

# 测试结果记录
record() {
    local test_name="$1"
    local expected_code="$2"
    local actual_code="$3"
    local response="$4"
    if [ "$expected_code" = "$actual_code" ]; then
        echo "  ✅ PASS [$test_name] code=$actual_code"
    else
        echo "  ❌ FAIL [$test_name] expected=$expected_code actual=$actual_code"
        echo "      Response: $(echo "$response" | head -c 300)"
    fi
}

# 保存 ID 到索引
save_id() {
    local key="$1"
    local value="$2"
    if [ -f "$IDS_FILE" ]; then
        $PYTHON3 -c "
import json, sys
with open('$IDS_FILE', 'r') as f:
    d = json.load(f)
d['$key'] = '$value'
with open('$IDS_FILE', 'w') as f:
    json.dump(d, f, indent=2, ensure_ascii=False)
"
    else
        echo "{\"$key\": \"$value\"}" > "$IDS_FILE"
    fi
}

# 读取 ID
get_id() {
    local key="$1"
    if [ -f "$IDS_FILE" ]; then
        $PYTHON3 -c "
import json
with open('$IDS_FILE', 'r') as f:
    d = json.load(f)
print(d.get('$key', ''))
"
    fi
}

echo "[utils.sh] 已加载. BASE=$BASE, USER=$USERNAME"
