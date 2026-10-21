-- 20261021_create_problem_comments.sql
--
-- 补建 problem_comments 表：问题域 GA 阻断项「评论功能尚未实现」的落地存储。
--
-- 背景（2026-10-21）：
--   GET/POST /api/v1/problems/:id/comments 两个端点此前恒返回 503/5003
--   （handlers/problem/handler.go 的显式 unready 桩），问题处理团队没有地方留
--   处理纪要，只能把结论塞进 root_cause/workaround 字段，详情页也看不到协作过程。
--
--   本表与 ticket_comments 同构但独立：problem_id 与 ticket_id 是不同聚合根，
--   复用同一张表会让「问题的处理纪要」与「工单的客户沟通」共享可见性规则，
--   而两者读者角色不同（前者是处理团队，后者含报障人）。
--
-- 租户隔离口径：
--   本表直接带 tenant_id 列，查询一律 tenant_id + problem_id 双谓词，
--   并通过 JOIN problems 校验问题归属（跨租户 problem_id 不得命中评论）。
--
-- 幂等：IF NOT EXISTS，可重复执行。配套 ent schema：
--   ent/schema/problem_comment.go（Problem.comments / User.problem_comments 双边）。

CREATE TABLE IF NOT EXISTS problem_comments (
    id          BIGSERIAL PRIMARY KEY,
    problem_id  BIGINT NOT NULL REFERENCES problems (id) ON DELETE CASCADE,
    user_id     BIGINT NOT NULL,
    content     TEXT NOT NULL,
    is_internal BOOLEAN NOT NULL DEFAULT FALSE,
    mentions    JSONB NOT NULL DEFAULT '[]'::jsonb,
    attachments JSONB NOT NULL DEFAULT '[]'::jsonb,
    tenant_id   BIGINT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 列表查询：同一问题的评论按时间正序分页
CREATE INDEX IF NOT EXISTS problem_comments_problem_created_idx
    ON problem_comments (tenant_id, problem_id, created_at);

-- 评论人维度（「我参与过的问题」类查询）
CREATE INDEX IF NOT EXISTS problem_comments_user_idx
    ON problem_comments (tenant_id, user_id);
