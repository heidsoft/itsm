-- Phase B6（2026-11-17）：ToolInvocation 业务单据强关联列。
-- 替代 AI 审计底账 inputRef 字符串弱关联（格式漂移即断链），
-- 让工具调用记录可直接按 ticket_id / change_id 反查。
-- 两列均可空：对话类工具调用没有业务单据，不强制非空。
-- 迁移幂等（IF NOT EXISTS），可安全重放。

ALTER TABLE tool_invocations
  ADD COLUMN IF NOT EXISTS ticket_id INTEGER,
  ADD COLUMN IF NOT EXISTS change_id INTEGER;

-- 反查索引：按工单/变更追踪 AI 工具调用链（审计与故障复盘的高频路径）。
CREATE INDEX IF NOT EXISTS tool_invocations_tenant_ticket_idx
  ON tool_invocations (tenant_id, ticket_id)
  WHERE ticket_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS tool_invocations_tenant_change_idx
  ON tool_invocations (tenant_id, change_id)
  WHERE change_id IS NOT NULL;
