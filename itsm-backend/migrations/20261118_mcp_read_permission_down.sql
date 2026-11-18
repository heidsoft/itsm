-- 回滚 MCP 通道权限码播种（2026-11-18）。
--
-- 移除 mcp:read 权限码及其角色关联。
-- 影响：回滚后 MCP 端点仍然注册，但任何角色的调用都会被 RequirePermission
-- 判 403「权限不足」（DB 里查不到该码）——即通道不可用，与未部署 MCP 同等。
-- 不影响 human 主体的其它权限码，也不影响既有角色授权。

DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE code = 'mcp:read'
);

DELETE FROM permissions WHERE code = 'mcp:read';
