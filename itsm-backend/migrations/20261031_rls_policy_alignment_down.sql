-- 20261031_rls_policy_alignment_down.sql
--
-- 回滚 20261031_rls_policy_alignment.sql —— **刻意只做部分回滚**。
--
-- 为什么不把 tenant_isolation_<table> 全删掉：
--   这些表已经 ENABLE ROW LEVEL SECURITY。删掉策略后，在 enforce 模式下
--   它们对非 owner 恒返回 0 行（fail closed），业务会大面积读不到数据。
--   "删策略"不是回滚，是把一个安全缺陷换成一个可用性故障。
--
-- 本脚本只恢复 knowledge_article_reviews 的历史形态：这是本迁移唯一
-- 改变了「策略名 + 谓词语义」的表。恢复它会重新引入 app.tenant_id 与
-- 驱动注入的 app.current_tenant 不一致的问题（enforce 下恒 0 行），
-- 因此只应在「确认新策略导致该表业务异常、且需要先回到旧行为排查」时使用，
-- 并在排查结束后重新执行正向迁移。
--
-- 幂等：表/策略不存在则跳过。

DO $$
BEGIN
    IF to_regclass('public.knowledge_article_reviews') IS NULL THEN
        RAISE NOTICE 'skip knowledge_article_reviews rollback: table missing';
        RETURN;
    END IF;

    DROP POLICY IF EXISTS tenant_isolation_knowledge_article_reviews ON knowledge_article_reviews;

    IF NOT EXISTS (
        SELECT 1 FROM pg_policies
        WHERE schemaname = 'public' AND tablename = 'knowledge_article_reviews'
    ) THEN
        CREATE POLICY knowledge_article_reviews_tenant_policy ON knowledge_article_reviews
            USING (tenant_id = current_setting('app.tenant_id', TRUE)::bigint);
        RAISE NOTICE 'restored historical knowledge_article_reviews policy (app.tenant_id — legacy, enforce 下恒 0 行)';
    ELSE
        RAISE NOTICE 'knowledge_article_reviews already has a policy, leaving it as-is';
    END IF;
END $$;

DO $$
BEGIN
    RAISE NOTICE '注意：其它表的 tenant_isolation_<table> 策略刻意保留（删除会让 RLS 表恒 0 行，不是回滚）';
END $$;
