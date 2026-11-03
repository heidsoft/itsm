-- 20261031_rls_roles_bootstrap_down.sql
--
-- 回滚 20261031_rls_roles_bootstrap.sql：撤销授权，并在安全时删除角色。
--
-- 角色是**集群级**对象，不是库级对象。直接 DROP 可能影响共用同一集群的
-- 其他数据库，或因为角色仍拥有对象而失败。因此本脚本：
--   1. 先撤销本迁移授予的 schema / 表 / 序列 / 默认权限；
--   2. 再尝试 DROP ROLE，失败时如实 WARNING 并保留角色，交由运维判断。
--
-- 幂等：角色不存在、授权不存在时跳过。

DO $$
DECLARE
    target record;
BEGIN
    FOR target IN
        SELECT v.rolname
        FROM (VALUES ('itsm_app'), ('itsm_admin')) AS v(rolname)
        WHERE to_regrole(v.rolname) IS NOT NULL
    LOOP
        BEGIN
            EXECUTE format('REVOKE ALL PRIVILEGES ON ALL TABLES    IN SCHEMA public FROM %I', target.rolname);
            EXECUTE format('REVOKE ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public FROM %I', target.rolname);
            EXECUTE format('REVOKE ALL PRIVILEGES ON SCHEMA public FROM %I', target.rolname);
            EXECUTE format('ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE ALL ON TABLES    FROM %I', target.rolname);
            EXECUTE format('ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE ALL ON SEQUENCES FROM %I', target.rolname);
            RAISE NOTICE 'revoked public schema privileges from %', target.rolname;
        EXCEPTION WHEN insufficient_privilege THEN
            RAISE WARNING 'cannot revoke privileges from % (needs ownership): %', target.rolname, SQLERRM;
        END;

        BEGIN
            EXECUTE format('DROP ROLE %I', target.rolname);
            RAISE NOTICE 'dropped role %', target.rolname;
        EXCEPTION
            WHEN dependent_objects_still_exist THEN
                RAISE WARNING 'role % still owns objects or has role memberships; left in place (resolve manually, do not CASCADE)', target.rolname;
            WHEN insufficient_privilege THEN
                RAISE WARNING 'cannot drop role %: %', target.rolname, SQLERRM;
        END;
    END LOOP;

    RAISE NOTICE '注意：回滚角色不等于回滚 RLS 策略；策略仍由 009/019/20261028/20261031 维护';
END $$;
