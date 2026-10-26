-- 20261026_add_ai_llm_calls_tenant_request.sql
--
-- LLM 调用按租户归集：ai_llm_calls 增补 tenant_id / request_id。
-- 此前该表没有任何租户维度（tenant_guard 按「LLM 网关级可观测性，与租户无关」豁免），
-- 后果是 AI 辅助的成本与延迟只有全平台口径：租户看不到自己的 token 消耗，
-- 平台无法按租户做计量与配额告警；/ai/metrics 还把平台平均延迟混在单租户指标里，
-- 属于跨租户归因。
--   - 写路径：service.LLMObserver 在网关每次调用时从 ctx 回填归属
--     （common/tenantctx + middleware.RequestIDFromContext）。
--   - 存量行无法回填：写入时没有任何可 join 的关联字段。tenant_id=0 是
--     「系统调用/无法归属的历史行」哨兵值，按租户聚合的读路径一律排除，
--     租户无数据时如实报空窗口，不冒充完整。
--
-- 本文件面向「跳过 auto-migrate、只跑版本化迁移」的库，幂等可重复执行。

ALTER TABLE ai_llm_calls
    ADD COLUMN IF NOT EXISTS tenant_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS request_id TEXT NOT NULL DEFAULT '';

COMMENT ON COLUMN ai_llm_calls.tenant_id IS '租户归属；0=系统调用或无法归属的历史行（哨兵值，按租户聚合时排除）';
COMMENT ON COLUMN ai_llm_calls.request_id IS '写入时所在请求 ID（X-Request-Id），用于把一次请求的多次模型调用串起来';

-- 按租户聚合与窗口扫描的读路径都以 (tenant_id, created_at) 过滤
CREATE INDEX IF NOT EXISTS idx_ai_llm_calls_tenant_created
    ON ai_llm_calls (tenant_id, created_at);
