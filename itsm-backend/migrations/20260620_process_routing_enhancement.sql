-- ITSM Process Routing Enhancement Migration
-- Date: 2026-06-20
-- Description: Add multi-dimensional routing fields to process_bindings table

-- Step 1: Add new columns to process_bindings table
ALTER TABLE process_bindings 
ADD COLUMN IF NOT EXISTS department_id INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS team_id INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS scenario VARCHAR(100) DEFAULT '',
ADD COLUMN IF NOT EXISTS category VARCHAR(50) DEFAULT '',
ADD COLUMN IF NOT EXISTS conditions JSONB DEFAULT '{}',
ADD COLUMN IF NOT EXISTS approval_chain_id VARCHAR(100) DEFAULT '',
ADD COLUMN IF NOT EXISTS sla_policy_id VARCHAR(100) DEFAULT '',
ADD COLUMN IF NOT EXISTS overrides JSONB DEFAULT '{}';

-- Step 2: Create composite index for efficient routing queries
CREATE INDEX IF NOT EXISTS idx_process_binding_routing 
ON process_bindings(tenant_id, business_type, is_active, department_id, team_id, scenario);

-- Step 3: Create index for department-specific queries
CREATE INDEX IF NOT EXISTS idx_process_binding_department 
ON process_bindings(tenant_id, department_id, is_active);

-- Step 4: Create index for scenario-based queries
CREATE INDEX IF NOT EXISTS idx_process_binding_scenario 
ON process_bindings(tenant_id, scenario, is_active);

-- Step 5: Create domain configuration table for inheritance support
CREATE TABLE IF NOT EXISTS domain_configs (
    id BIGSERIAL PRIMARY KEY,
    config_key VARCHAR(255) NOT NULL,
    config_type VARCHAR(255) NOT NULL,
    config_value JSONB NOT NULL DEFAULT '{}',
    inherit_mode VARCHAR(50) NOT NULL DEFAULT 'inherit',
    tenant_id INTEGER NOT NULL DEFAULT 0,
    department_id INTEGER NOT NULL DEFAULT 0,
    team_id INTEGER NOT NULL DEFAULT 0,
    parent_config_id INTEGER,
    version INTEGER NOT NULL DEFAULT 1,
    is_active BOOLEAN NOT NULL DEFAULT true,
    description VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT domain_configs_tenant_id_check CHECK (tenant_id >= 0),
    CONSTRAINT domain_configs_scope_unique UNIQUE (tenant_id, config_type, config_key, department_id, team_id)
);

CREATE INDEX IF NOT EXISTS idx_domain_config_lookup
ON domain_configs(tenant_id, config_type, config_key);

CREATE INDEX IF NOT EXISTS idx_domain_config_scope
ON domain_configs(tenant_id, department_id, team_id);

-- Step 6: Backfill existing data (set defaults for new columns)
UPDATE process_bindings 
SET 
    department_id = COALESCE(department_id, 0),
    team_id = COALESCE(team_id, 0),
    scenario = COALESCE(scenario, ''),
    category = COALESCE(category, ''),
    conditions = COALESCE(conditions, '{}'),
    approval_chain_id = COALESCE(approval_chain_id, ''),
    sla_policy_id = COALESCE(sla_policy_id, ''),
    overrides = COALESCE(overrides, '{}')
WHERE department_id IS NULL 
   OR team_id IS NULL 
   OR scenario IS NULL 
   OR category IS NULL;

-- Step 7: Insert sample routing rules for demonstration

-- Operations Department: Alert Handling
INSERT INTO process_bindings (
    business_type, business_sub_type, process_definition_key, 
    department_id, scenario, category, priority, is_active, tenant_id,
    created_at, updated_at
) VALUES 
-- P0 Alert (Critical)
('incident', 'alert_p0', 'incident_emergency_flow', 
 (SELECT id FROM departments WHERE code = 'OPS' LIMIT 1), 
 'alert_handling', 'operations', 100, true, 1,
 NOW(), NOW()),
-- P1 Alert (High)
('incident', 'alert_p1', 'incident_emergency_flow',
 (SELECT id FROM departments WHERE code = 'OPS' LIMIT 1),
 'alert_handling', 'operations', 90, true, 1,
 NOW(), NOW()),
-- P2 Alert (Medium)
('incident', 'alert_p2', 'incident_general_flow',
 (SELECT id FROM departments WHERE code = 'OPS' LIMIT 1),
 'alert_handling', 'operations', 80, true, 1,
 NOW(), NOW());

-- Operations Department: Change Release
INSERT INTO process_bindings (
    business_type, business_sub_type, process_definition_key,
    department_id, scenario, category, priority, is_active, tenant_id,
    created_at, updated_at
) VALUES 
-- Normal Change
('change', 'normal', 'change_normal_flow',
 (SELECT id FROM departments WHERE code = 'OPS' LIMIT 1),
 'change_release', 'operations', 70, true, 1,
 NOW(), NOW()),
-- Emergency Change
('change', 'emergency', 'change_emergency_flow',
 (SELECT id FROM departments WHERE code = 'OPS' LIMIT 1),
 'change_release', 'operations', 80, true, 1,
 NOW(), NOW()),
-- Standard Change
('change', 'standard', 'change_normal_flow',
 (SELECT id FROM departments WHERE code = 'OPS' LIMIT 1),
 'change_release', 'operations', 60, true, 1,
 NOW(), NOW());

-- R&D Department: Code Release
INSERT INTO process_bindings (
    business_type, business_sub_type, process_definition_key,
    department_id, scenario, category, priority, is_active, tenant_id,
    created_at, updated_at
) VALUES 
-- Production Release
('release', 'production', 'release_approval_flow',
 (SELECT id FROM departments WHERE code = 'RD' LIMIT 1),
 'code_release', 'rd', 90, true, 1,
 NOW(), NOW()),
-- Test Release
('release', 'testing', 'release_test_flow',
 (SELECT id FROM departments WHERE code = 'RD' LIMIT 1),
 'code_release', 'rd', 70, true, 1,
 NOW(), NOW());

-- Finance Department: Expense Approval
INSERT INTO process_bindings (
    business_type, business_sub_type, process_definition_key,
    department_id, scenario, category, priority, is_active, tenant_id,
    conditions, created_at, updated_at
) VALUES 
-- Large Expense (>100000)
('service_request', 'expense', 'expense_approval_flow',
 (SELECT id FROM departments WHERE code = 'FIN' LIMIT 1),
 'expense_approval', 'finance', 100, true, 1,
 '{"min_amount": 100000}',
 NOW(), NOW()),
-- Normal Expense (<=100000)
('service_request', 'expense', 'expense_approval_flow',
 (SELECT id FROM departments WHERE code = 'FIN' LIMIT 1),
 'expense_approval', 'finance', 80, true, 1,
 '{"max_amount": 100000}',
 NOW(), NOW());

-- Global Default Bindings (lowest priority)
INSERT INTO process_bindings (
    business_type, business_sub_type, process_definition_key,
    department_id, team_id, scenario, category, 
    priority, is_default, is_active, tenant_id,
    created_at, updated_at
) VALUES 
-- Default Ticket Flow
('ticket', '', 'ticket_general_flow',
 0, 0, '', '',
 10, true, true, 1,
 NOW(), NOW()),
-- Default Incident Flow
('incident', '', 'incident_emergency_flow',
 0, 0, '', '',
 10, true, true, 1,
 NOW(), NOW()),
-- Default Change Flow
('change', '', 'change_normal_flow',
 0, 0, '', '',
 10, true, true, 1,
 NOW(), NOW());

-- Verification: Count routing rules
DO $$
DECLARE
    total_count INTEGER;
    dept_count INTEGER;
    global_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO total_count FROM process_bindings WHERE is_active = true;
    SELECT COUNT(*) INTO dept_count FROM process_bindings WHERE department_id > 0 AND is_active = true;
    SELECT COUNT(*) INTO global_count FROM process_bindings WHERE department_id = 0 AND is_active = true;
    
    RAISE NOTICE 'Migration completed:';
    RAISE NOTICE '  Total active bindings: %', total_count;
    RAISE NOTICE '  Department-specific bindings: %', dept_count;
    RAISE NOTICE '  Global bindings: %', global_count;
END $$;
