-- 20261031_rls_policy_alignment.sql
--
-- RLS enforcement 实路（第 2 段）：把已启用 RLS 的表的策略姿态统一成
-- 「一个表 = 一个 tenant_isolation_<table> 策略 + get_current_tenant_id() 谓词」。
--
-- 为什么需要这一段：策略散落在 009/019（权威内联流）、20260828/20260831/
-- 20261014/20261015/20261028（磁盘流）里，谓词与命名各写各的。实测两处真实缺陷：
--
--   1. knowledge_article_reviews 的策略读的是 **app.tenant_id**，
--      而驱动（database/rls/driver.go）注入的是 **app.current_tenant**。
--      在 RLS_MODE=enforce 下这条策略永远取到 NULL，表对非 owner 恒返回 0 行——
--      不是"隔离生效"，是"数据读不到"，而且没有任何报错。
--   2. 002_pilot_policies.sql 留下的策略名是 tenant_isolation（无表名后缀），
--      与 20261028 的 tenant_isolation_changes 并存时同一张表会有两条策略；
--      运维无法从策略名判断它守的是哪张表、哪套谓词。
--
-- 本迁移只做姿态统一，不新增受 RLS 保护的表（扩面是 P2-1 的独立变更窗）：
--   * 对每张「已 ENABLE RLS 且有 tenant_id 列」的表，重建
--     tenant_isolation_<table>，谓词统一为 tenant_id = get_current_tenant_id()；
--   * 删除同一张表上**其它**「谓词引用 tenant_id」的策略（遗留命名/错误 GUC）。
--     只删租户相关策略，不动业务自定义策略（例如按状态的可见性策略）。
--   * 末尾自检：任何一张 RLS 表缺少规范策略即 RAISE EXCEPTION，避免账本记
--     「已应用」而某张表仍在用错误谓词。
--
-- 事务性：DROP POLICY + CREATE POLICY 在同一事务内完成，且 DDL 取表的
-- ACCESS EXCLUSIVE 锁，期间不会有查询看到"策略已删、未建"的中间态。
-- 迁移器以事务执行；手工 psql 执行时必须加 -1。
--
-- 幂等：可重复执行。策略已规范时重建结果相同。
--
-- 回滚：见 20261031_rls_policy_alignment_down.sql。回滚只删除本迁移创建的
-- 规范策略并恢复 knowledge_article_reviews 的历史策略形态——把策略删空会让
-- 表在 enforce 下恒返回 0 行，因此 down 文件必须恢复，而不是一删了之。

-- ============================================================================
-- 0. 前置守卫：helper 只由权威迁移流（009/019）声明。
-- ============================================================================
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'get_current_tenant_id') THEN
        RAISE EXCEPTION 'missing helper get_current_tenant_id(): replay the canonical migration stream (009/019) before this file';
    END IF;
END $$;

-- ============================================================================
-- 1. 逐表统一策略
-- ============================================================================
DO $$
DECLARE
    target       record;
    canonical    text;
    legacy_names text[];
    legacy_name  text;
BEGIN
    FOR target IN
        SELECT c.relname AS table_name
        FROM pg_class c
        WHERE c.relkind = 'r'
          AND c.relnamespace = 'public'::regnamespace
          AND c.relrowsecurity
          AND EXISTS (
              SELECT 1 FROM information_schema.columns col
              WHERE col.table_schema = 'public'
                AND col.table_name = c.relname
                AND col.column_name = 'tenant_id'
          )
        ORDER BY c.relname
    LOOP
        canonical := 'tenant_isolation_' || target.table_name;

        -- PostgreSQL 标识符上限 63 字节；超长会被静默截断，导致策略名与
        -- 表名对不上（自检与运维脚本都会失效）。显式失败比截断好。
        IF length(canonical) > 63 THEN
            RAISE EXCEPTION 'policy name % exceeds 63 chars for table %; shorten the table name or the policy prefix',
                canonical, target.table_name;
        END IF;

        -- 1.1 先收集同一张表上的其它租户策略名。
        --     先物化成数组再删，避免边遍历 pg_policies 边删行。
        SELECT array_agg(p.policyname ORDER BY p.policyname)
        INTO legacy_names
        FROM pg_policies p
        WHERE p.schemaname = 'public'
          AND p.tablename = target.table_name
          AND p.policyname <> canonical
          AND (
                (p.qual       IS NOT NULL AND p.qual       ILIKE '%tenant_id%')
             OR (p.with_check IS NOT NULL AND p.with_check ILIKE '%tenant_id%')
          );

        -- 1.2 重建规范策略（DROP + CREATE 保证谓词/角色/命令形态一致，
        --     而不是在旧策略上"尽量修补"）。
        EXECUTE format('DROP POLICY IF EXISTS %I ON public.%I', canonical, target.table_name);
        EXECUTE format(
            'CREATE POLICY %I ON public.%I USING (tenant_id = get_current_tenant_id()) WITH CHECK (tenant_id = get_current_tenant_id())',
            canonical, target.table_name
        );

        -- 1.3 删除遗留租户策略。
        IF legacy_names IS NOT NULL THEN
            FOREACH legacy_name IN ARRAY legacy_names LOOP
                EXECUTE format('DROP POLICY %I ON public.%I', legacy_name, target.table_name);
                RAISE NOTICE 'dropped non-canonical tenant policy %.%', target.table_name, legacy_name;
            END LOOP;
        END IF;

        RAISE NOTICE 'normalized RLS policy on % (legacy policies dropped: %)',
            target.table_name, COALESCE(array_length(legacy_names, 1), 0);
    END LOOP;
END $$;

-- ============================================================================
-- 2. 结果自检：规范策略必须存在且谓词走 helper。
--
-- 这一段防的是"迁移看起来成功、实际漏了一张表"。漏表的后果不是报错，
-- 而是 enforce 下该表恒 0 行（有 RLS 无策略）或继续用错误 GUC（读不到）。
-- ============================================================================
DO $$
DECLARE
    offenders text;
BEGIN
    SELECT string_agg(c.relname, ', ' ORDER BY c.relname)
    INTO offenders
    FROM pg_class c
    WHERE c.relkind = 'r'
      AND c.relnamespace = 'public'::regnamespace
      AND c.relrowsecurity
      AND EXISTS (
          SELECT 1 FROM information_schema.columns col
          WHERE col.table_schema = 'public'
            AND col.table_name = c.relname
            AND col.column_name = 'tenant_id'
      )
      AND NOT EXISTS (
          SELECT 1 FROM pg_policies p
          WHERE p.schemaname = 'public'
            AND p.tablename = c.relname
            AND p.policyname = 'tenant_isolation_' || c.relname
            AND p.qual LIKE '%get_current_tenant_id()%'
      );

    IF offenders IS NOT NULL THEN
        RAISE EXCEPTION 'RLS policy alignment incomplete: tables without canonical tenant policy: %', offenders;
    END IF;

    RAISE NOTICE 'RLS policy alignment ok: every RLS-enabled tenant table has tenant_isolation_<table> using get_current_tenant_id()';
END $$;
