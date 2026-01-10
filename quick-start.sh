#!/bin/bash

# 简化的CMDB启动脚本
set -e

echo "🚀 启动ITSM CMDB系统..."

# 检查Docker
if ! command -v docker &> /dev/null; then
    echo "❌ Docker未安装"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo "❌ Docker Compose未安装"
    exit 1
fi

# 创建必要目录
mkdir -p logs nginx/conf.d monitoring scripts

# 生成基础配置
cat > .env << 'EOF'
DATABASE_URL=postgres://postgres:password@postgres:5432/itsm_cmdb?sslmode=disable
REDIS_URL=redis://redis:6379
PORT=8080
GIN_MODE=release
NEXT_PUBLIC_API_URL=http://localhost:8080
EOF

# 启动基础服务
echo "📦 启动基础服务..."
docker-compose up -d postgres redis

# 等待数据库启动
echo "⏳ 等待数据库启动..."
sleep 10

# 构建并启动应用
echo "🔨 构建应用..."
docker-compose build cmdb-backend

echo "🚀 启动CMDB服务..."
docker-compose up -d cmdb-backend

echo "✅ CMDB后端服务已启动"
echo "🌐 API地址: http://localhost:8080"
echo "🔍 健康检查: http://localhost:8080/health"

# 显示日志
echo "📋 查看服务日志:"
docker-compose logs -f cmdb-backend
