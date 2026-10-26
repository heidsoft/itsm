-- 20261027_prod_schema_guards.sql
--
-- 生产库结构护栏补齐，对应审计：output/prod-db-schema-audit-2026-10-26.md（F1–F5）。
-- 现状：库结构不缺表，缺约束与索引这两层护栏——84/175 表无业务唯一键、
-- 热路径 role_permissions 每请求全表扫（seq_scan 2185 / idx_scan 0）、
-- 94 个外键列无索引、软删 10 张表中仅 changes 有 deleted_at 索引。
-- 当前库数据量极低（≈1658 行），故全部使用 CREATE INDEX CONCURRENTLY（不阻塞读写，
-- 或耗时也可忽略）；存量重复前置检查见各分组注释，全部实测 0 重复。
--
-- 与 output/prod-db-schema-audit-2026-10-26-fixes.sql 的差异（执行前实测修正）：
--   1. assets 无 deleted_at 列（软删仅 10 张表，assets 不在其中），
--      F3 的 uq_assets_tenant_number 不能带 WHERE 子句，改为全量唯一索引；
--   2. 原脚本两处 GROUP BY 常量笔误不影响本文件（前置检查已单独跑过，0 重复）。
--
-- 刻意不做（避免制造新账）：
--   F6 冗余索引清理：这些索引多数由 ent 自动迁移声明（schema.go），
--   手删会在下一次 ent migrate 时被加回来或造成 schema diff 漂移；
--   空表上冗余索引无 IO 成本，收益为零。留给 ent schema 侧统一收口。
--   F7（类型统一 / RLS 扩展 / 迁移账本回填）：见审计报告，属流程项。
--
-- 本文件幂等可重复执行；CONCURRENTLY 不能在事务块内，执行器需逐条下发。

-- ============================================================================
-- F1.【P0】role_permissions：热路径覆盖索引 + 业务唯一键
-- 热路径：middleware/rbac.go WHERE role_id=$1 AND tenant_id=$2，
-- 现状 seq_scan 2185 / idx_scan 0，仅 (id)(tenant_id) 两个索引。
-- ============================================================================
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_rp_tenant_role_perm
    ON role_permissions (tenant_id, role_id) INCLUDE (permission_id);

-- 业务唯一键：阻止同一角色重复授权（实测无重复行）
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS uq_rp_role_permission
    ON role_permissions (role_id, permission_id);

ANALYZE role_permissions;

-- ============================================================================
-- F2.【P0】roles / permissions：code 租户内唯一 + 查询索引
-- role.Code()+TenantID() / permissions.code 授权查询此前只能走全表扫。
-- ============================================================================
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS uq_roles_tenant_code
    ON roles (tenant_id, code);

-- 注：permissions 为引用表（全局读、course 内 code 唯一），部分租户行为：
--     实测 code 无重复（含跨租户）。
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS uq_permissions_tenant_code
    ON permissions (tenant_id, code);

ANALYZE roles;
ANALYZE permissions;

-- ============================================================================
-- F3.【P0/P1】业务编号唯一键（部分唯一索引仅约束未软删行，避免与软删历史冲突）
-- 对照组：tickets/incidents 均已有 (tenant_id, xxx_number)。
-- ============================================================================
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS uq_changes_tenant_number
    ON changes (tenant_id, change_number) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS uq_problems_tenant_number
    ON problems (tenant_id, problem_number) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS uq_releases_tenant_number
    ON releases (tenant_id, release_number);
-- assets 无 deleted_at（软删表集合不含 assets），故为全量唯一索引
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS uq_assets_tenant_number
    ON assets (tenant_id, asset_number);
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS uq_system_configs_tenant_key
    ON system_configs (tenant_id, key) WHERE deleted_at IS NULL;

-- ============================================================================
-- F4.【P1】工单/事件/问题子表的外键列索引
-- 删父行时 FK NO ACTION 校验此前需全表扫子表；子表侧 JOIN 同样退化。
-- ============================================================================
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tkt_att_ticket    ON ticket_attachments (ticket_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tkt_cmt_ticket    ON ticket_comments (ticket_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tkt_appr_ticket   ON ticket_approvals (ticket_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tkt_appr_approver ON ticket_approvals (approver_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tkt_wfr_ticket    ON ticket_workflow_records (ticket_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tkt_notif_ticket  ON ticket_notifications (ticket_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tkt_notif_user    ON ticket_notifications (user_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tkt_cc_ticket     ON ticket_ccs (ticket_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tkt_tag_ticket    ON ticket_tags (ticket_tags);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_inc_evt_incident  ON incident_events (incident_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_inc_metric_inc    ON incident_metrics (incident_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_inc_alert_inc     ON incident_alerts (incident_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_inc_rule_exec_inc ON incident_rule_executions (incident_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_inc_related_inc   ON incident_related_incidents (incident_id);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_prb_cmt_problem   ON problem_comments (problem_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_prb_cmt_user      ON problem_comments (user_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_rca_ticket        ON root_cause_analyses (ticket_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_sla_alert_hist_tk ON sla_alert_histories (ticket_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_feishu_sync_tk    ON feishu_ticket_syncs (ticket_id);

-- ============================================================================
-- F5.【P1】软删索引统一为 (tenant_id) WHERE deleted_at IS NULL
-- 10 张软删表中此前仅 changes 有 deleted_at 部分索引。
-- tickets 是全库最热写入表，此前连 deleted_at 索引都没有。
-- ============================================================================
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tickets_tenant_active
    ON tickets (tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_incidents_tenant_active
    ON incidents (tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_problems_tenant_active
    ON problems (tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_knowledge_articles_active
    ON knowledge_articles (tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_service_requests_active
    ON service_requests (tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_applications_active
    ON applications (tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_departments_active
    ON departments (tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_teams_active
    ON teams (tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_system_configs_active
    ON system_configs (tenant_id) WHERE deleted_at IS NULL;
-- changes 由旧的单列部分索引升级为 (tenant_id) 部分索引：
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_changes_tenant_active
    ON changes (tenant_id) WHERE deleted_at IS NULL;
-- 新索引确认有效后删除旧单列索引（下方验证步骤确认）：
-- DROP INDEX CONCURRENTLY IF EXISTS idx_changes_deleted_at;
