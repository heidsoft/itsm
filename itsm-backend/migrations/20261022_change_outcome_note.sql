-- 20261022_change_outcome_note.sql
--
-- 变更实施结果留痕：
--   - changes 增补 outcome_note：failed 记失败原因、rolled_back 记回滚原因、
--     completed 记完成说明；
--   - 配套：新增 POST /api/v1/changes/:id/fail（此前状态机允许
--     in_progress/failed → failed，但没有任何 API 入口写这个状态，
--     实施失败只能谎报 completed 或直接 rolled_back，统计与 PIR 全被污染）；
--     失败/回滚必须携带原因，原因写入本列。
--
-- 为什么不是复用 justification / implementation_plan：
--   justification 是"为什么要做这个变更"（事前），outcome_note 是"实际做成了
--   什么样"（事后）。两者同列会让事后结论覆盖事前的变更理由，审计不可回溯。
--
-- 本文件面向「跳过 auto-migrate、只跑版本化迁移」的库，幂等可重复执行。

ALTER TABLE changes
    ADD COLUMN IF NOT EXISTS outcome_note TEXT NULL;

COMMENT ON COLUMN changes.outcome_note IS '实施结果说明（失败原因/回滚原因/完成说明）';
