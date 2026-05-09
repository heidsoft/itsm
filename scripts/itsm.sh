#!/bin/bash
#
# ITSM 统一快速启动脚本
# 支持本地开发和 Docker 容器部署自动检测
#

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_info() { echo "[INFO] $1"; }
print_success() { echo "[SUCCESS] $1"; }
print_warning() { echo "[WARNING] $1"; }
print_error() { echo "[ERROR] $1"; }

# Banner
echo "=========================================="
echo "  ITSM 统一快速启动脚本 v1.0"
echo "  支持本地开发 / Docker 容器部署"
echo "=========================================="
echo ""

# 检测运行环境
detect_environment() {
    if [ -f /.dockerenv ] || grep -q docker /proc/1/cgroup 2>/dev/null; then
        echo "DOCKER"
    else
        echo "LOCAL"
    fi
}

# 设置 API URL
setup_api_url() {
    ENV=$(detect_environment)
    if [ "$ENV" = "DOCKER" ]; then
        export NEXT_PUBLIC_API_URL=${NEXT_PUBLIC_API_URL:-http://host.docker.internal:8090}
        print_info "检测到 Docker 环境，API URL: $NEXT_PUBLIC_API_URL"
    else
        export NEXT_PUBLIC_API_URL=${NEXT_PUBLIC_API_URL:-http://localhost:8090}
        print_info "检测到本地开发环境，API URL: $NEXT_PUBLIC_API_URL"
    fi
}

# 启动依赖服务 (仅本地开发需要)
start_dependencies() {
    local env=$(detect_environment)
    
    if [ "$env" = "DOCKER" ]; then
        print_info "Docker 环境，跳过依赖启动（由 docker-compose 管理）"
        return 0
    fi
    
    print_info "检查依赖服务..."
    
    # PostgreSQL
    if ! docker ps | grep -q "itsm-postgres-dev\|postgres.*5432"; then
        print_warning "启动 PostgreSQL..."
        docker run -d --name itsm-postgres-dev \
            -e POSTGRES_DB=itsm \
            -e POSTGRES_USER=itsm \
            -e POSTGRES_PASSWORD=itsm_password_2026 \
            -p 5432:5432 \
            --restart unless-stopped \
            postgres:17-alpine
        print_success "PostgreSQL 已启动 (端口 5432)"
        sleep 3
    else
        print_info "PostgreSQL 已运行"
    fi
    
    # Redis
    if ! docker ps | grep -q "itsm-redis-dev\|redis.*6379"; then
        print_warning "启动 Redis..."
        docker run -d --name itsm-redis-dev \
            -p 6379:6379 \
            --restart unless-stopped \
            redis:7-alpine
        print_success "Redis 已启动 (端口 6379)"
    else
        print_info "Redis 已运行"
    fi
}

# 启动后端
start_backend() {
    local env=$(detect_environment)
    
    print_info "准备启动后端服务..."
    
    if [ "$env" = "DOCKER" ]; then
        print_info "后端由 Docker 容器管理"
        # 初始化数据库
        print_info "初始化数据库..."
        docker exec itsm-backend ./main migrate 2>/dev/null || print_warning "数据库初始化完成或无需初始化"
    else
        # 本地开发 - 启动后端
        cd "$SCRIPT_DIR/itsm-backend"
        
        # 检查是否已运行
        if curl -s http://localhost:8090/health > /dev/null 2>&1; then
            print_warning "后端已在运行 (端口 8090)"
        else
            print_info "启动后端服务..."
            
            # 后台启动后端
            nohup go run main.go serve > backend.log 2>&1 &
            BACKEND_PID=$!
            
            # 等待后端启动
            print_info "等待后端启动..."
            for i in {1..30}; do
                if curl -s http://localhost:8090/health > /dev/null 2>&1; then
                    break
                fi
                sleep 1
            done
            
            if curl -s http://localhost:8090/health > /dev/null 2>&1; then
                print_success "后端已启动 (PID: $BACKEND_PID, 端口 8090)"
            else
                print_error "后端启动失败，请检查 logs/backend.log"
            fi
        fi
    fi
}

# 启动前端
start_frontend() {
    local env=$(detect_environment)
    
    if [ "$env" = "DOCKER" ]; then
        print_info "前端由 Docker 容器管理 (端口 3000)"
        return 0
    fi
    
    cd "$SCRIPT_DIR/itsm-frontend"
    
    # 检查是否已运行
    if curl -s http://localhost:3000 > /dev/null 2>&1; then
        print_warning "前端已在运行 (端口 3000)"
        return 0
    fi
    
    print_info "安装前端依赖..."
    if [ ! -d "node_modules" ]; then
        pnpm install --ignore-scripts
    fi
    
    print_info "启动前端服务..."
    nohup pnpm dev > frontend.log 2>&1 &
    FRONTEND_PID=$!
    
    print_info "等待前端启动..."
    for i in {1..60}; do
        if curl -s http://localhost:3000 > /dev/null 2>&1; then
            break
        fi
        sleep 1
    done
    
    if curl -s http://localhost:3000 > /dev/null 2>&1; then
        print_success "前端已启动 (PID: $FRONTEND_PID, 端口 3000)"
    else
        print_warning "前端启动可能需要更长时间，请查看 logs/frontend.log"
    fi
}

# 验证服务状态
verify_services() {
    print_info "验证服务状态..."
    
    local backend_ok=false
    local frontend_ok=false
    
    if curl -s -o /dev/null -w "%{http_code}" http://localhost:8090/health | grep -q "200\|404\|405"; then
        backend_ok=true
    fi
    
    if curl -s -o /dev/null -w "%{http_code}" http://localhost:3000 | grep -q "200\|304\|307\|308"; then
        frontend_ok=true
    fi
    
    echo ""
    echo "═══════════════════════════════════════════════════════════"
    echo -e "  ${GREEN}✓${NC} 后端服务:  $(if $backend_ok; then echo -e "${GREEN}运行中 (http://localhost:8090)${NC}"; else echo -e "${RED}未运行${NC}"; fi)"
    echo -e "  ${GREEN}✓${NC} 前端服务:  $(if $frontend_ok; then echo -e "${GREEN}运行中 (http://localhost:3000)${NC}"; else echo -e "${RED}未运行${NC}"; fi)"
    echo -e "  ${GREEN}✓${NC} 登录账号:  ${CYAN}admin / admin123${NC}"
    echo "═══════════════════════════════════════════════════════════"
    echo ""
    
    if $backend_ok && $frontend_ok; then
        print_success "所有服务已就绪！"
    else
        print_warning "部分服务可能未完全启动，请稍后重试"
    fi
}

# 显示使用帮助
show_help() {
    echo ""
    echo "使用方法: $0 [命令]"
    echo ""
    echo "命令:"
    echo "  start       启动所有服务 (默认)"
    echo "  stop        停止所有服务"
    echo "  restart     重启所有服务"
    echo "  status      查看服务状态"
    echo "  logs        查看日志"
    echo "  help        显示帮助"
    echo ""
    echo "示例:"
    echo "  $0 start     # 启动服务"
    echo "  $0 status    # 查看状态"
    echo ""
}

# 停止服务
stop_services() {
    print_info "停止服务..."
    
    local env=$(detect_environment)
    
    if [ "$env" = "DOCKER" ]; then
        print_warning "Docker 环境，请使用 docker compose down"
    else
        # 停止本地服务
        pkill -f "go run main.go" && print_success "后端已停止" || true
        pkill -f "pnpm dev" && print_success "前端已停止" || true
        
        # 可选：停止依赖服务
        if [ "$1" = "--clean" ]; then
            docker stop itsm-postgres-dev itsm-redis-dev 2>/dev/null || true
            docker rm itsm-postgres-dev itsm-redis-dev 2>/dev/null || true
            print_success "依赖服务已清理"
        fi
    fi
}

# 查看状态
show_status() {
    local env=$(detect_environment)
    
    print_info "运行环境: $(detect_environment)"
    print_info "API URL: $NEXT_PUBLIC_API_URL"
    echo ""
    
    if [ "$env" = "DOCKER" ]; then
        docker compose -f "$SCRIPT_DIR/docker-compose.yml" ps
    else
        echo "后端: $(curl -s -o /dev/null -w '%{http_code}' http://localhost:8090/health 2>/dev/null || echo 'down')"
        echo "前端: $(curl -s -o /dev/null -w '%{http_code}' http://localhost:3000 2>/dev/null || echo 'down')"
    fi
}

# 查看日志
show_logs() {
    local env=$(detect_environment)
    
    if [ "$env" = "DOCKER" ]; then
        echo "Docker 容器日志 (Ctrl+C 退出):"
        docker compose -f "$SCRIPT_DIR/docker-compose.yml" logs -f
    else
        echo "按 Ctrl+C 退出日志查看"
        echo ""
        echo "=== 后端日志 ==="
        tail -f "$SCRIPT_DIR/itsm-backend/backend.log" 2>/dev/null &
        echo "=== 前端日志 ==="
        tail -f "$SCRIPT_DIR/itsm-frontend/frontend.log" 2>/dev/null &
        wait
    fi
}

# 主入口
main() {
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    cd "$SCRIPT_DIR"
    
    setup_api_url
    
    case "${1:-start}" in
        start)
            start_dependencies
            start_backend
            start_frontend
            verify_services
            ;;
        stop)
            stop_services "${2:-}"
            ;;
        restart)
            stop_services
            sleep 2
            start_dependencies
            start_backend
            start_frontend
            verify_services
            ;;
        status)
            show_status
            ;;
        logs)
            show_logs
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            print_error "未知命令: $1"
            show_help
            exit 1
            ;;
    esac
}

main "$@"