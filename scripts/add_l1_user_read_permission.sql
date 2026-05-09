-- ========================================================
-- 为 l1_support 添加缺失的 user:read 权限 (最终修复)
-- ========================================================
-- 问题: l1_support 角色缺少 permission_id=30 (user:read)
-- 导致 /api/v1/auth/me 返回 403
-- ========================================================
-- 查看当前权限状态
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
-- 添加 user:read 权限 (permission_id=30)
DO $$
DECLARE l1_role_id BIGINT;
BEGIN
SELECT id INTO l1_role_id
FROM roles
WHERE code = 'l1_support'
    AND tenant_id = 1;
-- 检查是否已存在
IF NOT EXISTS (
    SELECT 1
    FROM role_permissions
    WHERE role_id = l1_role_id
        AND permission_id = 30
) THEN
INSERT INTO role_permissions (
        role_id,
        permission_id,
        permission_definition_role_permissions
    )
VALUES (l1_role_id, 30, NULL);
RAISE NOTICE 'Added user:read (permission_id=30) to l1_support';
ELSE RAISE NOTICE 'user:read permission already exists for l1_support';
END IF;
END $$;
-- 验证权限已添加
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
-- 确认 permission_id=30 已添加
SELECT rp.permission_id,
    p.resource,
    p.action
FROM role_permissions rp
    JOIN roles r ON rp.role_id = r.id
    JOIN permissions p ON rp.permission_id = p.id
WHERE r.code = 'l1_support'
    AND r.tenant_id = 1
ORDER BY rp.permission_id;