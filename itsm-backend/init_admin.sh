#!/bin/bash

# ITSM 数据库初始化脚本：创建初始管理员账号（幂等，可重复执行）。
#
# 幂等语义（R3 修复）：admin 已存在时跳过——不重置口令、不调整角色；
# 重跑不得把已提为 super_admin 的账号降级，也不得覆盖运维改过的口令。
#
# 安全语义（R3 修复）：
#   - 连接口令走 PGPASSWORD 环境变量、ADMIN_PASSWORD 只经 stdin 进入 psql，
#     均不出现在进程列表（ps 可见的 argv）；
#   - 默认租户缺失时 fail closed，不再回退 tenant_id=1；
#   - id 不再硬编码（users.id 为整型，由数据库默认生成），email 不再硬编码，
#     可用 ADMIN_EMAIL 覆盖，默认与 Go seeder 的产品基线一致。

set -euo pipefail

DB_HOST=${DB_HOST:-"localhost"}
DB_PORT=${DB_PORT:-"5432"}
DB_USER=${DB_USER:-"itsm"}
DB_NAME=${DB_NAME:-"itsm"}
# Both credentials are REQUIRED - no hardcoded defaults.
DB_PASSWORD=${DB_PASSWORD:?"DB_PASSWORD must be set. Run: DB_PASSWORD=your_db_password $0"}
ADMIN_PASSWORD=${ADMIN_PASSWORD:?"ADMIN_PASSWORD must be set. Run: ADMIN_PASSWORD=your_secure_password $0"}
ADMIN_EMAIL=${ADMIN_EMAIL:-"admin@example.com"}

# SQL 字面量转义（单引号翻倍）。值只经 stdin 进 psql，不进 argv。
ADMIN_PASSWORD_SQL=$(printf '%s' "$ADMIN_PASSWORD" | sed "s/'/''/g")
ADMIN_EMAIL_SQL=$(printf '%s' "$ADMIN_EMAIL" | sed "s/'/''/g")

echo "🚀 开始初始化 ITSM 管理员账号..."
echo "数据库：${DB_NAME}@${DB_HOST}:${DB_PORT}"

# heredoc 不加引号以便注入上面两个转义后的值；SQL 体内的 $$ 必须写成 \$\$，
# 否则 bash 会把 $$ 展开为当前 shell 的 PID。
PGPASSWORD="$DB_PASSWORD" psql \
    -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
    -v ON_ERROR_STOP=1 << EOSQL
DO \$\$
DECLARE
    admin_password TEXT := crypt('${ADMIN_PASSWORD_SQL}', gen_salt('bf'));
    tenant_id BIGINT;
BEGIN
    SELECT id INTO tenant_id FROM tenants WHERE code = 'default' LIMIT 1;

    IF tenant_id IS NULL THEN
        RAISE EXCEPTION '默认租户（code=default）不存在：请先完成系统初始化（租户播种），本脚本不做租户回退';
    END IF;

    IF EXISTS (SELECT 1 FROM users WHERE username = 'admin') THEN
        RAISE NOTICE 'ℹ️  管理员账号已存在，跳过（不重置口令、不调整角色）';
    ELSE
        INSERT INTO users (username, email, password_hash, role, name, department, active, tenant_id, created_at, updated_at)
        VALUES (
            'admin',
            '${ADMIN_EMAIL_SQL}',
            admin_password,
            'admin',
            '系统管理员',
            'IT 部门',
            true,
            tenant_id,
            NOW(),
            NOW()
        );
        RAISE NOTICE '✅ 管理员账号创建成功！用户名：admin';
    END IF;
END
\$\$;

SELECT '用户统计:' as 信息, COUNT(*) as 数量 FROM users
UNION ALL
SELECT '管理员数量:', COUNT(*) FROM users WHERE role = 'admin';
EOSQL

echo ""
echo "✅ 数据库初始化完成！"
echo ""
echo "🔐 管理员账号："
echo "   用户名：admin"
echo "   密码：（由 ADMIN_PASSWORD 环境变量设置）"
echo ""
echo "⚠️  请确保 ADMIN_PASSWORD 已通过环境变量设置，切勿使用弱密码！"
