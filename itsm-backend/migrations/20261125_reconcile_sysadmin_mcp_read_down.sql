-- 回滚 20261125 的 sysadmin MCP 授权补偿。
--
-- 精确移除 sysadmin 角色对 mcp:read 的关联，不动其它角色、其它码、权限码本身。
-- 注意：由于 SQL 无法区分「本次补入」与「本次之前就存在」的关联，本 down
-- 会移除所有 sysadmin 的 mcp:read 关联——这与「全量补齐」的上行决策对称。
-- 若升级前有管理员主动配置过该关联，回滚后需人工恢复。
--
-- 不影响 agent / audit_admin 的 MCP 授权，也不删除 permissions 行。

BEGIN;

DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT p.id FROM permissions p WHERE p.code = 'mcp:read'
)
AND role_id IN (
    SELECT r.id FROM roles r WHERE r.code = 'sysadmin'
);

COMMIT;
