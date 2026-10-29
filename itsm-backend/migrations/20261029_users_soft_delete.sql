-- 20261029_users_soft_delete.sql
--
-- users 域「删除/停用语义分离」（审计 P3：删除与停用语义未分离）。
--
-- 现状：users 表没有 deleted_at，管理员「删除用户」走 service/user_service.go
-- DeleteUser 只写 active=false + 吊销存量 token——与「停用用户」
-- ChangeUserStatus(false) 完全同态。后果：
--   ① 被删用户在管理列表以「已停用」出现（无法区分删除与停用）；
--   ② toggle「启用」就能把一个已删除的账号复活；
--   ③ 审计日志上「删除了用户 Z」之后 Z 仍可见，语义无法自洽。
-- 改造（本批代码侧）：
--   删除 = active=false + deleted_at=now（读路径经
--        database.RegisterSoftDeleteInterceptors 对 UserQuery 附加
--        DeletedAtIsNil()，管理列表/搜索/接口与 WithXxx 关联查询统一屏蔽）；
--   停用 = active=false（语义不变，仍显示并可重新启用）。
--
-- 存量数据策略：不回填。库里既有的 active=false 行分不清是历史删除还是停用，
-- 一律继续按「停用」对待（deleted_at 保持 NULL）。若未来确有「批量区分」需求，
-- 由产品另行决策。
--
-- 模式与 20261024_change_soft_delete.sql 一致：
--   部分索引只收 deleted_at IS NULL——读路径每条 UserQuery 都带该过滤，
--   墓碑不进入索引。当前 users 体量有限，此索引满足按「未删用户」的查询形态。
--
-- 幂等可重复执行；ent schema 同步增补（field.Time("deleted_at")），
-- auto-migrate 路径与本迁移等价。

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL;

COMMENT ON COLUMN users.deleted_at IS '删除时间（软删标记，NULL 表示未删除；停用不置此列）';

CREATE INDEX IF NOT EXISTS idx_users_deleted_at
    ON users (deleted_at)
    WHERE deleted_at IS NULL;
