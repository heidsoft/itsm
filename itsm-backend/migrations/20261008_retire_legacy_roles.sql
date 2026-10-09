-- 20261008_retire_legacy_roles.sql
--
-- 角色模型重设（2026-10-07 决策：遗留角色退役，词表按 ITSM 产品定位重设）。
-- 不可逆（删除行无法还原）；前滚修复。幂等：重复执行只在仍有退役值时改数据。
--
-- 做三件事：
--   1) users.role 的退役值改到语义等价的新码（security → security_admin）；
--   2) 删除 roles 表里的退役角色行及其 role_permissions 授权；
--      有 user_roles 成员的角色一律跳过并 RAISE WARNING，等运维改绑后重跑补齐；
--   3) 已部署 BPMN 定义 XML 里的退役角色码换成新词表等价码。
--      不做这步，candidateGroups/assignee 解析不到人，存量流程待办会永久卡死。
--
-- 执行前实测（本地生产库 itsm_prod，2026-10-08，只读取证）：
--   users.role 命中退役码 1 行（只有 security；其余 16 个码 0 行）；
--   users.msp_role / change_approval_chains.role / change_review_members.role /
--     process_tasks.assignee / process_tasks.candidate_groups /
--     incident_escalation_rules.target_group / service_catalogs.approvers /
--     ticket_views.group_config 全部 0 命中；
--   roles 退役码 33 行（security 1 + 16 个岗位码 × 租户 1/2），user_roles 成员 0 行，
--     role_permissions 993 行；
--   process_definitions 36 行（18 个 key × 2 租户）的 bpmn_xml 元数据含退役码，
--     角色上下文只有两类：bpmn:metaData 的 assignee_type/notify_roles/target_role/
--     participant_roles，以及 camunda:assignee。
--
-- 已知不改的退役码出现位置（都不是角色语义，改了反而破坏数据）：
--   - camunda:topic="security-scan"（cloud_security_scan_flow 的连接器 topic）；
--   - bpmn:metaData name="expert_type" 的 dba,network_eng（专家技能分类，无 Go 消费者）；
--   - bpmn:metaData name="action" 的 l3_expert_handle（动作标识，\M 不匹配下划线，天然不受影响）。
--   dba 只改 camunda:assignee="dba" 这一处角色用法。

DO $$
DECLARE
    mapping     RECORD;
    changed     INTEGER;
    total_rows  INTEGER := 0;
BEGIN
    FOR mapping IN
        SELECT * FROM (VALUES
            ('it_director',   'it_admin'),
            ('ops_director',  'it_admin'),
            ('l3_expert',     'it_admin'),
            ('sd_manager',    'manager'),
            ('ops_manager',   'manager'),
            ('rd_manager',    'manager'),
            ('l1_support',    'agent'),
            ('l2_support',    'technician'),
            ('developer',     'technician'),
            ('ops_engineer',  'technician')
        ) AS m(from_code, to_code)
    LOOP
        UPDATE process_definitions p
        SET bpmn_xml = to_jsonb(encode(convert_to(
                regexp_replace(
                    convert_from(decode(p.bpmn_xml #>> '{}', 'base64'), 'UTF8'),
                    '\m' || mapping.from_code || '\M', mapping.to_code, 'g'),
                'UTF8'), 'base64')),
            updated_at = now()
        WHERE convert_from(decode(p.bpmn_xml #>> '{}', 'base64'), 'UTF8')
              ~ ('\m' || mapping.from_code || '\M');

        GET DIAGNOSTICS changed = ROW_COUNT;
        total_rows := total_rows + changed;
    END LOOP;

    -- dba 只出现在容量任务的 assignee；expert_type 分类保持原样。
    UPDATE process_definitions p
    SET bpmn_xml = to_jsonb(encode(convert_to(
            replace(
                convert_from(decode(p.bpmn_xml #>> '{}', 'base64'), 'UTF8'),
                'camunda:assignee="dba"', 'camunda:assignee="technician"'),
            'UTF8'), 'base64')),
        updated_at = now()
    WHERE convert_from(decode(p.bpmn_xml #>> '{}', 'base64'), 'UTF8')
          LIKE '%camunda:assignee="dba"%';

    GET DIAGNOSTICS changed = ROW_COUNT;
    total_rows := total_rows + changed;

    -- 收尾不变量：角色上下文里不得再有任何已映射的退役码。
    IF EXISTS (
        SELECT 1 FROM process_definitions p
        WHERE convert_from(decode(p.bpmn_xml #>> '{}', 'base64'), 'UTF8')
              ~ '\m(it_director|ops_director|l3_expert|sd_manager|ops_manager|rd_manager|l1_support|l2_support|developer|ops_engineer)\M'
    ) THEN
        RAISE EXCEPTION 'BPMN 定义仍含退役角色码，迁移未完成';
    END IF;

    RAISE NOTICE 'BPMN 定义改写行数: %（含重复执行时为 0）', total_rows;
END $$;

DO $$
DECLARE
    retired   TEXT[] := ARRAY[
        'security', 'it_director', 'ops_director', 'ops_manager', 'ops_engineer',
        'dba', 'network_eng', 'sd_manager', 'l1_support', 'l2_support', 'l3_expert',
        'rd_manager', 'developer', 'qa_engineer', 'dept_manager', 'team_lead', 'guest'
    ];
    renamed   INTEGER;
    dropped   INTEGER;
    cleaned   INTEGER;
    holders   TEXT;
BEGIN
    -- 1) 主角色改值：退役码 → 新词表等价码。
    UPDATE users
    SET role = 'security_admin', updated_at = now()
    WHERE role = 'security';
    GET DIAGNOSTICS renamed = ROW_COUNT;
    RAISE NOTICE 'users.role 改值行数: %', renamed;

    -- 2) 有成员的退役角色先摆出来，不删（删了会让人失去授权）。
    SELECT string_agg(r.code, ',' ORDER BY r.code) INTO holders
    FROM roles r
    WHERE r.code = ANY(retired)
      AND EXISTS (SELECT 1 FROM user_roles ur WHERE ur.role_id = r.id);
    IF holders IS NOT NULL THEN
        RAISE WARNING '退役角色 [%] 仍有 user_roles 成员，本次保留；改绑成员后重跑本迁移即可补齐删除', holders;
    END IF;

    -- role_permissions.role_id 对 roles 无外键，必须显式清理，否则留下悬空授权行。
    DELETE FROM role_permissions rp
    USING roles r
    WHERE rp.role_id = r.id
      AND r.code = ANY(retired)
      AND NOT EXISTS (SELECT 1 FROM user_roles ur WHERE ur.role_id = r.id);
    GET DIAGNOSTICS cleaned = ROW_COUNT;

    DELETE FROM roles r
    WHERE r.code = ANY(retired)
      AND NOT EXISTS (SELECT 1 FROM user_roles ur WHERE ur.role_id = r.id);
    GET DIAGNOSTICS dropped = ROW_COUNT;

    RAISE NOTICE '退役角色删除: roles % 行, role_permissions % 行', dropped, cleaned;
END $$;
