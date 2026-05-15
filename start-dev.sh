#!/bin/bash
# ITSM 开发环境启动脚本

set -e

echo "=========================================="
echo "ITSM 开发环境启动"
echo "=========================================="

# 切换到项目根目录
cd "$(dirname "$0")"

# 检查 Docker 是否运行
if ! docker info > /dev/null 2>&1; then
    echo "错误: Docker 未运行，请先启动 Docker Desktop"
    exit 1
fi

echo "1. 启动基础设施服务 (PostgreSQL, Redis)..."
docker compose -f docker-compose.dev.yml up -d postgres redis

# 等待数据库就绪
echo "   等待数据库启动..."
sleep 5

# 检查数据库健康状态
for i in {1..30}; do
    if docker compose -f docker-compose.dev.yml exec -T postgres pg_isready -U dev -d itsm_dev > /dev/null 2>&1; then
        echo "   PostgreSQL 就绪"
        break
    fi
    echo "   等待中... ($i/30)"
    sleep 2
done

# 启动后端
echo "2. 启动后端服务..."
docker compose -f docker-compose.dev.yml up -d --build itsm-backend

# 等待后端启动
echo "   等待后端启动..."
for i in {1..30}; do
    if curl -s http://localhost:8090/api/v1/health > /dev/null 2>&1; then
        echo "   后端服务就绪 (http://localhost:8090)"
        break
    fi
    echo "   等待中... ($i/30)"
    sleep 2
done

# 启动前端
echo "3. 启动前端服务..."
docker compose -f docker-compose.dev.yml up -d --build itsm-frontend

echo ""
echo "=========================================="
echo "服务状态:"
echo "=========================================="
docker compose -f docker-compose.dev.yml ps

echo ""
echo "=========================================="
echo "访问地址:"
echo "=========================================="
echo "- 前端: http://localhost:3000"
echo "- 后端: http://localhost:8090"
echo "- API文档: http://localhost:8090/swagger/index.html"
echo ""
echo "默认账号: admin / admin123"
echo "=========================================="
