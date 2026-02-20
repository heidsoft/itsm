-- ITSM 系统初始化数据
-- 执行方式: psql -h localhost -U postgres -d itsm -f init_data.sql

-- 部门初始化数据
INSERT INTO departments (name, code, description, tenant_id, created_at, updated_at)
SELECT * FROM (VALUES
    ('IT部门', 'IT', '信息技术部门', 1, NOW(), NOW()),
    ('运维部门', 'OPS', '运维支持部门', 1, NOW(), NOW()),
    ('客服部门', 'CS', '客户服务部门', 1, NOW(), NOW())
) AS v(name, code, description, tenant_id, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM departments WHERE code = 'IT');

-- 团队初始化数据
INSERT INTO teams (name, code, description, status, tenant_id, created_at, updated_at)
SELECT * FROM (VALUES
    ('一线支持', 'T1', '一线技术支持团队', 'active', 1, NOW(), NOW()),
    ('二线支持', 'T2', '二线技术支持团队', 'active', 1, NOW(), NOW())
) AS v(name, code, description, status, tenant_id, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM teams WHERE code = 'T1');

-- 角色初始化数据
INSERT INTO roles (name, code, description, tenant_id, created_at, updated_at)
SELECT * FROM (VALUES
    ('管理员', 'admin', '系统管理员', 1, NOW(), NOW()),
    ('工程师', 'engineer', 'IT工程师', 1, NOW(), NOW()),
    ('用户', 'user', '普通用户', 1, NOW(), NOW())
) AS v(name, code, description, tenant_id, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM roles WHERE code = 'admin');

-- 验证数据
SELECT 'Departments:' as info, COUNT(*) as count FROM departments
UNION ALL
SELECT 'Teams:', COUNT(*) FROM teams
UNION ALL
SELECT 'Roles:', COUNT(*) FROM roles;
