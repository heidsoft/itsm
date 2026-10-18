-- 20261018_fix_bpmn_approval_result_variable.sql
--
-- 已部署 BPMN 定义的条件表达式读的是蛇形审批变量 approval_result，而引擎完成任务时
-- 写入的是驼峰 approvalResult（service/bpmn_process_executor.go、service/bpmn_approval_bridge_service.go、
-- handlers/bpmn/workflow.go 三条写入路径全部是驼峰）。变量名对不上时条件恒为 false，
-- 审批驳回后网关不会走驳回分支，流程会静默停在错误路径上，不报错也没有日志。
--
-- 内置模板文件已在同批代码修复（service/bpmn/service_request_flow.bpmn 6 处、
-- service/bpmn/incident_emergency_flow_v1.1.bpmn 1 处），并由
-- service/bpmn_template_variable_guard_test.go 锁死语料。但部署逻辑对已存在的 key
-- 直接跳过（bpmn_template_service.go isTemplateDeployed），文件修复不会落到存量行，
-- 所以存量租户必须靠本迁移改写 process_definitions.bpmn_xml。
--
-- 范围：只改 approval_result 这一个变量名。process_definitions.bpmn_xml 是 base64 文本
-- 存进 JSONB（见 20261008_retire_legacy_roles.sql 的同款解码/再编码），因此解码后整体
-- 替换再写回。approval_result 在仓库里没有任何活的写入方（dto/bpmn_process_trigger_dto.go
-- 的 ConvertToMap 无调用者，service/common/event/event.go 是事件 JSON 标签，不是流程变量），
-- 所以不存在"改了把正确条件弄坏"的别名用法。
--
-- 幂等：WHERE 只命中仍含 approval_result 的行，重复执行改写 0 行。
-- 收尾不变量：process_definitions 里不得再出现 approval_result，否则 RAISE EXCEPTION。
-- 影响行数由本迁移自己 RAISE NOTICE 输出。本文件属磁盘自发现流，版本日期晚于收养
-- cutoff（20260908），因此不会被登记为"已执行"，而是在下一次后端启动时自动应用；
-- 对已有数据的部署，启动前请对目标库做一次备份。
-- 回滚见 20261018_fix_bpmn_approval_result_variable_down.sql。

DO $$
DECLARE
    affected  INTEGER;
    total     INTEGER := 0;
BEGIN
    UPDATE process_definitions p
    SET bpmn_xml = to_jsonb(encode(convert_to(
            replace(
                convert_from(decode(p.bpmn_xml #>> '{}', 'base64'), 'UTF8'),
                'approval_result', 'approvalResult'),
            'UTF8'), 'base64')),
        updated_at = now()
    WHERE convert_from(decode(p.bpmn_xml #>> '{}', 'base64'), 'UTF8')
          LIKE '%approval_result%';

    GET DIAGNOSTICS affected = ROW_COUNT;
    total := total + affected;
    RAISE NOTICE 'BPMN 定义审批变量改写行数: %（重复执行为 0）', total;

    IF EXISTS (
        SELECT 1 FROM process_definitions p
        WHERE convert_from(decode(p.bpmn_xml #>> '{}', 'base64'), 'UTF8')
              LIKE '%approval_result%'
    ) THEN
        RAISE EXCEPTION 'process_definitions 仍含 approval_result，迁移未完成';
    END IF;
END $$;
