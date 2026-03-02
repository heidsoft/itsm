#!/bin/bash
# =============================================
# ITSM 一键初始化脚本
# 使用方式: ./scripts/init.sh
# =============================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  ITSM 系统初始化脚本${NC}"
echo -e "${GREEN}========================================${NC}"

# 检查环境
check_env() {
    echo -e "\n${YELLOW}[1/5] 检查环境...${NC}"

    # 检查 PostgreSQL
    if ! command -v psql &> /dev/null; then
        echo -e "${RED}错误: 未安装 psql 命令${NC}"
        exit 1
    fi

    # 检查 Go
    if ! command -v go &> /dev/null; then
        echo -e "${RED}错误: 未安装 Go${NC}"
        exit 1
    fi

    echo -e "${GREEN}✓ 环境检查通过${NC}"
}

# 读取配置
read_config() {
    echo -e "\n${YELLOW}[2/5] 读取配置...${NC}"

    # 尝试读取 .env 文件
    if [ -f ".env" ]; then
        source .env
    fi

    # 默认值
    DB_HOST=${DB_HOST:-localhost}
    DB_PORT=${DB_PORT:-5432}
    DB_USER=${DB_USER:-dev}
    DB_NAME=${DB_NAME:-itsm}
    export DB_PASSWORD=${DB_PASSWORD:-StrongPassword123!}
    export JWT_SECRET=${JWT_SECRET:-itsm-jwt-secret-$(date +%s)}
    export ADMIN_PASSWORD=${ADMIN_PASSWORD:-admin123}

    echo -e "  数据库: ${DB_HOST}:${DB_PORT}/${DB_NAME}"
    echo -e "  用户:   ${DB_USER}"
    echo -e "  管理员: admin / ${ADMIN_PASSWORD}"
}

# 初始化数据库
init_db() {
    echo -e "\n${YELLOW}[3/5] 初始化数据库...${NC}"

    # 创建数据库（如果不存在）
    if ! PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -lqt | cut -d \| -f 1 | grep -qw "$DB_NAME"; then
        echo "  创建数据库: $DB_NAME"
        PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -c "CREATE DATABASE $DB_NAME;"
    fi

    # 执行 SQL 初始化
    echo "  执行种子数据..."
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f config/seed/seed_data.sql 2>/dev/null || true

    echo -e "${GREEN}✓ 数据库初始化完成${NC}"
}

# 启动后端
start_backend() {
    echo -e "\n${YELLOW}[4/5] 启动后端服务...${NC}"

    cd itsm-backend

    # 后台运行
    P_PASSWORD" JWT_SECRETGPASSWORD="$DB="$JWT_SECRET" nohup go run main.go > ../itsm-backend.log 2>&1 &
    BACKEND_PID=$!

    # 等待服务启动
    echo "  等待服务启动..."
    for i in {1..30}; do
        if curl -s http://localhost:8090/api/v1/auth/login > /dev/null 2>&1; then
            echo -e "${GREEN}✓ 后端服务启动成功 (PID: $BACKEND_PID)${NC}"
            break
        fi
        sleep 1
    done

    cd ..
}

# 验证
verify() {
    echo -e "\n${YELLOW}[5/5] 验证安装...${NC}"

    # 测试登录
    RESPONSE=$(curl -s -X POST http://localhost:8090/api/v1/auth/login \
        -H "Content-Type: application/json" \
        -d '{"username":"admin","password":"'$ADMIN_PASSWORD'"}')

    if echo "$RESPONSE" | grep -q '"code":0'; then
        echo -e "${GREEN}✓ 登录验证成功${NC}"
    else
        echo -e "${RED}✗ 登录验证失败${NC}"
        exit 1
    fi

    echo -e "\n${GREEN}========================================${NC}"
    echo -e "${GREEN}  初始化完成！${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo "  前端地址: http://localhost:3000"
    echo "  后端地址: http://localhost:8090"
    echo "  API 文档: http://localhost:8090/swagger"
    echo ""
    echo "  管理员账户: admin / $ADMIN_PASSWORD"
    echo ""
}

# 主流程
main() {
    check_env
    read_config
    init_db
    start_backend
    verify
}

main "$@"
