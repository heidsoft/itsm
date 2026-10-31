-- 20261031_ai_vector_bigint.sql
--
-- P2-4：引用列 int4 -> int8 类型统一（AI/向量域 + 全库 tenant_id 不变量）。
--
-- 背景（DB 审计 output/prod-db-schema-audit-2026-10-26.md §3-P2-4 / §5 第 7 条）：
-- 全库 138 张带 tenant_id 的表里有 136 张是 bigint，只有 ai_feedbacks 与
-- vectors 是 integer；它们的引用列（user_id / item_id / object_id）同样是
-- integer。int4 与 int8 混用在 JOIN 时触发隐式转换，会让 tenant_id 上的索引
-- 在部分执行计划里失效——这不是"空间优化"，是把索引建了却用不上。
--
-- 根因不是手滑：这两张表由 020_add_ai_vector_observability_storage 的手写
-- DDL 创建（`tenant_id INT`），而 ent 侧 field.Int 生成的是 bigint，两条建表
-- 链路没有共同规范。同类隐患还有两处**当前 dev/prod 恰好被 ent 抢先建表而
-- 掩盖**的手写 DDL：20260620 的 domain_configs、20260614 的
-- ci_relationships.tenant_id 都写的是 INTEGER，只是因为 ent 先建了 bigint 表、
-- `CREATE TABLE IF NOT EXISTS` / `ADD COLUMN IF NOT EXISTS` 变成空操作才没暴露。
-- 在「磁盘迁移先于 ent 建表」的 fresh 库上，这两处会真的落成 int4。
-- 020 与 20260614/20260620 均属已应用迁移，不得改（改动会让 checksum 与历史
-- 账本不一致），因此修正只能以本磁盘迁移的形式追加。
--
-- 分两段执行：
--   A. 泛化不变量：public 下所有 tenant_id 为 integer 的表统一转 bigint。
--      tenant_id 按定义引用 tenants.id（bigint），不存在"合法的 int4 tenant_id"，
--      因此这是纯结构修正，不改任何读写语义。
--   B. 显式目标：ai_feedbacks.user_id / item_id、vectors.object_id。
--      这三列不是通用的"所有 *_id"规则（item_id/object_id 语义按域不同），
--      只按审计点名处理，不做猜测式批量转换。
--
-- 执行姿态：
--   * 逐列守卫：表/列不存在跳过（fresh 库 ent 建表可能已是 bigint）；
--     已是 bigint 跳过，重复执行安全。
--   * ALTER COLUMN TYPE 会取 ACCESS EXCLUSIVE 锁并重写表。受影响表都是
--     AI 遥测/向量/配置表，体量小（审计实测全库 ≈1658 行），重写耗时可忽略；
--     若未来某库这些表很大，应在低峰执行，本文件刻意不引入 CONCURRENTLY
--     （类型变更没有 CONCURRENTLY 形式）。
--   * 依赖对象（vectors 上的 UNIQUE(tenant_id, object_type, object_id)、
--     ai_feedbacks_tenant_idx 等）由 PostgreSQL 在重写时自动重建，无需手工
--     DROP/CREATE；刻意不写「先删索引再重建」，避免中途失败留下缺索引状态。
--   * public 下无视图（2026-10-31 实测 0 个），不存在视图依赖阻断。
--
-- 回滚：见 20261031_ai_vector_bigint_down.sql。回滚只回退本迁移的**显式目标列**，
-- 不把泛化的 tenant_id 修正回 int4——那属于回归而非修复，且 down 文件无法
-- 知道哪些表原本就是 int4。回滚前会检查存量值域，越界则拒绝（int8 -> int4 有损）。

-- ============================================================================
-- A. 泛化不变量：public.*.tenant_id 不得为 integer
--
-- 障碍：PostgreSQL 拒绝修改「被 policy 定义引用的列」的类型
-- （ERROR: cannot alter type of a column used in a policy definition）。
-- 已启用 RLS 的表其 tenant_isolation_<table> 谓词就写在 tenant_id 上，
-- 因此必须先把 policy 摘下来、改完类型再按原定义装回去。
--
-- 为什么捕获 pg_policies 重建，而不是「按 tenant_isolation_<table> 约定手写
-- 一条 USING (tenant_id = get_current_tenant_id())」：
--   * 约定并不统一——002 的 FORCE 试点与 019 的 helper 对齐留下过不同谓词，
--     按名字猜谓词会把一条租户隔离策略悄悄换成另一条；
--   * 租户隔离策略的正确性不容"差不多"：重建后谓词必须逐字符等于原定义。
-- pg_policies 提供 permissive / roles / cmd / qual / with_check，足够无损重建。
--
-- 事务性：DROP POLICY → ALTER → CREATE POLICY 三步必须原子。迁移器以事务执行
-- 本文件，失败会整体回滚；手工用 psql 执行时必须加 -1（单事务），否则中途失败
-- 会留下「策略已删、类型未改」的表。
-- ============================================================================
DO $$
DECLARE
    target    record;
    policy    record;
    policies  jsonb;
    roles_sql text;
BEGIN
    FOR target IN
        SELECT c.table_name
        FROM information_schema.columns c
        JOIN information_schema.tables t
          ON t.table_schema = c.table_schema
         AND t.table_name = c.table_name
         AND t.table_type = 'BASE TABLE'
        WHERE c.table_schema = 'public'
          AND c.column_name = 'tenant_id'
          AND c.data_type = 'integer'
        ORDER BY c.table_name
    LOOP
        -- 1. 捕获该表全部 policy 定义（不只 tenant_id 相关：一次改列可能被
        --    多条策略引用，全量摘除再全量装回最不容易漏）。
        SELECT jsonb_agg(jsonb_build_object(
                   'name',       p.policyname,
                   'permissive', p.permissive,
                   'roles',      to_jsonb(p.roles),
                   'cmd',        p.cmd,
                   'qual',       p.qual,
                   'check',      p.with_check
               ) ORDER BY p.policyname)
        INTO policies
        FROM pg_policies p
        WHERE p.schemaname = 'public'
          AND p.tablename = target.table_name;

        -- 2. 摘除
        IF policies IS NOT NULL THEN
            FOR policy IN SELECT value FROM jsonb_array_elements(policies) LOOP
                EXECUTE format('DROP POLICY %I ON public.%I',
                               policy.value->>'name', target.table_name);
            END LOOP;
        END IF;

        -- 3. 改类型
        EXECUTE format('ALTER TABLE public.%I ALTER COLUMN tenant_id TYPE bigint', target.table_name);

        -- 4. 按原定义装回
        IF policies IS NOT NULL THEN
            FOR policy IN SELECT value FROM jsonb_array_elements(policies) LOOP
                -- PUBLIC 是关键字而不是角色名；quote_ident('public') 会变成
                -- "public" 并报 role does not exist。
                SELECT string_agg(
                           CASE WHEN lower(r) = 'public' THEN 'PUBLIC' ELSE quote_ident(r) END,
                           ', '
                       )
                INTO roles_sql
                FROM jsonb_array_elements_text(policy.value->'roles') AS r;

                EXECUTE format(
                    'CREATE POLICY %I ON public.%I AS %s FOR %s TO %s %s %s',
                    policy.value->>'name',
                    target.table_name,
                    lower(policy.value->>'permissive'),
                    policy.value->>'cmd',
                    roles_sql,
                    CASE WHEN policy.value->>'qual' IS NOT NULL
                         THEN 'USING (' || (policy.value->>'qual') || ')' ELSE '' END,
                    CASE WHEN policy.value->>'check' IS NOT NULL
                         THEN 'WITH CHECK (' || (policy.value->>'check') || ')' ELSE '' END
                );
            END LOOP;
        END IF;

        RAISE NOTICE '%.tenant_id converted integer -> bigint (policies recreated: %)',
            target.table_name, COALESCE(jsonb_array_length(policies), 0);
    END LOOP;
END $$;

-- ============================================================================
-- B. 显式目标：AI/向量域引用列
-- ============================================================================
DO $$
DECLARE
    target record;
BEGIN
    IF to_regclass('public.ai_feedbacks') IS NULL THEN
        RAISE NOTICE 'skip ai_feedbacks reference columns: table missing';
    ELSE
        FOR target IN
            SELECT c.column_name
            FROM information_schema.columns c
            WHERE c.table_schema = 'public'
              AND c.table_name = 'ai_feedbacks'
              AND c.column_name IN ('user_id', 'item_id')
              AND c.data_type = 'integer'
            ORDER BY c.column_name
        LOOP
            EXECUTE format('ALTER TABLE public.ai_feedbacks ALTER COLUMN %I TYPE bigint', target.column_name);
            RAISE NOTICE 'ai_feedbacks.% converted integer -> bigint', target.column_name;
        END LOOP;

        -- 缺列不是本迁移的职责（列由 020/022 定义），但静默跳过会让「类型统一」
        -- 看起来成功而实际没覆盖，因此对缺失列如实告警。
        FOR target IN
            SELECT expected.column_name
            FROM (VALUES ('tenant_id'), ('user_id'), ('item_id')) AS expected(column_name)
            WHERE NOT EXISTS (
                SELECT 1 FROM information_schema.columns c
                WHERE c.table_schema = 'public'
                  AND c.table_name = 'ai_feedbacks'
                  AND c.column_name = expected.column_name
            )
        LOOP
            RAISE WARNING 'ai_feedbacks.% 不存在，无法确认其类型（表结构可能落后于 020/022）', target.column_name;
        END LOOP;
    END IF;

    IF to_regclass('public.vectors') IS NULL THEN
        RAISE NOTICE 'skip vectors reference columns: table missing';
    ELSE
        FOR target IN
            SELECT c.column_name
            FROM information_schema.columns c
            WHERE c.table_schema = 'public'
              AND c.table_name = 'vectors'
              AND c.column_name IN ('object_id')
              AND c.data_type = 'integer'
            ORDER BY c.column_name
        LOOP
            EXECUTE format('ALTER TABLE public.vectors ALTER COLUMN %I TYPE bigint', target.column_name);
            RAISE NOTICE 'vectors.% converted integer -> bigint', target.column_name;
        END LOOP;

        FOR target IN
            SELECT expected.column_name
            FROM (VALUES ('tenant_id'), ('object_id')) AS expected(column_name)
            WHERE NOT EXISTS (
                SELECT 1 FROM information_schema.columns c
                WHERE c.table_schema = 'public'
                  AND c.table_name = 'vectors'
                  AND c.column_name = expected.column_name
            )
        LOOP
            RAISE WARNING 'vectors.% 不存在，无法确认其类型（表结构可能落后于 020）', target.column_name;
        END LOOP;
    END IF;
END $$;

-- ============================================================================
-- C. 结果自检：不变量未达成必须显式失败。
--
-- 前面两段是「尽力转换 + 如实告警」，但 P2-4 的目标是硬不变量：
-- tenant_id 与点名引用列不能停在 int4。若因权限/锁/异常路径导致没转成功，
-- 本迁移必须失败而不是记一笔"已应用"——否则账本会撒谎，而索引失效是无声的。
-- ============================================================================
DO $$
DECLARE
    offenders text;
BEGIN
    SELECT string_agg(table_name || '.' || column_name, ', ' ORDER BY table_name, column_name)
    INTO offenders
    FROM information_schema.columns
    WHERE table_schema = 'public'
      AND data_type = 'integer'
      AND (    column_name = 'tenant_id'
            OR (table_name = 'ai_feedbacks' AND column_name IN ('user_id', 'item_id'))
            OR (table_name = 'vectors'      AND column_name = 'object_id') );

    IF offenders IS NOT NULL THEN
        RAISE EXCEPTION 'P2-4 未完成：以下引用列仍为 integer: %', offenders;
    END IF;

    RAISE NOTICE 'P2-4 ok: public.tenant_id 与 ai_feedbacks/vectors 引用列均为 bigint';
END $$;
