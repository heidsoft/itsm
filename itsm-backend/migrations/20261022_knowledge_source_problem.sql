-- 20261022_knowledge_source_problem.sql
--
-- 问题→知识沉淀闭环落地：
--   - knowledge_articles 增补 source_problem_id：记录文章由哪个问题沉淀而来；
--   - 配套端点：POST /api/v1/problems/:id/knowledge（沉淀）、
--     GET /api/v1/problems/:id/knowledge（反查）。
--
-- 设计取舍：用裸 ID 而非外键/ent edge 是刻意的。
-- 沉淀后的知识是独立资产，问题被删除不应级联删知识；
-- 引用完整性（沉淀与反查时校验问题存在且同租户）由业务层保证。
--
-- 本文件面向「跳过 auto-migrate、只跑版本化迁移」的库，全部幂等可重复执行。

-- ---------- 1. knowledge_articles 增补来源字段 ----------

ALTER TABLE knowledge_articles
    ADD COLUMN IF NOT EXISTS source_problem_id INT NULL;

-- ---------- 2. 反查索引 ----------

-- 问题详情页「沉淀了哪些知识」的查询路径：按租户 + 来源问题过滤。
CREATE INDEX IF NOT EXISTS idx_knowledge_articles_source_problem
    ON knowledge_articles (tenant_id, source_problem_id);
