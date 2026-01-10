#!/bin/bash

# 本地开发启动脚本
set -e

echo "🚀 启动ITSM CMDB本地开发环境..."

# 检查Go环境
if ! command -v go &> /dev/null; then
    echo "❌ Go未安装，请先安装Go 1.21+"
    exit 1
fi

# 检查Node环境
if ! command -v node &> /dev/null; then
    echo "❌ Node.js未安装，请先安装Node.js 18+"
    exit 1
fi

# 检查PostgreSQL
if ! command -v psql &> /dev/null; then
    echo "⚠️  PostgreSQL客户端未安装，将使用SQLite"
    export DATABASE_URL="sqlite:///tmp/itsm_cmdb.db"
else
    export DATABASE_URL="postgres://postgres:password@localhost:5432/itsm_cmdb?sslmode=disable"
fi

# 设置环境变量
export PORT=8080
export GIN_MODE=debug
export NEXT_PUBLIC_API_URL=http://localhost:8080

cd itsm-backend

# 安装Go依赖
echo "📦 安装Go依赖..."
go mod tidy

# 启动后端服务
echo "🚀 启动CMDB后端服务..."
go run cmd/cmdb/main.go &
BACKEND_PID=$!

# 等待后端启动
sleep 5

# 检查后端健康状态
if curl -f http://localhost:8080/health > /dev/null 2>&1; then
    echo "✅ CMDB后端服务启动成功"
else
    echo "❌ CMDB后端服务启动失败"
    kill $BACKEND_PID 2>/dev/null || true
    exit 1
fi

echo "🌐 服务地址:"
echo "  - API: http://localhost:8080"
echo "  - 健康检查: http://localhost:8080/health"
echo "  - API文档: http://localhost:8080/swagger/index.html"

echo ""
echo "📋 测试API:"
echo "curl http://localhost:8080/health"
echo "curl http://localhost:8080/api/v1/cmdb/classes"

echo ""
echo "按 Ctrl+C 停止服务"

# 等待用户中断
trap "echo '🛑 停止服务...'; kill $BACKEND_PID 2>/dev/null || true; exit 0" INT
wait $BACKEND_PID
