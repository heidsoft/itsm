-- 20261028_rls_core_tables.sql
--
-- 生产库 RLS 纵深防御扩面：把 5 张事务型主表补进 RLS 策略姿态。
-- 对应审计：output/prod-db-schema-audit-2026-10-26.md §3-P2-1 / §5 第 5 步
-- （175 张表只有 21 张启用 RLS，且 tickets/changes/incidents/problems/
--  configuration_items 五张最核心的事务表全部不在列）。
--
-- 行为定位：纯结构护栏，零运行时行为变化。
--   应用连接以 superuser itsm 执行（superuser 无条件绕过 RLS，FORCE 也拦不住），
--   因此本迁移不改任何读写结果；当后续按 database/rls/ 的三档方案
--   （RLS_MODE=enforce + itsm_app 非 owner 角色）真正启用 enforcement 时，
--   这批策略即时生效，无需再写 DDL。既有 21 张表（009/019 批次）就是这一
--   姿态：ENABLE + 非 FORCE + helper 谓词。
--
-- 谓词口径与既有 21 张表完全一致：
--   policy 名 tenant_isolation_<table>；
--   谓词 tenant_id = get_current_tenant_id()（helper 已由 019 批次对齐到
--   app.current_tenant —— 即 database/rls/driver.go 注入的会话变量；
--   未 SET / RESET / 非法值 => NULL fail closed）。
--
-- 注意：changes 在 002_pilot_policies.sql（R1 试点）里走 FORCE 试点形态，
-- 但该试点未应用于本库（prod 无该策略）；本迁移按 21 张表的统一姿态补齐，
-- 不引入 FORCE——应用侧仍走 per-connection SET 的 driver 机制，FORCE 语义
-- 留给 itsm_app 角色切换时一并决策。
--
-- 本文件幂等可重复执行；DDL 均可在事务块内执行（无 CONCURRENTLY / 数据回填）。

-- ============================================================================
-- 0. 前置守卫：helper 只由权威迁移流（migration/migrations.go 的
--    009_enable_rls_tenant_isolation，对齐见 019）声明，本文件不重复定义，
--    避免 helper 出现两份可漂移的来源。
-- ============================================================================
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'get_current_tenant_id') THEN
        RAISE EXCEPTION 'missing helper get_current_tenant_id(): replay the canonical migration stream (009) before this file';
    END IF;
END $$;

-- ============================================================================
-- 1. tickets —— 全库最热写入表
-- ============================================================================
ALTER TABLE tickets ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_tickets ON tickets;
CREATE POLICY tenant_isolation_tickets ON tickets
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

-- ============================================================================
-- 2. incidents
-- ============================================================================
ALTER TABLE incidents ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_incidents ON incidents;
CREATE POLICY tenant_isolation_incidents ON incidents
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

-- ============================================================================
-- 3. problems
-- ============================================================================
ALTER TABLE problems ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_problems ON problems;
CREATE POLICY tenant_isolation_problems ON problems
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

-- ============================================================================
-- 4. changes —— 事务主表中唯一被 002 试点触碰过、但策略未落本库的一张
-- ============================================================================
ALTER TABLE changes ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_changes ON changes;
CREATE POLICY tenant_isolation_changes ON changes
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

-- ============================================================================
-- 5. configuration_items —— CMDB 主体
-- ============================================================================
ALTER TABLE configuration_items ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_configuration_items ON configuration_items;
CREATE POLICY tenant_isolation_configuration_items ON configuration_items
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

COMMENT ON POLICY tenant_isolation_tickets ON tickets IS
    'tenant RLS scaffold; active under RLS enforcement rollout (helper reads app.current_tenant, 019-aligned)';
COMMENT ON POLICY tenant_isolation_incidents ON incidents IS
    'tenant RLS scaffold; active under RLS enforcement rollout (helper reads app.current_tenant, 019-aligned)';
COMMENT ON POLICY tenant_isolation_problems ON problems IS
    'tenant RLS scaffold; active under RLS enforcement rollout (helper reads app.current_tenant, 019-aligned)';
COMMENT ON POLICY tenant_isolation_changes ON changes IS
    'tenant RLS scaffold; active under RLS enforcement rollout (helper reads app.current_tenant, 019-aligned)';
COMMENT ON POLICY tenant_isolation_configuration_items ON configuration_items IS
    'tenant RLS scaffold; active under RLS enforcement rollout (helper reads app.current_tenant, 019-aligned)';
