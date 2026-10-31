-- 20261031_ai_vector_bigint_down.sql
--
-- 回滚 20261031_ai_vector_bigint.sql 的**显式目标列**：bigint -> integer。
--
-- 刻意不对称：正向迁移里那段泛化的 tenant_id 修正（public.*.tenant_id
-- integer -> bigint）不在回滚范围内。原因有两条：
--   1. down 文件无法知道哪些表在本迁移之前本来就是 bigint，盲目全量回退会把
--      136 张正确建表的 tenant_id 一起降级；
--   2. 回到 int4 tenant_id 是回归而不是修复——那正是 P2-4 要消除的缺陷。
-- 需要回滚的应急场景几乎都发生在显式目标列（AI 遥测/向量写入异常），
-- 回退这两张表即可；tenant_id 的类型统一应当保留。
--
-- 这是**有损**回滚。int8 -> int4 在 PostgreSQL 里不是隐式转换，越界值会让
-- ALTER 直接报 22003；但更危险的情形是「运维为了回滚成功，先手动删数据」。
-- 因此本脚本先显式检查值域，越界就带明确信息拒绝执行，把决策权交回给人，
-- 而不是让一次回滚顺手截断租户 ID。
--
-- 幂等：列已是 integer 或表/列不存在则跳过。

DO $$
DECLARE
    target  record;
    max_val bigint;
BEGIN
    IF to_regclass('public.ai_feedbacks') IS NOT NULL THEN
        FOR target IN
            SELECT c.column_name
            FROM information_schema.columns c
            WHERE c.table_schema = 'public'
              AND c.table_name = 'ai_feedbacks'
              AND c.column_name IN ('user_id', 'item_id')
              AND c.data_type = 'bigint'
            ORDER BY c.column_name
        LOOP
            EXECUTE format('SELECT max(%I) FROM public.ai_feedbacks', target.column_name) INTO max_val;
            IF max_val IS NOT NULL AND max_val > 2147483647 THEN
                RAISE EXCEPTION '拒绝回滚 ai_feedbacks.%：存在超过 int4 上限的值 %，回滚会截断用户/条目 ID', target.column_name, max_val;
            END IF;
            EXECUTE format('ALTER TABLE public.ai_feedbacks ALTER COLUMN %I TYPE integer', target.column_name);
            RAISE NOTICE 'ai_feedbacks.% reverted bigint -> integer', target.column_name;
        END LOOP;
    ELSE
        RAISE NOTICE 'skip ai_feedbacks revert: table missing';
    END IF;

    IF to_regclass('public.vectors') IS NOT NULL THEN
        FOR target IN
            SELECT c.column_name
            FROM information_schema.columns c
            WHERE c.table_schema = 'public'
              AND c.table_name = 'vectors'
              AND c.column_name IN ('object_id')
              AND c.data_type = 'bigint'
            ORDER BY c.column_name
        LOOP
            EXECUTE format('SELECT max(%I) FROM public.vectors', target.column_name) INTO max_val;
            IF max_val IS NOT NULL AND max_val > 2147483647 THEN
                RAISE EXCEPTION '拒绝回滚 vectors.%：存在超过 int4 上限的值 %，回滚会截断对象 ID', target.column_name, max_val;
            END IF;
            EXECUTE format('ALTER TABLE public.vectors ALTER COLUMN %I TYPE integer', target.column_name);
            RAISE NOTICE 'vectors.% reverted bigint -> integer', target.column_name;
        END LOOP;
    ELSE
        RAISE NOTICE 'skip vectors revert: table missing';
    END IF;

    RAISE NOTICE '注意：tenant_id 的 int8 统一未回滚（回到 int4 是回归，不是修复）';
END $$;
