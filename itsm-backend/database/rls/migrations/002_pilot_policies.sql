-- =============================================================================
-- RLS Migration 002 — Enable Row-Level Security on Pilot Tables
--
-- 目的：在 changes 和 vectors 两张表上启用 RLS 策略，作为 R1 里程碑的
-- 双表试点。其他 100 张业务表将在 R2 里程碑批量启用。
--
-- 策略：使用 SESSION 变量 `app.current_tenant` 承载 tenant_id：
--   - itsm_app 连接 → SET SESSION app.current_tenant = <tid>，policy 匹配
--   - itsm_admin 连接 → BYPASSRLS 属性生效，policy 不适用
--   - 未 SET 或 RESET 后 → current_setting(...) 返回空字符串，
--     NULLIF(...)::bigint 结果为 NULL，任何行都不匹配 → 拒绝所有查询
--
-- 关键修正（vs R0 POC）：
--   POC 版本使用 `current_setting(name, true)::bigint`，RESET 后转换报错。
--   本版改用 `NULLIF(current_setting(name, true), '')::bigint`：
--   - RESET 后返回 '' → NULLIF → NULL → policy 不匹配 → 拒绝
--   - 从未 SET 过 → 返回 '' 同理
--   - 已 SET 有效值 → 正常转 bigint 匹配
--
-- 幂等性：`IF NOT EXISTS` + `DROP POLICY IF EXISTS`，重复执行安全。
-- 回滚：见 002_pilot_policies_rollback.sql
-- =============================================================================

-- ---------------------------------------------------------------------------
-- 1. changes 表
-- ---------------------------------------------------------------------------

ALTER TABLE changes ENABLE ROW LEVEL SECURITY;

-- FORCE 让表所有者也受约束（防止 owner 视角意外看到全部数据）
ALTER TABLE changes FORCE ROW LEVEL SECURITY;

-- 幂等：先 DROP 后 CREATE
DROP POLICY IF EXISTS tenant_isolation ON changes;

CREATE POLICY tenant_isolation ON changes
    USING       (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::bigint)
    WITH CHECK  (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::bigint);

-- ---------------------------------------------------------------------------
-- 2. vectors 表（AI 检索索引）
-- ---------------------------------------------------------------------------

-- vectors 表在部分环境可能因 pgvector 未启用而不存在，加存在性判断
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema = 'public' AND table_name = 'vectors') THEN
        EXECUTE 'ALTER TABLE vectors ENABLE ROW LEVEL SECURITY';
        EXECUTE 'ALTER TABLE vectors FORCE ROW LEVEL SECURITY';
        EXECUTE 'DROP POLICY IF EXISTS tenant_isolation ON vectors';
        EXECUTE $POLICY$
            CREATE POLICY tenant_isolation ON vectors
                USING       (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::bigint)
                WITH CHECK  (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::bigint)
        $POLICY$;
        RAISE NOTICE 'RLS enabled on vectors';
    ELSE
        RAISE NOTICE 'vectors table not found, skipping (pgvector not enabled?)';
    END IF;
END $$;

-- ---------------------------------------------------------------------------
-- 3. 验证语句（可选运行）
-- ---------------------------------------------------------------------------
-- SELECT tablename, rowsecurity, forcerowsecurity
-- FROM pg_tables WHERE schemaname = 'public' AND tablename IN ('changes', 'vectors');
--
-- SELECT * FROM pg_policies WHERE tablename IN ('changes', 'vectors');
