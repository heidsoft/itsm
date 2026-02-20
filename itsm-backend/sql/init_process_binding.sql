-- BPMN流程绑定初始化数据
-- 将业务类型绑定到BPMN流程定义

INSERT INTO process_bindings (business_type, business_sub_type, process_definition_key, process_version, is_default, priority, is_active, tenant_id, created_at, updated_at)
SELECT * FROM (VALUES
    ('ticket', NULL, 'ticket_general_flow', 1, true, 1, true, 1, NOW(), NOW()),
    ('incident', NULL, 'incident_emergency_flow', 1, true, 1, true, 1, NOW(), NOW()),
    ('change', 'standard', 'change_normal_flow', 1, true, 1, true, 1, NOW(), NOW()),
    ('change', 'emergency', 'change_normal_flow', 1, false, 2, true, 1, NOW(), NOW()),
    ('problem', NULL, 'problem_management_flow', 1, true, 1, true, 1, NOW(), NOW()),
    ('service_request', NULL, 'service_request_flow', 1, true, 1, true, 1, NOW(), NOW())
) AS v(business_type, business_sub_type, process_definition_key, process_version, is_default, priority, is_active, tenant_id, created_at, updated_at)
WHERE NOT EXISTS (
    SELECT 1 FROM process_bindings
    WHERE business_type = v.business_type
    AND (business_sub_type IS NULL OR business_sub_type = v.business_sub_type)
);

-- 验证绑定数据
SELECT business_type, business_sub_type, process_definition_key, is_default, is_active
FROM process_bindings
ORDER BY business_type, priority;
