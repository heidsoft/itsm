-- ========================================================
-- 国内企业ITSM工作流 - 完整部署脚本
-- 执行步骤: 1) 注册新流程定义 2) 清理旧绑定 3) 创建新绑定
-- ========================================================
BEGIN;
-- ========================================================
-- 步骤1: 注册新的国内版流程定义
-- ========================================================
-- 检查是否已存在
SELECT key,
    name
FROM process_definitions
WHERE key LIKE '%_cn'
    AND tenant_id = 1;
-- 插入新的国内版流程定义 (如果不存在)
INSERT INTO process_definitions (
        key,
        name,
        version,
        description,
        bpmn_xml,
        tenant_id,
        is_active,
        created_at,
        updated_at
    )
SELECT 'incident_emergency_flow_cn',
    '紧急故障响应流程(国内企业版)',
    '2.0.0',
    '国内企业版紧急故障响应流程，包含多级升级机制',
    (
        SELECT pg_read_binary_file(
                '/Users/heidsoft/Downloads/research/itsm/itsm-backend/service/bpmn/incident_emergency_flow_cn.bpmn'
            )
    ),
    1,
    true,
    NOW(),
    NOW()
WHERE NOT EXISTS (
        SELECT 1
        FROM process_definitions
        WHERE key = 'incident_emergency_flow_cn'
            AND tenant_id = 1
    );
INSERT INTO process_definitions (
        key,
        name,
        version,
        description,
        bpmn_xml,
        tenant_id,
        is_active,
        created_at,
        updated_at
    )
SELECT 'change_normal_flow_cn',
    '标准变更流程(国内企业版)',
    '2.0.0',
    '国内企业版标准变更流程，低中高三档审批',
    (
        SELECT pg_read_binary_file(
                '/Users/heidsoft/Downloads/research/itsm/itsm-backend/service/bpmn/change_normal_flow_cn.bpmn'
            )
    ),
    1,
    true,
    NOW(),
    NOW()
WHERE NOT EXISTS (
        SELECT 1
        FROM process_definitions
        WHERE key = 'change_normal_flow_cn'
            AND tenant_id = 1
    );
INSERT INTO process_definitions (
        key,
        name,
        version,
        description,
        bpmn_xml,
        tenant_id,
        is_active,
        created_at,
        updated_at
    )
SELECT 'service_request_flow_cn',
    '服务请求流程(国内企业版)',
    '2.0.0',
    '国内企业版服务请求流程，服务目录化',
    (
        SELECT pg_read_binary_file(
                '/Users/heidsoft/Downloads/research/itsm/itsm-backend/service/bpmn/service_request_flow_cn.bpmn'
            )
    ),
    1,
    true,
    NOW(),
    NOW()
WHERE NOT EXISTS (
        SELECT 1
        FROM process_definitions
        WHERE key = 'service_request_flow_cn'
            AND tenant_id = 1
    );
INSERT INTO process_definitions (
        key,
        name,
        version,
        description,
        bpmn_xml,
        tenant_id,
        is_active,
        created_at,
        updated_at
    )
SELECT 'problem_management_flow_cn',
    '问题管理流程(国内企业版)',
    '2.0.0',
    '国内企业版问题管理流程，专注根因分析和预防',
    (
        SELECT pg_read_binary_file(
                '/Users/heidsoft/Downloads/research/itsm/itsm-backend/service/bpmn/problem_management_flow_cn.bpmn'
            )
    ),
    1,
    true,
    NOW(),
    NOW()
WHERE NOT EXISTS (
        SELECT 1
        FROM process_definitions
        WHERE key = 'problem_management_flow_cn'
            AND tenant_id = 1
    );
INSERT INTO process_definitions (
        key,
        name,
        version,
        description,
        bpmn_xml,
        tenant_id,
        is_active,
        created_at,
        updated_at
    )
SELECT 'release_approval_flow_cn',
    '发布审批流程(国内企业版)',
    '2.0.0',
    '国内企业版发布审批流程，含灰度发布',
    (
        SELECT pg_read_binary_file(
                '/Users/heidsoft/Downloads/research/itsm/itsm-backend/service/bpmn/release_approval_flow_cn.bpmn'
            )
    ),
    1,
    true,
    NOW(),
    NOW()
WHERE NOT EXISTS (
        SELECT 1
        FROM process_definitions
        WHERE key = 'release_approval_flow_cn'
            AND tenant_id = 1
    );
-- ========================================================
-- 步骤2: 清理重复的process_bindings记录
-- ========================================================
-- 按business_type分组，保留is_default=true或最早创建的记录
DELETE FROM process_bindings
WHERE tenant_id = 1
    AND is_active = true
    AND business_type IN (
        SELECT business_type
        FROM process_bindings
        WHERE tenant_id = 1
            AND is_active = true
        GROUP BY business_type
        HAVING COUNT(*) > 1
    )
    AND id NOT IN (
        SELECT id
        FROM (
                SELECT id,
                    ROW_NUMBER() OVER (
                        PARTITION BY business_type
                        ORDER BY is_default DESC,
                            created_at ASC
                    ) as rn
                FROM process_bindings
                WHERE tenant_id = 1
                    AND is_active = true
            ) sub
        WHERE rn = 1
    );
-- ========================================================
-- 步骤3: 创建新的流程绑定
-- ========================================================
-- 更新现有的incident绑定到国内版 (如果需要)
UPDATE process_bindings
SET process_definition_key = 'incident_emergency_flow',
    updated_at = NOW()
WHERE business_type = 'incident'
    AND is_default = true
    AND tenant_id = 1;
-- 更新现有的service_request绑定到国内版 (如果需要)
UPDATE process_bindings
SET process_definition_key = 'service_request_flow',
    updated_at = NOW()
WHERE business_type = 'service_request'
    AND is_default = true
    AND tenant_id = 1;
-- 更新现有的problem绑定到国内版 (如果需要)
UPDATE process_bindings
SET process_definition_key = 'problem_management_flow',
    updated_at = NOW()
WHERE business_type = 'problem'
    AND is_default = true
    AND tenant_id = 1;
-- 更新现有的release绑定到国内版 (如果需要)
UPDATE process_bindings
SET process_definition_key = 'release_approval_flow',
    updated_at = NOW()
WHERE business_type = 'release'
    AND is_default = true
    AND tenant_id = 1;
-- ========================================================
-- 步骤4: 更新approval_workflows配置 (可选)
-- ========================================================
-- 更新P0/P1事件审批配置 (id=1)
UPDATE approval_workflows
SET nodes = '[
  {"name": "自动分配", "type": "service_task", "level": 1, "action": "auto_assign", "timeout": 30, "approver_type": "system"},
  {"name": "一线响应", "type": "user_task", "level": 2, "action": "l1_response", "timeout": 30, "approver_type": "l1_support"},
  {"name": "问题判定", "type": "exclusive_gateway", "level": 3, "condition_variable": "problem_identified"},
  {"name": "L2诊断", "type": "user_task", "level": 4, "action": "l2_diagnosis", "timeout": 60, "approver_type": "l2_support"},
  {"name": "L3专家处理", "type": "user_task", "level": 5, "action": "l3_expert_handle", "timeout": 240, "approver_type": "l3_expert"},
  {"name": "重大事件指挥室", "type": "user_task", "level": 6, "action": "war_room_handle", "timeout": 480, "approver_type": "ops_director"},
  {"name": "通知相关方", "type": "service_task", "level": 7, "action": "notify_stakeholders", "timeout": 10},
  {"name": "事件关闭", "type": "user_task", "level": 8, "action": "close_incident", "timeout": 30}
]'
WHERE id = 1
    AND tenant_id = 1;
COMMIT;
-- ========================================================
-- 验证查询
-- ========================================================
-- 1. 验证流程定义
SELECT key,
    name,
    version
FROM process_definitions
WHERE tenant_id = 1
ORDER BY key;
-- 2. 验证流程绑定
SELECT business_type,
    process_definition_key,
    is_default,
    priority
FROM process_bindings
WHERE tenant_id = 1
    AND is_active = true
ORDER BY business_type;
-- 3. 验证审批配置
SELECT id,
    name,
    ticket_type,
    jsonb_array_length(nodes) as node_count
FROM approval_workflows
WHERE tenant_id = 1
    AND is_active = true;