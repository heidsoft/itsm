-- 补偿迁移：补齐存量 sysadmin 角色对 MCP 通道码 mcp:read 的授权。
--
-- 背景（codex 权限审计 P2-5）：
--   `20261118_mcp_read_permission.sql` 只给 agent / audit_admin 建了角色关联，
--   漏了 sysadmin。但 `internal/authz/roles.go` 的 BuiltinRolePermissionCodes
--   把 sysadmin 授权为 allPermissionCodes()（含 mcp:read）。
--   全新安装经 seeder 播种，sysadmin 正常；存量部署 ITSM_AUTO_SEED=false，
--   升级只跑迁移，sysadmin 没有该关联 → DBOnly configured 态不回退默认值 →
--   sysadmin 访问 MCP 得 403，super_admin 的 *:* 直通会掩盖这个问题。
--
-- 产品决策（用户拍板）：全量自动补齐 sysadmin 的 mcp:read 关联。
--   本迁移只做「新增缺失关联」，不删除任何既有授权，不改权限码 IsActive 状态。
--   管理员如果此前主动撤权，本次也会被补回——这是全量策略的既定取舍，
--   已在 UPGRADE.md 明示，并由 _down.sql 提供精确回退。
--
-- 幂等性：
--   - role_permissions 无 (role_id, permission_id) 唯一约束，用 NOT EXISTS 防重；
--   - 按租户 join（r.tenant_id = p.tenant_id），不跨租户串码；
--   - 权限码不存在（全新库 seeder 尚未跑）时插入 0 行，不报错；
--   - 可重复执行。

BEGIN;

INSERT INTO role_permissions (role_id, permission_id, tenant_id)
SELECT r.id, p.id, r.tenant_id
FROM roles r
JOIN permissions p
  ON p.code = 'mcp:read'
 AND p.tenant_id = r.tenant_id
WHERE r.code = 'sysadmin'
  AND NOT EXISTS (
      SELECT 1 FROM role_permissions rp
      WHERE rp.role_id = r.id
        AND rp.permission_id = p.id
        AND rp.tenant_id = r.tenant_id
  );

COMMIT;
