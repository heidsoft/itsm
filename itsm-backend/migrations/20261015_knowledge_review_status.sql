-- 20261015_knowledge_review_status.sql
--
-- 知识评审状态机（最小闭环）落地：
--   - knowledge_articles 增补 review_status / reviewer_id / reviewed_at / review_comment；
--   - 新建 knowledge_article_reviews 记录每一次评审动作的审计流水。
--
-- 关键设计：存量数据按 is_published 反向初始化 review_status。
-- 若新字段默认 'draft' 而不迁移存量，已发布的存量文章会在加字段瞬间
-- 被视为「未评审」，导致 RAG 检索大面积掉量（知识库是 AI 的唯一事实源，
-- 掉量会直接反映为 AI 回答质量下降）。
--
-- 本文件面向「跳过 auto-migrate、只跑版本化迁移」的库，全部幂等可重复执行。
-- 行级隔离策略与 20260828_create_alerts.sql 一致。

-- ---------- 1. knowledge_articles 增补评审字段 ----------

ALTER TABLE knowledge_articles
    ADD COLUMN IF NOT EXISTS review_status VARCHAR(20) NOT NULL DEFAULT 'draft';

ALTER TABLE knowledge_articles
    ADD COLUMN IF NOT EXISTS reviewer_id BIGINT NULL;

ALTER TABLE knowledge_articles
    ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMP NULL;

ALTER TABLE knowledge_articles
    ADD COLUMN IF NOT EXISTS review_comment TEXT NULL;

-- 存量数据反向初始化：已发布 → approved（视为历史已上线），未发布 → draft。
-- 这一步是本迁移的核心，不能省：字段默认 'draft'，若不回填，
-- 加字段瞬间所有已发布存量文章都会变成「未评审」，RAG 检索大面积掉量。
-- 幂等：重复执行时已回填的行其 review_status 已不等于 'draft'，不再被改写。
UPDATE knowledge_articles
SET review_status = 'approved'
WHERE review_status = 'draft'
  AND is_published = TRUE;

-- ---------- 2. 评审流水表 ----------

CREATE TABLE IF NOT EXISTS knowledge_article_reviews (
    id BIGSERIAL PRIMARY KEY,
    article_id BIGINT NOT NULL REFERENCES knowledge_articles(id) ON DELETE CASCADE,
    tenant_id BIGINT NOT NULL,
    from_status VARCHAR(20) NOT NULL DEFAULT 'draft',
    to_status VARCHAR(20) NOT NULL,
    action VARCHAR(20) NOT NULL,
    operator_id BIGINT NOT NULL,
    comment TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 查询索引：按文章取评审历史（时间序）与按租户清理
CREATE INDEX IF NOT EXISTS idx_knowledge_article_reviews_article_created
    ON knowledge_article_reviews (article_id, created_at);

CREATE INDEX IF NOT EXISTS idx_knowledge_article_reviews_tenant_article
    ON knowledge_article_reviews (tenant_id, article_id);

-- ---------- 3. 行级隔离（与其他租户表同口径） ----------

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE policyname = 'knowledge_article_reviews_tenant_policy') THEN
        EXECUTE 'ALTER TABLE knowledge_article_reviews ENABLE ROW LEVEL SECURITY';
        EXECUTE 'CREATE POLICY knowledge_article_reviews_tenant_policy ON knowledge_article_reviews '
                'USING (tenant_id = current_setting(''app.tenant_id'', TRUE)::bigint)';
    END IF;
END
$$;
