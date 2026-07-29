-- Migration: 20260729_ticket_workflow_records_created_at
-- Description: Align ticket_workflow_records timestamp column name with ent schema.
--             DB already has created_at (NOT NULL); ent schema previously used mixin.CreateTime{}
--             which generates a create_time column. This migration confirms the correct
--             column (created_at) exists and is NOT NULL, and that create_time does NOT exist.
-- Target: production and existing deployments
-- Author: heidsoft
-- Date: 2026-07-29

-- Step 1: Verify created_at column exists and is NOT NULL
DO $$
BEGIN
    -- Check created_at exists and is NOT NULL
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'ticket_workflow_records'
          AND column_name = 'created_at'
          AND is_nullable = 'NO'
    ) THEN
        RAISE EXCEPTION 'ticket_workflow_records.created_at must exist as NOT NULL';
    END IF;

    -- Ensure create_time does NOT exist (was the old ent mixin column name)
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'ticket_workflow_records'
          AND column_name = 'create_time'
    ) THEN
        RAISE EXCEPTION 'ticket_workflow_records.create_time should not exist (old ent mixin column)';
    END IF;

    RAISE NOTICE 'ticket_workflow_records schema check passed: created_at is NOT NULL, create_time does not exist';
END $$;
