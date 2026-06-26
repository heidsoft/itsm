-- ITSM Process Definition Approval/SLA Config Migration
-- Date: 2026-06-21
-- Description: Add approval_config / sla_config JSON columns to process_definitions
--              so the WorkflowDesigner "流程配置" tab values actually persist to DB.
-- ROLLBACK: 20260621_process_definition_approval_sla_config_down.sql
--           DROP COLUMN approval_config, sla_config; 备份: CREATE TABLE process_definitions_bak_20260621 AS SELECT * FROM process_definitions;
-- PR: #01 (v1.0 GA 拆分批次)

-- Step 1: Add approval_config column
ALTER TABLE process_definitions
ADD COLUMN IF NOT EXISTS approval_config JSONB DEFAULT '{}'::jsonb;

COMMENT ON COLUMN process_definitions.approval_config IS
'流程级审批配置（注意：审批组是节点级，走 BPMN userTask candidateGroups，不存这里）。Schema:
{
  "require_approval": bool,
  "approval_type": "single" | "parallel" | "sequential" | "conditional",
  "approvers": [int],              -- 流程级兜底用户 ID 列表（可选，主要走 BPMN 节点级）
  "auto_approve_roles": [string],  -- 自动通过的角色
  "escalation_rules": [
    {"after_hours": int, "to_user": int, "to_group": string, "notify": bool}
  ]
}';

-- Step 2: Add sla_config column
ALTER TABLE process_definitions
ADD COLUMN IF NOT EXISTS sla_config JSONB DEFAULT '{}'::jsonb;

COMMENT ON COLUMN process_definitions.sla_config IS
'流程级 SLA 配置。Schema:
{
  "response_time_hours": int,      -- 首次响应时长
  "resolution_time_hours": int,    -- 解决时长
  "business_hours_only": bool,     -- 仅计算工作时间
  "exclude_weekends": bool,
  "exclude_holidays": bool,
  "priority_overrides": {          -- 按优先级覆盖
    "P0": {"response_time_hours": int, "resolution_time_hours": int},
    ...
  }
}';

-- Step 3: Backfill existing rows so the column is never NULL going forward
UPDATE process_definitions
SET
    approval_config = COALESCE(approval_config, '{}'::jsonb),
    sla_config      = COALESCE(sla_config,      '{}'::jsonb)
WHERE approval_config IS NULL
   OR sla_config      IS NULL;

-- Step 4: Index on the JSONB for cheap filtering by require_approval / approval_type
CREATE INDEX IF NOT EXISTS idx_process_definitions_approval_require
ON process_definitions USING gin ((approval_config -> 'require_approval'));

CREATE INDEX IF NOT EXISTS idx_process_definitions_approval_type
ON process_definitions ((approval_config ->> 'approval_type'));

-- Step 5: Verification
DO $$
DECLARE
    total_count INTEGER;
    with_approval INTEGER;
BEGIN
    SELECT COUNT(*) INTO total_count FROM process_definitions;
    SELECT COUNT(*) INTO with_approval FROM process_definitions
        WHERE (approval_config ->> 'require_approval')::boolean = true;
    RAISE NOTICE 'Migration completed:';
    RAISE NOTICE '  Total process_definitions: %', total_count;
    RAISE NOTICE '  Rows with require_approval=true: %', with_approval;
END $$;