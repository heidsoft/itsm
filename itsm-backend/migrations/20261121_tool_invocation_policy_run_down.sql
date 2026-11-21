-- 回滚：移除 Agent 审计可回放列（策略版本 + 运行轮次）。
--
-- 影响面：仅丢失「这次放行依据哪版策略」与「这轮 Agent 做过什么」两个
-- 追溯维度；工具调用主体记录（工具名/参数/结果/审批态/权限校验结果）不受影响。
-- 因此回滚是安全的降级，不是数据损坏。

DROP INDEX IF EXISTS tool_invocations_tenant_policy_version_idx;
DROP INDEX IF EXISTS tool_invocations_tenant_run_idx;

ALTER TABLE tool_invocations
  DROP COLUMN IF EXISTS run_id,
  DROP COLUMN IF EXISTS policy_version;
