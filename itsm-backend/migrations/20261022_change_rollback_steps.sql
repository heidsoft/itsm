-- 20261022_change_rollback_steps.sql
--
-- 变更回滚清单：把「回滚计划」从一段自由文本升级为可执行、可逐项确认的清单。
--
-- 背景（2026-10-22）：
--   变更域晋升「可用」的最后一条上线门禁是「失败可回退」（能力契约链路 C）。
--   此前回滚只有两样东西：changes.rollback_plan（TEXT，提交门禁只校验非空）
--   和 POST /changes/:id/rollback（一次状态翻转 + outcome_note 记原因）。
--   也就是说：回滚计划写没写、写了几步、有没有真按顺序执行、每一步是否成功，
--   系统一概不知——审计只能看到「状态从 in_progress 变成 rolled_back」。
--   真实事故里最难回答的问题恰恰是这三件：回滚到第几步了？哪一步没回成？
--   谁确认的？没有逐项事实就无法回答，PIR 的「影响是否已复原」也只能靠猜。
--
-- 为什么是「一行一步」而不是复用 change_rollback_plans.rollback_steps(JSONB)：
--   逐项确认需要每步独立的 status / note / resolved_by / resolved_at，
--   并要能按行做租户谓词更新与并发 CAS。JSONB 数组只能整段覆盖，
--   并发确认会互相踩踏，也拿不到「谁在什么时候确认了第 3 步」这一行级事实。
--   （change_rollback_plans / change_rollback_executions 是 007 迁移预建、
--   至今零 Go 引用的空表，本表不复用它们，理由同上。）
--
-- 租户隔离口径：
--   本表带 tenant_id 列，所有查询/更新一律 (tenant_id, change_id) 双谓词；
--   change_approval_chains 同为该口径（changes 主表才有 RLS 策略）。
--
-- 幂等：IF NOT EXISTS，可重复执行。

CREATE TABLE IF NOT EXISTS change_rollback_steps (
    id          BIGSERIAL PRIMARY KEY,
    change_id   BIGINT NOT NULL REFERENCES changes (id) ON DELETE CASCADE,
    tenant_id   BIGINT NOT NULL,
    step_no     INTEGER NOT NULL,
    description TEXT NOT NULL,
    -- pending=待确认；done=已执行；skipped=不适用（须备注依据）；failed=执行失败（须备注现象）
    status      TEXT NOT NULL DEFAULT 'pending'
                CHECK (status IN ('pending', 'done', 'skipped', 'failed')),
    note        TEXT,
    resolved_by BIGINT,
    resolved_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 同一变更内步骤序号唯一：清单是「有序执行」的，重复序号无法排序与对账。
CREATE UNIQUE INDEX IF NOT EXISTS change_rollback_steps_change_step_uidx
    ON change_rollback_steps (change_id, step_no);

-- 主查询路径：按租户 + 变更读取整张清单（按序号）
CREATE INDEX IF NOT EXISTS change_rollback_steps_tenant_change_idx
    ON change_rollback_steps (tenant_id, change_id, step_no);

COMMENT ON TABLE change_rollback_steps IS '变更回滚清单（一行一步，逐项确认与留痕）';
COMMENT ON COLUMN change_rollback_steps.status IS 'pending/done/skipped/failed；skipped 与 failed 必须带 note';
COMMENT ON COLUMN change_rollback_steps.note IS '执行备注：证据、失败现象或不适用依据';
