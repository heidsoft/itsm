-- =============================================================================
-- RLS Migration 001 Rollback
--
-- 撤销 001_roles.sql 建立的角色与权限。仅用于紧急回滚，正常运维不应使用。
-- =============================================================================

REVOKE ALL ON ALL TABLES    IN SCHEMA public FROM itsm_app, itsm_admin;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM itsm_app, itsm_admin;
REVOKE USAGE ON SCHEMA public FROM itsm_app, itsm_admin;

ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES    FROM itsm_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE USAGE, SELECT                  ON SEQUENCES FROM itsm_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE ALL PRIVILEGES ON TABLES    FROM itsm_admin;
ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE ALL PRIVILEGES ON SEQUENCES FROM itsm_admin;

DROP ROLE IF EXISTS itsm_app;
DROP ROLE IF EXISTS itsm_admin;
