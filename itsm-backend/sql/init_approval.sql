-- 审批工作流初始化数据
-- 执行方式: psql -h localhost -p 5432 -U dev -d itsm -f init_approval.sql
-- 密码: StrongPassword123!

-- 审批工作流初始化数据
INSERT INTO approval_workflows (name, description, ticket_type, priority, nodes, is_active, tenant_id, status, created_at, updated_at)
SELECT * FROM (VALUES
    ('变更审批-标准', '标准变更审批流程，需要部门经理审批', 'change', 'medium',
     '[{"id":"node1","name":"部门经理审批","type":"approver","assignee_type":"role","assignee_value":"admin","step_order":1,"timeout_hours":24}]'::jsonb,
     true, 1, 'active', NOW(), NOW()),
    ('变更审批-紧急', '紧急变更审批流程，需要安全团队审批', 'change', 'urgent',
     '[{"id":"node1","name":"安全团队审批","type":"approver","assignee_type":"role","assignee_value":"security","step_order":1,"timeout_hours":2},{"id":"node2","name":"IT总监审批","type":"approver","assignee_type":"role","assignee_value":"admin","step_order":2,"timeout_hours":4}]'::jsonb,
     true, 1, 'active', NOW(), NOW()),
    ('服务请求审批-高权限', '高权限服务请求需要审批', 'service_request', 'high',
     '[{"id":"node1","name":"一线审批","type":"approver","assignee_type":"role","assignee_value":"engineer","step_order":1,"timeout_hours":8},{"id":"node2","name":"二线审批","type":"approver","assignee_type":"role","assignee_value":"admin","step_order":2,"timeout_hours":16}]'::jsonb,
     true, 1, 'active', NOW(), NOW()),
    ('工单审批-加急', '加急工单需要审批', 'ticket', 'urgent',
     '[{"id":"node1","name":"主管审批","type":"approver","assignee_type":"role","assignee_value":"admin","step_order":1,"timeout_hours":4}]'::jsonb,
     true, 1, 'active', NOW(), NOW())
) AS v(name, description, ticket_type, priority, nodes, is_active, tenant_id, status, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM approval_workflows WHERE name = '变更审批-标准');

-- 验证数据
SELECT 'approval_workflows:' as info, COUNT(*) as count FROM approval_workflows
UNION ALL
SELECT 'approval_records:', COUNT(*) FROM approval_records;
