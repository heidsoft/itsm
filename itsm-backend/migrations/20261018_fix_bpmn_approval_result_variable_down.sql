-- 20261018_fix_bpmn_approval_result_variable_down.sql
--
-- 回滚 20261018_fix_bpmn_approval_result_variable.sql：把驼峰 approvalResult 换回蛇形
-- approval_result，但只针对本修复前内置模板本身就是蛇形的两个 key
-- （service_request_flow、incident_emergency_flow_v1.1）。change_normal_flow 等原本
-- 就用驼峰的 key 不参与回滚，否则会把正确的定义改成错的。
--
-- 已知局限：
--   - 按 key 回滚会连同该 key 后续发布的版本一起改回蛇形，即修复前的错误语义会重新生效；
--     这是"恢复原状"的代价，不是数据损坏。
--   - 租户自行导入/编辑、且恰好使用驼峰 approvalResult 的同名 key 定义也会被改回蛇形。
--   - 主迁移的改写范围是全表（WHERE 含 approval_result），若存在其它 key 的蛇形定义，
--     本回滚无法定位并还原它们；执行前请按 key 备份 bpmn_xml。
--
-- 幂等：只命中含 approvalResult 的上述 key，重复执行改写 0 行。

DO $$
DECLARE
    affected INTEGER;
BEGIN
    UPDATE process_definitions p
    SET bpmn_xml = to_jsonb(encode(convert_to(
            replace(
                convert_from(decode(p.bpmn_xml #>> '{}', 'base64'), 'UTF8'),
                'approvalResult', 'approval_result'),
            'UTF8'), 'base64')),
        updated_at = now()
    WHERE p.key IN ('service_request_flow', 'incident_emergency_flow_v1.1')
      AND convert_from(decode(p.bpmn_xml #>> '{}', 'base64'), 'UTF8')
          LIKE '%approvalResult%';

    GET DIAGNOSTICS affected = ROW_COUNT;
    RAISE NOTICE 'BPMN 定义审批变量回滚行数: %（重复执行为 0）', affected;
END $$;
