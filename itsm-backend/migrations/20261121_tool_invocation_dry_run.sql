-- Agent 副作用预演（dry-run）落地所需的列与索引（2026-11-21）。
--
-- 背景：dry_run 列自 tool_invocations 建表起就存在（由 ent Schema.Create 生成），
-- 但长期**零消费方**——实体无字段、仓储不映射、服务不写入。本批把它接成真实能力
-- （POST /agent/tools/dry-run + POST /agent/tools/invocations/:id/review）后，
-- 需要保证「所有部署形态都真的有这一列」。
--
-- 为什么必须补这条迁移（而不是依赖 ent 自动建列）：
--   生产与 SaaS 的 backend/worker 都以 ITSM_AUTO_MIGRATE=false 运行
--   （docker-compose.prod.yml:206、deploy/k8s/01-configmap.yaml:10），
--   仅 bootstrap job 开启。凡是「表在建列之前就已存在」的实例
--   （例如从更早版本升级、或表由历史迁移手工建过），
--   在 AUTO_MIGRATE=false 下永远拿不到这一列——写入 dry_run 会直接报列不存在。
--
-- 幂等（IF NOT EXISTS），可安全重放；已有该列的实例执行后无变化。

ALTER TABLE tool_invocations
  ADD COLUMN IF NOT EXISTS dry_run BOOLEAN NOT NULL DEFAULT false;

-- 审批待办的查询路径：tenant_id + dry_run=false + approval_state IN (...)。
--
-- 为什么需要索引：此前只有 idx_tool_invocations_tenant（仅 tenant_id），
-- 待办查询在租户内过滤 dry_run 与审批态是纯扫。审批人每次刷新都走这条路径，
-- 而记录量随时间线性增长（每次工具调用都落一行，含只读）。
-- 部分索引只收 dry_run=false：预演行被排除在读面之外，索引里不需要它们。
CREATE INDEX IF NOT EXISTS tool_invocations_tenant_queue_idx
  ON tool_invocations (tenant_id, approval_state)
  WHERE dry_run = false;

COMMENT ON COLUMN tool_invocations.dry_run IS
  'true=预演（跑完判定链但不执行、不建待审批行）；false=真实调用。预演行 approval_state 恒为 none 并被审批待办排除';
