-- =============================================================================
-- RLS 角色引导（已迁移至版本化迁移流）
--
-- 本文件**不再包含可执行 DDL**，只保留一个指针，原因：
--   1. 它此前不在迁移流里（不会被自动执行），却写着硬编码占位口令
--      `REPLACE_IN_PRODUCTION`——照抄执行会得到一个「口令公开且可登录」的
--      角色，比不建角色更危险；
--   2. 角色模型（itsm_app / itsm_admin）现在由
--      `itsm-backend/migrations/20261031_rls_roles_bootstrap.sql` 统一维护，
--      由迁移器自动执行、进入 schema_migrations 账本并参与 checksum 校验。
--      两处维护必然漂移，因此这里只留指针。
--
-- 现行做法（见 docs/operations/rls-rollout.md）：
--   * 迁移创建的是 **NOLOGIN** 角色，不会写入任何口令；
--   * 运维在带外显式执行 `ALTER ROLE itsm_app LOGIN PASSWORD <password>`；
--   * 是否真正启用 enforcement 由启动时的 readiness 探测把关
--     （database/rls/readiness.go），角色存在不等于隔离生效。
--
-- 回滚：itsm-backend/migrations/20261031_rls_roles_bootstrap_down.sql
-- =============================================================================

DO $$
BEGIN
    RAISE NOTICE '001_roles.sql is retired: roles are managed by migrations/20261031_rls_roles_bootstrap.sql';
END $$;
