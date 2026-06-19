-- ========================================================
-- 修复用户角色关联脚本
-- 必须在有写入权限的数据库连接中执行
-- ========================================================
-- 1. 为 l1_engineer 分配 l1_support 角色
INSERT INTO user_roles (user_id, role_id)
SELECT u.id,
    r.id
FROM users u,
    roles r
WHERE u.username = 'l1_engineer'
    AND r.code = 'l1_support'
    AND u.tenant_id = r.tenant_id ON CONFLICT (user_id, role_id) DO NOTHING;
-- 2. 为 l2_engineer 分配 l2_support 角色
INSERT INTO user_roles (user_id, role_id)
SELECT u.id,
    r.id
FROM users u,
    roles r
WHERE u.username = 'l2_engineer'
    AND r.code = 'l2_support'
    AND u.tenant_id = r.tenant_id ON CONFLICT (user_id, role_id) DO NOTHING;
-- 3. 为 l3_expert 分配 l3_expert 角色
INSERT INTO user_roles (user_id, role_id)
SELECT u.id,
    r.id
FROM users u,
    roles r
WHERE u.username = 'l3_expert'
    AND r.code = 'l3_expert'
    AND u.tenant_id = r.tenant_id ON CONFLICT (user_id, role_id) DO NOTHING;
-- 4. 为 ops_manager 分配 ops_manager 角色
INSERT INTO user_roles (user_id, role_id)
SELECT u.id,
    r.id
FROM users u,
    roles r
WHERE u.username = 'ops_manager'
    AND r.code = 'ops_manager'
    AND u.tenant_id = r.tenant_id ON CONFLICT (user_id, role_id) DO NOTHING;
-- 5. 为 sd_manager 分配 sd_manager 角色
INSERT INTO user_roles (user_id, role_id)
SELECT u.id,
    r.id
FROM users u,
    roles r
WHERE u.username = 'sd_manager'
    AND r.code = 'sd_manager'
    AND u.tenant_id = r.tenant_id ON CONFLICT (user_id, role_id) DO NOTHING;
-- 6. 为 employee 分配 agent 角色 (有完整事件读写权限)
INSERT INTO user_roles (user_id, role_id)
SELECT u.id,
    r.id
FROM users u,
    roles r
WHERE u.username = 'employee'
    AND r.code = 'agent'
    AND u.tenant_id = r.tenant_id ON CONFLICT (user_id, role_id) DO NOTHING;
-- 验证角色分配结果
SELECT u.username,
    u.name,
    r.name as role_name,
    r.code as role_code
FROM users u
    LEFT JOIN user_roles ur ON u.id = ur.user_id
    LEFT JOIN roles r ON ur.role_id = r.id
WHERE u.username IN (
        'l1_engineer',
        'l2_engineer',
        'l3_expert',
        'ops_manager',
        'sd_manager',
        'employee'
    )
ORDER BY u.id;