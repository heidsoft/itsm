-- ========================================================
-- 国内企业ITSM工作流绑定配置
-- Domestic Enterprise ITSM Workflow Bindings
-- ========================================================
-- 注意: 已有incident_emergency_flow绑定到incident
-- 添加新的业务类型绑定
-- ========================================================
-- 1. 问题管理流程绑定 (problem)
-- ========================================================
INSERT INTO process_bindings (
        business_type,
        business_sub_type,
        process_definition_key,
        process_version,
        is_default,
        priority,
        is_active,
        tenant_id,
        process_definition_bindings,
        created_at,
        updated_at
    )
VALUES (
        'problem',
        NULL,
        'problem_management_flow_cn',
        2,
        true,
        NULL,
        true,
        1,
        '{"name": "问题管理流程(国内企业版)", "description": "专注根因分析和预防"}',
        NOW(),
        NOW()
    ) ON CONFLICT DO NOTHING;
-- ========================================================
-- 2. 发布审批流程绑定 (release)
-- ========================================================
INSERT INTO process_bindings (
        business_type,
        business_sub_type,
        process_definition_key,
        process_version,
        is_default,
        priority,
        is_active,
        tenant_id,
        process_definition_bindings,
        created_at,
        updated_at
    )
VALUES (
        'release',
        NULL,
        'release_approval_flow_cn',
        2,
        true,
        NULL,
        true,
        1,
        '{"name": "发布审批流程(国内企业版)", "description": "含严格测试和灰度发布"}',
        NOW(),
        NOW()
    ) ON CONFLICT DO NOTHING;
-- ========================================================
-- 3. 服务请求流程绑定 (service_request)
-- ========================================================
INSERT INTO process_bindings (
        business_type,
        business_sub_type,
        process_definition_key,
        process_version,
        is_default,
        priority,
        is_active,
        tenant_id,
        process_definition_bindings,
        created_at,
        updated_at
    )
VALUES (
        'service_request',
        NULL,
        'service_request_flow_cn',
        2,
        true,
        NULL,
        true,
        1,
        '{"name": "服务请求流程(国内企业版)", "description": "服务目录化、自动化派单"}',
        NOW(),
        NOW()
    ) ON CONFLICT DO NOTHING;
-- ========================================================
-- 4. 变更流程绑定 - 升级国内版
-- ========================================================
UPDATE process_bindings
SET process_definition_key = 'change_normal_flow_cn',
    process_definition_bindings = '{"name": "标准变更流程(国内企业版)", "description": "低中高三档审批"}',
    updated_at = NOW()
WHERE business_type = 'change'
    AND is_default = true;
-- ========================================================
-- 5. 紧急故障流程 - 添加国内版作为备选
-- ========================================================
INSERT INTO process_bindings (
        business_type,
        business_sub_type,
        process_definition_key,
        process_version,
        is_default,
        priority,
        is_active,
        tenant_id,
        process_definition_bindings,
        created_at,
        updated_at
    )
VALUES (
        'incident',
        'emergency',
        'incident_emergency_flow_cn',
        2,
        false,
        'P0,P1',
        true,
        1,
        '{"name": "紧急故障响应流程(国内企业版)", "description": "P0/P1故障响应"}',
        NOW(),
        NOW()
    ) ON CONFLICT DO NOTHING;
-- ========================================================
-- 验证绑定结果
-- ========================================================
SELECT business_type,
    business_sub_type,
    process_definition_key,
    is_default,
    priority,
    is_active
FROM process_bindings
WHERE tenant_id = 1
    AND is_active = true
ORDER BY business_type,
    is_default DESC;