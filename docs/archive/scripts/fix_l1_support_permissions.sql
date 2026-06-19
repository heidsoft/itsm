-- ========================================================
-- 修复 l1_support 角色权限缺失问题 (修复版 v3)
-- ========================================================
-- 问题原因: role_permissions 表没有唯一约束在 (role_id, permission_id)
-- 解决: 直接 INSERT，不使用 ON CONFLICT
-- ========================================================
-- 查看当前各角色权限数
SELECT r.name,
    r.code,
    COUNT(rp.id) as permission_count
FROM roles r
    LEFT JOIN role_permissions rp ON r.id = rp.role_id
WHERE r.tenant_id = 1
    AND r.code IN (
        'l1_support',
        'l2_support',
        'l3_expert',
        'technician',
        'agent'
    )
GROUP BY r.id,
    r.name,
    r.code
ORDER BY permission_count DESC;
-- 从 technician 角色复制权限到 l1_support
-- 直接从role_permissions表获取permission_id
DO $$
DECLARE l1_role_id BIGINT;
tech_permission_ids INTEGER [];
perm_id INTEGER;
BEGIN -- 获取 l1_support 角色ID
SELECT id INTO l1_role_id
FROM roles
WHERE code = 'l1_support'
    AND tenant_id = 1;
-- 获取 technician 角色的所有权限ID
FOR perm_id IN
SELECT rp.permission_id
FROM role_permissions rp
    JOIN roles r ON rp.role_id = r.id
WHERE r.code = 'technician'
    AND r.tenant_id = 1
    AND rp.permission_id IS NOT NULL LOOP -- 检查是否已存在
    IF NOT EXISTS (
        SELECT 1
        FROM role_permissions
        WHERE role_id = l1_role_id
            AND permission_id = perm_id
    ) THEN -- 插入新权限
INSERT INTO role_permissions (
        role_id,
        permission_id,
        permission_definition_role_permissions
    )
VALUES (l1_role_id, perm_id, NULL);
RAISE NOTICE 'Added permission % to l1_support',
perm_id;
END IF;
END LOOP;
END $$;
-- 从 agent 角色复制权限到 l1_support
DO $$
DECLARE l1_role_id BIGINT;
perm_id INTEGER;
BEGIN -- 获取 l1_support 角色ID
SELECT id INTO l1_role_id
FROM roles
WHERE code = 'l1_support'
    AND tenant_id = 1;
-- 获取 agent 角色的所有权限ID
FOR perm_id IN
SELECT rp.permission_id
FROM role_permissions rp
    JOIN roles r ON rp.role_id = r.id
WHERE r.code = 'agent'
    AND r.tenant_id = 1
    AND rp.permission_id IS NOT NULL LOOP -- 检查是否已存在
    IF NOT EXISTS (
        SELECT 1
        FROM role_permissions
        WHERE role_id = l1_role_id
            AND permission_id = perm_id
    ) THEN -- 插入新权限
INSERT INTO role_permissions (
        role_id,
        permission_id,
        permission_definition_role_permissions
    )
VALUES (l1_role_id, perm_id, NULL);
RAISE NOTICE 'Added permission % to l1_support from agent',
perm_id;
END IF;
END LOOP;
END $$;
-- 验证权限已分配
SELECT r.name,
    r.code,
    COUNT(rp.id) as permission_count
FROM roles r
    LEFT JOIN role_permissions rp ON r.id = rp.role_id
WHERE r.tenant_id = 1
    AND r.code = 'l1_support'
GROUP BY r.id,
    r.name,
    r.code;
-- 查看 l1_support 的具体权限列表
SELECT r.name as role_name,
    pd.resource,
    pd.action,
    pd.description
FROM role_permissions rp
    JOIN roles r ON rp.role_id = r.id
    JOIN permission_definitions pd ON rp.permission_id = pd.id
WHERE r.code = 'l1_support'
    AND r.tenant_id = 1
ORDER BY pd.resource,
    pd.action;