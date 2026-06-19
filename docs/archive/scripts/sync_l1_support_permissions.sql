-- l1_support 角色权限同步脚本
-- 为一线工程师角色分配完整的功能权限
BEGIN;
-- Step 1: 清除现有权限
DELETE FROM role_permissions
WHERE role_id = 11;
-- Step 2: 分配与 Agent (id=104) 类似的基础权限
-- 工单管理
INSERT INTO role_permissions (role_id, permission_id)
SELECT 11,
    p.id
FROM permissions p
WHERE p.tenant_id = 1
    AND p.resource = 'ticket'
    AND p.action IN ('read', 'write');
-- 通知
INSERT INTO role_permissions (role_id, permission_id)
SELECT 11,
    p.id
FROM permissions p
WHERE p.tenant_id = 1
    AND p.resource = 'notification'
    AND p.action IN ('read', 'write');
-- 仪表盘（关键！解决 403 问题）
INSERT INTO role_permissions (role_id, permission_id)
SELECT 11,
    p.id
FROM permissions p
WHERE p.tenant_id = 1
    AND p.resource = 'dashboard'
    AND p.action = 'read';
-- 知识库
INSERT INTO role_permissions (role_id, permission_id)
SELECT 11,
    p.id
FROM permissions p
WHERE p.tenant_id = 1
    AND p.resource = 'knowledge'
    AND p.action IN ('read', 'write');
-- CMDB
INSERT INTO role_permissions (role_id, permission_id)
SELECT 11,
    p.id
FROM permissions p
WHERE p.tenant_id = 1
    AND p.resource = 'cmdb'
    AND p.action = 'read';
-- 事件管理（一线工程师核心职责）
INSERT INTO role_permissions (role_id, permission_id)
SELECT 11,
    p.id
FROM permissions p
WHERE p.tenant_id = 1
    AND p.resource = 'incident'
    AND p.action IN ('read', 'write');
-- 服务目录
INSERT INTO role_permissions (role_id, permission_id)
SELECT 11,
    p.id
FROM permissions p
WHERE p.tenant_id = 1
    AND p.resource = 'service'
    AND p.action = 'read';
-- 服务请求
INSERT INTO role_permissions (role_id, permission_id)
SELECT 11,
    p.id
FROM permissions p
WHERE p.tenant_id = 1
    AND p.resource = 'service_request'
    AND p.action IN ('read', 'write');
-- 变更管理（一线只读）
INSERT INTO role_permissions (role_id, permission_id)
SELECT 11,
    p.id
FROM permissions p
WHERE p.tenant_id = 1
    AND p.resource = 'change'
    AND p.action = 'read';
-- 问题管理（一线只读）
INSERT INTO role_permissions (role_id, permission_id)
SELECT 11,
    p.id
FROM permissions p
WHERE p.tenant_id = 1
    AND p.resource = 'problem'
    AND p.action = 'read';
-- 用户信息（查看自己的用户信息）
INSERT INTO role_permissions (role_id, permission_id)
SELECT 11,
    p.id
FROM permissions p
WHERE p.tenant_id = 1
    AND p.resource = 'user'
    AND p.action = 'read';
-- 租户信息（查看租户列表）
INSERT INTO role_permissions (role_id, permission_id)
SELECT 11,
    p.id
FROM permissions p
WHERE p.tenant_id = 1
    AND p.resource = 'tenant'
    AND p.action = 'read';
-- SLA（只读）
INSERT INTO role_permissions (role_id, permission_id)
SELECT 11,
    p.id
FROM permissions p
WHERE p.tenant_id = 1
    AND p.resource = 'sla'
    AND p.action = 'read';
-- 组合管理
INSERT INTO role_permissions (role_id, permission_id)
SELECT 11,
    p.id
FROM permissions p
WHERE p.tenant_id = 1
    AND p.resource = 'groups'
    AND p.action = 'read';
-- BPMN 工作流（只读）
INSERT INTO role_permissions (role_id, permission_id)
SELECT 11,
    p.id
FROM permissions p
WHERE p.tenant_id = 1
    AND p.resource = 'workflow'
    AND p.action = 'read';
COMMIT;
-- 验证
SELECT r.code,
    r.name,
    COUNT(rp.id) as permission_count
FROM roles r
    LEFT JOIN role_permissions rp ON r.id = rp.role_id
WHERE r.code = 'l1_support'
GROUP BY r.id,
    r.code,
    r.name;