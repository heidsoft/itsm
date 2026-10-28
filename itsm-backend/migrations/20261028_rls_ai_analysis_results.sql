-- 20261028_rls_ai_analysis_results.sql
--
-- 给 AI 分析持久化表 ai_analysis_results 补 RLS 策略。
--
-- 背景：20260831_ai_analysis_result_expand.sql 的 RLS 意图落在**单数表**
-- ai_analysis_result（手写 DDL），而 ent 实体真实映射的是**复数表**
-- ai_analysis_results（ent/aianalysisresult/aianalysisresult.go）；单数表
-- 后由 20261009_drop_orphan_ai_analysis_result.sql 按用户决策下线，
-- 但 RLS 意图没有随之转移到复数表——真实存储 AI 分析结果（request_prompt、
-- result_json 等租户敏感内容）的表在 prod 中文策略全无。
-- 本文件把该意图对齐到 ent 真实表上。
--
-- 谓词与本批 20261028_rls_core_tables.sql 同一口径：
--   tenant_id = get_current_tenant_id()（helper 由 009 声明、019 对齐到
--   app.current_tenant —— database/rls/driver.go 注入的变量）；
--   未设/RESET/非法值 => NULL => fail closed 拒绝。
-- 与既有 21 张表姿态一致：ENABLE + 非 FORCE，等待 RLS enforcement 变更窗
-- （RLS_MODE=shadow→enforce + itsm_app 非 owner 角色）统一生效。
--
-- 存在性守卫 vs 顺序依赖：复数表由 ent auto-migrate 创建。若本文件在某环境
-- 先于 ent 初始化执行（历史上磁盘流与 ent 流的执行顺序不保证），无守卫的
-- ALTER 会以 42P01 中断整条初始化链（2025-10-01 20260501 的实证教训）。
-- DO 块守卫让本文件在任何顺序下安全跳过，幂等可重复执行。

DO $$
BEGIN
    IF to_regclass('public.ai_analysis_results') IS NULL THEN
        RAISE NOTICE 'ai_analysis_results 不存在（ent auto-migrate 尚未执行），跳过';
    ELSE
        EXECUTE 'ALTER TABLE ai_analysis_results ENABLE ROW LEVEL SECURITY';
        EXECUTE 'DROP POLICY IF EXISTS tenant_isolation_ai_analysis_results ON ai_analysis_results';
        EXECUTE $policy$
            CREATE POLICY tenant_isolation_ai_analysis_results ON ai_analysis_results
                USING (tenant_id = get_current_tenant_id())
                WITH CHECK (tenant_id = get_current_tenant_id())
        $policy$;
        EXECUTE 'COMMENT ON POLICY tenant_isolation_ai_analysis_results ON ai_analysis_results IS ' ||
                $c$'tenant RLS scaffold, intent restored from 20260831(singular)-20261009(dropped) chain; active under RLS enforcement rollout'$c$;
        RAISE NOTICE 'ai_analysis_results RLS policy ensured';
    END IF;
END $$;
