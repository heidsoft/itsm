-- =============================================================================
-- RLS R1 端到端集成测试脚本（不依赖 Go 客户端）
--
-- 目的：在 dev PostgreSQL 17 内验证 rls package 的运行时行为等价物：
--   1. AcquireConn 语义（SET SESSION app.current_tenant）
--   2. ReleaseConn 语义（DISCARD ALL 清理 session 变量）
--   3. 池复用场景下变量不泄漏
--   4. NULLIF-safe policy 处理 RESET / 未初始化 / DISCARD ALL 三种边界
--   5. 跨租户 SELECT / INSERT / UPDATE / DELETE 全线拦截
--
-- 运行方式：
--   docker cp <this_file> itsm-postgres-dev:/tmp/
--   docker exec itsm-postgres-dev psql -U itsm_user -d itsm -f /tmp/rls_r1_e2e.sql
--
-- 前置：migrations/001_roles.sql + 002_pilot_policies.sql 已执行
-- =============================================================================

-- 环境准备
\set ON_ERROR_STOP off
\echo === 环境探测 ===
SELECT current_user AS superuser_check, session_user;
SELECT tablename, rowsecurity FROM pg_tables WHERE tablename = 'changes';
SELECT policyname FROM pg_policies WHERE tablename = 'changes';

-- =====================================================================
-- 场景 1：切换到应用角色，验证基础隔离
-- =====================================================================
\echo
\echo === 场景 1：itsm_app 角色 + SET tenant=1 ===
SET ROLE itsm_app;
SET SESSION app.current_tenant = 1;

SELECT COUNT(*) AS visible_tenant_1 FROM changes;
-- 预期：> 0（假设 dev 库 tenant=1 已有数据）

\echo
\echo === 场景 2：切换到 tenant=999 应为 0 ===
SET SESSION app.current_tenant = 999;
SELECT COUNT(*) AS visible_tenant_999 FROM changes;
-- 预期：0

\echo
\echo === 场景 3：RESET 后应为 0（NULLIF 保护）===
RESET app.current_tenant;
SELECT COUNT(*) AS visible_after_reset FROM changes;
-- 预期：0，且不报 "invalid input syntax for type bigint" 错误

\echo
\echo === 场景 4：DISCARD ALL 后应为 0（模拟连接归还池后被复用）===
DISCARD ALL;
SET ROLE itsm_app;  -- DISCARD 会 RESET ROLE
SELECT COUNT(*) AS visible_after_discard FROM changes;
-- 预期：0

\echo
\echo === 场景 5：WITH CHECK 拦截跨租户 INSERT ===
SET SESSION app.current_tenant = 1;
INSERT INTO changes (title, description, type, priority, status, risk_level, tenant_id, created_at, updated_at, created_by)
VALUES ('rls_r1_probe_cross_tenant', 'attack', 'normal', 'low', 'draft', 'low', 2, NOW(), NOW(), 1);
-- 预期：ERROR: new row violates row-level security policy

\echo
\echo === 场景 6：WITH CHECK 允许同租户 INSERT ===
INSERT INTO changes (title, description, type, priority, status, risk_level, tenant_id, created_at, updated_at, created_by)
VALUES ('rls_r1_probe_same_tenant', 'legit', 'normal', 'low', 'draft', 'low', 1, NOW(), NOW(), 1);
-- 预期：INSERT 0 1

\echo
\echo === 场景 7：UPDATE 跨租户被 USING 拦截（0 rows affected）===
SET SESSION app.current_tenant = 999;
UPDATE changes SET description = 'HACKED' WHERE title = 'rls_r1_probe_same_tenant';
-- 预期：UPDATE 0

SET SESSION app.current_tenant = 1;
SELECT description FROM changes WHERE title = 'rls_r1_probe_same_tenant';
-- 预期：仍然是 'legit'

\echo
\echo === 场景 8：DELETE 跨租户被 USING 拦截 ===
SET SESSION app.current_tenant = 999;
DELETE FROM changes WHERE title = 'rls_r1_probe_same_tenant';
-- 预期：DELETE 0

SET SESSION app.current_tenant = 1;
SELECT COUNT(*) AS probe_still_present FROM changes WHERE title = 'rls_r1_probe_same_tenant';
-- 预期：1

\echo
\echo === 清理测试探针 ===
DELETE FROM changes WHERE title = 'rls_r1_probe_same_tenant';
SELECT COUNT(*) AS after_cleanup FROM changes WHERE title LIKE 'rls_r1_probe%';

\echo
\echo === 场景 9：BYPASSRLS 角色应看全部（模拟后台任务）===
RESET ROLE;
SET ROLE itsm_admin;
SELECT COUNT(*) AS admin_sees_all FROM changes;
-- 预期：>= 13（tenant=1 全量）

RESET ROLE;

\echo
\echo === 集成测试完成 ===
