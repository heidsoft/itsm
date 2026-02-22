#!/bin/bash
#
# ITSM 系统部署脚本
#
# 用法:
#   ./scripts/deploy.sh [选项]
#
# 选项:
#   init     - 初始化数据库和种子数据
#   migrate  - 运行数据库迁移
#   seed     - 运行种子数据
#   start    - 启动服务
#   stop     - 停止服务
#   restart  - 重启服务
#   logs     - 查看日志
#   clean    - 清理所有容器和数据
#   docker   - 使用 Docker 部署
#   help     - 显示帮助信息
#

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 项目根目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 显示帮助信息
show_help() {
    cat << EOF
ITSM 系统部署脚本

用法:
    ./scripts/deploy.sh [选项]

选项:
    init       初始化数据库和种子数据 (migrate + seed)
    migrate    运行数据库迁移 (创建/更新表结构)
    seed       运行种子数据 (创建初始数据)
    start      启动服务
    stop       停止服务
    restart    重启服务
    logs       查看服务日志
    clean      清理所有容器和数据 (谨慎使用!)
    docker     使用 Docker Compose 部署
    dev        开发模式启动 (前端+后端)
    help       显示此帮助信息

示例:
    # 首次部署
    ./scripts/deploy.sh init
    ./scripts/deploy.sh start

    # 开发模式
    ./scripts/deploy.sh dev

    # Docker 部署
    ./scripts/deploy.sh docker

EOF
}

# 检查依赖
check_dependencies() {
    log_info "检查依赖..."

    if [ "$USE_DOCKER" = "true" ]; then
        if ! command -v docker &> /dev/null; then
            log_error "Docker 未安装，请先安装 Docker"
            exit 1
        fi
        if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
            log_error "Docker Compose 未安装，请先安装 Docker Compose"
            exit 1
        fi
    else
        if ! command -v go &> /dev/null; then
            log_error "Go 未安装，请先安装 Go 1.21+"
            exit 1
        fi
        if ! command -v node &> /dev/null; then
            log_warn "Node.js 未安装，前端需要 Node.js"
        fi
        if ! command -v psql &> /dev/null; then
            log_warn "psql 未安装，数据库操作需要 PostgreSQL 客户端"
        fi
    fi

    log_info "依赖检查完成"
}

# 加载环境变量
load_env() {
    if [ -f "$PROJECT_ROOT/.env" ]; then
        log_info "加载 .env 文件"
        export $(cat "$PROJECT_ROOT/.env" | grep -v '^#' | xargs)
    else
        log_warn ".env 文件不存在，使用默认配置"
    fi
}

# 等待数据库就绪
wait_for_db() {
    local max_attempts=30
    local attempt=1

    log_info "等待数据库就绪..."

    while [ $attempt -le $max_attempts ]; do
        if PGPASSWORD="${DB_PASSWORD:-StrongPassword123!}" psql -h "${DB_HOST:-localhost}" -p "${DB_PORT:-5432}" -U "${DB_USER:-dev}" -d postgres -c "SELECT 1" &> /dev/null; then
            log_info "数据库已就绪"
            return 0
        fi
        log_info "等待数据库... ($attempt/$max_attempts)"
        sleep 2
        attempt=$((attempt + 1))
    done

    log_error "数据库连接超时"
    return 1
}

# 运行数据库迁移
run_migrate() {
    cd "$PROJECT_ROOT/itsm-backend"

    log_info "开始数据库迁移..."

    if [ "$USE_DOCKER" = "true" ]; then
        # Docker 环境下的迁移
        docker-compose exec itsm-backend go run -tags migrate main.go
    else
        # 本地环境的迁移
        go run -tags migrate main.go
    fi

    log_info "数据库迁移完成"
}

# 运行种子数据
run_seed() {
    cd "$PROJECT_ROOT/itsm-backend"

    log_info "开始填充种子数据..."

    if [ "$USE_DOCKER" = "true" ]; then
        docker-compose exec itsm-backend go run -tags seed main.go
    else
        # 种子数据在启动时自动运行，这里主要是额外的 SQL 数据
        if [ -f "sql/seed_test_data.sql" ]; then
            PGPASSWORD="${DB_PASSWORD:-StrongPassword123!}" psql -h "${DB_HOST:-localhost}" -p "${DB_PORT:-5432}" -U "${DB_USER:-dev}" -d "${DB_NAME:-itsm}" -f sql/seed_test_data.sql
        fi
    fi

    log_info "种子数据填充完成"
}

# 初始化
init_db() {
    log_info "初始化数据库..."
    run_migrate
    run_seed
    log_info "数据库初始化完成"
}

# 启动服务
start_service() {
    cd "$PROJECT_ROOT/itsm-backend"

    log_info "启动后端服务..."

    if [ "$USE_DOCKER" = "true" ]; then
        docker-compose up -d
    else
        # 后台运行 Go 服务
        nohup go run main.go > logs/app.log 2>&1 &
        echo $! > .app.pid
    fi

    log_info "后端服务已启动 (http://localhost:${SERVER_PORT:-8080})"
}

# 停止服务
stop_service() {
    cd "$PROJECT_ROOT/itsm-backend"

    log_info "停止服务..."

    if [ "$USE_DOCKER" = "true" ]; then
        docker-compose down
    else
        if [ -f .app.pid ]; then
            kill $(cat .app.pid) 2>/dev/null || true
            rm .app.pid
        fi
        pkill -f "go run main.go" || true
    fi

    log_info "服务已停止"
}

# 重启服务
restart_service() {
    stop_service
    sleep 2
    start_service
}

# 查看日志
show_logs() {
    cd "$PROJECT_ROOT/itsm-backend"

    if [ "$USE_DOCKER" = "true" ]; then
        docker-compose logs -f
    else
        if [ -f logs/app.log ]; then
            tail -f logs/app.log
        else
            log_error "日志文件不存在"
        fi
    fi
}

# 清理所有内容
clean_all() {
    log_warn "这将删除所有容器、卷和数据！"
    read -p "确定要继续吗？(yes/no): " confirm

    if [ "$confirm" != "yes" ]; then
        log_info "已取消"
        exit 0
    fi

    cd "$PROJECT_ROOT/itsm-backend"

    if [ -f docker-compose.yml ]; then
        docker-compose down -v
    fi

    if [ -f .app.pid ]; then
        kill $(cat .app.pid) 2>/dev/null || true
        rm .app.pid
    fi

    log_info "清理完成"
}

# Docker 部署
docker_deploy() {
    USE_DOCKER=true
    check_dependencies
    load_env

    log_info "使用 Docker 部署 ITSM 系统..."
    cd "$PROJECT_ROOT/itsm-backend"

    if [ ! -f docker-compose.yml ]; then
        log_error "docker-compose.yml 文件不存在"
        exit 1
    fi

    docker-compose up -d
    log_info "Docker 部署完成"
    log_info "访问地址:"
    log_info "  - 后端 API: http://localhost:8080"
    log_info "  - API 文档: http://localhost:8080/swagger/index.html"
}

# 开发模式
dev_mode() {
    log_info "启动开发模式..."

    # 启动数据库（如果需要）
    cd "$PROJECT_ROOT/itsm-backend"
    if docker compose ps postgres 2>/dev/null | grep -q "Exit"; then
        log_info "启动 PostgreSQL 容器..."
        docker compose up -d postgres
    fi

    # 等待数据库
    wait_for_db

    # 初始化数据库
    init_db

    # 启动后端
    log_info "启动后端服务..."
    cd itsm-backend
    go run main.go &
    BACKEND_PID=$!
    cd ..

    # 启动前端
    log_info "启动前端服务..."
    cd itsm-frontend
    if [ ! -d "node_modules" ]; then
        log_info "安装前端依赖..."
        npm install
    fi
    npm run dev &
    FRONTEND_PID=$!
    cd ..

    # 保存 PID
    echo $BACKEND_PID > .backend.pid
    echo $FRONTEND_PID > .frontend.pid

    log_info "开发模式已启动"
    log_info "  - 后端: http://localhost:8080"
    log_info "  - 前端: http://localhost:3000"
    log_info ""
    log_info "按 Ctrl+C 停止服务"

    # 等待信号
    trap "kill $BACKEND_PID $FRONTEND_PID 2>/dev/null; rm -f .backend.pid .frontend.pid; exit 0" INT TERM
    wait
}

# 创建必要的目录
create_dirs() {
    mkdir -p "$PROJECT_ROOT/itsm-backend/logs"
    mkdir -p "$PROJECT_ROOT/itsm-backend/data"
}

# 主函数
main() {
    local command="${1:-help}"

    create_dirs

    case $command in
        init)
            check_dependencies
            load_env
            wait_for_db
            init_db
            ;;
        migrate)
            check_dependencies
            load_env
            run_migrate
            ;;
        seed)
            check_dependencies
            load_env
            run_seed
            ;;
        start)
            check_dependencies
            load_env
            start_service
            ;;
        stop)
            stop_service
            ;;
        restart)
            restart_service
            ;;
        logs)
            show_logs
            ;;
        clean)
            clean_all
            ;;
        docker)
            docker_deploy
            ;;
        dev)
            check_dependencies
            dev_mode
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            log_error "未知命令: $command"
            show_help
            exit 1
            ;;
    esac
}

# 运行主函数
main "$@"
