-- Phase B6 回滚：移除业务单据强关联列及其反查索引。
-- 既有数据仅丢失 ticket_id/change_id 关联信息，不影响工具调用主体记录。

DROP INDEX IF EXISTS tool_invocations_tenant_change_idx;
DROP INDEX IF EXISTS tool_invocations_tenant_ticket_idx;

ALTER TABLE tool_invocations
  DROP COLUMN IF EXISTS change_id,
  DROP COLUMN IF EXISTS ticket_id;
