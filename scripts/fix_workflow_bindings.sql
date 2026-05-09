-- =====================================================
-- ITSM 工作流绑定修复脚本
-- 执行时间: 2026-05-01
-- 目的: 修复 incident 流程绑定错误
-- =====================================================
-- 1. 修复 process_bindings 表 - 将 incident 绑定改为 incident_emergency_flow
UPDATE process_bindings
SET process_definition_key = 'incident_emergency_flow',
    is_default = true,
    is_active = true
WHERE business_type = 'incident'
    AND tenant_id = 1;
-- 2. 确保 incident_emergency_flow 流程绑定存在（如果没有则创建）
INSERT INTO process_bindings (
        business_type,
        process_definition_key,
        process_version,
        is_default,
        is_active,
        tenant_id,
        created_at,
        updated_at
    )
SELECT 'incident',
    'incident_emergency_flow',
    1,
    true,
    true,
    1,
    NOW(),
    NOW()
WHERE NOT EXISTS (
        SELECT 1
        FROM process_bindings
        WHERE business_type = 'incident'
            AND process_definition_key = 'incident_emergency_flow'
            AND tenant_id = 1
    );
-- 3. 更新变更流程绑定
UPDATE process_bindings
SET process_definition_key = 'change_normal_flow',
    is_default = true,
    is_active = true
WHERE business_type = 'change'
    AND tenant_id = 1;
-- 4. 确保 change_normal_flow 流程绑定存在
INSERT INTO process_bindings (
        business_type,
        process_definition_key,
        process_version,
        is_default,
        is_active,
        tenant_id,
        created_at,
        updated_at
    )
SELECT 'change',
    'change_normal_flow',
    1,
    true,
    true,
    1,
    NOW(),
    NOW()
WHERE NOT EXISTS (
        SELECT 1
        FROM process_bindings
        WHERE business_type = 'change'
            AND process_definition_key = 'change_normal_flow'
            AND tenant_id = 1
    );
-- 5. 更新服务请求流程绑定
UPDATE process_bindings
SET process_definition_key = 'service_request_flow',
    is_default = true,
    is_active = true
WHERE business_type = 'service_request'
    AND tenant_id = 1;
-- 6. 确保 service_request_flow 流程绑定存在
INSERT INTO process_bindings (
        business_type,
        process_definition_key,
        process_version,
        is_default,
        is_active,
        tenant_id,
        created_at,
        updated_at
    )
SELECT 'service_request',
    'service_request_flow',
    1,
    true,
    true,
    1,
    NOW(),
    NOW()
WHERE NOT EXISTS (
        SELECT 1
        FROM process_bindings
        WHERE business_type = 'service_request'
            AND process_definition_key = 'service_request_flow'
            AND tenant_id = 1
    );
-- 7. 查看修复后的绑定
SELECT id,
    business_type,
    process_definition_key,
    is_default,
    is_active
FROM process_bindings
WHERE tenant_id = 1
ORDER BY business_type;
-- =====================================================
-- 执行完毕后，可以测试创建紧急事件工单验证工作流触发
-- =====================================================