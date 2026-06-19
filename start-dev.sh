#!/bin/sh
# ITSM 开发环境启动脚本

set -eu

cd "$(dirname "$0")"

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.dev.yml}"
WAIT_RETRIES="${WAIT_RETRIES:-30}"
WAIT_INTERVAL="${WAIT_INTERVAL:-2}"

BUILD=1
START_FRONTEND=1
INFRA_ONLY=0
WITH_MINIO=0
WITH_OLLAMA=0
WITH_OBSERVABILITY=0
SKIP_PORT_CHECK=0

usage() {
    cat <<'EOF'
ITSM 开发环境启动

用法:
  sh start-dev.sh [选项]
  ./start-dev.sh [选项]

选项:
  -h, --help              显示帮助，不启动服务
      --infra-only        只启动 PostgreSQL / Redis 基础设施
      --backend-only      启动基础设施和后端，不启动前端
      --no-frontend       同 --backend-only
      --no-build          使用已有镜像，不重新构建后端/前端
      --with-minio        同时启动 MinIO
      --with-ollama       同时启动 Ollama
      --with-observability 同时启动 Prometheus / Grafana
      --skip-port-check   跳过 3000/8090 端口冲突预检查

常用:
  sh start-dev.sh
  sh start-dev.sh --backend-only
  sh start-dev.sh --infra-only
  sh stop-dev.sh
EOF
}

compose() {
    docker compose -f "$COMPOSE_FILE" --profile dev "$@"
}

require_docker() {
    if ! docker info >/dev/null 2>&1; then
        echo "错误: Docker 未运行，请先启动 Docker Desktop"
        exit 1
    fi
}

check_port_conflict() {
    port="$1"
    expected_container="$2"
    label="$3"

    if [ "$SKIP_PORT_CHECK" = "1" ]; then
        return
    fi

    conflicts="$(docker ps --filter "publish=$port" --format '{{.Names}}' | grep -v "^${expected_container}$" || true)"
    if [ -n "$conflicts" ]; then
        echo "错误: ${label} 端口 ${port} 已被其他容器占用:"
        echo "$conflicts" | sed 's/^/  - /'
        echo ""
        echo "如需停止本项目旧版默认栈，可执行: sh stop-dev.sh --include-legacy"
        echo "如你确认端口占用可忽略，可使用: sh start-dev.sh --skip-port-check"
        exit 1
    fi
}

wait_for_postgres() {
    echo "   等待 PostgreSQL 启动..."
    i=1
    while [ "$i" -le "$WAIT_RETRIES" ]; do
        if compose exec -T postgres pg_isready -U itsm_user -d itsm >/dev/null 2>&1; then
            echo "   PostgreSQL 就绪"
            return
        fi
        echo "   等待中... (${i}/${WAIT_RETRIES})"
        sleep "$WAIT_INTERVAL"
        i=$((i + 1))
    done

    echo "错误: PostgreSQL 未在预期时间内就绪"
    compose logs --tail=80 postgres || true
    exit 1
}

wait_for_http() {
    label="$1"
    url="$2"
    service="$3"

    if ! command -v curl >/dev/null 2>&1; then
        echo "   未找到 curl，跳过 ${label} HTTP 健康检查"
        return
    fi

    echo "   等待 ${label} 启动..."
    i=1
    while [ "$i" -le "$WAIT_RETRIES" ]; do
        if curl -fsS "$url" >/dev/null 2>&1; then
            echo "   ${label} 就绪 (${url})"
            return
        fi
        echo "   等待中... (${i}/${WAIT_RETRIES})"
        sleep "$WAIT_INTERVAL"
        i=$((i + 1))
    done

    echo "错误: ${label} 未在预期时间内就绪"
    compose logs --tail=120 "$service" || true
    exit 1
}

up_app_service() {
    if [ "$BUILD" = "1" ]; then
        compose up -d --build "$@"
    else
        compose up -d "$@"
    fi
}

while [ "$#" -gt 0 ]; do
    case "$1" in
        -h|--help)
            usage
            exit 0
            ;;
        --infra-only)
            INFRA_ONLY=1
            START_FRONTEND=0
            ;;
        --backend-only|--no-frontend)
            START_FRONTEND=0
            ;;
        --no-build)
            BUILD=0
            ;;
        --with-minio)
            WITH_MINIO=1
            ;;
        --with-ollama)
            WITH_OLLAMA=1
            ;;
        --with-observability)
            WITH_OBSERVABILITY=1
            ;;
        --skip-port-check)
            SKIP_PORT_CHECK=1
            ;;
        *)
            echo "未知选项: $1"
            echo ""
            usage
            exit 2
            ;;
    esac
    shift
done

echo "=========================================="
echo "ITSM 开发环境启动"
echo "=========================================="

require_docker

if [ "$INFRA_ONLY" = "0" ]; then
    check_port_conflict 8090 itsm-backend-dev "后端"
fi
if [ "$START_FRONTEND" = "1" ]; then
    check_port_conflict 3000 itsm-frontend-dev "前端"
fi

echo "1. 启动基础设施服务 (PostgreSQL, Redis)..."
infra_services="postgres redis"
if [ "$WITH_MINIO" = "1" ]; then
    infra_services="$infra_services minio"
fi
compose up -d $infra_services
wait_for_postgres

if [ "$WITH_OLLAMA" = "1" ]; then
    echo "2. 启动 Ollama..."
    compose up -d ollama
fi

if [ "$WITH_OBSERVABILITY" = "1" ]; then
    echo "2. 启动可观测性组件 (Prometheus, Grafana)..."
    compose up -d prometheus grafana
fi

if [ "$INFRA_ONLY" = "1" ]; then
    echo ""
    echo "=========================================="
    echo "服务状态:"
    echo "=========================================="
    compose ps
    exit 0
fi

echo "2. 启动后端服务..."
up_app_service itsm-backend
wait_for_http "后端服务" "http://localhost:8090/api/v1/health" "itsm-backend"

if [ "$START_FRONTEND" = "1" ]; then
    echo "3. 启动前端服务..."
    up_app_service itsm-frontend
    wait_for_http "前端服务" "http://localhost:3000" "itsm-frontend"
else
    echo "3. 跳过前端服务"
fi

echo ""
echo "=========================================="
echo "服务状态:"
echo "=========================================="
compose ps

echo ""
echo "=========================================="
echo "访问地址:"
echo "=========================================="
if [ "$START_FRONTEND" = "1" ]; then
    echo "- 前端: http://localhost:3000"
fi
echo "- 后端: http://localhost:8090"
echo "- API文档: http://localhost:8090/swagger/index.html"
echo ""
echo "默认账号: admin / admin123"
echo "=========================================="
