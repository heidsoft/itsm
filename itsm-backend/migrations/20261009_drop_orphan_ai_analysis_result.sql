-- 20261009_drop_orphan_ai_analysis_result.sql
--
-- 下线孤儿表 ai_analysis_result（单数）。
--
-- 来源：20260831_ai_analysis_result_expand.sql 以手工 DDL 建了 ai_analysis_result，
-- 但 Ent 的类型 AIAnalysisResult 实际映射到复数表 ai_analysis_results
-- （ent/aianalysisresult/aianalysisresult.go: Table = "ai_analysis_results"）。
-- 单数表因此从未被写入：本地生产库实测 0 行，非生成代码里的引用数为 0
-- （grep -rn ai_analysis_result --include='*.go' 只命中 ent/ 生成包内的 Label 常量，
-- 那是 Prometheus 标签名，不是表名）。两张同义表长期并存会让后续 Agent 误查空表，
-- 并给出假的"AI 分析无历史"结论，故按用户决策（2026-10-08）彻底删除。
--
-- 不动的同类候选：approval_records / approval_chains / ticket_approvals 虽也 0 行，
-- 但仍是 Ent schema 拥有的表（ent/schema/approval_record.go 等），且在旧审批退役决策里
-- 明确保留为遗留审批历史的查询入口；DROP 它们需要「删 schema + 重生成 + 迁移」的独立批次。
--
-- 幂等：表不存在时整段跳过。不可逆（无 _down.sql）——只可能删掉 0 行表，
-- 若某安装在此表里有数据，说明存在本仓库未知的写入者，直接 fail closed 让运维处置。

DO $$
DECLARE
    orphan_rows BIGINT;
BEGIN
    IF to_regclass('public.ai_analysis_result') IS NULL THEN
        RAISE NOTICE 'ai_analysis_result 不存在，跳过（幂等重跑）';
        RETURN;
    END IF;

    EXECUTE 'SELECT count(*) FROM public.ai_analysis_result' INTO orphan_rows;
    IF orphan_rows > 0 THEN
        RAISE EXCEPTION
            'ai_analysis_result 含 % 行数据，与本迁移的"从未被写入"前提矛盾；'
            '先确认写入者再决定归档或合并，禁止静默删除', orphan_rows;
    END IF;

    DROP TABLE public.ai_analysis_result;
    RAISE NOTICE '孤儿表 ai_analysis_result 已下线（其索引与 RLS policy 随表删除）';
END $$;
