-- =============================================================================
-- RLS 角色引导回滚（已迁移至版本化迁移流）
--
-- 本文件不再包含可执行 DDL。回滚由
-- `itsm-backend/migrations/20261031_rls_roles_bootstrap_down.sql` 负责：
-- 先撤销 public schema 授权与默认权限，再在无依赖时删除角色，失败则如实告警。
-- =============================================================================

DO $$
BEGIN
    RAISE NOTICE '001_roles_rollback.sql is retired: use migrations/20261031_rls_roles_bootstrap_down.sql';
END $$;
