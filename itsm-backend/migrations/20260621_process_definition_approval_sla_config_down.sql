-- ROLLBACK for: 20260621_process_definition_approval_sla_config.sql
-- Generated: 2026-06-27
-- Description: Removes approval_config and sla_config JSONB columns from process_definitions.
-- WARNING: 会清空 approval_config / sla_config 数据，操作前请备份。
-- PR: #01 (v1.0 GA 拆分批次)

BEGIN;

-- Step 1: 备份（生产环境必做；开发环境可跳过）
CREATE TABLE IF NOT EXISTS process_definitions_bak_20260621 AS
SELECT * FROM process_definitions;

-- Step 2: 移除 sla_config 列
ALTER TABLE process_definitions DROP COLUMN IF EXISTS sla_config;

-- Step 3: 移除 approval_config 列
ALTER TABLE process_definitions DROP COLUMN IF EXISTS approval_config;

-- Step 4: 验证
DO $$
DECLARE
    has_approval boolean;
    has_sla boolean;
BEGIN
    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'process_definitions' AND column_name = 'approval_config'
    ) INTO has_approval;

    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'process_definitions' AND column_name = 'sla_config'
    ) INTO has_sla;

    IF has_approval OR has_sla THEN
        RAISE EXCEPTION 'Rollback failed: columns still exist (approval=%, sla=%)', has_approval, has_sla;
    END IF;

    RAISE NOTICE 'Rollback successful: approval_config and sla_config removed';
END $$;

COMMIT;
