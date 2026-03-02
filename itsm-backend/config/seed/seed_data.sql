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

    RAISE NOTICE 'Seed data initialized successfully!';
END $$;

-- 7. 验证数据
SELECT '=== 初始化数据统计 ===' as info;
SELECT 'departments:', COUNT(*) FROM departments WHERE tenant_id = (SELECT id FROM tenants WHERE code = 'default' LIMIT 1);
SELECT 'teams:', COUNT(*) FROM teams WHERE tenant_id = (SELECT id FROM tenants WHERE code = 'default' LIMIT 1);
SELECT 'roles:', COUNT(*) FROM roles WHERE tenant_id = (SELECT id FROM tenants WHERE code = 'default' LIMIT 1);
SELECT 'sla_definitions:', COUNT(*) FROM sla_definitions WHERE tenant_id = (SELECT id FROM tenants WHERE code = 'default' LIMIT 1);
SELECT 'service_catalogs:', COUNT(*) FROM service_catalogs WHERE tenant_id = (SELECT id FROM tenants WHERE code = 'default' LIMIT 1);

\echo '========================================'
\echo '初始化完成！'
\echo '默认管理员账户: admin / admin123'
\echo '========================================'
