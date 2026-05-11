#!/bin/bash

# ITSM 数据库初始化脚本

DB_HOST=${DB_HOST:-"localhost"}
DB_PORT=${DB_PORT:-"5432"}
DB_USER=${DB_USER:-"itsm"}
DB_NAME=${DB_NAME:-"itsm"}
DB_PASSWORD=${DB_PASSWORD:-"itsm_password_2026"}

# Admin password is REQUIRED - no hardcoded default
ADMIN_PASSWORD=${ADMIN_PASSWORD:?"ADMIN_PASSWORD must be set. Run: ADMIN_PASSWORD=your_secure_password $0"}

export DATABASE_URL="host=${DB_HOST} user=${DB_USER} password=${DB_PASSWORD} dbname=${DB_NAME} port=${DB_PORT} sslmode=disable"

echo "🚀 开始初始化 ITSM 数据库..."
echo "数据库：${DB_NAME}@${DB_HOST}:${DB_PORT}"

# 执行 SQL 初始化脚本
echo "📝 执行 SQL 初始化脚本..."
psql "host=${DB_HOST} user=${DB_USER} password=${DB_PASSWORD} dbname=${DB_NAME} port=${DB_PORT}" -v admin_pw="'${ADMIN_PASSWORD}'" << 'EOSQL'
-- 创建默认管理员账号（密码由 ADMIN_PASSWORD 环境变量提供）
DO $$
DECLARE
    admin_password TEXT := crypt(:'admin_pw', gen_salt('bf'));
    tenant_id BIGINT;
    role_id BIGINT;
BEGIN
    -- 获取默认租户 ID
    SELECT id INTO tenant_id FROM tenants WHERE code = 'default' LIMIT 1;

    -- 获取管理员角色 ID
    SELECT id INTO role_id FROM roles WHERE code = 'admin' LIMIT 1;

    -- 创建或更新管理员账号
    IF EXISTS (SELECT 1 FROM users WHERE username = 'admin') THEN
        -- 更新现有管理员密码
        UPDATE users SET
            password_hash = admin_password,
            role = 'admin',
            updated_at = NOW()
        WHERE username = 'admin';

        RAISE NOTICE 'ℹ️  管理员账号已存在，密码已重置为环境变量提供的密码';
    ELSE
        -- 创建新管理员账号
        INSERT INTO users (id, username, email, password_hash, role, name, department, active, tenant_id, created_at, updated_at)
        VALUES (
            'admin-001',
            'admin',
            'admin@itsm.com',
            admin_password,
            'admin',
            '系统管理员',
            'IT 部门',
            true,
            COALESCE(tenant_id, 1),
            NOW(),
            NOW()
        );

        RAISE NOTICE '✅ 管理员账号创建成功！';
        RAISE NOTICE '用户名：admin';
    END IF;
END $$;

-- 显示用户统计
SELECT '用户统计:' as 信息，COUNT(*) as 数量 FROM users
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
