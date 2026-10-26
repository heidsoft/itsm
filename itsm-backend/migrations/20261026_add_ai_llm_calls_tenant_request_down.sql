-- 20261026_add_ai_llm_calls_tenant_request_down.sql
--
-- 回滚前写入的 tenant_id/request_id 数据随列一起丢弃；
-- 读路径（按租户聚合）在回滚后应同步回退到平台口径，见 service/ai_telemetry.go。

DROP INDEX IF EXISTS idx_ai_llm_calls_tenant_created;
ALTER TABLE ai_llm_calls DROP COLUMN IF EXISTS request_id;
ALTER TABLE ai_llm_calls DROP COLUMN IF EXISTS tenant_id;
