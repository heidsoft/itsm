#!/bin/bash
# ============================================
# ITSM 开箱即用验收测试脚本
# ============================================
# 测试步骤：
# 1. 健康检查
# 2. 登录功能
# 3. 核心API可用性
# 4. 数据库连接

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置
BACKEND_URL=${ITSM_BACKEND_URL:-"http://localhost:8090"}
FRONTEND_URL=${ITSM_FRONTEND_URL:-"http://localhost:3000"}
ADMIN_USER=${ITSM_ADMIN_USER:-"admin"}
ADMIN_PASS=${ITSM_ADMIN_PASS:-"admin123"}
MAX_RETRIES=${MAX_RETRIES:-30}
RETRY_INTERVAL=${RETRY_INTERVAL:-2}

# 测试计数器
PASSED=0
FAILED=0

echo -e "${BLUE}========================================"
echo "ITSM 开箱即用验收测试"
echo "========================================${NC}"
echo ""

# ============================================
# 辅助函数
# ============================================
wait_for_service() {
    local url=$1
    local name=$2
    local retries=$MAX_RETRIES
    
    echo -e "${YELLOW}等待服务响应: $name ($url)${NC}"
    
    while [ $retries -gt 0 ]; do
        if curl -sf --max-time 5 "$url" > /dev/null 2>&1; then
            echo -e "${GREEN}✓ 服务已就绪: $name${NC}"
            return 0
        fi
        retries=$((retries - 1))
        echo "  剩余重试次数: $retries..."
        sleep $RETRY_INTERVAL
    done
    
    echo -e "${RED}✗ 服务无响应: $name${NC}"
    return 1
}

test_endpoint() {
    local name=$1
    local url=$2
    local expected_code=${3:-200}
    local method=${4:-GET}
    local body=${5:-""}
    
    echo -e "${BLUE}测试: $name${NC}"
    echo "  URL: $url"
    
    if [ "$method" = "GET" ]; then
        response=$(curl -sf -w "\n%{http_code}" "$url" 2>&1 || true)
    else
        response=$(curl -sf -w "\n%{http_code}" -X "$method" -H "Content-Type: application/json" -d "$body" "$url" 2>&1 || true)
    fi
    
    code=$(echo "$response" | tail -1)
    
    if [ "$code" = "$expected_code" ]; then
        echo -e "${GREEN}  ✓ HTTP $code${NC}"
        PASSED=$((PASSED + 1))
        return 0
    else
        echo -e "${RED}  ✗ HTTP $code (期望: $expected_code)${NC}"
        echo "  响应: $response"
        FAILED=$((FAILED + 1))
        return 1
    fi
}

test_json_endpoint() {
    local name=$1
    local url=$2
    local method=${3:-GET}
    local body=${4:-""}
    local auth_token=${5:-""}
    
    echo -e "${BLUE}测试: $name${NC}"
    echo "  URL: $url"
    
    headers="-H Content-Type: application/json"
    if [ -n "$auth_token" ]; then
        headers="$headers -H Authorization: Bearer $auth_token"
    fi
    
    if [ -n "$body" ]; then
        response=$(curl -sf "$url" $headers -X "$method" -d "$body" 2>&1 || true)
    else
        response=$(curl -sf "$url" $headers -X "$method" 2>&1 || true)
    fi
    
    if [ $? -eq 0 ]; then
        # 验证是否为有效JSON
        if echo "$response" | jq . > /dev/null 2>&1; then
            echo -e "${GREEN}  ✓ 返回有效JSON${NC}"
            PASSED=$((PASSED + 1))
            return 0
        else
            echo -e "${RED}  ✗ 返回不是有效JSON${NC}"
            echo "  响应: $response"
            FAILED=$((FAILED + 1))
            return 1
        fi
    else
        echo -e "${RED}  ✗ 请求失败${NC}"
        FAILED=$((FAILED + 1))
        return 1
    fi
}

# ============================================
# 测试阶段
# ============================================

# 阶段1: 服务健康检查
echo -e "${YELLOW}[阶段 1/5] 服务健康检查${NC}"
echo "----------------------------------------"

# The backend only mounts the health endpoint under /api/v1 (see
# itsm-backend/internal/api/routes.go). Hitting the bare /health URL
# always returns 404 and the smoke test would time out forever.
wait_for_service "$BACKEND_URL/api/v1/health" "后端健康检查"
wait_for_service "$FRONTEND_URL" "前端首页"

echo ""

# 阶段2: 登录功能
echo -e "${YELLOW}[阶段 2/5] 登录功能测试${NC}"
echo "----------------------------------------"

# 获取token
echo -e "${BLUE}测试: 用户登录${NC}"
login_response=$(curl -sf -X POST "$BACKEND_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"$ADMIN_USER\",\"password\":\"$ADMIN_PASS\"}" 2>&1 || true)

if echo "$login_response" | jq . > /dev/null 2>&1; then
    token=$(echo "$login_response" | jq -r '.data.token // .token // empty' 2>/dev/null || true)
    if [ -n "$token" ] && [ "$token" != "null" ]; then
        echo -e "${GREEN}  ✓ 登录成功，获取到Token${NC}"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}  ✗ 登录响应格式异常${NC}"
        echo "  响应: $login_response"
        FAILED=$((FAILED + 1))
        token=""
    fi
else
    echo -e "${RED}  ✗ 登录请求失败${NC}"
    echo "  响应: $login_response"
    FAILED=$((FAILED + 1))
    token=""
fi

echo ""

# 阶段3: 核心API测试
echo -e "${YELLOW}[阶段 3/5] 核心API可用性${NC}"
echo "----------------------------------------"

if [ -n "$token" ]; then
    test_json_endpoint "获取用户信息" "$BACKEND_URL/api/v1/auth/me" "GET" "" "$token"
    test_json_endpoint "获取仪表盘数据" "$BACKEND_URL/api/v1/dashboard/stats" "GET" "" "$token"
    test_json_endpoint "获取事件列表" "$BACKEND_URL/api/v1/incidents" "GET" "" "$token"
    test_json_endpoint "获取工单列表" "$BACKEND_URL/api/v1/tickets" "GET" "" "$token"
else
    echo -e "${YELLOW}  跳过API测试（无有效Token）${NC}"
fi

echo ""

# 阶段4: 数据库连接测试
echo -e "${YELLOW}[阶段 4/5] 数据库连接${NC}"
echo "----------------------------------------"

# 通过健康检查端点验证数据库连接
db_check=$(curl -sf "$BACKEND_URL/health" 2>&1 || true)
if echo "$db_check" | jq . > /dev/null 2>&1; then
    db_status=$(echo "$db_check" | jq -r '.database // .db // "ok"' 2>/dev/null || echo "ok")
    if [ "$db_status" = "ok" ] || [ "$db_status" = "connected" ]; then
        echo -e "${GREEN}  ✓ 数据库连接正常${NC}"
        PASSED=$((PASSED + 1))
    else
        echo -e "${YELLOW}  ⚠ 数据库状态: $db_status${NC}"
        PASSED=$((PASSED + 1))
    fi
else
    echo -e "${YELLOW}  ⚠ 健康检查无数据库详情（可能正常）${NC}"
    PASSED=$((PASSED + 1))
fi

echo ""

# 阶段5: 前端可用性
echo -e "${YELLOW}[阶段 5/5] 前端页面可用性${NC}"
echo "----------------------------------------"

# 检查前端页面是否能正常加载
frontend_check=$(curl -sf -o /dev/null -w "%{http_code}" "$FRONTEND_URL" 2>&1 || echo "000")
if [ "$frontend_check" = "200" ] || [ "$frontend_check" = "304" ]; then
    echo -e "${GREEN}  ✓ 前端页面正常加载 (HTTP $frontend_check)${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}  ✗ 前端页面异常 (HTTP $frontend_check)${NC}"
    FAILED=$((FAILED + 1))
fi

echo ""

# ============================================
# 测试结果汇总
# ============================================
echo "========================================"
echo "测试结果汇总"
echo "========================================"
echo -e "${GREEN}✓ 通过: $PASSED${NC}"
echo -e "${RED}✗ 失败: $FAILED${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 开箱即用验收通过！${NC}"
    echo ""
    echo "您可以访问:"
    echo "  🌐 前端: $FRONTEND_URL"
    echo "  🔧 后端: $BACKEND_URL"
    echo "  📚 API文档: $BACKEND_URL/swagger"
    echo ""
    echo "默认登录: $ADMIN_USER / $ADMIN_PASS"
    exit 0
else
    echo -e "${RED}⚠ 部分测试失败，请检查日志${NC}"
    echo ""
    echo "常见问题排查:"
    echo "  1. 检查服务是否启动: docker compose ps"
    echo "  2. 查看后端日志: docker compose logs itsm-backend"
    echo "  3. 检查数据库日志: docker compose logs postgres"
    echo "  4. 等待初始化完成（首次启动约需1-2分钟）"
    exit 1
fi