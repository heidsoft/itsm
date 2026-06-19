#!/bin/bash
# ========================================================
# 国内企业ITSM工作流 - 企业交付标准测试脚本
# ========================================================

BASE_URL="http://localhost:8080/api/v1"
TOKEN=""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
echo_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
echo_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# ========================================================
# 测试1: 登录获取Token
# ========================================================
test_login() {
    echo_info "测试1: 用户登录..."
    
    RESPONSE=$(curl -s -X POST "$BASE_URL/auth/login" \
        -H "Content-Type: application/json" \
        -d '{"username":"admin","password":"admin123"}')
    
    if echo "$RESPONSE" | grep -q "token"; then
        TOKEN=$(echo "$RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
        echo_info "✓ 登录成功"
        return 0
    else
        echo_error "✗ 登录失败: $RESPONSE"
        return 1
    fi
}

# ========================================================
# 测试2: 获取流程定义列表
# ========================================================
test_process_definitions() {
    echo_info "测试2: 获取流程定义..."
    
    RESPONSE=$(curl -s -X GET "$BASE_URL/process-definitions" \
        -H "Authorization: Bearer $TOKEN")
    
    if echo "$RESPONSE" | grep -q "incident_emergency_flow"; then
        echo_info "✓ 流程定义查询成功"
        echo "$RESPONSE" | jq '.data[] | "\(.key) - \(.name)"' 2>/dev/null || echo "$RESPONSE"
        return 0
    else
        echo_warn "⚠ 流程定义列表查询响应: $RESPONSE"
        return 1
    fi
}

# ========================================================
# 测试3: 创建事件(P0紧急)
# ========================================================
test_create_incident() {
    echo_info "测试3: 创建P0紧急事件..."
    
    RESPONSE=$(curl -s -X POST "$BASE_URL/incidents" \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        -d '{
            "title": "[TEST] P0故障测试 - 企业交付验证",
            "description": "测试紧急故障响应流程",
            "priority": "urgent",
            "severity": "critical",
            "category": "infrastructure"
        }')
    
    if echo "$RESPONSE" | grep -q '"id"'; then
        INCIDENT_ID=$(echo "$RESPONSE" | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
        echo_info "✓ P0事件创建成功, ID: $INCIDENT_ID"
        return 0
    else
        echo_error "✗ P0事件创建失败: $RESPONSE"
        return 1
    fi
}

# ========================================================
# 测试4: 获取事件列表
# ========================================================
test_list_incidents() {
    echo_info "测试4: 获取事件列表..."
    
    RESPONSE=$(curl -s -X GET "$BASE_URL/incidents?page=1&pageSize=10" \
        -H "Authorization: Bearer $TOKEN")
    
    if echo "$RESPONSE" | grep -q "data"; then
        echo_info "✓ 事件列表查询成功"
        return 0
    else
        echo_warn "⚠ 事件列表查询: $RESPONSE"
        return 1
    fi
}

# ========================================================
# 测试5: 获取变更列表
# ========================================================
test_list_changes() {
    echo_info "测试5: 获取变更列表..."
    
    RESPONSE=$(curl -s -X GET "$BASE_URL/changes?page=1&pageSize=10" \
        -H "Authorization: Bearer $TOKEN")
    
    if echo "$RESPONSE" | grep -q "data"; then
        echo_info "✓ 变更列表查询成功"
        return 0
    else
        echo_warn "⚠ 变更列表查询: $RESPONSE"
        return 1
    fi
}

# ========================================================
# 测试6: 获取服务请求列表
# ========================================================
test_list_service_requests() {
    echo_info "测试6: 获取服务请求列表..."
    
    RESPONSE=$(curl -s -X GET "$BASE_URL/service-requests?page=1&pageSize=10" \
        -H "Authorization: Bearer $TOKEN")
    
    if echo "$RESPONSE" | grep -q "data"; then
        echo_info "✓ 服务请求列表查询成功"
        return 0
    else
        echo_warn "⚠ 服务请求列表查询: $RESPONSE"
        return 1
    fi
}

# ========================================================
# 测试7: 获取发布列表
# ========================================================
test_list_releases() {
    echo_info "测试7: 获取发布列表..."
    
    RESPONSE=$(curl -s -X GET "$BASE_URL/releases?page=1&pageSize=10" \
        -H "Authorization: Bearer $TOKEN")
    
    if echo "$RESPONSE" | grep -q "data"; then
        echo_info "✓ 发布列表查询成功"
        return 0
    else
        echo_warn "⚠ 发布列表查询: $RESPONSE"
        return 1
    fi
}

# ========================================================
# 测试8: 获取流程实例
# ========================================================
test_process_instances() {
    echo_info "测试8: 获取流程实例..."
    
    RESPONSE=$(curl -s -X GET "$BASE_URL/process-instances?page=1&pageSize=10" \
        -H "Authorization: Bearer $TOKEN")
    
    if echo "$RESPONSE" | grep -q "data"; then
        echo_info "✓ 流程实例查询成功"
        return 0
    else
        echo_warn "⚠ 流程实例查询: $RESPONSE"
        return 1
    fi
}

# ========================================================
# 测试9: 获取统计信息
# ========================================================
test_stats() {
    echo_info "测试9: 获取统计信息..."
    
    RESPONSE=$(curl -s -X GET "$BASE_URL/stats" \
        -H "Authorization: Bearer $TOKEN")
    
    if echo "$RESPONSE" | grep -q "incidents"; then
        echo_info "✓ 统计接口正常"
        return 0
    else
        echo_warn "⚠ 统计接口: $RESPONSE"
        return 1
    fi
}

# ========================================================
# 主测试流程
# ========================================================
echo "========================================"
echo "国内企业ITSM工作流 - 企业交付标准测试"
echo "========================================"
echo ""

# 首先登录获取token
if ! test_login; then
    echo_error "登录失败，终止测试"
    exit 1
fi

echo ""
echo "--- API接口测试 ---"

# 执行API测试
test_process_definitions
test_list_incidents
test_list_changes
test_list_service_requests
test_list_releases
test_process_instances
test_stats

echo ""
echo "--- 创建流程测试 ---"

test_create_incident

echo ""
echo "========================================"
echo "测试完成"
echo "========================================"
echo ""
echo "请检查上述测试结果，确保所有API接口正常工作。"
echo "如果需要查看流程实例，请在数据库中执行:"
echo "  SELECT * FROM process_instances WHERE tenant_id = 1 ORDER BY created_at DESC LIMIT 10;"
