-- =============================================================================
-- RLS Migration 002 Rollback
--
-- 卸载 changes 与 vectors 两张表的 RLS policy。用于紧急回滚场景。
-- 卸载后表恢复到无 RLS 状态，业务代码继续依赖手动 TenantIDEQ 过滤。
-- =============================================================================

DROP POLICY IF EXISTS tenant_isolation ON changes;
ALTER TABLE changes NO FORCE ROW LEVEL SECURITY;
ALTER TABLE changes DISABLE ROW LEVEL SECURITY;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema = 'public' AND table_name = 'vectors') THEN
        EXECUTE 'DROP POLICY IF EXISTS tenant_isolation ON vectors';
        EXECUTE 'ALTER TABLE vectors NO FORCE ROW LEVEL SECURITY';
        EXECUTE 'ALTER TABLE vectors DISABLE ROW LEVEL SECURITY';
    END IF;
END $$;
