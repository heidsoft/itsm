-- =============================================
-- ITSM 种子数据初始化脚本
-- 使用方式: psql -h localhost -U dev -d itsm -f seed_data.sql
-- 可根据需要修改其中的数据
-- =============================================

-- 1. 创建默认租户（如果不存在）
INSERT INTO tenants (name, code, domain, status, type, created_at, updated_at)
SELECT 'Default Tenant', 'default', 'localhost', 'active', 'enterprise', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM tenants WHERE code = 'default');

-- 获取租户ID
DO $$
DECLARE
    tenant_id INTEGER;
BEGIN
    SELECT id INTO tenant_id FROM tenants WHERE code = 'default' LIMIT 1;

    -- 2. 部门数据（14个）
    INSERT INTO departments (name, code, description, tenant_id, created_at, updated_at)
    SELECT * FROM (VALUES
        ('信息技术部', 'IT', 'IT整体管理', tenant_id, NOW(), NOW()),
        ('IT基础架构', 'IT-INFRA', '基础设施运维', tenant_id, NOW(), NOW()),
        ('IT应用服务', 'IT-APP', '应用系统运维', tenant_id, NOW(), NOW()),
        ('IT安全', 'IT-SEC', '信息安全管理', tenant_id, NOW(), NOW()),
        ('IT项目管理', 'IT-PMO', 'IT项目管理', tenant_id, NOW(), NOW()),
        ('运营管理部', 'OPS', 'IT运营管理', tenant_id, NOW(), NOW()),
        ('服务台', 'OPS-SD', '一线服务支持', tenant_id, NOW(), NOW()),
        ('运维中心', 'OPS-NOC', '7x24运维监控', tenant_id, NOW(), NOW()),
        ('客户服务', 'OPS-CS', '客户服务体验', tenant_id, NOW(), NOW()),
        ('研发部', 'RD', '产品研发', tenant_id, NOW(), NOW()),
        ('测试部', 'QA', '质量保证', tenant_id, NOW(), NOW()),
        ('人力资源部', 'HR', '人力资源管理', tenant_id, NOW(), NOW()),
        ('财务部', 'FIN', '财务管理', tenant_id, NOW(), NOW()),
        ('行政部', 'ADMIN', '行政管理', tenant_id, NOW(), NOW())
    ) AS v(name, code, description, tenant_id, created_at, updated_at)
    WHERE NOT EXISTS (SELECT 1 FROM departments WHERE code = 'IT' AND tenant_id = tenant_id);

    -- 3. 团队数据（18个）
    INSERT INTO teams (name, code, description, status, tenant_id, created_at, updated_at)
    SELECT * FROM (VALUES
        ('服务台-L1', 'SD-L1', '一线服务支持', 'active', tenant_id, NOW(), NOW()),
        ('服务台-L2', 'SD-L2', '二线技术支持', 'active', tenant_id, NOW(), NOW()),
        ('服务台-L3', 'SD-L3', '三线技术专家', 'active', tenant_id, NOW(), NOW()),
        ('服务器运维', 'SERVER', '服务器运维管理', 'active', tenant_id, NOW(), NOW()),
        ('网络运维', 'NETWORK', '网络设备运维', 'active', tenant_id, NOW(), NOW()),
        ('数据库运维', 'DBA', '数据库运维管理', 'active', tenant_id, NOW(), NOW()),
        ('云平台运维', 'CLOUD', '云计算平台运维', 'active', tenant_id, NOW(), NOW()),
        ('ERP支持', 'ERP', 'ERP系统支持', 'active', tenant_id, NOW(), NOW()),
        ('CRM支持', 'CRM', 'CRM系统支持', 'active', tenant_id, NOW(), NOW()),
        ('OA支持', 'OA', 'OA办公系统支持', 'active', tenant_id, NOW(), NOW()),
        ('安全运营', 'SEC-OPS', '安全监控与响应', 'active', tenant_id, NOW(), NOW()),
        ('安全合规', 'SEC-COM', '安全合规管理', 'active', tenant_id, NOW(), NOW()),
        ('后端开发', 'BACKEND', '后端开发团队', 'active', tenant_id, NOW(), NOW()),
        ('前端开发', 'FRONTEND', '前端开发团队', 'active', tenant_id, NOW(), NOW()),
        ('移动开发', 'MOBILE', '移动端开发团队', 'active', tenant_id, NOW(), NOW()),
        ('测试团队', 'QA', '测试与质量保证', 'active', tenant_id, NOW(), NOW()),
        ('客户成功', 'CS', '客户成功管理', 'active', tenant_id, NOW(), NOW()),
        ('技术支持', 'TECH', '客户服务技术支持', 'active', tenant_id, NOW(), NOW())
    ) AS v(name, code, description, status, tenant_id, created_at, updated_at)
    WHERE NOT EXISTS (SELECT 1 FROM teams WHERE code = 'SD-L1' AND tenant_id = tenant_id);

    -- 4. 角色数据（20个）
    INSERT INTO roles (name, code, description, tenant_id, created_at, updated_at)
    SELECT * FROM (VALUES
        ('IT总监', 'it_director', 'IT部门总监', tenant_id, NOW(), NOW()),
        ('运维总监', 'ops_director', '运维部门总监', tenant_id, NOW(), NOW()),
        ('系统管理员', 'sysadmin', '系统管理员', tenant_id, NOW(), NOW()),
        ('安全管理员', 'security_admin', '安全管理角色', tenant_id, NOW(), NOW()),
        ('审计管理员', 'audit_admin', '审计管理角色', tenant_id, NOW(), NOW()),
        ('运维经理', 'ops_manager', '运维团队经理', tenant_id, NOW(), NOW()),
        ('运维工程师', 'ops_engineer', '运维工程师', tenant_id, NOW(), NOW()),
        ('DBA工程师', 'dba', '数据库管理员', tenant_id, NOW(), NOW()),
        ('网络安全工程师', 'network_eng', '网络工程师', tenant_id, NOW(), NOW()),
        ('服务台主管', 'sd_manager', '服务台主管', tenant_id, NOW(), NOW()),
        ('一线工程师', 'l1_support', '一线支持工程师', tenant_id, NOW(), NOW()),
        ('二线工程师', 'l2_support', '二线支持工程师', tenant_id, NOW(), NOW()),
        ('三线专家', 'l3_expert', '三线技术专家', tenant_id, NOW(), NOW()),
        ('研发经理', 'rd_manager', '研发团队经理', tenant_id, NOW(), NOW()),
        ('开发工程师', 'developer', '开发工程师', tenant_id, NOW(), NOW()),
        ('测试工程师', 'qa_engineer', '测试工程师', tenant_id, NOW(), NOW()),
        ('部门经理', 'dept_manager', '部门经理', tenant_id, NOW(), NOW()),
        ('团队主管', 'team_lead', '团队主管', tenant_id, NOW(), NOW()),
        ('普通用户', 'end_user', '普通终端用户', tenant_id, NOW(), NOW()),
        ('访客', 'guest', '访客用户', tenant_id, NOW(), NOW())
    ) AS v(name, code, description, tenant_id, created_at, updated_at)
    WHERE NOT EXISTS (SELECT 1 FROM roles WHERE code = 'it_director' AND tenant_id = tenant_id);

    -- 5. SLA 定义（6个）
    INSERT INTO sla_definitions (name, description, service_type, priority, response_time, resolution_time, is_active, tenant_id, created_at, updated_at)
    SELECT * FROM (VALUES
        ('SLA-P0-紧急', 'P0紧急级别SLA', 'incident', 'urgent', 15, 120, true, tenant_id, NOW(), NOW()),
        ('SLA-P1-高', 'P1高级别SLA', 'incident', 'high', 30, 240, true, tenant_id, NOW(), NOW()),
        ('SLA-P2-中', 'P2中级别SLA', 'incident', 'medium', 120, 480, true, tenant_id, NOW(), NOW()),
        ('SLA-P3-低', 'P3低级别SLA', 'incident', 'low', 240, 1440, true, tenant_id, NOW(), NOW()),
        ('SLA-服务请求', '服务请求标准SLA', 'service_request', 'medium', 480, 4320, true, tenant_id, NOW(), NOW()),
        ('SLA-变更', '变更请求SLA', 'change', 'high', 60, 1440, true, tenant_id, NOW(), NOW())
    ) AS v(name, description, service_type, priority, response_time, resolution_time, is_active, tenant_id, created_at, updated_at)
    WHERE NOT EXISTS (SELECT 1 FROM sla_definitions WHERE name = 'SLA-P0-紧急' AND tenant_id = tenant_id);

    -- 6. 服务目录（22个）
    INSERT INTO service_catalogs (name, description, category, service_type, requires_approval, delivery_time, status, is_active, tenant_id, created_at, updated_at)
    SELECT * FROM (VALUES
        ('云服务器 ECS', '弹性云服务器', '云计算', 'vm', true, 1, 'active', true, tenant_id, NOW(), NOW()),
        ('云数据库 RDS', 'MySQL/PostgreSQL数据库', '数据库', 'rds', true, 1, 'active', true, tenant_id, NOW(), NOW()),
        ('对象存储 OSS', '海量云存储', '存储', 'oss', false, 0, 'active', true, tenant_id, NOW(), NOW()),
        ('CDN 加速', '内容分发加速', '网络', 'network', false, 0, 'active', true, tenant_id, NOW(), NOW()),
        ('负载均衡 SLB', '流量分发服务', '网络', 'network', true, 1, 'active', true, tenant_id, NOW(), NOW()),
        ('VPN 网关', 'VPN加密通道', '安全', 'security', true, 2, 'active', true, tenant_id, NOW(), NOW()),
        ('企业邮箱', '企业域名邮箱', '通讯', 'custom', false, 1, 'active', true, tenant_id, NOW(), NOW()),
        ('企业网盘', '文件存储共享', '协作', 'custom', false, 0, 'active', true, tenant_id, NOW(), NOW()),
        ('视频会议', '高清视频会议', '通讯', 'custom', false, 0, 'active', true, tenant_id, NOW(), NOW()),
        ('企业IM', '即时通讯工具', '通讯', 'custom', false, 0, 'active', true, tenant_id, NOW(), NOW()),
        ('漏洞扫描', 'Web漏洞扫描', '安全', 'security', true, 1, 'active', true, tenant_id, NOW(), NOW()),
        ('渗透测试', '安全渗透测试', '安全', 'security', true, 5, 'active', true, tenant_id, NOW(), NOW()),
        ('等保合规', '等级保护咨询', '安全', 'security', true, 30, 'active', true, tenant_id, NOW(), NOW()),
        ('IT服务台', 'IT问题咨询支持', '支持', 'custom', false, 0, 'active', true, tenant_id, NOW(), NOW()),
        ('软件安装', '标准软件安装', '支持', 'custom', false, 1, 'active', true, tenant_id, NOW(), NOW()),
        ('账户申请', '新员工账户开通', '支持', 'custom', true, 1, 'active', true, tenant_id, NOW(), NOW()),
        ('网络接入', '网络接入申请', '支持', 'custom', true, 2, 'active', true, tenant_id, NOW(), NOW()),
        ('域名申请', '内部域名注册', '支持', 'custom', true, 3, 'active', true, tenant_id, NOW(), NOW()),
        ('代码仓库', 'Git代码仓库', '开发', 'custom', false, 0, 'active', true, tenant_id, NOW(), NOW()),
        ('CI/CD流水线', '自动化部署', '开发', 'custom', false, 0, 'active', true, tenant_id, NOW(), NOW()),
        ('测试环境', '预发布测试环境', '开发', 'custom', true, 2, 'active', true, tenant_id, NOW(), NOW()),
        ('API网关', 'API接口管理', '开发', 'custom', true, 3, 'active', true, tenant_id, NOW(), NOW())
    ) AS v(name, description, category, service_type, requires_approval, delivery_time, status, is_active, tenant_id, created_at, updated_at)
    WHERE NOT EXISTS (SELECT 1 FROM service_catalogs WHERE name = '云服务器 ECS' AND tenant_id = tenant_id);

    -- 7. 菜单数据（完整层级结构）
    -- 先清理旧菜单，再重建完整结构
    DELETE FROM menus WHERE tenant_id = tenant_id;

    -- 7.1 顶层菜单 (parent_id = NULL)
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled) VALUES
    ('服务台', '/dashboard', 'LayoutDashboard', NULL, '', 10, tenant_id, true, true),
    ('服务请求', '/service-requests', 'FileText', NULL, 'ticket:read', 20, tenant_id, true, true),
    ('我的请求', '/my-requests', 'User', NULL, 'ticket:read', 25, tenant_id, true, true),
    ('事件管理', '/incidents', 'AlertCircle', NULL, 'incident:read', 30, tenant_id, true, true),
    ('问题管理', '/problems', 'HelpCircle', NULL, 'problem:read', 40, tenant_id, true, true),
    ('变更管理', '/changes', 'BarChart3', NULL, 'change:read', 50, tenant_id, true, true),
    ('知识库', '/knowledge', 'Book', NULL, 'knowledge:read', 60, tenant_id, true, true),
    ('服务目录', '/service-catalog', 'BookOpen', NULL, 'service:read', 70, tenant_id, true, true),
    ('CMDB', '/cmdb', 'Database', NULL, 'cmdb:read', 80, tenant_id, true, true),
    ('资产管理', '/assets', 'Monitor', NULL, 'asset:read', 90, tenant_id, true, true),
    ('SLA管理', '/sla', 'Clock', NULL, 'sla:read', 100, tenant_id, true, true),
    ('报表中心', '/reports', 'TrendingUp', NULL, 'report:read', 110, tenant_id, true, true),
    ('工作流', '/workflow', 'GitMerge', NULL, 'workflow:read', 120, tenant_id, true, true),
    ('AI助手', '/ai/chat', 'Bot', NULL, 'ai:use', 130, tenant_id, true, true),
    ('访问管理', '/access', 'Key', NULL, 'access:read', 150, tenant_id, true, true),
    ('服务台调度', '/shifts', 'Calendar', NULL, 'helpdesk:manage', 160, tenant_id, true, true),
    ('客户管理', '/msp', 'Building', NULL, 'msp:read', 200, tenant_id, true, true),
    ('发布管理', '/releases', 'Rocket', NULL, 'release:read', 210, tenant_id, true, true),
    ('系统管理', '/admin', 'Settings', NULL, 'admin:write', 300, tenant_id, true, true),
    ('待我审批', '/approvals/pending', 'CheckCircle', NULL, 'approval:read', 400, tenant_id, true, true),
    ('通知中心', '/notifications', 'Bell', NULL, '', 410, tenant_id, true, true),
    ('个人设置', '/profile', 'User', NULL, '', 420, tenant_id, true, true);

    -- 7.2 服务请求子菜单
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '服务请求模板', '/tickets/templates', 'FileText', id, 'ticket:write', 21, tenant_id, true, true FROM menus WHERE path = '/service-requests' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '工单统计', '/tickets/analytics', 'BarChart3', id, 'ticket:read', 22, tenant_id, true, true FROM menus WHERE path = '/service-requests' AND parent_id IS NULL;

    -- 7.3 事件管理子菜单
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '重大事件', '/incidents/major', 'AlertTriangle', id, 'incident:read', 31, tenant_id, true, true FROM menus WHERE path = '/incidents' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '事件分析', '/incidents/analytics', 'TrendingUp', id, 'incident:read', 32, tenant_id, true, true FROM menus WHERE path = '/incidents' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '事件转问题', '/incidents/convert', 'ArrowRight', id, 'incident:manage', 33, tenant_id, true, true FROM menus WHERE path = '/incidents' AND parent_id IS NULL;

    -- 7.4 问题管理子菜单
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '已知错误', '/problems/known-errors', 'AlertCircle', id, 'problem:read', 41, tenant_id, true, true FROM menus WHERE path = '/problems' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '根因分析', '/problems/root-cause', 'Search', id, 'problem:analyze', 42, tenant_id, true, true FROM menus WHERE path = '/problems' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '问题链接', '/problems/links', 'Link', id, 'problem:manage', 43, tenant_id, true, true FROM menus WHERE path = '/problems' AND parent_id IS NULL;

    -- 7.5 变更管理子菜单
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '标准变更', '/changes/standard', 'CheckCircle', id, 'change:read', 51, tenant_id, true, true FROM menus WHERE path = '/changes' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '紧急变更', '/changes/emergency', 'Zap', id, 'change:manage', 52, tenant_id, true, true FROM menus WHERE path = '/changes' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '变更审批', '/changes/approvals', 'GitMerge', id, 'change:approve', 53, tenant_id, true, true FROM menus WHERE path = '/changes' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '评审组', '/admin/change-review', 'Users', id, 'change:manage', 54, tenant_id, true, true FROM menus WHERE path = '/changes' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '变更日历', '/changes/calendar', 'Calendar', id, 'change:read', 55, tenant_id, true, true FROM menus WHERE path = '/changes' AND parent_id IS NULL;

    -- 7.6 知识库子菜单
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '文章管理', '/knowledge/articles', 'FileText', id, 'knowledge:write', 61, tenant_id, true, true FROM menus WHERE path = '/knowledge' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '文章分类', '/knowledge/categories', 'Tag', id, 'knowledge:manage', 62, tenant_id, true, true FROM menus WHERE path = '/knowledge' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '知识审核', '/knowledge/review', 'CheckSquare', id, 'knowledge:approve', 63, tenant_id, true, true FROM menus WHERE path = '/knowledge' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT 'AI知识推荐', '/knowledge/ai-recommend', 'Sparkles', id, 'knowledge:read', 64, tenant_id, true, true FROM menus WHERE path = '/knowledge' AND parent_id IS NULL;

    -- 7.7 服务目录子菜单
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '服务项管理', '/service-catalog/items', 'List', id, 'service:write', 71, tenant_id, true, true FROM menus WHERE path = '/service-catalog' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '服务类别', '/service-catalog/categories', 'Folder', id, 'service:write', 72, tenant_id, true, true FROM menus WHERE path = '/service-catalog' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '服务请求模板', '/service-catalog/templates', 'FileText', id, 'service:write', 73, tenant_id, true, true FROM menus WHERE path = '/service-catalog' AND parent_id IS NULL;

    -- 7.8 CMDB子菜单
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '配置项列表', '/cmdb/cis', 'Server', id, 'cmdb:read', 81, tenant_id, true, true FROM menus WHERE path = '/cmdb' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT 'CI类型管理', '/cmdb/types', 'Database', id, 'cmdb:write', 82, tenant_id, true, true FROM menus WHERE path = '/cmdb' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '云资源', '/cmdb/cloud-resources', 'Cloud', id, 'cmdb:read', 83, tenant_id, true, true FROM menus WHERE path = '/cmdb' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '云账号', '/cmdb/cloud-accounts', 'Key', id, 'cmdb:manage', 84, tenant_id, true, true FROM menus WHERE path = '/cmdb' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '云服务', '/cmdb/cloud-services', 'Boxes', id, 'cmdb:read', 85, tenant_id, true, true FROM menus WHERE path = '/cmdb' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '同步对账', '/cmdb/reconciliation', 'RefreshCw', id, 'cmdb:write', 86, tenant_id, true, true FROM menus WHERE path = '/cmdb' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '关系图谱', '/cmdb/relationships', 'GitBranch', id, 'cmdb:read', 87, tenant_id, true, true FROM menus WHERE path = '/cmdb' AND parent_id IS NULL;

    -- 7.9 资产管理子菜单
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '资产列表', '/assets/list', 'Server', id, 'asset:read', 91, tenant_id, true, true FROM menus WHERE path = '/assets' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '软件许可证', '/assets/licenses', 'Key', id, 'license:manage', 92, tenant_id, true, true FROM menus WHERE path = '/assets' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '资产分类', '/assets/categories', 'Tag', id, 'asset:write', 93, tenant_id, true, true FROM menus WHERE path = '/assets' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '维保管理', '/assets/maintenance', 'Wrench', id, 'asset:manage', 94, tenant_id, true, true FROM menus WHERE path = '/assets' AND parent_id IS NULL;

    -- 7.10 SLA管理子菜单
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT 'SLA监控', '/sla/monitor', 'Activity', id, 'sla:read', 101, tenant_id, true, true FROM menus WHERE path = '/sla' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT 'SLA定义', '/sla/definitions', 'FileText', id, 'sla:write', 102, tenant_id, true, true FROM menus WHERE path = '/sla' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '升级规则', '/sla/escalations', 'ArrowUpCircle', id, 'sla:write', 103, tenant_id, true, true FROM menus WHERE path = '/sla' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT 'SLA报表', '/sla/reports', 'BarChart3', id, 'sla:read', 104, tenant_id, true, true FROM menus WHERE path = '/sla' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '目标管理', '/sla/targets', 'Target', id, 'sla:manage', 105, tenant_id, true, true FROM menus WHERE path = '/sla' AND parent_id IS NULL;

    -- 7.11 报表中心子菜单
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '工单报表', '/reports/tickets', 'FileText', id, 'report:read', 111, tenant_id, true, true FROM menus WHERE path = '/reports' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '事件报表', '/reports/incidents', 'AlertCircle', id, 'report:read', 112, tenant_id, true, true FROM menus WHERE path = '/reports' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '问题报表', '/reports/problems', 'HelpCircle', id, 'report:read', 113, tenant_id, true, true FROM menus WHERE path = '/reports' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '变更报表', '/reports/changes', 'BarChart3', id, 'report:read', 114, tenant_id, true, true FROM menus WHERE path = '/reports' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT 'SLA报表', '/reports/sla', 'Calendar', id, 'report:read', 115, tenant_id, true, true FROM menus WHERE path = '/reports' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT 'CMDB质量', '/reports/cmdb-quality', 'Database', id, 'report:read', 116, tenant_id, true, true FROM menus WHERE path = '/reports' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '服务目录使用', '/reports/catalog-usage', 'BookOpen', id, 'report:read', 117, tenant_id, true, true FROM menus WHERE path = '/reports' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '运维报表', '/reports/operations', 'TrendingUp', id, 'report:read', 118, tenant_id, true, true FROM menus WHERE path = '/reports' AND parent_id IS NULL;

    -- 7.12 工作流子菜单
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '工作流列表', '/workflow/list', 'List', id, 'workflow:read', 121, tenant_id, true, true FROM menus WHERE path = '/workflow' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '流程设计器', '/workflow/designer', 'Edit', id, 'workflow:write', 122, tenant_id, true, true FROM menus WHERE path = '/workflow' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '流程实例', '/workflow/instances', 'Play', id, 'workflow:read', 123, tenant_id, true, true FROM menus WHERE path = '/workflow' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '版本管理', '/workflow/versions', 'History', id, 'workflow:write', 124, tenant_id, true, true FROM menus WHERE path = '/workflow' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '监控仪表盘', '/workflow/dashboard', 'Activity', id, 'workflow:read', 125, tenant_id, true, true FROM menus WHERE path = '/workflow' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '审计日志', '/workflow/audit', 'ClipboardList', id, 'workflow:read', 126, tenant_id, true, true FROM menus WHERE path = '/workflow' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '自动化规则', '/workflow/automation', 'Zap', id, 'workflow:write', 127, tenant_id, true, true FROM menus WHERE path = '/workflow' AND parent_id IS NULL;

    -- 7.13 AI助手子菜单
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT 'AI对话', '/ai/chat', 'MessageSquare', id, 'ai:use', 131, tenant_id, true, true FROM menus WHERE path = '/ai/chat' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT 'AI创建工单', '/tickets/ai-create', 'Sparkles', id, 'ai:use', 132, tenant_id, true, true FROM menus WHERE name = 'AI对话' AND path = '/ai/chat';
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '故障分析', '/ai/analyze', 'Search', id, 'ai:use', 133, tenant_id, true, true FROM menus WHERE name = 'AI对话' AND path = '/ai/chat';
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '智能推荐', '/ai/recommend', 'Lightbulb', id, 'ai:use', 134, tenant_id, true, true FROM menus WHERE name = 'AI对话' AND path = '/ai/chat';

    -- 7.14 访问管理子菜单
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '权限申请', '/access/requests', 'Send', id, 'access:request', 151, tenant_id, true, true FROM menus WHERE path = '/access' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '权限审批', '/access/approvals', 'CheckCircle', id, 'access:approve', 152, tenant_id, true, true FROM menus WHERE path = '/access' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '权限审计', '/access/audit', 'ClipboardList', id, 'access:audit', 153, tenant_id, true, true FROM menus WHERE path = '/access' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '角色申请', '/access/role-requests', 'Shield', id, 'access:request', 154, tenant_id, true, true FROM menus WHERE path = '/access' AND parent_id IS NULL;

    -- 7.15 服务台调度子菜单
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '班次管理', '/shifts/schedules', 'Calendar', id, 'helpdesk:manage', 161, tenant_id, true, true FROM menus WHERE path = '/shifts' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '交接班', '/shifts/handoffs', 'ArrowLeftRight', id, 'helpdesk:manage', 162, tenant_id, true, true FROM menus WHERE path = '/shifts' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '值班记录', '/shifts/logs', 'FileText', id, 'helpdesk:read', 163, tenant_id, true, true FROM menus WHERE path = '/shifts' AND parent_id IS NULL;

    -- 7.16 客户管理(MSP)子菜单
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '客户仪表盘', '/msp/dashboard', 'LayoutDashboard', id, 'msp:read', 201, tenant_id, true, true FROM menus WHERE path = '/msp' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '分配管理', '/msp/allocations', 'GitBranch', id, 'msp:manage', 202, tenant_id, true, true FROM menus WHERE path = '/msp' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '客户合同', '/msp/contracts', 'FileText', id, 'msp:read', 203, tenant_id, true, true FROM menus WHERE path = '/msp' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT 'SLA报告', '/msp/sla-reports', 'BarChart3', id, 'msp:read', 204, tenant_id, true, true FROM menus WHERE path = '/msp' AND parent_id IS NULL;

    -- 7.17 发布管理子菜单
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '发布计划', '/releases/plans', 'Calendar', id, 'release:read', 211, tenant_id, true, true FROM menus WHERE path = '/releases' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '发布阶段', '/releases/phases', 'GitBranch', id, 'release:manage', 212, tenant_id, true, true FROM menus WHERE path = '/releases' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '回滚计划', '/releases/rollbacks', 'RotateCcw', id, 'release:manage', 213, tenant_id, true, true FROM menus WHERE path = '/releases' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '发布评审', '/releases/reviews', 'CheckSquare', id, 'release:approve', 214, tenant_id, true, true FROM menus WHERE path = '/releases' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '发布历史', '/releases/history', 'History', id, 'release:read', 215, tenant_id, true, true FROM menus WHERE path = '/releases' AND parent_id IS NULL;

    -- 7.18 系统管理子菜单
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '系统概览', '/admin/overview', 'LayoutDashboard', id, 'admin:write', 301, tenant_id, true, true FROM menus WHERE path = '/admin' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '用户管理', '/admin/users', 'Users', id, 'user:read', 302, tenant_id, true, true FROM menus WHERE path = '/admin' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '角色管理', '/admin/roles', 'Shield', id, 'role:read', 303, tenant_id, true, true FROM menus WHERE path = '/admin' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '组管理', '/admin/groups', 'Users', id, 'group:read', 304, tenant_id, true, true FROM menus WHERE path = '/admin' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '租户管理', '/admin/tenants', 'Building', id, 'tenant:manage', 305, tenant_id, true, true FROM menus WHERE path = '/admin' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '部门管理', '/admin/departments', 'Building', id, 'department:manage', 306, tenant_id, true, true FROM menus WHERE path = '/admin' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '团队管理', '/admin/teams', 'Users', id, 'team:manage', 307, tenant_id, true, true FROM menus WHERE path = '/admin' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '工单分类', '/admin/ticket-categories', 'Tag', id, 'ticket:category:manage', 308, tenant_id, true, true FROM menus WHERE path = '/admin' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '工单分配规则', '/admin/tickets/assignment', 'GitBranch', id, 'ticket:manage', 309, tenant_id, true, true FROM menus WHERE path = '/admin' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '自动化规则', '/admin/tickets/automation', 'Zap', id, 'ticket:manage', 310, tenant_id, true, true FROM menus WHERE path = '/admin' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '审批管理', '/admin/approvals', 'GitMerge', id, 'approval:manage', 311, tenant_id, true, true FROM menus WHERE path = '/admin' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '审批链', '/admin/approval-chains', 'Link', id, 'approval:manage', 312, tenant_id, true, true FROM menus WHERE path = '/admin' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '权限管理', '/admin/permissions', 'Lock', id, 'permission:manage', 313, tenant_id, true, true FROM menus WHERE path = '/admin' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '系统配置', '/admin/system-config', 'Settings', id, 'system:config', 314, tenant_id, true, true FROM menus WHERE path = '/admin' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '通知配置', '/admin/notifications', 'Bell', id, 'system:config', 315, tenant_id, true, true FROM menus WHERE path = '/admin' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '操作日志', '/admin/audit-logs', 'ClipboardList', id, 'system:audit', 316, tenant_id, true, true FROM menus WHERE path = '/admin' AND parent_id IS NULL;
    INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled)
    SELECT '连接器市场', '/admin/connectors', 'Plug', id, 'connector:read', 317, tenant_id, true, true FROM menus WHERE path = '/admin' AND parent_id IS NULL;

    RAISE NOTICE '完整菜单结构已初始化';

    RAISE NOTICE 'Seed data initialized successfully!';
END $$;

-- 8. 验证数据
SELECT '=== 初始化数据统计 ===' as info;
SELECT 'departments:', COUNT(*) FROM departments WHERE tenant_id = (SELECT id FROM tenants WHERE code = 'default' LIMIT 1);
SELECT 'teams:', COUNT(*) FROM teams WHERE tenant_id = (SELECT id FROM tenants WHERE code = 'default' LIMIT 1);
SELECT 'roles:', COUNT(*) FROM roles WHERE tenant_id = (SELECT id FROM tenants WHERE code = 'default' LIMIT 1);
SELECT 'sla_definitions:', COUNT(*) FROM sla_definitions WHERE tenant_id = (SELECT id FROM tenants WHERE code = 'default' LIMIT 1);
SELECT 'service_catalogs:', COUNT(*) FROM service_catalogs WHERE tenant_id = (SELECT id FROM tenants WHERE code = 'default' LIMIT 1);
SELECT 'menus:', COUNT(*) FROM menus WHERE tenant_id = (SELECT id FROM tenants WHERE code = 'default' LIMIT 1);

\echo '========================================'
\echo '初始化完成！'
\echo '默认管理员账户: admin / admin123'
\echo '========================================'
