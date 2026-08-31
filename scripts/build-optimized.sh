#!/bin/bash
# ITSM 优化构建脚本
# 用法: ./scripts/build-optimized.sh [options]
# 选项:
#   --no-cache    不使用缓存构建
#   --parallel    并行构建前后端（默认）
#   --sequential  顺序构建
#   --push        构建后推送到registry

set -euo pipefail

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 默认参数
NO_CACHE=""
PARALLEL=true
PUSH=false
ENV_FILE=".env.prod"
COMPOSE_FILE="docker-compose.prod.yml"
VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "latest")}

# 解析参数
while [[ $# -gt 0 ]]; do
    case $1 in
        --no-cache)
            NO_CACHE="--no-cache"
            shift
            ;;
        --parallel)
            PARALLEL=true
            shift
            ;;
        --sequential)
            PARALLEL=false
            shift
            ;;
        --push)
            PUSH=true
            shift
            ;;
        --version)
            VERSION="$2"
            shift 2
            ;;
        --env-file)
            ENV_FILE="$2"
            shift 2
            ;;
        *)
            log_error "未知参数: $1"
            exit 1
            ;;
    esac
done

# 检查环境文件
if [ ! -f "$ENV_FILE" ]; then
    log_error "环境文件不存在: $ENV_FILE"
    exit 1
fi

# 加载环境变量
set -a
source "$ENV_FILE"
set +a

log_info "开始构建 ITSM 系统"
log_info "版本: $VERSION"
log_info "环境文件: $ENV_FILE"
log_info "并行构建: $PARALLEL"

# 记录开始时间
START_TIME=$(date +%s)

# 设置BuildKit
export DOCKER_BUILDKIT=1
export COMPOSE_DOCKER_CLI_BUILD=1

# 构建参数
BUILD_ARGS=(
    "VERSION=$VERSION"
    "BUILDKIT_INLINE_CACHE=1"
)

# 创建构建缓存目录
CACHE_DIR=".build-cache"
mkdir -p "$CACHE_DIR"

# 函数：构建后端
build_backend() {
    log_info "开始构建后端..."
    local backend_start=$(date +%s)
    
    docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" \
        build $NO_CACHE \
        --build-arg "VERSION=$VERSION" \
        itsm-backend
    
    local backend_end=$(date +%s)
    local backend_duration=$((backend_end - backend_start))
    log_success "后端构建完成，耗时: ${backend_duration}秒"
}

# 函数：构建前端
build_frontend() {
    log_info "开始构建前端..."
    local frontend_start=$(date +%s)
    
    docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" \
        build $NO_CACHE \
        --build-arg "VERSION=$VERSION" \
        itsm-frontend
    
    local frontend_end=$(date +%s)
    local frontend_duration=$((frontend_end - frontend_start))
    log_success "前端构建完成，耗时: ${frontend_duration}秒"
}

# 执行构建
if [ "$PARALLEL" = true ]; then
    log_info "并行构建前后端..."
    
    # 并行启动构建
    build_backend &
    BACKEND_PID=$!
    
    build_frontend &
    FRONTEND_PID=$!
    
    # 等待构建完成
    wait $BACKEND_PID
    BACKEND_EXIT=$?
    
    wait $FRONTEND_PID
    FRONTEND_EXIT=$!
    
    # 检查构建结果
    if [ $BACKEND_EXIT -ne 0 ]; then
        log_error "后端构建失败"
        exit 1
    fi
    
    if [ $FRONTEND_EXIT -ne 0 ]; then
        log_error "前端构建失败"
        exit 1
    fi
else
    log_info "顺序构建前后端..."
    build_backend
    build_frontend
fi

# 记录结束时间
END_TIME=$(date +%s)
TOTAL_DURATION=$((END_TIME - START_TIME))

log_success "构建完成！"
log_info "总耗时: ${TOTAL_DURATION}秒"

# 显示镜像信息
log_info "构建的镜像:"
docker images | grep -E "itsm-backend|itsm-frontend" | head -5

# 推送镜像（如果启用）
if [ "$PUSH" = true ]; then
    log_info "推送镜像到registry..."
    docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" push
    log_success "镜像推送完成"
fi

# 清理旧镜像（保留最近3个版本）
log_info "清理旧镜像..."
docker images --format "table {{.Repository}}\t{{.Tag}}\t{{.ID}}\t{{.CreatedAt}}" | \
    grep -E "itsm-backend|itsm-frontend" | \
    sort -k4 -r | \
    tail -n +4 | \
    awk '{print $3}' | \
    xargs -r docker rmi 2>/dev/null || true

log_success "构建流程完成！"
