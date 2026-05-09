-- ========================================================
-- 为 l1_support 角色添加缺失的基础权限
-- ========================================================
-- 问题: l1_support 角色缺少访问 /api/v1/auth/me 的 user:read 权限
-- 解决: 添加 technician 角色的所有权限到 l1_support
-- ========================================================
-- 查看当前各角色权限数
SELECT r.name,
    r.code,
    COUNT(rp.id) as permission_count
FROM roles r
    LEFT JOIN role_permissions rp ON r.id = rp.role_id
WHERE r.tenant_id = 1
    AND r.code IN ('l1_support', 'technician', 'agent')
GROUP BY r.id,
    r.name,
    r.code
ORDER BY permission_count DESC;
-- 为 l1_support 添加 technician 的所有权限
DO $$
DECLARE l1_role_id BIGINT;
perm_id INTEGER;
BEGIN -- 获取 l1_support 角色ID
SELECT id INTO l1_role_id
FROM roles
WHERE code = 'l1_support'
    AND tenant_id = 1;
-- 获取 technician 角色的所有权限ID并添加到 l1_support
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
    ) THEN
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
-- 为 l1_support 添加 agent 的所有权限
DO $$
DECLARE l1_role_id BIGINT;
perm_id INTEGER;
BEGIN
SELECT id INTO l1_role_id
FROM roles
WHERE code = 'l1_support'
    AND tenant_id = 1;
FOR perm_id IN
SELECT rp.permission_id
FROM role_permissions rp
    JOIN roles r ON rp.role_id = r.id
WHERE r.code = 'agent'
    AND r.tenant_id = 1
    AND rp.permission_id IS NOT NULL LOOP IF NOT EXISTS (
        SELECT 1
        FROM role_permissions
        WHERE role_id = l1_role_id
            AND permission_id = perm_id
    ) THEN
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
-- 添加 l2_support 和 l3_expert 的所有权限
DO $$
DECLARE l1_role_id BIGINT;
perm_id INTEGER;
BEGIN
SELECT id INTO l1_role_id
FROM roles
WHERE code = 'l1_support'
    AND tenant_id = 1;
-- 从 l2_support 复制
FOR perm_id IN
SELECT rp.permission_id
FROM role_permissions rp
    JOIN roles r ON rp.role_id = r.id
WHERE r.code = 'l2_support'
    AND r.tenant_id = 1
    AND rp.permission_id IS NOT NULL LOOP IF NOT EXISTS (
        SELECT 1
        FROM role_permissions
        WHERE role_id = l1_role_id
            AND permission_id = perm_id
    ) THEN
INSERT INTO role_permissions (
        role_id,
        permission_id,
        permission_definition_role_permissions
    )
VALUES (l1_role_id, perm_id, NULL);
END IF;
END LOOP;
-- 从 l3_expert 复制
FOR perm_id IN
SELECT rp.permission_id
FROM role_permissions rp
    JOIN roles r ON rp.role_id = r.id
WHERE r.code = 'l3_expert'
    AND r.tenant_id = 1
    AND rp.permission_id IS NOT NULL LOOP IF NOT EXISTS (
        SELECT 1
        FROM role_permissions
        WHERE role_id = l1_role_id
            AND permission_id = perm_id
    ) THEN
INSERT INTO role_permissions (
        role_id,
        permission_id,
        permission_definition_role_permissions
    )
VALUES (l1_role_id, perm_id, NULL);
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
SELECT rp.permission_id,
    CASE
        WHEN rp.permission_id = 1 THEN 'ticket:*'
        WHEN rp.permission_id = 2 THEN 'ticket:read'
        WHEN rp.permission_id = 3 THEN 'ticket:write'
        WHEN rp.permission_id = 4 THEN 'incident:*'
        WHEN rp.permission_id = 5 THEN 'incident:read'
        WHEN rp.permission_id = 6 THEN 'incident:write'
        WHEN rp.permission_id = 7 THEN 'service_request:*'
        WHEN rp.permission_id = 8 THEN 'service_request:read'
        WHEN rp.permission_id = 9 THEN 'service_request:write'
        WHEN rp.permission_id = 10 THEN 'service_catalog:read'
        WHEN rp.permission_id = 11 THEN 'service_catalog:write'
        WHEN rp.permission_id = 12 THEN 'problem:*'
        WHEN rp.permission_id = 13 THEN 'problem:read'
        WHEN rp.permission_id = 14 THEN 'problem:write'
        WHEN rp.permission_id = 15 THEN 'change:*'
        WHEN rp.permission_id = 16 THEN 'change:read'
        WHEN rp.permission_id = 17 THEN 'change:write'
        WHEN rp.permission_id = 18 THEN 'user:*'
        WHEN rp.permission_id = 19 THEN 'user:read'
        WHEN rp.permission_id = 20 THEN 'user:write'
        WHEN rp.permission_id = 21 THEN 'role:*'
        WHEN rp.permission_id = 22 THEN 'role:read'
        WHEN rp.permission_id = 23 THEN 'role:write'
        WHEN rp.permission_id = 24 THEN 'dashboard:*'
        WHEN rp.permission_id = 25 THEN 'dashboard:read'
        WHEN rp.permission_id = 26 THEN 'knowledge:*'
        WHEN rp.permission_id = 27 THEN 'knowledge:read'
        WHEN rp.permission_id = 28 THEN 'knowledge:write'
        WHEN rp.permission_id = 29 THEN 'cmdb:*'
        WHEN rp.permission_id = 30 THEN 'cmdb:read'
        WHEN rp.permission_id = 31 THEN 'cmdb:write'
        WHEN rp.permission_id = 32 THEN 'sla:*'
        WHEN rp.permission_id = 33 THEN 'sla:read'
        WHEN rp.permission_id = 34 THEN 'sla:write'
        WHEN rp.permission_id = 35 THEN 'bpmn:*'
        WHEN rp.permission_id = 36 THEN 'bpmn:read'
        WHEN rp.permission_id = 37 THEN 'bpmn:write'
        ELSE 'permission_' || rp.permission_id
    END as permission_name
FROM role_permissions rp
    JOIN roles r ON rp.role_id = r.id
WHERE r.code = 'l1_support'
    AND r.tenant_id = 1
ORDER BY rp.permission_id;