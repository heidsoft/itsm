-- ITSM 菜单结构修复 - 清理重复 + 完善 ITIL v4
-- 执行前请备份数据

BEGIN;

-- ============================================
-- 1. 删除所有现有菜单
-- ============================================
DELETE FROM menus WHERE tenant_id = 1;

-- ============================================
-- 2. 顶层菜单 (parent_id = NULL)
-- ============================================
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled) VALUES
-- 主菜单区 (1-99): 服务台、工单、事件、问题、变更
('服务台', '/dashboard', 'LayoutDashboard', NULL, '', 10, 1, true, true),
('服务请求', '/service-requests', 'FileText', NULL, 'ticket:read', 20, 1, true, true),
('我的请求', '/my-requests', 'User', NULL, 'ticket:read', 25, 1, true, true),
('事件管理', '/incidents', 'AlertCircle', NULL, 'incident:read', 30, 1, true, true),
('问题管理', '/problems', 'HelpCircle', NULL, 'problem:read', 40, 1, true, true),
('变更管理', '/changes', 'BarChart3', NULL, 'change:read', 50, 1, true, true),

-- 服务运营区 (60-99): 知识库、服务目录、CMDB、资产
('知识库', '/knowledge', 'Book', NULL, 'knowledge:read', 60, 1, true, true),
('服务目录', '/service-catalog', 'BookOpen', NULL, 'service:read', 70, 1, true, true),
('CMDB', '/cmdb', 'Database', NULL, 'cmdb:read', 80, 1, true, true),
('资产管理', '/assets', 'Monitor', NULL, 'asset:read', 90, 1, true, true),

-- 运维支持区 (100-149): SLA、报表、工作流、AI
('SLA管理', '/sla', 'Clock', NULL, 'sla:read', 100, 1, true, true),
('报表中心', '/reports', 'TrendingUp', NULL, 'report:read', 110, 1, true, true),
('工作流', '/workflow', 'GitMerge', NULL, 'workflow:read', 120, 1, true, true),
('AI助手', '/ai/chat', 'Bot', NULL, 'ai:use', 130, 1, true, true),

-- 扩展模块 (150-199): 访问管理、服务台调度
('访问管理', '/access', 'Key', NULL, 'access:read', 150, 1, true, true),
('服务台调度', '/shifts', 'Calendar', NULL, 'helpdesk:manage', 160, 1, true, true),

-- MSP与发布 (200-249)
('客户管理', '/msp', 'Building', NULL, 'msp:read', 200, 1, true, true),
('发布管理', '/releases', 'Rocket', NULL, 'release:read', 210, 1, true, true),

-- 系统管理 (300+)
('系统管理', '/admin', 'Settings', NULL, 'admin:write', 300, 1, true, true);

-- ============================================
-- 3. 服务请求子菜单
-- ============================================
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '服务请求模板', '/tickets/templates', 'FileText', id, 'ticket:write', 21, 1, true, true FROM menus WHERE path = '/service-requests';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '工单统计', '/tickets/analytics', 'BarChart3', id, 'ticket:read', 22, 1, true, true FROM menus WHERE path = '/service-requests';

-- ============================================
-- 4. 事件管理子菜单
-- ============================================
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '重大事件', '/incidents/major', 'AlertTriangle', id, 'incident:read', 31, 1, true, true FROM menus WHERE path = '/incidents';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '事件分析', '/incidents/analytics', 'TrendingUp', id, 'incident:read', 32, 1, true, true FROM menus WHERE path = '/incidents';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '事件转问题', '/incidents/convert', 'ArrowRight', id, 'incident:manage', 33, 1, true, true FROM menus WHERE path = '/incidents';

-- ============================================
-- 5. 问题管理子菜单
-- ============================================
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '已知错误', '/problems/known-errors', 'AlertCircle', id, 'problem:read', 41, 1, true, true FROM menus WHERE path = '/problems';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '根因分析', '/problems/root-cause', 'Search', id, 'problem:analyze', 42, 1, true, true FROM menus WHERE path = '/problems';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '问题链接', '/problems/links', 'Link', id, 'problem:manage', 43, 1, true, true FROM menus WHERE path = '/problems';

-- ============================================
-- 6. 变更管理子菜单
-- ============================================
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '标准变更', '/changes/standard', 'CheckCircle', id, 'change:read', 51, 1, true, true FROM menus WHERE path = '/changes';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '紧急变更', '/changes/emergency', 'Zap', id, 'change:manage', 52, 1, true, true FROM menus WHERE path = '/changes';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '变更审批', '/changes/approvals', 'GitMerge', id, 'change:approve', 53, 1, true, true FROM menus WHERE path = '/changes';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT 'CAB会议', '/changes/cab', 'Users', id, 'change:manage', 54, 1, true, true FROM menus WHERE path = '/changes';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '变更日历', '/changes/calendar', 'Calendar', id, 'change:read', 55, 1, true, true FROM menus WHERE path = '/changes';

-- ============================================
-- 7. 知识库子菜单
-- ============================================
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '文章管理', '/knowledge/articles', 'FileText', id, 'knowledge:write', 61, 1, true, true FROM menus WHERE path = '/knowledge';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '文章分类', '/knowledge/categories', 'Tag', id, 'knowledge:manage', 62, 1, true, true FROM menus WHERE path = '/knowledge';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '知识审核', '/knowledge/review', 'CheckSquare', id, 'knowledge:approve', 63, 1, true, true FROM menus WHERE path = '/knowledge';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT 'AI知识推荐', '/knowledge/ai-recommend', 'Sparkles', id, 'knowledge:read', 64, 1, true, true FROM menus WHERE path = '/knowledge';

-- ============================================
-- 8. 服务目录子菜单
-- ============================================
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '服务项管理', '/service-catalog/items', 'List', id, 'service:write', 71, 1, true, true FROM menus WHERE path = '/service-catalog';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '服务类别', '/service-catalog/categories', 'Folder', id, 'service:write', 72, 1, true, true FROM menus WHERE path = '/service-catalog';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '服务请求模板', '/service-catalog/templates', 'FileText', id, 'service:write', 73, 1, true, true FROM menus WHERE path = '/service-catalog';

-- ============================================
-- 9. CMDB子菜单
-- ============================================
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '配置项列表', '/cmdb/cis', 'Server', id, 'cmdb:read', 81, 1, true, true FROM menus WHERE path = '/cmdb';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT 'CI类型管理', '/cmdb/types', 'Database', id, 'cmdb:write', 82, 1, true, true FROM menus WHERE path = '/cmdb';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '云资源', '/cmdb/cloud-resources', 'Cloud', id, 'cmdb:read', 83, 1, true, true FROM menus WHERE path = '/cmdb';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '云账号', '/cmdb/cloud-accounts', 'Key', id, 'cmdb:manage', 84, 1, true, true FROM menus WHERE path = '/cmdb';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '云服务', '/cmdb/cloud-services', 'Boxes', id, 'cmdb:read', 85, 1, true, true FROM menus WHERE path = '/cmdb';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '同步对账', '/cmdb/reconciliation', 'RefreshCw', id, 'cmdb:write', 86, 1, true, true FROM menus WHERE path = '/cmdb';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '关系图谱', '/cmdb/relationships', 'GitBranch', id, 'cmdb:read', 87, 1, true, true FROM menus WHERE path = '/cmdb';

-- ============================================
-- 10. 资产管理子菜单
-- ============================================
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '资产列表', '/assets/list', 'Server', id, 'asset:read', 91, 1, true, true FROM menus WHERE path = '/assets';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '软件许可证', '/assets/licenses', 'Key', id, 'license:manage', 92, 1, true, true FROM menus WHERE path = '/assets';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '资产分类', '/assets/categories', 'Tag', id, 'asset:write', 93, 1, true, true FROM menus WHERE path = '/assets';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '维保管理', '/assets/maintenance', 'Wrench', id, 'asset:manage', 94, 1, true, true FROM menus WHERE path = '/assets';

-- ============================================
-- 11. SLA管理子菜单
-- ============================================
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT 'SLA监控', '/sla/monitor', 'Activity', id, 'sla:read', 101, 1, true, true FROM menus WHERE path = '/sla';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT 'SLA定义', '/sla/definitions', 'FileText', id, 'sla:write', 102, 1, true, true FROM menus WHERE path = '/sla';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '升级规则', '/sla/escalations', 'ArrowUpCircle', id, 'sla:write', 103, 1, true, true FROM menus WHERE path = '/sla';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT 'SLA报表', '/sla/reports', 'BarChart3', id, 'sla:read', 104, 1, true, true FROM menus WHERE path = '/sla';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '目标管理', '/sla/targets', 'Target', id, 'sla:manage', 105, 1, true, true FROM menus WHERE path = '/sla';

-- ============================================
-- 12. 报表中心子菜单
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
SELECT 'SLA报表', '/reports/sla', 'Calendar', id, 'report:read', 115, 1, true, true FROM menus WHERE path = '/reports';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT 'CMDB质量', '/reports/cmdb-quality', 'Database', id, 'report:read', 116, 1, true, true FROM menus WHERE path = '/reports';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '服务目录使用', '/reports/catalog-usage', 'BookOpen', id, 'report:read', 117, 1, true, true FROM menus WHERE path = '/reports';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '运维报表', '/reports/operations', 'TrendingUp', id, 'report:read', 118, 1, true, true FROM menus WHERE path = '/reports';

-- ============================================
-- 13. 工作流子菜单
-- ============================================
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '工作流列表', '/workflow/list', 'List', id, 'workflow:read', 121, 1, true, true FROM menus WHERE path = '/workflow';

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
SELECT '自动化规则', '/workflow/automation', 'Zap', id, 'workflow:write', 127, 1, true, true FROM menus WHERE path = '/workflow';

-- ============================================
-- 14. AI助手子菜单
-- ============================================
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT 'AI对话', '/ai/chat', 'MessageSquare', id, 'ai:use', 131, 1, true, true FROM menus WHERE path = '/ai/chat';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT 'AI创建工单', '/tickets/ai-create', 'Sparkles', id, 'ai:use', 132, 1, true, true FROM menus WHERE path = '/ai/chat';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '故障分析', '/ai/analyze', 'Search', id, 'ai:use', 133, 1, true, true FROM menus WHERE path = '/ai/chat';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '智能推荐', '/ai/recommend', 'Lightbulb', id, 'ai:use', 134, 1, true, true FROM menus WHERE path = '/ai/chat';

-- ============================================
-- 15. 访问管理子菜单
-- ============================================
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '权限申请', '/access/requests', 'Send', id, 'access:request', 151, 1, true, true FROM menus WHERE path = '/access';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '权限审批', '/access/approvals', 'CheckCircle', id, 'access:approve', 152, 1, true, true FROM menus WHERE path = '/access';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '权限审计', '/access/audit', 'ClipboardList', id, 'access:audit', 153, 1, true, true FROM menus WHERE path = '/access';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '角色申请', '/access/role-requests', 'Shield', id, 'access:request', 154, 1, true, true FROM menus WHERE path = '/access';

-- ============================================
-- 16. 服务台调度子菜单
-- ============================================
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '班次管理', '/shifts/schedules', 'Calendar', id, 'helpdesk:manage', 161, 1, true, true FROM menus WHERE path = '/shifts';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '交接班', '/shifts/handoffs', 'ArrowLeftRight', id, 'helpdesk:manage', 162, 1, true, true FROM menus WHERE path = '/shifts';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '值班记录', '/shifts/logs', 'FileText', id, 'helpdesk:read', 163, 1, true, true FROM menus WHERE path = '/shifts';

-- ============================================
-- 17. 客户管理(MSP)子菜单
-- ============================================
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '客户仪表盘', '/msp/dashboard', 'LayoutDashboard', id, 'msp:read', 201, 1, true, true FROM menus WHERE path = '/msp';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '分配管理', '/msp/allocations', 'GitBranch', id, 'msp:manage', 202, 1, true, true FROM menus WHERE path = '/msp';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '客户合同', '/msp/contracts', 'FileText', id, 'msp:read', 203, 1, true, true FROM menus WHERE path = '/msp';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT 'SLA报告', '/msp/sla-reports', 'BarChart3', id, 'msp:read', 204, 1, true, true FROM menus WHERE path = '/msp';

-- ============================================
-- 18. 发布管理子菜单
-- ============================================
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '发布计划', '/releases/plans', 'Calendar', id, 'release:read', 211, 1, true, true FROM menus WHERE path = '/releases';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '发布阶段', '/releases/phases', 'GitBranch', id, 'release:manage', 212, 1, true, true FROM menus WHERE path = '/releases';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '回滚计划', '/releases/rollbacks', 'RotateCcw', id, 'release:manage', 213, 1, true, true FROM menus WHERE path = '/releases';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '发布评审', '/releases/reviews', 'CheckSquare', id, 'release:approve', 214, 1, true, true FROM menus WHERE path = '/releases';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '发布历史', '/releases/history', 'History', id, 'release:read', 215, 1, true, true FROM menus WHERE path = '/releases';

-- ============================================
-- 19. 系统管理子菜单
-- ============================================
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '系统概览', '/admin/overview', 'LayoutDashboard', id, 'admin:write', 301, 1, true, true FROM menus WHERE path = '/admin';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '用户管理', '/admin/users', 'Users', id, 'user:read', 302, 1, true, true FROM menus WHERE path = '/admin';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '角色管理', '/admin/roles', 'Shield', id, 'role:read', 303, 1, true, true FROM menus WHERE path = '/admin';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '组管理', '/admin/groups', 'Users', id, 'group:read', 304, 1, true, true FROM menus WHERE path = '/admin';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '租户管理', '/admin/tenants', 'Building', id, 'tenant:manage', 305, 1, true, true FROM menus WHERE path = '/admin';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '部门管理', '/admin/departments', 'Building', id, 'department:manage', 306, 1, true, true FROM menus WHERE path = '/admin';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '团队管理', '/admin/teams', 'Users', id, 'team:manage', 307, 1, true, true FROM menus WHERE path = '/admin';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '工单分类', '/admin/ticket-categories', 'Tag', id, 'ticket:category:manage', 308, 1, true, true FROM menus WHERE path = '/admin';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '工单分配规则', '/admin/tickets/assignment', 'GitBranch', id, 'ticket:manage', 309, 1, true, true FROM menus WHERE path = '/admin';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '自动化规则', '/admin/tickets/automation', 'Zap', id, 'ticket:manage', 310, 1, true, true FROM menus WHERE path = '/admin';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '审批管理', '/admin/approvals', 'GitMerge', id, 'approval:manage', 311, 1, true, true FROM menus WHERE path = '/admin';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '审批链', '/admin/approval-chains', 'Link', id, 'approval:manage', 312, 1, true, true FROM menus WHERE path = '/admin';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '权限管理', '/admin/permissions', 'Lock', id, 'permission:manage', 313, 1, true, true FROM menus WHERE path = '/admin';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '系统配置', '/admin/system-config', 'Settings', id, 'system:config', 314, 1, true, true FROM menus WHERE path = '/admin';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '通知配置', '/admin/notifications', 'Bell', id, 'system:config', 315, 1, true, true FROM menus WHERE path = '/admin';

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '操作日志', '/admin/audit-logs', 'ClipboardList', id, 'system:audit', 316, 1, true, true FROM menus WHERE path = '/admin';

-- ============================================
-- 20. 独立功能页（无父菜单）
-- ============================================
INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled) VALUES
('待我审批', '/approvals/pending', 'CheckCircle', NULL, 'approval:read', 400, 1, true, true),
('通知中心', '/notifications', 'Bell', NULL, '', 410, 1, true, true),
('个人设置', '/profile', 'User', NULL, '', 420, 1, true, true);

COMMIT;

-- ============================================
-- 验证查询
-- ============================================
SELECT
  m.id,
  m.name,
  m.path,
  p.name as parent_name,
  m.permission_code,
  m.sort_order
FROM menus m
LEFT JOIN menus p ON m.parent_id = p.id
WHERE m.tenant_id = 1
ORDER BY coalesce(m.parent_id, m.id), m.sort_order;

-- 统计
SELECT
  (SELECT COUNT(*) FROM menus WHERE tenant_id = 1 AND parent_id IS NULL) as root_menus,
  (SELECT COUNT(*) FROM menus WHERE tenant_id = 1 AND parent_id IS NOT NULL) as child_menus,
  (SELECT COUNT(*) FROM menus WHERE tenant_id = 1) as total_menus;
