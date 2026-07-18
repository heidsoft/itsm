-- =============================================================================
-- RLS Migration 001 — Roles Bootstrap
--
-- 目的：为 Row-Level Security 建立最小权限模型。
--
-- 角色约定：
--   itsm_app   ← 应用连接使用（普通角色，policy 生效）
--   itsm_admin ← 迁移/后台任务/MSP 运维使用（BYPASSRLS，policy 不生效）
--
-- 执行方式：
--   psql -U <superuser> -d itsm -f 001_roles.sql
--
-- 幂等性：使用 DO 块 + pg_roles 探测，重复执行安全。
-- 回滚：见 001_roles_rollback.sql（本目录）
-- =============================================================================

DO $$
BEGIN
    -- ------------------------------------------------------------------------
    -- 1. 应用角色（普通角色，受 RLS 约束）
    -- ------------------------------------------------------------------------
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'itsm_app') THEN
        CREATE ROLE itsm_app LOGIN PASSWORD 'REPLACE_IN_PRODUCTION';
        RAISE NOTICE 'created role itsm_app';
    ELSE
        RAISE NOTICE 'role itsm_app already exists, skipping';
    END IF;

    -- ------------------------------------------------------------------------
    -- 2. 管理角色（BYPASSRLS，用于迁移/后台任务）
    -- ------------------------------------------------------------------------
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'itsm_admin') THEN
        CREATE ROLE itsm_admin LOGIN BYPASSRLS PASSWORD 'REPLACE_IN_PRODUCTION';
        RAISE NOTICE 'created role itsm_admin';
    ELSE
        -- 已存在则确保有 BYPASSRLS
        ALTER ROLE itsm_admin BYPASSRLS;
        RAISE NOTICE 'role itsm_admin already exists, ensured BYPASSRLS';
    END IF;
END $$;

-- ---------------------------------------------------------------------------
-- 3. 权限：itsm_app 需要业务表的 CRUD，但不能改 schema。
--    itsm_admin 拥有全部权限（用于 migration + 应急运维）。
-- ---------------------------------------------------------------------------
GRANT USAGE ON SCHEMA public TO itsm_app, itsm_admin;

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES    IN SCHEMA public TO itsm_app;
GRANT USAGE, SELECT                  ON ALL SEQUENCES IN SCHEMA public TO itsm_app;

GRANT ALL PRIVILEGES ON ALL TABLES    IN SCHEMA public TO itsm_admin;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO itsm_admin;

-- 未来新建的表也自动授权
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES    TO itsm_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT USAGE, SELECT                  ON SEQUENCES TO itsm_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT ALL PRIVILEGES ON TABLES    TO itsm_admin;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT ALL PRIVILEGES ON SEQUENCES TO itsm_admin;

-- ---------------------------------------------------------------------------
-- 4. GUC 变量登记（PostgreSQL 15+ 语法友好；14 也可用）
--    这一步不是必需，policy 引用未登记的自定义变量也能工作，但登记后
--    pg_settings 中可见，便于运维查询。
-- ---------------------------------------------------------------------------
-- 无需 DDL，`SET SESSION app.current_tenant = ...` 直接可用。
