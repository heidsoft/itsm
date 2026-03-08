#!/bin/bash
#
# ITSM 系统部署脚本
#
# 用法:
#   ./scripts/deploy.sh [选项]
#
# 选项:
#   install      - 一键安装（需配置数据库，从 Release 下载安装包）
#   standonline  - 一键在线部署（自动启动数据库 Docker 容器）⭐ 推荐
#   init         - 初始化数据库和种子数据
#   migrate      - 运行数据库迁移
#   seed         - 运行种子数据
#   start        - 启动服务
#   stop         - 停止服务
#   restart      - 重启服务
#   logs         - 查看日志
#   clean        - 清理所有容器和数据
#   docker       - 使用 Docker 部署
#   help         - 显示帮助信息
#
# 环境变量:
#   VERSION      - 指定版本（默认: latest）
#   INSTALL_DIR  - 安装目录（默认: ~/.itsm）
#   GITHUB_REPO  - GitHub 仓库（默认: heidsoft/itsm）
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
    install    一键安装部署（前端+后端）⭐ 推荐
    init       初始化数据库和种子数据 (migrate + seed)
    migrate    运行数据库迁移 (创建/更新表结构)
    seed       运行种子数据 (创建初始数据)
    start      启动服务（仅后端）
    stop       停止服务
    restart    重启服务
    logs       查看服务日志
    clean      清理所有容器和数据 (谨慎使用!)
    docker     使用 Docker Compose 部署
    dev        开发模式启动 (前端+后端)
    help       显示此帮助信息

示例:
    # 一键安装（需先配置数据库，从 Release 下载安装包）
    ./scripts/deploy.sh install

    # 指定版本部署
    VERSION=v1.0.0 ./scripts/deploy.sh install

    # 一键在线部署（自动启动数据库 Docker 容器）⭐ 推荐生产使用
    ./scripts/deploy.sh standonline

    # 指定版本在线部署
    VERSION=v1.0.0 ./scripts/deploy.sh standonline

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

# 检测操作系统
detect_os() {
    case "$(uname -s)" in
        Linux*)     echo "linux";;
        Darwin*)    echo "darwin";;
        MINGW*|MSYS*|CYGWIN*) echo "windows";;
        *)          echo "unknown";;
    esac
}

# 检测 CPU 架构
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)   echo "amd64";;
        aarch64|arm64)  echo "arm64";;
        *)              echo "amd64";;
    esac
}

# 获取最新版本号
get_latest_version() {
    local repo="${GITHUB_REPO:-heidsoft/itsm}"
    curl -sL "https://api.github.com/repos/${repo}/releases/latest" | grep -o '"tag_name": "v[^"]*"' | cut -d'"' -f4
}

# 一键安装部署（从 Release 下载）
install_all() {
    log_info "========== ITSM 一键安装部署 =========="

    # 配置变量
    local os=$(detect_os)
    local arch=$(detect_arch)
    local version="${VERSION:-latest}"
    local install_dir="${INSTALL_DIR:-$HOME/.itsm}"
    local repo="${GITHUB_REPO:-heidsoft/itsm}"

    # 如果是 latest，获取最新版本号
    if [ "$version" = "latest" ]; then
        log_info "获取最新版本..."
        version=$(get_latest_version)
        if [ -z "$version" ]; then
            log_error "无法获取最新版本，请检查网络连接"
            exit 1
        fi
        log_info "最新版本: $version"
    fi

    # 1. 检查依赖
    check_dependencies
    load_env

    # 2. 等待数据库
    wait_for_db

    # 3. 初始化数据库
    log_info "初始化数据库..."
    run_migrate

    # 4. 创建安装目录
    log_info "创建安装目录: $install_dir"
    mkdir -p "$install_dir"/{backend,frontend,logs}
    cd "$install_dir"

    # 5. 下载后端
    log_info "下载后端服务 ($os/$arch)..."
    local backend_file="itsm-${os}-${arch}"
    local backend_url="https://github.com/${repo}/releases/download/${version}/${backend_file}.zip"

    if curl -fSL "$backend_url" -o "${backend_file}.zip" 2>/dev/null; then
        unzip -o "${backend_file}.zip" -d backend/ 2>/dev/null || true
        rm -f "${backend_file}.zip"
        # 查找并移动二进制文件
        local backend_bin=$(find backend -name "itsm-*" -type f 2>/dev/null | head -1)
        if [ -n "$backend_bin" ] && [ -x "$backend_bin" ]; then
            mv "$backend_bin" backend/itsm
        fi
        log_info "✓ 后端下载完成"
    else
        log_warn "后端下载失败，尝试本地构建..."
        cd "$PROJECT_ROOT/itsm-backend"
        go build -ldflags="-s -w" -o itsm main.go
        cp itsm "$install_dir/backend/"
        cd "$install_dir"
    fi

    # 6. 下载前端
    log_info "下载前端服务..."
    local frontend_url="https://github.com/${repo}/releases/download/${version}/itsm-${version}-frontend.zip"

    if curl -fSL "$frontend_url" -o "frontend.zip" 2>/dev/null; then
        unzip -o "frontend.zip" -d frontend/ 2>/dev/null || true
        rm -f frontend.zip
        log_info "✓ 前端下载完成"
    else
        log_warn "前端下载失败，请从 Release 页面手动下载"
    fi

    # 7. 启动后端
    log_info "启动后端服务..."
    cd "$install_dir/backend"
    chmod +x itsm 2>/dev/null || true
    nohup ./itsm > ../logs/backend.log 2>&1 &
    echo $! > .app.pid
    log_info "✓ 后端启动完成"

    # 8. 启动前端
    log_info "启动前端服务..."
    cd "$install_dir/frontend"
    if [ -d ".next" ]; then
        PORT=3000 nohup npm run start > ../logs/frontend.log 2>&1 &
        echo $! > ../.frontend.pid
        log_info "✓ 前端启动完成"
    else
        log_warn "前端未正确安装，跳过启动"
    fi

    # 9. 显示完成信息
    sleep 2

    echo ""
    log_info "=========================================="
    log_info "         部署完成！"
    log_info "=========================================="
    echo ""
    log_info "访问地址:"
    log_info "  - 前端: ${GREEN}http://localhost:3000${NC}"
    log_info "  - 后端: ${GREEN}http://localhost:${SERVER_PORT:-8080}${NC}"
    log_info ""
    log_info "登录账号:"
    log_info "  - 用户名: ${GREEN}admin${NC}"
    log_info "  - 密码:   ${GREEN}admin123${NC}"
    echo ""
    log_info "安装位置: $install_dir"
    log_info ""
    log_info "管理命令:"
    log_info "  - 查看日志: tail -f $install_dir/logs/backend.log"
    log_info "  - 停止服务: cd $install_dir/backend && ./itsm stop"
    echo ""
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

# 一键在线部署（Docker Compose，包含数据库）
standonline() {
    local version="${VERSION:-latest}"
    local repo="${GITHUB_REPO:-heidsoft/itsm}"

    log_info "========== ITSM 一键在线部署 =========="
    log_info "版本: ${version}"

    # 检查 Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装，请先安装 Docker"
        exit 1
    fi
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        log_error "Docker Compose 未安装，请先安装 Docker Compose"
        exit 1
    fi

    # 获取最新版本
    if [ "$version" = "latest" ]; then
        version=$(get_latest_version)
        if [ -z "$version" ]; then
            log_error "无法获取最新版本"
            exit 1
        fi
        log_info "最新版本: $version"
    fi

    # 创建临时目录
    local temp_dir=$(mktemp -d)
    cd "$temp_dir"

    # 下载后端（从 backends.zip 中提取 Linux amd64 版本用于 Docker 容器）
    log_info "下载后端服务..."
    local backends_zip="itsm-${version}-backends.zip"
    local backend_url="https://github.com/${repo}/releases/download/${version}/${backends_zip}"

    if curl -fSL "$backend_url" -o "$backends_zip" 2>/dev/null; then
        unzip -o "$backends_zip" -d backends/
        # Docker 容器使用 linux/amd64
        local backend_bin=$(find backends -name "itsm-linux-amd64" -type f 2>/dev/null | head -1)
        if [ -n "$backend_bin" ]; then
            cp "$backend_bin" itsm-backend
            chmod +x itsm-backend
            log_info "✓ 后端下载完成 (linux/amd64)"
        else
            log_error "未找到 linux/amd64 平台的二进制"
            rm -rf "$temp_dir"
            exit 1
        fi
    else
        log_error "后端下载失败，请检查版本是否正确"
        rm -rf "$temp_dir"
        exit 1
    fi

    # 下载前端（可选）
    log_info "下载前端服务..."
    local frontend_url="https://github.com/${repo}/releases/download/${version}/itsm-${version}-frontend.zip"
    if curl -fSL "$frontend_url" -o frontend.zip 2>/dev/null; then
        unzip -o frontend.zip -d frontend 2>/dev/null
        log_info "✓ 前端下载完成"
    else
        log_warn "前端下载失败（该版本可能未包含前端），跳过"
    fi

    # 下载后端后添加执行权限并复制到项目目录（注意：docker-compose 在 itsm-backend 目录下运行）
    log_info "配置服务..."
    mkdir -p "$PROJECT_ROOT/itsm-backend/deploy"

    # 查找解压后的 linux amd64 二进制（注意目录结构是嵌套的）
    local linux_bin=$(find backends -name "itsm-linux-amd64" -type f 2>/dev/null | head -1)
    if [ -n "$linux_bin" ]; then
        cp "$linux_bin" "$PROJECT_ROOT/itsm-backend/deploy/itsm"
        chmod +x "$PROJECT_ROOT/itsm-backend/deploy/itsm"
        log_info "✓ 后端二进制已复制"
    fi

    # 复制前端（处理 frontend-standalone 目录结构）
    if [ -d "frontend-standalone" ]; then
        cp -r frontend-standalone "$PROJECT_ROOT/itsm-backend/deploy/frontend"
    fi

    # 创建配置文件（使用实际默认值）
    local db_password="${DB_PASSWORD:-StrongPassword123!}"
    local jwt_secret="${JWT_SECRET:-itsm-jwt-secret-2024}"
    cat > "$PROJECT_ROOT/itsm-backend/deploy/config.yaml" << CONFIG_EOF
database:
  host: postgres
  port: 5432
  user: dev
  password: "${db_password}"
  dbname: itsm
  sslmode: disable

server:
  port: 8080
  mode: release
  jwt_secret: "${jwt_secret}"
  cors_origins:
    - "http://localhost:3000"
    - "http://localhost:8080"

redis:
  url: "redis://redis:6379"

log:
  level: info
CONFIG_EOF

    # 使用 docker-compose 部署
    cd "$PROJECT_ROOT/itsm-backend"
    load_env

    # 生成 docker-compose 配置
    cat > docker-compose.standalone.yml << 'COMPOSE_EOF'
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    container_name: itsm-postgres
    environment:
      POSTGRES_DB: itsm
      POSTGRES_USER: dev
      POSTGRES_PASSWORD: ${DB_PASSWORD:-StrongPassword123!}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U dev -d itsm"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: itsm-redis
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

  itsm-backend:
    image: debian:stable-slim
    container_name: itsm-backend
    user: root
    working_dir: /app
    volumes:
      - ./deploy/itsm:/app/itsm
      - ./deploy/config.yaml:/app/config.yaml
    environment:
      - DATABASE_URL=postgres://dev:${DB_PASSWORD:-StrongPassword123!}@postgres:5432/itsm?sslmode=disable
      - REDIS_URL=redis://redis:6379
      - LOG_LEVEL=info
      - JWT_SECRET=${JWT_SECRET:-itsm-jwt-secret-2024}
      - ADMIN_PASSWORD=${ADMIN_PASSWORD:-admin123}
    ports:
      - "8080:8080"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_started
    entrypoint: ["/bin/sh", "-c", "chmod +x /app/itsm && cd /app && /app/itsm"]

  # 前端服务 - 使用 Node.js 运行 Next.js standalone
  frontend:
    image: node:22-alpine
    container_name: itsm-frontend
    working_dir: /app
    volumes:
      - ./deploy/frontend:/app
    environment:
      - PORT=3000
      - HOSTNAME=0.0.0.0
      - NEXT_PUBLIC_API_URL=http://itsm-backend:8080
    ports:
      - "3000:3000"
    command: node server.js
    depends_on:
      - itsm-backend

volumes:
  postgres_data:
  redis_data:
COMPOSE_EOF

    log_info "启动服务（包含数据库）..."
    docker-compose -f docker-compose.standalone.yml up -d

    # 等待服务启动
    sleep 5

    # 清理临时目录
    rm -rf "$temp_dir"

    echo ""
    log_info "=========================================="
    log_info "         部署完成！"
    log_info "=========================================="
    echo ""
    log_info "访问地址:"
    log_info "  - 前端: ${GREEN}http://localhost:3000${NC}"
    log_info "  - 后端: ${GREEN}http://localhost:8080${NC}"
    log_info ""
    log_info "数据库:"
    log_info "  - 地址: localhost:5432"
    log_info "  - 用户: dev"
    log_info "  - 密码: ${DB_PASSWORD:-StrongPassword123!}"
    log_info ""
    log_info "登录账号:"
    log_info "  - 用户名: ${GREEN}admin${NC}"
    log_info "  - 密码:   ${GREEN}admin123${NC}"
    echo ""
    log_info "管理命令:"
    log_info "  - 查看日志: docker-compose -f docker-compose.standalone.yml logs -f"
    log_info "  - 停止服务: docker-compose -f docker-compose.standalone.yml down"
    log_info ""
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
        install)
            install_all
            ;;
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
        standonline)
            standonline
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
