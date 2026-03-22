-- 修复菜单层级结构 (简化版)

BEGIN;

-- 1. 添加新的工作流子菜单
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
VALUES
  ('流程设计器', '/workflow/designer', 'Settings', 14, 'workflow:write', 201, 1, true, true),
  ('流程实例', '/workflow/instances', 'Play', 14, 'workflow:read', 202, 1, true, true),
  ('版本管理', '/workflow/versions', 'History', 14, 'workflow:write', 203, 1, true, true),
  ('监控仪表盘', '/workflow/dashboard', 'BarChart3', 14, 'workflow:read', 204, 1, true, true),
  ('审计日志', '/workflow/audit', 'ClipboardList', 14, 'workflow:read', 205, 1, true, true),
  ('SLA监控', '/workflow/sla', 'Clock', 14, 'workflow:read', 206, 1, true, true);

-- 2. 添加MSP子菜单
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
VALUES ('分配管理', '/msp/management', 'Settings', 13, 'msp:manage', 131, 1, true, true);

-- 3. 添加enterprise父菜单和子菜单
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
VALUES ('企业组织', '/enterprise', 'Building', NULL, 'admin:write', 235, 1, true, true);

-- 更新现有菜单的parent_id
UPDATE menus SET parent_id = (SELECT id FROM menus WHERE path='/enterprise' AND tenant_id=1)
WHERE path IN ('/enterprise/departments', '/enterprise/teams') AND tenant_id = 1;

-- 4. 添加admin父菜单
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
VALUES ('系统管理', '/admin', 'Shield', NULL, 'admin:write', 208, 1, true, true);

-- 更新admin子菜单的parent_id
UPDATE menus SET parent_id = (SELECT id FROM menus WHERE path='/admin' AND tenant_id=1)
WHERE path IN ('/admin/users', '/admin/roles', '/admin/groups', '/admin/sla-definitions', '/admin/ticket-categories', '/admin/system-config', '/admin/approvals', '/admin/cmdb-types') AND tenant_id = 1;

-- 5. 添加缺失的admin子菜单
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
VALUES
  ('租户管理', '/admin/tenants', 'Building', (SELECT id FROM menus WHERE path='/admin' AND tenant_id=1), 'admin:write', 212, 1, true, true),
  ('审批链管理', '/admin/approval-chains', 'GitMerge', (SELECT id FROM menus WHERE path='/admin' AND tenant_id=1), 'admin:write', 262, 1, true, true),
  ('工单模板', '/tickets/templates', 'FileText', 2, 'ticket:write', 25, 1, true, true),
  ('票据统计', '/tickets/analytics', 'BarChart3', 2, 'ticket:read', 26, 1, true, true);

COMMIT;

-- 显示菜单层级
SELECT
  m.id,
  m.name,
  m.path,
  p.name as parent_name,
  m.permission_code
FROM menus m
LEFT JOIN menus p ON m.parent_id = p.id
WHERE m.tenant_id = 1
ORDER BY coalesce(m.parent_id, m.id), m.sort_order;
