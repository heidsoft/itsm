#!/bin/sh
# ITSM 开发环境停止脚本

set -eu

cd "$(dirname "$0")"

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.dev.yml}"
REMOVE_VOLUMES=0
INCLUDE_LEGACY=0
REMOVE_ORPHANS=0

usage() {
    cat <<'EOF'
ITSM 开发环境停止

用法:
  sh stop-dev.sh [选项]
  ./stop-dev.sh [选项]

选项:
  -h, --help         显示帮助
      --volumes      同时删除开发环境 Docker volumes（会清空开发库数据）
      --remove-orphans
                     同时删除同 compose 项目下的孤儿容器
      --include-legacy
                     同时停止旧版默认 docker-compose.yml 栈

常用:
  sh stop-dev.sh
  sh stop-dev.sh --volumes
  sh stop-dev.sh --include-legacy
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

while [ "$#" -gt 0 ]; do
    case "$1" in
        -h|--help)
            usage
            exit 0
            ;;
        --volumes)
            REMOVE_VOLUMES=1
            ;;
        --remove-orphans)
            REMOVE_ORPHANS=1
            ;;
        --include-legacy)
            INCLUDE_LEGACY=1
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
echo "ITSM 开发环境停止"
echo "=========================================="

require_docker

down_args=""
if [ "$REMOVE_ORPHANS" = "1" ]; then
    down_args="$down_args --remove-orphans"
fi
if [ "$REMOVE_VOLUMES" = "1" ]; then
    down_args="$down_args --volumes"
fi

# shellcheck disable=SC2086
compose down $down_args

if [ "$INCLUDE_LEGACY" = "1" ]; then
    echo ""
    echo "停止旧版默认 docker-compose.yml 栈..."
    # shellcheck disable=SC2086
    docker compose -f docker-compose.yml down $down_args || true
fi

echo ""
echo "当前 ITSM 容器:"
docker ps -a --filter "name=itsm-" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" || true
echo "=========================================="
