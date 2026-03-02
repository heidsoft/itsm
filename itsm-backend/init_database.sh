#!/bin/bash

# ITSM 数据库完整初始化脚本

echo "🚀 ITSM 数据库初始化脚本"
echo "========================"
echo ""

# 数据库配置
DB_HOST=${DB_HOST:-"localhost"}
DB_PORT=${DB_PORT:-"5432"}
DB_USER=${DB_USER:-"dev"}
DB_PASSWORD=${DB_PASSWORD:-"dev_password_2026"}
DB_NAME=${DB_NAME:-"itsm"}

export PGPASSWORD=${DB_PASSWORD}

echo "📋 数据库配置:"
echo "   主机：${DB_HOST}:${DB_PORT}"
echo "   用户：${DB_USER}"
echo "   数据库：${DB_NAME}"
echo ""

# 检查数据库连接
echo "🔍 检查数据库连接..."
if ! psql -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d ${DB_NAME} -c "SELECT 1;" > /dev/null 2>&1; then
    echo "❌ 无法连接到数据库！"
    echo ""
    echo "请检查:"
    echo "1. PostgreSQL 服务是否运行"
    echo "2. 数据库配置是否正确"
    echo "3. 防火墙设置"
    echo ""
    echo "启动 PostgreSQL:"
    echo "  sudo systemctl start postgresql"
    echo "  或"
    echo "  sudo service postgresql start"
    exit 1
fi
echo "✅ 数据库连接成功！"
echo ""

# 执行初始化 SQL
echo "📝 执行 SQL 初始化脚本..."
psql -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d ${DB_NAME} << 'EOSQL'
-- 创建默认管理员账号（密码：admin123）
DO $$
DECLARE
    admin_password TEXT := crypt('admin123', gen_salt('bf'));
    tenant_id BIGINT;
BEGIN
    -- 获取默认租户 ID
    SELECT id INTO tenant_id FROM tenants WHERE code = 'default' LIMIT 1;
    
    -- 创建或更新管理员账号
    IF EXISTS (SELECT 1 FROM users WHERE username = 'admin') THEN
        -- 更新现有管理员密码
        UPDATE users SET 
            password_hash = admin_password,
            role = 'admin',
            updated_at = NOW()
        WHERE username = 'admin';
        
        RAISE NOTICE 'ℹ️  管理员账号已存在，密码已重置为：admin123';
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
        
        RAISE NOTICE '✅ 默认管理员账号创建成功！';
    END IF;
    
    -- 显示用户统计
    RAISE NOTICE '';
    RAISE NOTICE '📊 用户统计:';
END $$;

-- 显示用户信息
SELECT '管理员账号:' as 信息，username as 用户名，email as 邮箱，role as 角色，active as 状态 
FROM users WHERE username = 'admin';

SELECT '用户总数:' as 信息，COUNT(*) as 数量 FROM users;
SELECT '管理员数量:' as 信息，COUNT(*) as 数量 FROM users WHERE role = 'admin';
EOSQL

echo ""
echo "✅ 数据库初始化完成！"
echo ""
echo "🔐 默认管理员账号："
echo "   用户名：admin"
echo "   密码：admin123"
echo ""
echo "⚠️  建议首次登录后修改默认密码！"
echo ""
echo "🚀 下一步:"
echo "1. 启动后端服务：cd itsm-backend && go run main.go"
echo "2. 访问管理后台：http://localhost:8080"
echo "3. 使用 admin/admin123 登录"
