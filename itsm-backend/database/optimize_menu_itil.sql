-- ITSM 菜单层级优化 - 基于 ITIL v4 标准
-- 执行前请备份数据

BEGIN;

-- ============================================
-- 1. 删除现有菜单（保留结构仅删除数据）
-- ============================================
DELETE FROM menus WHERE tenant_id = 1;

-- ============================================
-- 2. 创建完整的 ITIL v4 菜单结构
-- ============================================

-- 顶层菜单 (按 ITIL 优先级排序)
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled) VALUES
-- 主菜单区 (sort_order: 1-99)
('服务台', '/dashboard', 'LayoutDashboard', NULL, '', 10, 1, true, true),
('服务请求', '/service-requests', 'FileText', NULL, 'ticket:read', 20, 1, true, true),
('我的请求', '/my-requests', 'User', NULL, 'ticket:read', 25, 1, true, true),
('事件管理', '/incidents', 'AlertCircle', NULL, 'incident:read', 30, 1, true, true),
('问题管理', '/problems', 'HelpCircle', NULL, 'problem:read', 40, 1, true, true),
('变更管理', '/changes', 'BarChart3', NULL, 'change:read', 50, 1, true, true),
('知识库', '/knowledge', 'Book', NULL, 'knowledge:read', 60, 1, true, true),
('服务目录', '/service-catalog', 'BookOpen', NULL, 'service:read', 70, 1, true, true),
('CMDB', '/cmdb', 'Database', NULL, 'cmdb:read', 80, 1, true, true),
('资产管理', '/assets', 'Monitor', NULL, 'asset:read', 90, 1, true, true),
('SLA管理', '/sla-dashboard', 'Calendar', NULL, 'sla:read', 100, 1, true, true),
('报表中心', '/reports', 'TrendingUp', NULL, 'report:read', 110, 1, true, true),
('工作流', '/workflow', 'GitMerge', NULL, 'workflow:read', 120, 1, true, true),
('AI助手', '/ai/chat', 'Bot', NULL, 'ai:view', 130, 1, true, true),

-- 运维工具区 (sort_order: 200-299)
('客户管理', '/msp', 'Building', NULL, 'msp:read', 200, 1, true, true),
('发布管理', '/releases', 'Rocket', NULL, 'release:read', 210, 1, true, true),

-- 系统管理区 (sort_order: 300-399)
('系统管理', '/admin', 'Settings', NULL, 'admin:write', 300, 1, true, true);

-- ============================================
-- 3. 子菜单 - 服务请求管理
-- ============================================
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '服务请求模板', '/tickets/templates', 'FileText', id, 'ticket:write', 21, 1, true, true FROM menus WHERE path = '/service-requests';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '工单统计', '/tickets/analytics', 'BarChart3', id, 'ticket:read', 22, 1, true, true FROM menus WHERE path = '/service-requests';

-- ============================================
-- 4. 子菜单 - 事件管理
-- ============================================
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '重大事件', '/incidents/major', 'AlertTriangle', id, 'incident:read', 31, 1, true, true FROM menus WHERE path = '/incidents';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '事件分析', '/incidents/analytics', 'TrendingUp', id, 'incident:read', 32, 1, true, true FROM menus WHERE path = '/incidents';

-- ============================================
-- 5. 子菜单 - 问题管理
-- ============================================
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '已知错误', '/problems/known-errors', 'AlertCircle', id, 'problem:read', 41, 1, true, true FROM menus WHERE path = '/problems';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '根因分析', '/problems/root-cause', 'Search', id, 'problem:read', 42, 1, true, true FROM menus WHERE path = '/problems';

-- ============================================
-- 6. 子菜单 - 变更管理
-- ============================================
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '标准变更', '/changes/standard', 'CheckCircle', id, 'change:read', 51, 1, true, true FROM menus WHERE path = '/changes';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '紧急变更', '/changes/emergency', 'Zap', id, 'change:read', 52, 1, true, true FROM menus WHERE path = '/changes';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '变更审批', '/changes/approvals', 'GitMerge', id, 'change:read', 53, 1, true, true FROM menus WHERE path = '/changes';

-- ============================================
-- 7. 子菜单 - 知识库
-- ============================================
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '文章管理', '/knowledge/articles', 'FileText', id, 'knowledge:write', 61, 1, true, true FROM menus WHERE path = '/knowledge';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '文章分类', '/knowledge/categories', 'Tag', id, 'knowledge:write', 62, 1, true, true FROM menus WHERE path = '/knowledge';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '知识审核', '/knowledge/review', 'CheckSquare', id, 'knowledge:write', 63, 1, true, true FROM menus WHERE path = '/knowledge';

-- ============================================
-- 8. 子菜单 - 服务目录
-- ============================================
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '服务项管理', '/admin/service-catalogs', 'List', id, 'service:write', 71, 1, true, true FROM menus WHERE path = '/service-catalog';

-- ============================================
-- 9. 子菜单 - CMDB
-- ============================================
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '配置项列表', '/cmdb/cis', 'Server', id, 'cmdb:read', 81, 1, true, true FROM menus WHERE path = '/cmdb';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT 'CI类型管理', '/admin/cmdb-types', 'Database', id, 'cmdb:write', 82, 1, true, true FROM menus WHERE path = '/cmdb';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '云资源', '/cmdb/cloud-resources', 'Cloud', id, 'cmdb:read', 83, 1, true, true FROM menus WHERE path = '/cmdb';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '云账号', '/cmdb/cloud-accounts', 'Key', id, 'cmdb:read', 84, 1, true, true FROM menus WHERE path = '/cmdb';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '云服务', '/cmdb/cloud-services', 'Boxes', id, 'cmdb:read', 85, 1, true, true FROM menus WHERE path = '/cmdb';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '同步对账', '/cmdb/reconciliation', 'RefreshCw', id, 'cmdb:write', 86, 1, true, true FROM menus WHERE path = '/cmdb';

-- ============================================
-- 10. 子菜单 - 资产管理
-- ============================================
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '资产列表', '/assets', 'Server', id, 'asset:read', 91, 1, true, true FROM menus WHERE path = '/assets';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '许可证管理', '/licenses', 'Key', id, 'license:read', 92, 1, true, true FROM menus WHERE path = '/assets';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '资产分类', '/assets/categories', 'Tag', id, 'asset:write', 93, 1, true, true FROM menus WHERE path = '/assets';

-- ============================================
-- 11. 子菜单 - SLA管理
-- ============================================
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT 'SLA监控', '/sla-dashboard', 'Activity', id, 'sla:read', 101, 1, true, true FROM menus WHERE path = '/sla-dashboard';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT 'SLA配置', '/admin/sla-definitions', 'Calendar', id, 'sla:write', 102, 1, true, true FROM menus WHERE path = '/sla-dashboard';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '升级规则', '/admin/escalation-rules', 'ArrowUpCircle', id, 'sla:write', 103, 1, true, true FROM menus WHERE path = '/sla-dashboard';

-- ============================================
-- 12. 子菜单 - 报表中心
-- ============================================
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '工单报表', '/reports/tickets', 'FileText', id, 'report:read', 111, 1, true, true FROM menus WHERE path = '/reports';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '事件报表', '/reports/incidents', 'AlertCircle', id, 'report:read', 112, 1, true, true FROM menus WHERE path = '/reports';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '问题报表', '/reports/problems', 'HelpCircle', id, 'report:read', 113, 1, true, true FROM menus WHERE path = '/reports';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '变更报表', '/reports/changes', 'BarChart3', id, 'report:read', 114, 1, true, true FROM menus WHERE path = '/reports';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT 'SLA报表', '/reports/sla-performance', 'Calendar', id, 'report:read', 115, 1, true, true FROM menus WHERE path = '/reports';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT 'CMDB质量', '/reports/cmdb-quality', 'Database', id, 'report:read', 116, 1, true, true FROM menus WHERE path = '/reports';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '服务目录使用', '/reports/service-catalog-usage', 'BookOpen', id, 'report:read', 117, 1, true, true FROM menus WHERE path = '/reports';

-- ============================================
-- 13. 子菜单 - 工作流
-- ============================================
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '工作流列表', '/workflow', 'List', id, 'workflow:read', 121, 1, true, true FROM menus WHERE path = '/workflow';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '流程设计器', '/workflow/designer', 'Edit', id, 'workflow:write', 122, 1, true, true FROM menus WHERE path = '/workflow';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '流程实例', '/workflow/instances', 'Play', id, 'workflow:read', 123, 1, true, true FROM menus WHERE path = '/workflow';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '版本管理', '/workflow/versions', 'History', id, 'workflow:write', 124, 1, true, true FROM menus WHERE path = '/workflow';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '监控仪表盘', '/workflow/dashboard', 'Activity', id, 'workflow:read', 125, 1, true, true FROM menus WHERE path = '/workflow';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '审计日志', '/workflow/audit', 'ClipboardList', id, 'workflow:read', 126, 1, true, true FROM menus WHERE path = '/workflow';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '工单自动化', '/workflow/ticket-approval', 'Zap', id, 'workflow:write', 127, 1, true, true FROM menus WHERE path = '/workflow';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '自动化规则', '/workflow/automation', 'Settings', id, 'workflow:write', 128, 1, true, true FROM menus WHERE path = '/workflow';

-- ============================================
-- 14. 子菜单 - 客户管理 (原MSP)
-- ============================================
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '客户仪表盘', '/msp', 'LayoutDashboard', id, 'msp:read', 201, 1, true, true FROM menus WHERE path = '/msp';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '分配管理', '/msp/management', 'Settings', id, 'msp:manage', 202, 1, true, true FROM menus WHERE path = '/msp';

-- ============================================
-- 15. 子菜单 - 发布管理
-- ============================================
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '发布计划', '/releases', 'Calendar', id, 'release:read', 211, 1, true, true FROM menus WHERE path = '/releases';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '新建发布', '/releases/new', 'Plus', id, 'release:write', 212, 1, true, true FROM menus WHERE path = '/releases';

-- ============================================
-- 16. 子菜单 - 系统管理
-- ============================================
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '系统概览', '/admin', 'LayoutDashboard', id, 'admin:write', 301, 1, true, true FROM menus WHERE path = '/admin';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '用户管理', '/admin/users', 'Users', id, 'user:read', 302, 1, true, true FROM menus WHERE path = '/admin';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '角色管理', '/admin/roles', 'Shield', id, 'role:read', 303, 1, true, true FROM menus WHERE path = '/admin';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '组管理', '/admin/groups', 'Users', id, 'group:read', 304, 1, true, true FROM menus WHERE path = '/admin';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '租户管理', '/admin/tenants', 'Building', id, 'admin:write', 305, 1, true, true FROM menus WHERE path = '/admin';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '部门管理', '/enterprise/departments', 'Building', id, 'department:read', 306, 1, true, true FROM menus WHERE path = '/admin';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '团队管理', '/enterprise/teams', 'Users', id, 'team:read', 307, 1, true, true FROM menus WHERE path = '/admin';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '工单分类', '/admin/ticket-categories', 'Tag', id, 'ticket:write', 308, 1, true, true FROM menus WHERE path = '/admin';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '工单分配规则', '/admin/tickets/assignment-rules', 'GitBranch', id, 'ticket:write', 309, 1, true, true FROM menus WHERE path = '/admin';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '自动化规则', '/admin/tickets/automation-rules', 'Zap', id, 'ticket:write', 310, 1, true, true FROM menus WHERE path = '/admin';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '审批管理', '/admin/approvals', 'GitMerge', id, 'approval:read', 311, 1, true, true FROM menus WHERE path = '/admin';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '审批链管理', '/admin/approval-chains', 'Link', id, 'admin:write', 312, 1, true, true FROM menus WHERE path = '/admin';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '权限管理', '/admin/permissions', 'Lock', id, 'admin:write', 313, 1, true, true FROM menus WHERE path = '/admin';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '系统配置', '/admin/system-config', 'Settings', id, 'system:write', 314, 1, true, true FROM menus WHERE path = '/admin';

-- ============================================
-- 17. 特殊页面（无子菜单）
-- ============================================
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled) VALUES
('待我审批', '/approvals/pending', 'CheckCircle', NULL, 'approval:read', 400, 1, true, true),
('AI创建工单', '/tickets/ai-create', 'Sparkles', NULL, 'ticket:write', 401, 1, true, true);

COMMIT;

-- ============================================
-- 验证查询
-- ============================================
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
