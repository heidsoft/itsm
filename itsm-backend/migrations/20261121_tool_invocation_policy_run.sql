-- Agent 审计可回放（2026-11-21）：策略版本 + 运行轮次标识。
--
-- 背景：战略文档 C 段要求「执行证据、策略版本与回放」，而此前
--   - permissions.description 里的策略信封没有版本号，事后无法回答
--     「这次放行依据的是哪一版策略」；
--   - tool_invocations / ai_feedbacks(ai_audit) / auditlogs 三处审计各自可查，
--     但一次 Agent 会话的多次工具调用无法作为整体取出。
--
-- 两列都是加列、可空等价（Default 空串），存量行不需要回填业务值：
--   - policy_version 空串 = 当时没有策略参与判定（human 主体走角色 RBAC），
--     这是真实语义而不是缺失；只有「取不到指纹」才写 'unknown'。
--   - run_id 空串 = 该调用不属于任何可标识的运行轮次（如单发 HTTP 工具调用）。
--
-- 迁移幂等（IF NOT EXISTS），可安全重放。

ALTER TABLE tool_invocations
  ADD COLUMN IF NOT EXISTS policy_version VARCHAR DEFAULT '',
  ADD COLUMN IF NOT EXISTS run_id VARCHAR DEFAULT '';

-- 按轮次取出「这一轮 Agent 做过什么」是审计复盘的唯一入口路径。
-- 部分索引：绝大多数调用没有 run_id（单发 HTTP），全量索引会白白膨胀。
CREATE INDEX IF NOT EXISTS tool_invocations_tenant_run_idx
  ON tool_invocations (tenant_id, run_id)
  WHERE run_id <> '';

-- 「这个权限码历史上按哪个版本放行过」是策略变更影响面的排查路径。
CREATE INDEX IF NOT EXISTS tool_invocations_tenant_policy_version_idx
  ON tool_invocations (tenant_id, policy_version)
  WHERE policy_version <> '';

-- 存量行的策略版本无从追溯（信封当时无版本概念），刻意留空串而非填 'unknown'：
-- 空串表达「历史上没有这个概念」，'unknown' 表达「有这个概念但我们没取到」。
-- 二者混用会让「有多少调用缺策略版本」这个指标失去意义。
COMMENT ON COLUMN tool_invocations.policy_version IS
  '放行判据的策略指纹（决策输入哈希）；空串=无策略参与判定；unknown=未能读取';
COMMENT ON COLUMN tool_invocations.run_id IS
  '运行轮次标识，跨请求串联同一轮 Agent 行为；空串=不属于任何运行轮次';
