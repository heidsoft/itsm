-- ========================================================
-- 多角色协作测试 - 创建测试用户 (优化版)
-- ========================================================
-- 修复说明:
-- 1. 使用已验证的bcrypt哈希(admin用户相同)
-- 2. 包含完整的user_roles关联
-- 3. 幂等设计，可安全重复执行
-- 4. 添加详细的错误处理和日志
-- 
-- 执行: psql -h localhost -U postgres -d <dbname> -f scripts/init_test_users.sql
-- ========================================================
-- 查看当前用户
SELECT '=== 当前用户列表 ===' as info;
SELECT id,
    username,
    name,
    role
FROM users
WHERE tenant_id = 1
ORDER BY id;
-- ========================================================
-- 获取admin用户的密码哈希 (确保与admin密码一致)
-- ========================================================
DO $$
DECLARE admin_password_hash TEXT;
BEGIN
SELECT password_hash INTO admin_password_hash
FROM users
WHERE username = 'admin'
    AND tenant_id = 1
LIMIT 1;
IF admin_password_hash IS NULL THEN RAISE EXCEPTION 'Admin user not found! Please run initial data migration first.';
END IF;
RAISE NOTICE 'Using admin password hash for test users';
END $$;
-- 获取密码哈希用于后续INSERT
SELECT password_hash INTO :admin_hash
FROM users
WHERE username = 'admin'
    AND tenant_id = 1
LIMIT 1;
-- ========================================================
-- 创建测试用户 (使用动态密码哈希)
-- ========================================================
DO $$
DECLARE admin_password_hash TEXT;
BEGIN
SELECT password_hash INTO admin_password_hash
FROM users
WHERE username = 'admin'
    AND tenant_id = 1
LIMIT 1;
-- 1. 创建测试用户 - 一线工程师 (L1 Support)
INSERT INTO users (
        username,
        password_hash,
        name,
        role,
        tenant_id,
        email,
        phone,
        active,
        created_at,
        updated_at
    )
VALUES (
        'l1_engineer',
        admin_password_hash,
        -- 使用与admin相同的哈希
        '一线工程师-张三',
        'l1_support',
        1,
        'zhangsan@example.com',
        '13800138001',
        true,
        NOW(),
        NOW()
    ) ON CONFLICT (username) DO
UPDATE
SET password_hash = EXCLUDED.password_hash,
    name = EXCLUDED.name,
    active = EXCLUDED.active;
RAISE NOTICE 'l1_engineer created/updated';
-- 2. 创建测试用户 - 二线工程师 (L2 Support)
INSERT INTO users (
        username,
        password_hash,
        name,
        role,
        tenant_id,
        email,
        phone,
        active,
        created_at,
        updated_at
    )
VALUES (
        'l2_engineer',
        admin_password_hash,
        '二线工程师-李四',
        'l2_support',
        1,
        'lisi@example.com',
        '13800138002',
        true,
        NOW(),
        NOW()
    ) ON CONFLICT (username) DO
UPDATE
SET password_hash = EXCLUDED.password_hash,
    name = EXCLUDED.name,
    active = EXCLUDED.active;
RAISE NOTICE 'l2_engineer created/updated';
-- 3. 创建测试用户 - 运维经理 (Ops Manager)
INSERT INTO users (
        username,
        password_hash,
        name,
        role,
        tenant_id,
        email,
        phone,
        active,
        created_at,
        updated_at
    )
VALUES (
        'ops_manager',
        admin_password_hash,
        '运维经理-王五',
        'ops_manager',
        1,
        'wangwu@example.com',
        '13800138003',
        true,
        NOW(),
        NOW()
    ) ON CONFLICT (username) DO
UPDATE
SET password_hash = EXCLUDED.password_hash,
    name = EXCLUDED.name,
    active = EXCLUDED.active;
RAISE NOTICE 'ops_manager created/updated';
-- 4. 创建测试用户 - 三线专家 (L3 Expert)
INSERT INTO users (
        username,
        password_hash,
        name,
        role,
        tenant_id,
        email,
        phone,
        active,
        created_at,
        updated_at
    )
VALUES (
        'l3_expert',
        admin_password_hash,
        '三线专家-赵六',
        'l3_expert',
        1,
        'zhaoliu@example.com',
        '13800138004',
        true,
        NOW(),
        NOW()
    ) ON CONFLICT (username) DO
UPDATE
SET password_hash = EXCLUDED.password_hash,
    name = EXCLUDED.name,
    active = EXCLUDED.active;
RAISE NOTICE 'l3_expert created/updated';
-- 5. 创建测试用户 - 服务台主管 (SD Manager)
INSERT INTO users (
        username,
        password_hash,
        name,
        role,
        tenant_id,
        email,
        phone,
        active,
        created_at,
        updated_at
    )
VALUES (
        'sd_manager',
        admin_password_hash,
        '服务台主管-钱七',
        'sd_manager',
        1,
        'qianqi@example.com',
        '13800138005',
        true,
        NOW(),
        NOW()
    ) ON CONFLICT (username) DO
UPDATE
SET password_hash = EXCLUDED.password_hash,
    name = EXCLUDED.name,
    active = EXCLUDED.active;
RAISE NOTICE 'sd_manager created/updated';
-- 6. 创建测试用户 - 普通员工 (End User + Agent双角色)
INSERT INTO users (
        username,
        password_hash,
        name,
        role,
        tenant_id,
        email,
        phone,
        active,
        created_at,
        updated_at
    )
VALUES (
        'employee',
        admin_password_hash,
        '普通员工-孙八',
        'end_user',
        1,
        'sunba@example.com',
        '13800138006',
        true,
        NOW(),
        NOW()
    ) ON CONFLICT (username) DO
UPDATE
SET password_hash = EXCLUDED.password_hash,
    name = EXCLUDED.name,
    active = EXCLUDED.active;
RAISE NOTICE 'employee created/updated';
END $$;
-- ========================================================
-- 验证用户创建
-- ========================================================
SELECT '=== 创建后的用户 ===' as info;
SELECT id,
    username,
    name,
    role
FROM users
WHERE tenant_id = 1
    AND username IN (
        'l1_engineer',
        'l2_engineer',
        'l3_expert',
        'ops_manager',
        'sd_manager',
        'employee'
    )
ORDER BY id;
-- ========================================================
-- 为测试用户分配角色 (user_roles表关联)
-- ========================================================
DO $$
DECLARE end_user_role_id BIGINT;
agent_role_id BIGINT;
l1_role_id BIGINT;
l2_role_id BIGINT;
l3_role_id BIGINT;
ops_role_id BIGINT;
sd_role_id BIGINT;
sysadmin_role_id BIGINT;
BEGIN -- 查找角色ID
SELECT id INTO end_user_role_id
FROM roles
WHERE code = 'end_user'
    AND tenant_id = 1
LIMIT 1;
SELECT id INTO agent_role_id
FROM roles
WHERE code = 'agent'
    AND tenant_id = 1
LIMIT 1;
SELECT id INTO l1_role_id
FROM roles
WHERE code = 'l1_support'
    AND tenant_id = 1
LIMIT 1;
SELECT id INTO l2_role_id
FROM roles
WHERE code = 'l2_support'
    AND tenant_id = 1
LIMIT 1;
SELECT id INTO l3_role_id
FROM roles
WHERE code = 'l3_expert'
    AND tenant_id = 1
LIMIT 1;
SELECT id INTO ops_role_id
FROM roles
WHERE code = 'ops_manager'
    AND tenant_id = 1
LIMIT 1;
SELECT id INTO sd_role_id
FROM roles
WHERE code = 'sd_manager'
    AND tenant_id = 1
LIMIT 1;
SELECT id INTO sysadmin_role_id
FROM roles
WHERE code = 'sysadmin'
    AND tenant_id = 1
LIMIT 1;
-- 分配角色 - L1工程师
IF l1_role_id IS NOT NULL THEN
INSERT INTO user_roles (user_id, role_id)
SELECT u.id,
    l1_role_id
FROM users u
WHERE u.username = 'l1_engineer' ON CONFLICT (user_id, role_id) DO NOTHING;
RAISE NOTICE 'l1_engineer -> l1_support';
END IF;
-- 分配角色 - L2工程师
IF l2_role_id IS NOT NULL THEN
INSERT INTO user_roles (user_id, role_id)
SELECT u.id,
    l2_role_id
FROM users u
WHERE u.username = 'l2_engineer' ON CONFLICT (user_id, role_id) DO NOTHING;
RAISE NOTICE 'l2_engineer -> l2_support';
END IF;
-- 分配角色 - L3专家
IF l3_role_id IS NOT NULL THEN
INSERT INTO user_roles (user_id, role_id)
SELECT u.id,
    l3_role_id
FROM users u
WHERE u.username = 'l3_expert' ON CONFLICT (user_id, role_id) DO NOTHING;
RAISE NOTICE 'l3_expert -> l3_expert';
END IF;
-- 分配角色 - 运维经理
IF ops_role_id IS NOT NULL THEN
INSERT INTO user_roles (user_id, role_id)
SELECT u.id,
    ops_role_id
FROM users u
WHERE u.username = 'ops_manager' ON CONFLICT (user_id, role_id) DO NOTHING;
RAISE NOTICE 'ops_manager -> ops_manager';
END IF;
-- 分配角色 - 服务台主管
IF sd_role_id IS NOT NULL THEN
INSERT INTO user_roles (user_id, role_id)
SELECT u.id,
    sd_role_id
FROM users u
WHERE u.username = 'sd_manager' ON CONFLICT (user_id, role_id) DO NOTHING;
RAISE NOTICE 'sd_manager -> sd_manager';
END IF;
-- 分配角色 - 员工(双角色: end_user + agent)
IF agent_role_id IS NOT NULL THEN
INSERT INTO user_roles (user_id, role_id)
SELECT u.id,
    agent_role_id
FROM users u
WHERE u.username = 'employee' ON CONFLICT (user_id, role_id) DO NOTHING;
RAISE NOTICE 'employee -> agent';
END IF;
IF end_user_role_id IS NOT NULL THEN
INSERT INTO user_roles (user_id, role_id)
SELECT u.id,
    end_user_role_id
FROM users u
WHERE u.username = 'employee' ON CONFLICT (user_id, role_id) DO NOTHING;
RAISE NOTICE 'employee -> end_user';
END IF;
-- Admin关联sysadmin
IF sysadmin_role_id IS NOT NULL THEN
INSERT INTO user_roles (user_id, role_id)
SELECT u.id,
    sysadmin_role_id
FROM users u
WHERE u.username = 'admin' ON CONFLICT (user_id, role_id) DO NOTHING;
RAISE NOTICE 'admin -> sysadmin';
END IF;
END $$;
-- ========================================================
-- 验证角色分配
-- ========================================================
SELECT '=== 最终角色分配 ===' as info;
SELECT u.username,
    u.name,
    COALESCE(string_agg(r.name, ', '), '无角色') as roles,
    COALESCE(string_agg(r.code, ', '), 'none') as role_codes
FROM users u
    LEFT JOIN user_roles ur ON u.id = ur.user_id
    LEFT JOIN roles r ON ur.role_id = r.id
WHERE u.username IN (
        'l1_engineer',
        'l2_engineer',
        'l3_expert',
        'ops_manager',
        'sd_manager',
        'employee',
        'admin'
    )
GROUP BY u.id,
    u.username,
    u.name
ORDER BY u.id;
-- ========================================================
-- 测试密码验证
-- ========================================================
SELECT '=== 密码哈希验证 ===' as info;
SELECT username,
    CASE
        WHEN password_hash = (
            SELECT password_hash
            FROM users
            WHERE username = 'admin'
            LIMIT 1
        ) THEN '✓ 与admin相同'
        ELSE '✗ 不匹配'
    END as password_status
FROM users
WHERE username IN ('l1_engineer', 'employee');
-- ========================================================
-- 1. 创建测试用户 - 一线工程师 (L1 Support)
-- ========================================================
INSERT INTO users (
        username,
        password_hash,
        name,
        role,
        tenant_id,
        email,
        phone,
        active,
        created_at,
        updated_at
    )
VALUES (
        'l1_engineer',
        '$2a$10$N9qo8uLOickgx2ZMRZoMye0HvRB/OG.QbKvVgjSgxQJdCJhT0O0S',
        '一线工程师-张三',
        'l1_support',
        1,
        'zhangsan@example.com',
        '13800138001',
        true,
        NOW(),
        NOW()
    ) ON CONFLICT (username) DO NOTHING;
-- ========================================================
-- 2. 创建测试用户 - 二线工程师 (L2 Support)
-- ========================================================
INSERT INTO users (
        username,
        password_hash,
        name,
        role,
        tenant_id,
        email,
        phone,
        active,
        created_at,
        updated_at
    )
VALUES (
        'l2_engineer',
        '$2a$10$N9qo8uLOickgx2ZMRZoMye0HvRB/OG.QbKvVgjSgxQJdCJhT0O0S',
        '二线工程师-李四',
        'l2_support',
        1,
        'lisi@example.com',
        '13800138002',
        true,
        NOW(),
        NOW()
    ) ON CONFLICT (username) DO NOTHING;
-- ========================================================
-- 3. 创建测试用户 - 运维经理 (Ops Manager)
-- ========================================================
INSERT INTO users (
        username,
        password_hash,
        name,
        role,
        tenant_id,
        email,
        phone,
        active,
        created_at,
        updated_at
    )
VALUES (
        'ops_manager',
        '$2a$10$N9qo8uLOickgx2ZMRZoMye0HvRB/OG.QbKvVgjSgxQJdCJhT0O0S',
        '运维经理-王五',
        'ops_manager',
        1,
        'wangwu@example.com',
        '13800138003',
        true,
        NOW(),
        NOW()
    ) ON CONFLICT (username) DO NOTHING;
-- ========================================================
-- 4. 创建测试用户 - 三线专家 (L3 Expert)
-- ========================================================
INSERT INTO users (
        username,
        password_hash,
        name,
        role,
        tenant_id,
        email,
        phone,
        active,
        created_at,
        updated_at
    )
VALUES (
        'l3_expert',
        '$2a$10$N9qo8uLOickgx2ZMRZoMye0HvRB/OG.QbKvVgjSgxQJdCJhT0O0S',
        '三线专家-赵六',
        'l3_expert',
        1,
        'zhaoliu@example.com',
        '13800138004',
        true,
        NOW(),
        NOW()
    ) ON CONFLICT (username) DO NOTHING;
-- ========================================================
-- 5. 创建测试用户 - 服务台主管 (SD Manager)
-- ========================================================
INSERT INTO users (
        username,
        password_hash,
        name,
        role,
        tenant_id,
        email,
        phone,
        active,
        created_at,
        updated_at
    )
VALUES (
        'sd_manager',
        '$2a$10$N9qo8uLOickgx2ZMRZoMye0HvRB/OG.QbKvVgjSgxQJdCJhT0O0S',
        '服务台主管-钱七',
        'sd_manager',
        1,
        'qianqi@example.com',
        '13800138005',
        true,
        NOW(),
        NOW()
    ) ON CONFLICT (username) DO NOTHING;
-- ========================================================
-- 6. 创建测试用户 - 普通员工 (End User)
-- ========================================================
INSERT INTO users (
        username,
        password_hash,
        name,
        role,
        tenant_id,
        email,
        phone,
        active,
        created_at,
        updated_at
    )
VALUES (
        'employee',
        '$2a$10$N9qo8uLOickgx2ZMRZoMye0HvRB/OG.QbKvVgjSgxQJdCJhT0O0S',
        '普通员工-孙八',
        'end_user',
        1,
        'sunba@example.com',
        '13800138006',
        true,
        NOW(),
        NOW()
    ) ON CONFLICT (username) DO NOTHING;
-- ========================================================
-- 验证用户创建
-- ========================================================
SELECT id,
    username,
    name,
    role
FROM users
WHERE tenant_id = 1
ORDER BY id;
-- ========================================================
-- 为测试用户分配角色 (user_roles表关联)
-- ========================================================
-- 获取角色ID
DO $$
DECLARE end_user_role_id BIGINT;
l1_role_id BIGINT;
l2_role_id BIGINT;
l3_role_id BIGINT;
ops_role_id BIGINT;
sd_role_id BIGINT;
admin_role_id BIGINT;
l1_user_id BIGINT;
l2_user_id BIGINT;
l3_user_id BIGINT;
ops_user_id BIGINT;
sd_user_id BIGINT;
employee_user_id BIGINT;
admin_user_id BIGINT;
BEGIN -- 查找角色ID
SELECT id INTO end_user_role_id
FROM roles
WHERE code = 'end_user'
    AND tenant_id = 1
LIMIT 1;
SELECT id INTO l1_role_id
FROM roles
WHERE code = 'l1_support'
    AND tenant_id = 1
LIMIT 1;
SELECT id INTO l2_role_id
FROM roles
WHERE code = 'l2_support'
    AND tenant_id = 1
LIMIT 1;
SELECT id INTO l3_role_id
FROM roles
WHERE code = 'l3_expert'
    AND tenant_id = 1
LIMIT 1;
SELECT id INTO ops_role_id
FROM roles
WHERE code = 'ops_manager'
    AND tenant_id = 1
LIMIT 1;
SELECT id INTO sd_role_id
FROM roles
WHERE code = 'sd_manager'
    AND tenant_id = 1
LIMIT 1;
SELECT id INTO admin_role_id
FROM roles
WHERE code = 'sysadmin'
    AND tenant_id = 1
LIMIT 1;
-- 查找用户ID
SELECT id INTO l1_user_id
FROM users
WHERE username = 'l1_engineer'
LIMIT 1;
SELECT id INTO l2_user_id
FROM users
WHERE username = 'l2_engineer'
LIMIT 1;
SELECT id INTO l3_user_id
FROM users
WHERE username = 'l3_expert'
LIMIT 1;
SELECT id INTO ops_user_id
FROM users
WHERE username = 'ops_manager'
LIMIT 1;
SELECT id INTO sd_user_id
FROM users
WHERE username = 'sd_manager'
LIMIT 1;
SELECT id INTO employee_user_id
FROM users
WHERE username = 'employee'
LIMIT 1;
SELECT id INTO admin_user_id
FROM users
WHERE username = 'admin'
LIMIT 1;
-- 分配角色
IF l1_user_id IS NOT NULL
AND l1_role_id IS NOT NULL THEN
INSERT INTO user_roles (user_id, role_id)
VALUES (l1_user_id, l1_role_id) ON CONFLICT DO NOTHING;
RAISE NOTICE 'l1_engineer assigned to l1_support role';
END IF;
IF l2_user_id IS NOT NULL
AND l2_role_id IS NOT NULL THEN
INSERT INTO user_roles (user_id, role_id)
VALUES (l2_user_id, l2_role_id) ON CONFLICT DO NOTHING;
RAISE NOTICE 'l2_engineer assigned to l2_support role';
END IF;
IF l3_user_id IS NOT NULL
AND l3_role_id IS NOT NULL THEN
INSERT INTO user_roles (user_id, role_id)
VALUES (l3_user_id, l3_role_id) ON CONFLICT DO NOTHING;
RAISE NOTICE 'l3_expert assigned to l3_expert role';
END IF;
IF ops_user_id IS NOT NULL
AND ops_role_id IS NOT NULL THEN
INSERT INTO user_roles (user_id, role_id)
VALUES (ops_user_id, ops_role_id) ON CONFLICT DO NOTHING;
RAISE NOTICE 'ops_manager assigned to ops_manager role';
END IF;
IF sd_user_id IS NOT NULL
AND sd_role_id IS NOT NULL THEN
INSERT INTO user_roles (user_id, role_id)
VALUES (sd_user_id, sd_role_id) ON CONFLICT DO NOTHING;
RAISE NOTICE 'sd_manager assigned to sd_manager role';
END IF;
IF employee_user_id IS NOT NULL
AND end_user_role_id IS NOT NULL THEN
INSERT INTO user_roles (user_id, role_id)
VALUES (employee_user_id, end_user_role_id) ON CONFLICT DO NOTHING;
RAISE NOTICE 'employee assigned to end_user role';
END IF;
IF admin_user_id IS NOT NULL
AND admin_role_id IS NOT NULL THEN
INSERT INTO user_roles (user_id, role_id)
VALUES (admin_user_id, admin_role_id) ON CONFLICT DO NOTHING;
RAISE NOTICE 'admin assigned to sysadmin role';
END IF;
END $$;
-- 验证角色分配
SELECT u.username,
    u.name,
    r.name as role_name,
    r.code as role_code
FROM users u
    LEFT JOIN user_roles ur ON u.id = ur.user_id
    LEFT JOIN roles r ON ur.role_id = r.id
WHERE u.tenant_id = 1
    AND u.username IN (
        'l1_engineer',
        'l2_engineer',
        'l3_expert',
        'ops_manager',
        'sd_manager',
        'employee',
        'admin'
    )
ORDER BY u.id;