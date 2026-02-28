#!/bin/bash
# 自动化规则数据初始化脚本
# 用于初始化 ITSM 系统的工单自动化规则示例数据

set -e

# 数据库配置
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-itsm_cmdb}"
DB_USER="${DB_USER:-postgres}"
export PGPASSWORD="${DB_PASSWORD:-password}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SQL_FILE="${SCRIPT_DIR}/../itsm-backend/sql/init_automation_rules.sql"

echo "======================================"
echo "ITSM 自动化规则数据初始化"
echo "======================================"
echo "数据库主机：${DB_HOST}"
echo "数据库端口：${DB_PORT}"
echo "数据库名称：${DB_NAME}"
echo "数据库用户：${DB_USER}"
echo "SQL 文件：${SQL_FILE}"
echo "======================================"

# 检查 SQL 文件是否存在
if [ ! -f "${SQL_FILE}" ]; then
    echo "❌ 错误：SQL 文件不存在：${SQL_FILE}"
    exit 1
fi

# 检查 psql 是否可用
if ! command -v psql &> /dev/null; then
    echo "❌ 错误：psql 命令未找到，请确保 PostgreSQL 客户端已安装"
    exit 1
fi

# 执行 SQL
echo "📝 执行自动化规则初始化 SQL..."
psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -f "${SQL_FILE}"

echo ""
echo "✅ 自动化规则数据初始化完成！"
echo ""

# 验证数据
echo "📊 验证自动化规则数据："
psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -c "SELECT id, name, priority, is_active FROM ticket_automation_rules ORDER BY priority;"

# 清理环境变量
unset PGPASSWORD

echo ""
echo "======================================"
echo "初始化成功！"
echo "======================================"
