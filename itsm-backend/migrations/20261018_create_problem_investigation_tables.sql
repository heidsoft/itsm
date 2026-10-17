-- 20261018_create_problem_investigation_tables.sql
--
-- 补建「问题调查」子系统的 4 张关联表。
--
-- 背景（2026-10-18 功能点验证缺陷 P0-3）：
--   service/problem_investigation_repository.go 用手写 SQL 访问下列 4 张表，
--   文件头注释也写明它们是「无 Ent schema 的关联表」，应由迁移创建。
--   但 migrations/ 与 migration/ 全量检索均无任何语句创建它们，于是：
--     - 全新环境（bootstrap 全量迁移）→ 表不存在；
--     - 现网 itsm_prod 同样不存在；
--     - POST /problem-investigation/root-cause-analysis 必然 500：
--       pq: relation "problem_root_cause_analyses" does not exist (42P01)
--
--   注意：库中已存在的 root_cause_analyses 是**另一套 schema**
--   （ticket_id / root_causes jsonb / analysis_summary，面向工单），
--   与本节 problem_id 语义不兼容，不可复用。
--
-- 租户隔离口径：
--   这 4 张表**不加 tenant_id 列**。仓库层所有查询都通过
--   JOIN problems p ON <fk> = p.id WHERE p.tenant_id = $N 完成隔离
--   （见 repository 中每一条 SELECT/UPDATE/DELETE）。因此也不启用 RLS——
--   没有租户列可做策略键，启用只会制造「策略存在但永不命中」的假象。
--   隔离正确性依赖 problems.tenant_id，这一点由既有 problems 表承担。
--
-- 幂等：全部 IF NOT EXISTS，可重复执行。

-- ------------------------------------------------------------
-- 1. problem_investigations：一个问题一条调查记录（服务层已做 1:1 前置校验，
--    这里用 UNIQUE 把 check-then-insert 的竞态一并关掉）
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS problem_investigations (
    id                        BIGSERIAL PRIMARY KEY,
    problem_id                BIGINT NOT NULL REFERENCES problems (id) ON DELETE CASCADE,
    investigator_id           BIGINT NOT NULL,
    status                    VARCHAR(40) NOT NULL DEFAULT 'in_progress',
    start_date                TIMESTAMPTZ NULL,
    estimated_completion_date TIMESTAMPTZ NULL,
    actual_completion_date    TIMESTAMPTZ NULL,
    investigation_summary     TEXT NULL,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT problem_investigations_problem_id_key UNIQUE (problem_id)
);

CREATE INDEX IF NOT EXISTS problem_investigations_investigator_idx
    ON problem_investigations (investigator_id);

-- ------------------------------------------------------------
-- 2. problem_investigation_steps：调查步骤（按 step_number 排序展示）
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS problem_investigation_steps (
    id               BIGSERIAL PRIMARY KEY,
    investigation_id BIGINT NOT NULL REFERENCES problem_investigations (id) ON DELETE CASCADE,
    step_number      INTEGER NOT NULL,
    step_title       VARCHAR(255) NOT NULL,
    step_description TEXT NULL,
    status           VARCHAR(40) NOT NULL DEFAULT 'pending',
    assigned_to      BIGINT NULL,
    start_date       TIMESTAMPTZ NULL,
    completion_date  TIMESTAMPTZ NULL,
    notes            TEXT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT problem_investigation_steps_investigation_step_key
        UNIQUE (investigation_id, step_number)
);

CREATE INDEX IF NOT EXISTS problem_investigation_steps_assigned_idx
    ON problem_investigation_steps (assigned_to);

-- ------------------------------------------------------------
-- 3. problem_root_cause_analyses：一个问题一条根因分析
--    （CreateRootCauseAnalysis 已做「已存在则拒绝」前置校验，UNIQUE 收口竞态）
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS problem_root_cause_analyses (
    id                     BIGSERIAL PRIMARY KEY,
    problem_id             BIGINT NOT NULL REFERENCES problems (id) ON DELETE CASCADE,
    analyst_id             BIGINT NOT NULL,
    analysis_method        VARCHAR(100) NOT NULL,
    root_cause_description TEXT NOT NULL,
    contributing_factors   TEXT NULL,
    evidence               TEXT NULL,
    -- 0..1 置信度；服务层按 <0.3 / <0.7 / 其余 映射 low/medium/high
    confidence_level       DOUBLE PRECISION NULL,
    analysis_date          TIMESTAMPTZ NULL,
    reviewed_by            BIGINT NULL,
    review_date            TIMESTAMPTZ NULL,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT problem_root_cause_analyses_problem_id_key UNIQUE (problem_id)
);

CREATE INDEX IF NOT EXISTS problem_root_cause_analyses_analyst_idx
    ON problem_root_cause_analyses (analyst_id);

-- ------------------------------------------------------------
-- 4. problem_solutions：一个问题多条解决方案
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS problem_solutions (
    id                     BIGSERIAL PRIMARY KEY,
    problem_id             BIGINT NOT NULL REFERENCES problems (id) ON DELETE CASCADE,
    solution_type          VARCHAR(50) NOT NULL,
    solution_description   TEXT NOT NULL,
    proposed_by            BIGINT NOT NULL,
    proposed_date          TIMESTAMPTZ NULL,
    status                 VARCHAR(40) NOT NULL DEFAULT 'proposed',
    priority               VARCHAR(20) NOT NULL DEFAULT 'medium',
    estimated_effort_hours INTEGER NULL,
    estimated_cost         DOUBLE PRECISION NULL,
    risk_assessment        TEXT NULL,
    approval_status        VARCHAR(40) NOT NULL DEFAULT 'pending',
    approved_by            BIGINT NULL,
    approval_date          TIMESTAMPTZ NULL,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS problem_solutions_problem_idx
    ON problem_solutions (problem_id);
