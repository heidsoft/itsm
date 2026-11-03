-- 20261031_rls_roles_bootstrap.sql
--
-- RLS enforcement 实路（第 3 段）：把 database/rls/migrations/001_roles.sql 的
-- 角色模型版本化进磁盘迁移流，并去掉其中的硬编码口令。
--
-- 背景：RLS_MODE=enforce 只有在「连接角色不是 superuser、不带 BYPASSRLS、
-- 且不是策略表 owner（或策略 FORCE）」时才真正过滤行。当前默认连接账号是
-- 部署时的 owner/superuser，因此 enforce 一直是**名义开启、实际无效**。
-- database/rls/migrations/001_roles.sql 早就定义了正确的双角色模型，但它
-- 不在迁移流里（不会被自动执行），而且写着一个占位口令——一旦有人照抄执行，就得到一个
-- 「口令公开且可登录」的角色，比不建更糟。
--
-- 本迁移的角色约定：
--   itsm_app   —— 应用请求路径（NOSUPERUSER / NOBYPASSRLS，策略生效）
--   itsm_admin —— 迁移 / 后台任务 / MSP 运维（BYPASSRLS，策略不生效）
--
-- 刻意不做的事：
--   * **不设置口令、不授予 LOGIN**。角色默认 NOLOGIN，无法被连接使用，
--     直到运维显式执行 `ALTER ROLE itsm_app LOGIN PASSWORD <password>`（见
--     docs/operations/rls-rollout.md）。口令永远不进仓库。
--   * **不修改当前连接角色**。迁移只创建/校准这两个角色，不把应用切过去；
--     切换是独立变更窗（改 DB_USER / DB_APP_ROLE_USER 并重启）。
--
-- 权限不足时的姿态：诚实降级。
--   CREATE ROLE / ALTER ROLE ... BYPASSRLS 需要 CREATEROLE 或 superuser。
--   托管 PostgreSQL 上应用账号常常没有这些权限。此时本迁移 RAISE WARNING
--   并继续——角色创建是运维前置条件，不是业务 schema 变更；而 enforce 是否
--   真正生效由启动时的 readiness 探测把关（database/rls/readiness.go），
--   不会因为这里"跳过了"而被误认为已隔离。
--
-- 幂等：重复执行安全。
-- 回滚：见 20261031_rls_roles_bootstrap_down.sql（撤销授权；角色本身按依赖
--       关系决定是否删除，见该文件说明）。

-- ============================================================================
-- 1. 角色本身
-- ============================================================================
DO $$
DECLARE
    created text[] := ARRAY[]::text[];
BEGIN
    -- 1.1 itsm_app：普通角色，受 RLS 约束
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'itsm_app') THEN
        BEGIN
            CREATE ROLE itsm_app NOLOGIN NOSUPERUSER NOBYPASSRLS NOCREATEDB NOCREATEROLE;
            created := created || 'itsm_app'::text;
        EXCEPTION WHEN insufficient_privilege THEN
            RAISE WARNING 'cannot create role itsm_app (needs CREATEROLE/superuser): %', SQLERRM;
        END;
    ELSE
        -- 已存在也要校准：如果历史上被设成 superuser/BYPASSRLS，那 RLS 对它无效。
        BEGIN
            ALTER ROLE itsm_app NOSUPERUSER NOBYPASSRLS NOCREATEDB NOCREATEROLE;
        EXCEPTION WHEN insufficient_privilege THEN
            RAISE WARNING 'cannot verify itsm_app attributes (needs superuser): %', SQLERRM;
        END;
    END IF;

    -- 1.2 itsm_admin：BYPASSRLS，供迁移/后台任务跨租户使用
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'itsm_admin') THEN
        BEGIN
            CREATE ROLE itsm_admin NOLOGIN NOSUPERUSER BYPASSRLS NOCREATEDB NOCREATEROLE;
            created := created || 'itsm_admin'::text;
        EXCEPTION WHEN insufficient_privilege THEN
            RAISE WARNING 'cannot create role itsm_admin with BYPASSRLS (needs superuser): %', SQLERRM;
        END;
    ELSE
        BEGIN
            ALTER ROLE itsm_admin NOSUPERUSER BYPASSRLS NOCREATEDB NOCREATEROLE;
        EXCEPTION WHEN insufficient_privilege THEN
            RAISE WARNING 'cannot ensure itsm_admin BYPASSRLS (needs superuser): %', SQLERRM;
        END;
    END IF;

    IF array_length(created, 1) > 0 THEN
        RAISE NOTICE 'created roles % as NOLOGIN (set LOGIN + password out of band; see docs/operations/rls-rollout.md)', created;
    END IF;
END $$;

-- ============================================================================
-- 2. 授权
--
-- 只对**已存在**的角色授权；角色创建失败时跳过而不是让迁移整体失败。
-- 授权失败同样降级为 WARNING：没有授权的 itsm_app 连不上业务表，
-- 运维会在切换连接角色时立刻发现，而不是被一次迁移阻断。
-- ============================================================================
DO $$
BEGIN
    IF to_regrole('itsm_app') IS NULL THEN
        RAISE NOTICE 'skip itsm_app grants: role missing';
        RETURN;
    END IF;

    BEGIN
        GRANT USAGE ON SCHEMA public TO itsm_app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES    IN SCHEMA public TO itsm_app;
        GRANT USAGE, SELECT                  ON ALL SEQUENCES IN SCHEMA public TO itsm_app;

        -- 未来由当前角色创建的表/序列自动授权。注意：ALTER DEFAULT PRIVILEGES
        -- 只作用于「执行者」未来创建的对象，因此本迁移应以表 owner 身份执行。
        ALTER DEFAULT PRIVILEGES IN SCHEMA public
            GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO itsm_app;
        ALTER DEFAULT PRIVILEGES IN SCHEMA public
            GRANT USAGE, SELECT ON SEQUENCES TO itsm_app;

        RAISE NOTICE 'granted itsm_app CRUD on public schema (existing + future objects)';
    EXCEPTION WHEN insufficient_privilege THEN
        RAISE WARNING 'cannot grant itsm_app privileges (needs table ownership): %', SQLERRM;
    END;
END $$;

DO $$
BEGIN
    IF to_regrole('itsm_admin') IS NULL THEN
        RAISE NOTICE 'skip itsm_admin grants: role missing';
        RETURN;
    END IF;

    BEGIN
        GRANT USAGE ON SCHEMA public TO itsm_admin;
        GRANT ALL PRIVILEGES ON ALL TABLES    IN SCHEMA public TO itsm_admin;
        GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO itsm_admin;

        ALTER DEFAULT PRIVILEGES IN SCHEMA public
            GRANT ALL PRIVILEGES ON TABLES TO itsm_admin;
        ALTER DEFAULT PRIVILEGES IN SCHEMA public
            GRANT ALL PRIVILEGES ON SEQUENCES TO itsm_admin;

        RAISE NOTICE 'granted itsm_admin ALL on public schema (existing + future objects)';
    EXCEPTION WHEN insufficient_privilege THEN
        RAISE WARNING 'cannot grant itsm_admin privileges (needs table ownership): %', SQLERRM;
    END;
END $$;

-- ============================================================================
-- 3. 结果自检：角色存在时其安全属性必须正确。
--
-- 这里刻意用 WARNING 而不是 EXCEPTION：托管库上应用账号可能无权修改角色
-- 属性（ALTER ROLE ... NOSUPERUSER 需要 superuser）。此时让迁移失败会
-- 阻断整个启动，而问题角色此刻还没被使用。真正的闸门是启动时的
-- readiness 探测——一旦把连接切到 itsm_app，superuser/BYPASSRLS 会被
-- 直接拒绝启动（database/rls/readiness.go）。
-- ============================================================================
DO $$
DECLARE
    bad text;
BEGIN
    SELECT string_agg(rolname || '(superuser=' || rolsuper || ',bypassrls=' || rolbypassrls || ')', ', ')
    INTO bad
    FROM pg_roles
    WHERE rolname = 'itsm_app'
      AND (rolsuper OR rolbypassrls);

    IF bad IS NOT NULL THEN
        RAISE WARNING 'itsm_app must be NOSUPERUSER NOBYPASSRLS, got %: RLS would not filter its rows', bad;
    ELSIF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'itsm_app') THEN
        RAISE NOTICE 'RLS roles ok: itsm_app exists with NOSUPERUSER NOBYPASSRLS';
    END IF;
END $$;
