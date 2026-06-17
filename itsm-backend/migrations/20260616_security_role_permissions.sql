-- 修复 security 角色缺少基础读取权限的问题
-- 为 security 角色补充 ticket:read/list, notification:read/list, knowledge:read/list 等权限

-- 获取 security 角色 ID（所有租户），并插入缺失的权限
INSERT INTO role_permissions (role_id, permission_id, created_at, updated_at)
SELECT r.id, p.id, NOW(), NOW()
FROM roles r
CROSS JOIN permissions p
WHERE r.code = 'security'
AND (
    (p.resource_type IN ('ticket', 'notification', 'knowledge_article') AND p.action IN ('read', 'list'))
    OR (p.resource_type = 'incident' AND p.action IN ('read', 'list'))
    OR (p.resource_type = 'problem' AND p.action IN ('read', 'list'))
    OR (p.resource_type = 'change' AND p.action IN ('read', 'list'))
    OR (p.resource_type = 'dashboard' AND p.action = 'read')
    OR (p.resource_type = 'cmdb' AND p.action = 'read')
    OR (p.resource_type = 'sla' AND p.action = 'read')
)
AND NOT EXISTS (
    SELECT 1 FROM role_permissions rp
    WHERE rp.role_id = r.id AND rp.permission_id = p.id
);
