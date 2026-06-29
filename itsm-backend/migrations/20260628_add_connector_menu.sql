-- 2026-06-28 添加连接器市场菜单
-- 适用于所有租户

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '连接器市场', '/admin/connectors', 'Plug', id, 'connector:read', 317, t.id, true, true 
FROM menus m
CROSS JOIN tenants t
WHERE m.path = '/admin' AND m.tenant_id = t.id
ON CONFLICT (tenant_id, path) DO NOTHING;

-- 如果是单租户环境，也可以直接执行：
-- INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
-- SELECT '连接器市场', '/admin/connectors', 'Plug', id, 'connector:read', 317, 1, true, true FROM menus WHERE path = '/admin' AND tenant_id = 1 ON CONFLICT DO NOTHING;
