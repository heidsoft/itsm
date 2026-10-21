-- 20261021_provisioning_task_recovery.sql
--
-- 交付任务失败恢复能力（服务请求域 Pilot → GA 的阻断项「交付失败即终态」）落地。
--
-- 背景（2026-10-21）：
--   交付任务（provisioning_tasks）此前只有 status/error_message 两个字段：
--     · 无法知道重试了几次，也无法阻止无限重试；
--     · provider 永久性故障（凭据吊销、配额封禁、地域下线）时自动重试永远不可能
--       成功，而 failed 是终态——交付链路只能靠改数据库收场；
--     · 资源已在平台外建好时，系统仍显示「失败」，与实际不符。
--
--   本次补齐两件事：尝试计数（attempt_count / last_attempt_at）与人工接管留痕
--   （resolved_by_user_id / resolution_note）。
--
-- 兼容性：
--   四个字段全部可空或带默认值，存量行无需回填——存量 failed 任务
--   attempt_count=0，因此仍可重试；存量 running 任务 last_attempt_at 为 NULL，
--   会被执行层的租约判定视为「无人持有」并允许重新领取（这是有意的：
--   升级前崩溃留下的僵尸任务必须能自愈）。
--
-- 幂等：IF NOT EXISTS，可重复执行。配套 ent schema：ent/schema/provisioningtask.go。

ALTER TABLE provisioning_tasks
    ADD COLUMN IF NOT EXISTS attempt_count INTEGER NOT NULL DEFAULT 0;

ALTER TABLE provisioning_tasks
    ADD COLUMN IF NOT EXISTS last_attempt_at TIMESTAMPTZ;

ALTER TABLE provisioning_tasks
    ADD COLUMN IF NOT EXISTS resolved_by_user_id BIGINT;

ALTER TABLE provisioning_tasks
    ADD COLUMN IF NOT EXISTS resolution_note TEXT;

-- 「卡住的交付」巡检：按租户找出 running 且租约过期的任务。
CREATE INDEX IF NOT EXISTS provisioning_tasks_tenant_status_attempt_idx
    ON provisioning_tasks (tenant_id, status, last_attempt_at);

-- 约束：attempt_count 不得为负（ent schema 已声明 Min(0)，此处对裸 SQL 写入兜底）。
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'provisioning_tasks_attempt_count_non_negative'
    ) THEN
        ALTER TABLE provisioning_tasks
            ADD CONSTRAINT provisioning_tasks_attempt_count_non_negative
            CHECK (attempt_count >= 0);
    END IF;
END $$;
