#!/bin/bash

# ITSM 前后端API对接测试脚本
# 逐个模块测试前端到后端的API对接

set -e

BASE_URL="${API_BASE_URL:-http://localhost:8090}"
TOKEN="${AUTH_TOKEN:-}"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试结果统计
PASSED=0
FAILED=0
SKIPPED=0

# 打印测试结果
print_result() {
    local module=$1
    local endpoint=$2
    local status=$3
    local message=$4
    
    if [ "$status" = "PASS" ]; then
        echo -e "${GREEN}✓${NC} [$module] $endpoint - $message"
        ((PASSED++))
    elif [ "$status" = "FAIL" ]; then
        echo -e "${RED}✗${NC} [$module] $endpoint - $message"
        ((FAILED++))
    else
        echo -e "${YELLOW}⊘${NC} [$module] $endpoint - $message"
        ((SKIPPED++))
    fi
}

# 测试API端点
test_endpoint() {
    local method=$1
    local endpoint=$2
    local data=$3
    local expected_code=${4:-200}
    
    local response
    local status_code
    
    if [ -n "$data" ]; then
        response=$(curl -s -w "\n%{http_code}" -X "$method" \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $TOKEN" \
            -d "$data" \
            "$BASE_URL$endpoint")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $TOKEN" \
            "$BASE_URL$endpoint")
    fi
    
    status_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$status_code" = "$expected_code" ]; then
        echo "PASS"
    else
        echo "FAIL: HTTP $status_code - $body"
    fi
}

# 检查后端服务是否运行
check_backend() {
    echo "检查后端服务状态..."
    local health=$(curl -s "$BASE_URL/api/v1/health" || echo "ERROR")
    if [[ "$health" == *"status"* ]] || [[ "$health" == *"ok"* ]]; then
        echo -e "${GREEN}后端服务运行正常${NC}"
        return 0
    else
        echo -e "${RED}后端服务未运行或无法访问${NC}"
        echo "请确保后端服务在 $BASE_URL 运行"
        return 1
    fi
}

# 登录获取Token
login() {
    echo "尝试登录获取Token..."
    local login_response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d '{"username":"admin","password":"password"}' \
        "$BASE_URL/api/v1/login")
    
    TOKEN=$(echo "$login_response" | grep -o '"token":"[^"]*' | cut -d'"' -f4 || echo "")
    
    if [ -n "$TOKEN" ]; then
        echo -e "${GREEN}登录成功${NC}"
        return 0
    else
        echo -e "${YELLOW}登录失败，将使用提供的TOKEN或跳过需要认证的测试${NC}"
        return 1
    fi
}

echo "=========================================="
echo "ITSM 前后端API对接测试"
echo "=========================================="
echo "后端地址: $BASE_URL"
echo ""

# 检查后端服务
if ! check_backend; then
    echo "无法连接到后端服务，退出测试"
    exit 1
fi

# 尝试登录
login

echo ""
echo "开始测试各个模块..."
echo ""

# ==================== 1. Dashboard模块 ====================
echo "--- Dashboard模块 ---"
result=$(test_endpoint "GET" "/api/v1/dashboard/overview" "" 200)
print_result "Dashboard" "GET /overview" "$([ "$result" = "PASS" ] && echo "PASS" || echo "FAIL")" "$result"

result=$(test_endpoint "GET" "/api/v1/dashboard/kpi-metrics" "" 200)
print_result "Dashboard" "GET /kpi-metrics" "$([ "$result" = "PASS" ] && echo "PASS" || echo "FAIL")" "$result"

result=$(test_endpoint "GET" "/api/v1/dashboard/ticket-trend?days=7" "" 200)
print_result "Dashboard" "GET /ticket-trend" "$([ "$result" = "PASS" ] && echo "PASS" || echo "FAIL")" "$result"

result=$(test_endpoint "GET" "/api/v1/dashboard/incident-distribution" "" 200)
print_result "Dashboard" "GET /incident-distribution" "$([ "$result" = "PASS" ] && echo "PASS" || echo "FAIL")" "$result"

# ==================== 2. Tickets模块 ====================
echo ""
echo "--- Tickets模块 ---"
result=$(test_endpoint "GET" "/api/v1/tickets?page=1&page_size=10" "" 200)
print_result "Tickets" "GET /tickets" "$([ "$result" = "PASS" ] && echo "PASS" || echo "FAIL")" "$result"

result=$(test_endpoint "GET" "/api/v1/tickets/stats" "" 200)
print_result "Tickets" "GET /stats" "$([ "$result" = "PASS" ] && echo "PASS" || echo "FAIL")" "$result"

# ==================== 3. Incidents模块 ====================
echo ""
echo "--- Incidents模块 ---"
result=$(test_endpoint "GET" "/api/v1/incidents?page=1&page_size=10" "" 200)
print_result "Incidents" "GET /incidents" "$([ "$result" = "PASS" ] && echo "PASS" || echo "FAIL")" "$result"

result=$(test_endpoint "GET" "/api/v1/incidents/stats" "" 200)
print_result "Incidents" "GET /stats" "$([ "$result" = "PASS" ] && echo "PASS" || echo "FAIL")" "$result"

# ==================== 4. Workflow模块 ====================
echo ""
echo "--- Workflow模块 ---"
result=$(test_endpoint "GET" "/api/v1/bpmn/process-definitions?page=1&page_size=10" "" 200)
print_result "Workflow" "GET /process-definitions" "$([ "$result" = "PASS" ] && echo "PASS" || echo "FAIL")" "$result"

result=$(test_endpoint "GET" "/api/v1/bpmn/process-instances?page=1&page_size=10" "" 200)
print_result "Workflow" "GET /process-instances" "$([ "$result" = "PASS" ] && echo "PASS" || echo "FAIL")" "$result"

# ==================== 5. Enterprise Management模块 ====================
echo ""
echo "--- Enterprise Management模块 ---"

# Departments
result=$(test_endpoint "GET" "/api/v1/departments/tree" "" 200)
print_result "Departments" "GET /tree" "$([ "$result" = "PASS" ] && echo "PASS" || echo "FAIL")" "$result"

# Projects
result=$(test_endpoint "GET" "/api/v1/projects" "" 200)
print_result "Projects" "GET /projects" "$([ "$result" = "PASS" ] && echo "PASS" || echo "FAIL")" "$result"

# Applications
result=$(test_endpoint "GET" "/api/v1/applications" "" 200)
print_result "Applications" "GET /applications" "$([ "$result" = "PASS" ] && echo "PASS" || echo "FAIL")" "$result"

result=$(test_endpoint "GET" "/api/v1/applications/microservices" "" 200)
print_result "Applications" "GET /microservices" "$([ "$result" = "PASS" ] && echo "PASS" || echo "FAIL")" "$result"

# Teams
result=$(test_endpoint "GET" "/api/v1/teams" "" 200)
print_result "Teams" "GET /teams" "$([ "$result" = "PASS" ] && echo "PASS" || echo "FAIL")" "$result"

# Tags
result=$(test_endpoint "GET" "/api/v1/tags" "" 200)
print_result "Tags" "GET /tags" "$([ "$result" = "PASS" ] && echo "PASS" || echo "FAIL")" "$result"

# ==================== 测试总结 ====================
echo ""
echo "=========================================="
echo "测试总结"
echo "=========================================="
echo -e "${GREEN}通过: $PASSED${NC}"
echo -e "${RED}失败: $FAILED${NC}"
echo -e "${YELLOW}跳过: $SKIPPED${NC}"
echo "总计: $((PASSED + FAILED + SKIPPED))"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}所有测试通过！${NC}"
    exit 0
else
    echo -e "${RED}有 $FAILED 个测试失败${NC}"
    exit 1
fi

