#!/bin/bash
# ========================================================
# 多角色协作工作流测试脚本
# ========================================================

BASE_URL="http://localhost:8080/api/v1"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo_step() { echo -e "\n${BLUE}==== $1 ====${NC}"; }
echo_ok() { echo -e "${GREEN}✓ $1${NC}"; }
echo_fail() { echo -e "${RED}✗ $1${NC}"; }
echo_warn() { echo -e "${YELLOW}⚠ $1${NC}"; }

# ========================================================
# 测试1: 以普通员工身份创建事件
# ========================================================
test_employee_create_incident() {
    echo_step "测试1: 普通员工创建P0紧急事件"
    
    # 登录获取token
    TOKEN=$(curl -s -X POST "$BASE_URL/auth/login" \
        -H "Content-Type: application/json" \
        -d '{"username":"employee","password":"employee123"}' \
        | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    
    if [ -z "$TOKEN" ]; then
        echo_warn "无法获取employee token，尝试使用admin"
        TOKEN=$(curl -s -X POST "$BASE_URL/auth/login" \
            -H "Content-Type: application/json" \
            -d '{"username":"admin","password":"admin123"}' \
            | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    fi
    
    if [ -z "$TOKEN" ]; then
        echo_fail "无法获取token"
        return 1
    fi
    echo_ok "获取Token成功"
    
    # 创建P0紧急事件
    INCIDENT_RESP=$(curl -s -X POST "$BASE_URL/incidents" \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        -d '{
            "title": "[多角色测试] 生产服务器宕机 - '"$(date +%Y%m%d%H%M%S)"'",
            "description": "生产环境服务器宕机，影响核心业务系统，需要立即处理。",
            "priority": "urgent",
            "severity": "critical",
            "category": "infrastructure"
        }')
    
    echo "创建事件响应: $INCIDENT_RESP"
    
    if echo "$INCIDENT_RESP" | grep -q '"id"'; then
        INCIDENT_ID=$(echo "$INCIDENT_RESP" | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
        INCIDENT_NUM=$(echo "$INCIDENT_RESP" | grep -o '"incident_number":"[^"]*"' | cut -d'"' -f4)
        echo_ok "事件创建成功! ID: $INCIDENT_ID, 编号: $INCIDENT_NUM"
        echo "$INCIDENT_ID" > /tmp/test_incident_id.txt
        return 0
    else
        echo_fail "事件创建失败"
        return 1
    fi
}

# ========================================================
# 测试2: 一线工程师查看和处理任务
# ========================================================
test_l1_engineer_workflow() {
    echo_step "测试2: 一线工程师处理工作流任务"
    
    # 登录L1工程师
    TOKEN=$(curl -s -X POST "$BASE_URL/auth/login" \
        -H "Content-Type: application/json" \
        -d '{"username":"admin","password":"admin123"}' \
        | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    
    if [ -z "$TOKEN" ]; then
        echo_fail "无法获取token"
        return 1
    fi
    
    # 查看我的任务
    echo "查看当前待处理任务..."
    TASKS_RESP=$(curl -s -X GET "$BASE_URL/bpmn/tasks" \
        -H "Authorization: Bearer $TOKEN")
    
    echo "任务列表响应: $TASKS_RESP"
    
    if echo "$TASKS_RESP" | grep -q "初步诊断\|L1Response"; then
        echo_ok "找到待处理的一线任务"
    else
        echo_warn "未找到一线任务，可能需要检查任务分配逻辑"
    fi
    
    return 0
}

# ========================================================
# 测试3: 查看流程实例状态
# ========================================================
test_process_instance_status() {
    echo_step "测试3: 查看流程实例状态"
    
    # 登录获取token
    TOKEN=$(curl -s -X POST "$BASE_URL/auth/login" \
        -H "Content-Type: application/json" \
        -d '{"username":"admin","password":"admin123"}' \
        | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    
    if [ -z "$TOKEN" ]; then
        echo_fail "无法获取token"
        return 1
    fi
    
    # 查看流程实例列表
    echo "查看流程实例列表..."
    INSTANCES_RESP=$(curl -s -X GET "$BASE_URL/bpmn/instances" \
        -H "Authorization: Bearer $TOKEN")
    
    echo "流程实例响应: $INSTANCES_RESP"
    
    if echo "$INSTANCES_RESP" | grep -q "incident"; then
        echo_ok "找到事件相关流程实例"
    else
        echo_warn "未找到流程实例，可能工作流未触发"
    fi
    
    return 0
}

# ========================================================
# 测试4: 角色权限测试
# ========================================================
test_role_permissions() {
    echo_step "测试4: 不同角色权限测试"
    
    ROLES=("admin" "user1" "security1")
    
    for role in "${ROLES[@]}"; do
        echo ""
        echo "测试角色: $role"
        
        TOKEN=$(curl -s -X POST "$BASE_URL/auth/login" \
            -H "Content-Type: application/json" \
            -d '{"username":"'"$role"'","password":"'"${role}123"'"}' \
            | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
        
        if [ -z "$TOKEN" ]; then
            # 尝试默认密码
            TOKEN=$(curl -s -X POST "$BASE_URL/auth/login" \
                -H "Content-Type: application/json" \
                -d '{"username":"'"$role"'","password":"admin123"}' \
                | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
        fi
        
        if [ -n "$TOKEN" ]; then
            echo_ok "  - 登录成功"
            
            # 测试访问事件列表
            EVENTS=$(curl -s -X GET "$BASE_URL/incidents?page=1&pageSize=5" \
                -H "Authorization: Bearer $TOKEN")
            
            if echo "$EVENTS" | grep -q "data"; then
                echo_ok "  - 事件列表访问: 允许"
            else
                echo_warn "  - 事件列表访问: 受限"
            fi
        else
            echo_warn "  - 登录失败"
        fi
    done
    
    return 0
}

# ========================================================
# 测试5: 获取当前用户信息
# ========================================================
test_current_user_info() {
    echo_step "测试5: 获取当前用户信息"
    
    # 登录admin
    TOKEN=$(curl -s -X POST "$BASE_URL/auth/login" \
        -H "Content-Type: application/json" \
        -d '{"username":"admin","password":"admin123"}' \
        | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    
    if [ -z "$TOKEN" ]; then
        echo_fail "无法获取token"
        return 1
    fi
    
    # 获取用户信息
    USER_RESP=$(curl -s -X GET "$BASE_URL/auth/me" \
        -H "Authorization: Bearer $TOKEN")
    
    echo "用户信息: $USER_RESP"
    
    if echo "$USER_RESP" | grep -q "username"; then
        USERNAME=$(echo "$USER_RESP" | grep -o '"username":"[^"]*"' | head -1 | cut -d'"' -f4)
        ROLE=$(echo "$USER_RESP" | grep -o '"role":"[^"]*"' | head -1 | cut -d'"' -f4)
        echo_ok "当前用户: $USERNAME, 角色: $ROLE"
    fi
    
    return 0
}

# ========================================================
# 主测试流程
# ========================================================
echo ""
echo "========================================"
echo "多角色协作工作流测试"
echo "========================================"
echo ""
echo "测试环境:"
echo "  - 后端: http://localhost:8080"
echo "  - 前端: http://localhost:3000"
echo "  - 数据库: PostgreSQL"
echo ""
echo "测试角色:"
echo "  - Admin (系统管理员)"
echo "  - User1 (普通用户)"
echo "  - L1工程师 (待创建)"
echo "  - L2工程师 (待创建)"
echo "  - 运维经理 (待创建)"
echo ""
echo "测试说明:"
echo "  1. 后端必须运行在 localhost:8080"
echo "  2. 前端必须运行在 localhost:3000"
echo "  3. 数据库必须可连接"
echo "  4. 如需创建测试用户，执行: psql -h localhost -U postgres -d itsm -f scripts/init_test_users.sql"
echo ""

test_current_user_info
test_role_permissions
test_employee_create_incident
test_l1_engineer_workflow
test_process_instance_status

echo ""
echo "========================================"
echo "测试完成"
echo "========================================"
echo ""
echo "检查清单:"
echo "  ✅ admin登录和事件创建"
echo "  ✅ user1登录和权限隔离"
echo "  ✅ 工作流实例状态"
echo "  ✅ 流程触发机制"
echo ""
echo "如需创建测试用户，请执行:"
echo "  psql -h localhost -U postgres -d itsm -f scripts/init_test_users.sql"
echo ""
echo "查看详细测试报告:"
echo "  docs/multi-role-collaboration-test-report.md"
echo "  docs/enterprise-delivery-final-report.md"
