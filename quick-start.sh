#!/bin/bash
# ============================================
# ITSM 一键启动脚本
# ============================================
# 功能：
# 1. 检查 Docker 环境
# 2. 一键启动所有服务
# 3. 等待服务就绪
# 4. 运行验收测试

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# 配置
COMPOSE_FILE="docker-compose.yml"
PROJECT_NAME="itsm_oob"
BACKEND_URL="http://localhost:8090"
FRONTEND_URL="http://localhost:3000"
MAX_WAIT=180
HEALTH_CHECK_INTERVAL=5

echo -e "${BLUE}"
echo "╔════════════════════════════════════════════╗"
echo "║   AI-Native ITSM 一键启动脚本              ║"
echo "║   AI First, Not AI After                   ║"
echo "╚════════════════════════════════════════════╝"
echo -e "${NC}"
echo ""

# ============================================
# 检查 Docker 环境
# ============================================
check_docker() {
    echo -e "${YELLOW}[1/5] 检查 Docker 环境...${NC}"
    
    if ! command -v docker &> /dev/null; then
        echo -e "${RED}✗ Docker 未安装${NC}"
        echo "请先安装 Docker: https://docs.docker.com/get-docker/"
        exit 1
    fi
    
    if ! docker info &> /dev/null; then
        echo -e "${RED}✗ Docker 未运行${NC}"
        echo "请先启动 Docker"
        exit 1
    fi
    
    if ! command -v docker compose &> /dev/null && ! docker-compose --version &> /dev/null; then
        echo -e "${RED}✗ Docker Compose 未安装${NC}"
        echo "请先安装 Docker Compose"
        exit 1
    fi
    
    echo -e "${GREEN}✓ Docker 环境就绪${NC}"
    echo ""
}

# ============================================
# 停止已有服务
# ============================================
cleanup_existing() {
    echo -e "${YELLOW}[2/5] 清理已有服务...${NC}"
    
    docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" down -v 2>/dev/null || true
    docker compose -f "$COMPOSE_FILE" down -v 2>/dev/null || true
    
    echo -e "${GREEN}✓ 已清理${NC}"
    echo ""
}

# ============================================
# 启动服务
# ============================================
start_services() {
    echo -e "${YELLOW}[3/5] 启动服务...${NC}"
    
    # 检查是否有 docker compose v2
    if docker compose version &> /dev/null; then
        DOCKER_COMPOSE="docker compose"
    else
        DOCKER_COMPOSE="docker-compose"
    fi
    
    echo -e "${CYAN}执行: $DOCKER_COMPOSE -f $COMPOSE_FILE up -d --build${NC}"
    $DOCKER_COMPOSE -f "$COMPOSE_FILE" up -d --build
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ 服务启动成功${NC}"
    else
        echo -e "${RED}✗ 服务启动失败${NC}"
        echo "请检查日志: docker compose -f $COMPOSE_FILE logs"
        exit 1
    fi
    echo ""
}

# ============================================
# 等待服务就绪
# ============================================
wait_for_services() {
    echo -e "${YELLOW}[4/5] 等待服务就绪...${NC}"
    echo "（首次启动可能需要 1-3 分钟初始化数据库）"
    echo ""
    
    local elapsed=0
    local backend_ready=false
    local frontend_ready=false
    
    while [ $elapsed -lt $MAX_WAIT ]; do
        # 检查后端
        if [ "$backend_ready" = "false" ]; then
            if curl -sf --max-time 5 "$BACKEND_URL/health" > /dev/null 2>&1; then
                echo -e "${GREEN}✓ 后端服务已就绪 ($elapsed s)${NC}"
                backend_ready=true
            fi
        fi
        
        # 检查前端
        if [ "$frontend_ready" = "false" ]; then
            if curl -sf --max-time 5 "$FRONTEND_URL" > /dev/null 2>&1; then
                echo -e "${GREEN}✓ 前端服务已就绪 ($elapsed s)${NC}"
                frontend_ready=true
            fi
        fi
        
        # 都就绪了
        if [ "$backend_ready" = "true" ] && [ "$frontend_ready" = "true" ]; then
            echo ""
            echo -e "${GREEN}✓ 所有服务已就绪${NC}"
            return 0
        fi
        
        # 进度显示
        remaining=$((MAX_WAIT - elapsed))
        echo -ne "\r  等待中... ${elapsed}s / ${MAX_WAIT}s (后端: $backend_ready, 前端: $frontend_ready)  "
        
        sleep $HEALTH_CHECK_INTERVAL
        elapsed=$((elapsed + HEALTH_CHECK_INTERVAL))
    done
    
    echo ""
    echo -e "${YELLOW}⚠ 服务启动超时，部分服务可能仍在初始化${NC}"
    echo "请稍后访问或检查日志"
}

# ============================================
# 验收测试
# ============================================
run_smoke_test() {
    echo ""
    echo -e "${YELLOW}[5/5] 运行验收测试...${NC}"
    echo ""
    
    chmod +x scripts/smoke-test.sh 2>/dev/null || true
    ./scripts/smoke-test.sh
}

# ============================================
# 主流程
# ============================================
main() {
    check_docker
    cleanup_existing
    start_services
    wait_for_services
    run_smoke_test
    
    echo ""
    echo -e "${GREEN}╔════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║   🎉 启动完成！                          ║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "访问地址:"
    echo -e "  🌐 前端:    ${CYAN}$FRONTEND_URL${NC}"
    echo -e "  🔧 后端:    ${CYAN}$BACKEND_URL${NC}"
    echo -e "  📚 API文档: ${CYAN}$BACKEND_URL/swagger${NC}"
    echo ""
    echo -e "默认登录: ${YELLOW}admin / admin123${NC}"
    echo ""
    echo "常用命令:"
    echo "  查看日志:  docker compose -f $COMPOSE_FILE logs -f"
    echo "  停止服务:  docker compose -f $COMPOSE_FILE down"
    echo "  完全清理:  docker compose -f $COMPOSE_FILE down -v"
    echo ""
}

main "$@"