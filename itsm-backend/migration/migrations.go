package migration

// LegacyMigrations documents the pre-unified migration history. These versions
// were superseded by the Ent schema and must never be replayed by active
// migration entry points.
var LegacyMigrations = []Migration{
	{
		Version:     "001_initial_schema",
		Description: "Initial schema created by Ent (handled separately)",
		RollbackSQL: "",
	},
	{
		Version:     "002_add_notification_preferences",
		Description: "Add user notification preferences table",
		RollbackSQL: `DROP TABLE IF EXISTS user_notification_preferences;`,
	},
	{
		Version:     "003_add_audit_indexes",
		Description: "Add indexes for audit log queries",
		RollbackSQL: `DROP INDEX IF EXISTS idx_audit_tenant_created; DROP INDEX IF EXISTS idx_audit_entity;`,
	},
	{
		Version:     "004_add_sla_calendar",
		Description: "Add SLA business hours calendar",
		RollbackSQL: `DROP TABLE IF EXISTS sla_calendar;`,
	},
	{
		Version:     "005_add_external_id_mapping",
		Description: "Add external system ID mapping table",
		RollbackSQL: `DROP TABLE IF EXISTS external_id_mappings;`,
	},
	{
		Version:     "006_add_change_approvals",
		Description: "Add change approvals table for change workflow",
		RollbackSQL: `DROP TABLE IF EXISTS change_approvals;`,
	},
}

// RegisteredMigrations is the single canonical active migration stream used
// by bootstrap, migrate up, and migration status.
var RegisteredMigrations = []Migration{
	{
		Version:     "007_add_change_execution_tables",
		Description: "Add tenant-scoped change execution and rollback tables (irreversible; forward-fix only)",
		RollbackSQL: "",
	},
	{
		Version:     "008_add_initialization_ledger",
		Description: "Add production initialization installation, run, component-attempt, and managed-record ledgers",
		RollbackSQL: "",
	},
	{
		Version:     "009_enable_rls_tenant_isolation",
		Description: "Enable RLS row-level tenant isolation on all tenant-scoped tables",
		RollbackSQL: "",
	},
	{
		Version:     "010_add_ticket_types",
		Description: "Add ticket_types table for structured ticket type definitions with JSON config fields",
		RollbackSQL: "",
	},
}

// PostSchemaMigrations returns a defensive copy of the canonical active stream.
func PostSchemaMigrations() []Migration {
	result := make([]Migration, len(RegisteredMigrations))
	copy(result, RegisteredMigrations)
	return result
}

// GetMigrationSQL returns the SQL for a specific migration
func GetMigrationSQL(version string) string {
	switch version {
	case "002_add_notification_preferences":
		return `
CREATE TABLE IF NOT EXISTS user_notification_preferences (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id INTEGER NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
    notification_type VARCHAR(50) NOT NULL,
    channel VARCHAR(20) NOT NULL DEFAULT 'email',
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, notification_type, channel)
);

CREATE INDEX IF NOT EXISTS idx_notification_prefs_user ON user_notification_preferences(user_id);
CREATE INDEX IF NOT EXISTS idx_notification_prefs_tenant ON user_notification_preferences(tenant_id);
`
	case "003_add_audit_indexes":
		return `
CREATE INDEX IF NOT EXISTS idx_audit_tenant_created ON audit_log(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_entity ON audit_log(entity_type, entity_id);
`
	case "004_add_sla_calendar":
		return `
CREATE TABLE IF NOT EXISTS sla_calendar (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    timezone VARCHAR(50) NOT NULL DEFAULT 'UTC',
    working_days VARCHAR(20) NOT NULL DEFAULT '1,2,3,4,5',
    working_hours_start TIME NOT NULL DEFAULT '09:00',
    working_hours_end TIME NOT NULL DEFAULT '18:00',
    holidays JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sla_calendar_tenant ON sla_calendar(tenant_id);
`
	case "005_add_external_id_mapping":
		return `
CREATE TABLE IF NOT EXISTS external_id_mappings (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
    entity_type VARCHAR(50) NOT NULL,
    entity_id INTEGER NOT NULL,
    external_system VARCHAR(100) NOT NULL,
    external_id VARCHAR(255) NOT NULL,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(entity_type, entity_id, external_system)
);

CREATE INDEX IF NOT EXISTS idx_ext_mapping_tenant ON external_id_mappings(tenant_id);
CREATE INDEX IF NOT EXISTS idx_ext_mapping_external ON external_id_mappings(external_system, external_id);
`
	case "006_add_change_approvals":
		return `
CREATE TABLE IF NOT EXISTS change_approvals (
    id SERIAL PRIMARY KEY,
    change_id INTEGER NOT NULL REFERENCES changes(id) ON DELETE CASCADE,
    approver_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    comment TEXT,
    approved_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_change_approvals_change ON change_approvals(change_id);
CREATE INDEX IF NOT EXISTS idx_change_approvals_approver ON change_approvals(approver_id);
CREATE INDEX IF NOT EXISTS idx_change_approvals_status ON change_approvals(status);
`
	case "007_add_change_execution_tables":
		return `
CREATE TABLE IF NOT EXISTS change_approvals (
    id BIGSERIAL PRIMARY KEY,
    change_id BIGINT NOT NULL REFERENCES changes(id) ON DELETE CASCADE,
    tenant_id BIGINT NOT NULL,
    approver_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending',
    comment TEXT,
    approved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE change_approvals ADD COLUMN IF NOT EXISTS tenant_id BIGINT NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS change_approval_chains (
    id BIGSERIAL PRIMARY KEY,
    change_id BIGINT NOT NULL REFERENCES changes(id) ON DELETE CASCADE,
    tenant_id BIGINT NOT NULL,
    level INTEGER NOT NULL,
    approver_id BIGINT NOT NULL,
    role TEXT DEFAULT 'approver',
    status TEXT DEFAULT 'pending',
    is_required BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS change_risk_assessments (
    id BIGSERIAL PRIMARY KEY,
    change_id BIGINT NOT NULL REFERENCES changes(id) ON DELETE CASCADE,
    tenant_id BIGINT NOT NULL,
    risk_level TEXT NOT NULL DEFAULT 'medium',
    risk_description TEXT,
    impact_analysis TEXT,
    mitigation_measures TEXT,
    contingency_plan TEXT,
    risk_owner TEXT,
    risk_review_date TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS change_rollback_plans (
    id BIGSERIAL PRIMARY KEY,
    change_id BIGINT NOT NULL REFERENCES changes(id) ON DELETE CASCADE,
    tenant_id BIGINT NOT NULL,
    trigger_conditions JSONB,
    rollback_steps JSONB,
    responsible TEXT,
    estimated_time INTEGER,
    communication_plan TEXT,
    test_plan TEXT,
    approval_required BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS change_rollback_executions (
    id BIGSERIAL PRIMARY KEY,
    change_id BIGINT NOT NULL REFERENCES changes(id) ON DELETE CASCADE,
    tenant_id BIGINT NOT NULL,
    rollback_plan_id BIGINT NOT NULL REFERENCES change_rollback_plans(id) ON DELETE CASCADE,
    trigger_reason TEXT,
    initiated_by BIGINT,
    status TEXT NOT NULL DEFAULT 'initiated',
    start_time TIMESTAMPTZ,
    end_time TIMESTAMPTZ,
    result TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS change_implementation_plans (
    id BIGSERIAL PRIMARY KEY,
    change_id BIGINT NOT NULL REFERENCES changes(id) ON DELETE CASCADE,
    tenant_id BIGINT NOT NULL,
    phase TEXT,
    description TEXT,
    tasks JSONB,
    responsible TEXT,
    start_date TIMESTAMPTZ,
    end_date TIMESTAMPTZ,
    prerequisites JSONB,
    dependencies JSONB,
    success_criteria TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE change_approval_chains ADD COLUMN IF NOT EXISTS tenant_id BIGINT;
ALTER TABLE change_risk_assessments ADD COLUMN IF NOT EXISTS tenant_id BIGINT;
ALTER TABLE change_rollback_plans ADD COLUMN IF NOT EXISTS tenant_id BIGINT;
ALTER TABLE change_rollback_executions ADD COLUMN IF NOT EXISTS tenant_id BIGINT;
ALTER TABLE change_implementation_plans ADD COLUMN IF NOT EXISTS tenant_id BIGINT;

UPDATE change_approvals ca
SET tenant_id = c.tenant_id
FROM changes c
WHERE ca.change_id = c.id AND (ca.tenant_id IS NULL OR ca.tenant_id = 0);

UPDATE change_approval_chains t SET tenant_id = c.tenant_id
FROM changes c WHERE t.change_id = c.id AND (t.tenant_id IS NULL OR t.tenant_id = 0);
UPDATE change_risk_assessments t SET tenant_id = c.tenant_id
FROM changes c WHERE t.change_id = c.id AND (t.tenant_id IS NULL OR t.tenant_id = 0);
UPDATE change_rollback_plans t SET tenant_id = c.tenant_id
FROM changes c WHERE t.change_id = c.id AND (t.tenant_id IS NULL OR t.tenant_id = 0);
UPDATE change_rollback_executions t SET tenant_id = c.tenant_id
FROM changes c WHERE t.change_id = c.id AND (t.tenant_id IS NULL OR t.tenant_id = 0);
UPDATE change_implementation_plans t SET tenant_id = c.tenant_id
FROM changes c WHERE t.change_id = c.id AND (t.tenant_id IS NULL OR t.tenant_id = 0);

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM change_approvals WHERE tenant_id IS NULL OR tenant_id = 0)
       OR EXISTS (SELECT 1 FROM change_approval_chains WHERE tenant_id IS NULL OR tenant_id = 0)
       OR EXISTS (SELECT 1 FROM change_risk_assessments WHERE tenant_id IS NULL OR tenant_id = 0)
       OR EXISTS (SELECT 1 FROM change_rollback_plans WHERE tenant_id IS NULL OR tenant_id = 0)
       OR EXISTS (SELECT 1 FROM change_rollback_executions WHERE tenant_id IS NULL OR tenant_id = 0)
       OR EXISTS (SELECT 1 FROM change_implementation_plans WHERE tenant_id IS NULL OR tenant_id = 0) THEN
        RAISE EXCEPTION 'change execution tables contain rows without a resolvable tenant';
    END IF;
END $$;

ALTER TABLE change_approvals ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE change_approvals ALTER COLUMN tenant_id DROP DEFAULT;
ALTER TABLE change_approval_chains ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE change_approval_chains ALTER COLUMN tenant_id DROP DEFAULT;
ALTER TABLE change_risk_assessments ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE change_risk_assessments ALTER COLUMN tenant_id DROP DEFAULT;
ALTER TABLE change_rollback_plans ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE change_rollback_plans ALTER COLUMN tenant_id DROP DEFAULT;
ALTER TABLE change_rollback_executions ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE change_rollback_executions ALTER COLUMN tenant_id DROP DEFAULT;
ALTER TABLE change_implementation_plans ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE change_implementation_plans ALTER COLUMN tenant_id DROP DEFAULT;

CREATE INDEX IF NOT EXISTS idx_change_approvals_tenant_id ON change_approvals(tenant_id);
CREATE INDEX IF NOT EXISTS idx_change_approval_chains_change_id ON change_approval_chains(change_id);
CREATE INDEX IF NOT EXISTS idx_change_approval_chains_tenant_id ON change_approval_chains(tenant_id);
CREATE INDEX IF NOT EXISTS idx_change_risk_assessments_change_id ON change_risk_assessments(change_id);
CREATE INDEX IF NOT EXISTS idx_change_risk_assessments_tenant_id ON change_risk_assessments(tenant_id);
CREATE INDEX IF NOT EXISTS idx_change_rollback_plans_change_id ON change_rollback_plans(change_id);
CREATE INDEX IF NOT EXISTS idx_change_rollback_plans_tenant_id ON change_rollback_plans(tenant_id);
CREATE INDEX IF NOT EXISTS idx_change_rollback_executions_change_id ON change_rollback_executions(change_id);
CREATE INDEX IF NOT EXISTS idx_change_rollback_executions_tenant_id ON change_rollback_executions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_change_rollback_executions_plan_id ON change_rollback_executions(rollback_plan_id);
CREATE INDEX IF NOT EXISTS idx_change_implementation_plans_change_id ON change_implementation_plans(change_id);
CREATE INDEX IF NOT EXISTS idx_change_implementation_plans_tenant_id ON change_implementation_plans(tenant_id);
`
	case "008_add_initialization_ledger":
		return `
CREATE TABLE IF NOT EXISTS initialization_installations (
    id BIGSERIAL PRIMARY KEY,
    scope_type VARCHAR(16) NOT NULL CHECK (scope_type IN ('platform', 'tenant')),
    scope_id BIGINT NOT NULL,
    component VARCHAR(100) NOT NULL,
    installed_version VARCHAR(64) NOT NULL DEFAULT '',
    source_checksum VARCHAR(128) NOT NULL DEFAULT '',
    status VARCHAR(24) NOT NULL CHECK (status IN ('pending', 'running', 'succeeded', 'failed', 'rolling_back')),
    fencing_token BIGINT NOT NULL DEFAULT 0,
    lease_owner VARCHAR(255) NOT NULL DEFAULT '',
    heartbeat_at TIMESTAMPTZ,
    lease_expires_at TIMESTAMPTZ,
    last_run_id BIGINT,
    error_code VARCHAR(100) NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    result_summary JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(scope_type, scope_id, component)
);

CREATE TABLE IF NOT EXISTS initialization_runs (
    id BIGSERIAL PRIMARY KEY,
    scope_type VARCHAR(16) NOT NULL CHECK (scope_type IN ('platform', 'tenant', 'batch')),
    scope_id BIGINT NOT NULL,
    target_version VARCHAR(64) NOT NULL,
    release_version VARCHAR(64) NOT NULL,
    requested_by VARCHAR(255) NOT NULL,
    executor_id VARCHAR(255) NOT NULL,
    status VARCHAR(24) NOT NULL CHECK (status IN ('pending', 'running', 'succeeded', 'failed', 'partial')),
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    result_summary JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS initialization_component_attempts (
    id BIGSERIAL PRIMARY KEY,
    run_id BIGINT NOT NULL REFERENCES initialization_runs(id) ON DELETE CASCADE,
    scope_type VARCHAR(16) NOT NULL,
    scope_id BIGINT NOT NULL,
    component VARCHAR(100) NOT NULL,
    attempt INTEGER NOT NULL,
    from_version VARCHAR(64) NOT NULL DEFAULT '',
    target_version VARCHAR(64) NOT NULL,
    source_checksum VARCHAR(128) NOT NULL,
    fencing_token BIGINT NOT NULL,
    status VARCHAR(24) NOT NULL CHECK (status IN ('pending', 'running', 'succeeded', 'failed', 'rolling_back')),
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    error_code VARCHAR(100) NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    result_summary JSONB NOT NULL DEFAULT '{}'::jsonb,
    rollback_metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(run_id, component, attempt)
);

CREATE TABLE IF NOT EXISTS initialization_managed_records (
    id BIGSERIAL PRIMARY KEY,
    scope_type VARCHAR(16) NOT NULL,
    scope_id BIGINT NOT NULL,
    component VARCHAR(100) NOT NULL,
    source_key VARCHAR(255) NOT NULL,
    source_version VARCHAR(64) NOT NULL,
    manifest_checksum VARCHAR(128) NOT NULL,
    ownership_mode VARCHAR(24) NOT NULL DEFAULT 'managed',
    managed_fields JSONB NOT NULL DEFAULT '[]'::jsonb,
    last_applied_values JSONB NOT NULL DEFAULT '{}'::jsonb,
    local_modified_at TIMESTAMPTZ,
    deprecated BOOLEAN NOT NULL DEFAULT false,
    stable_key_aliases JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(scope_type, scope_id, component, source_key)
);

CREATE INDEX IF NOT EXISTS idx_init_installations_status_lease
    ON initialization_installations(status, lease_expires_at);
CREATE INDEX IF NOT EXISTS idx_init_runs_scope_started
    ON initialization_runs(scope_type, scope_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_init_attempts_run_component
    ON initialization_component_attempts(run_id, component, attempt);
CREATE INDEX IF NOT EXISTS idx_init_managed_scope_component
    ON initialization_managed_records(scope_type, scope_id, component);
`
	case "009_enable_rls_tenant_isolation":
		return `
-- Enable RLS on all tenant-scoped tables
-- Policy: users can only see rows where tenant_id matches current_setting('app.current_tenant_id')

-- Helper function to get current tenant_id safely
CREATE OR REPLACE FUNCTION get_current_tenant_id() RETURNS INTEGER AS $$
BEGIN
    RETURN NULLIF(current_setting('app.current_tenant_id', true)::INTEGER, 0);
END;
$$ LANGUAGE plpgsql STABLE;

-- Teams
ALTER TABLE teams ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_teams ON teams;
CREATE POLICY tenant_isolation_teams ON teams
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

-- Roles
ALTER TABLE roles ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_roles ON roles;
CREATE POLICY tenant_isolation_roles ON roles
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

-- Users
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_users ON users;
CREATE POLICY tenant_isolation_users ON users
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

-- SLA Policies
ALTER TABLE sla_policies ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_sla_policies ON sla_policies;
CREATE POLICY tenant_isolation_sla_policies ON sla_policies
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

-- Service Catalogs
ALTER TABLE service_catalogs ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_service_catalogs ON service_catalogs;
CREATE POLICY tenant_isolation_service_catalogs ON service_catalogs
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

-- CI Types
ALTER TABLE ci_types ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_ci_types ON ci_types;
CREATE POLICY tenant_isolation_ci_types ON ci_types
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

-- Standard Changes
ALTER TABLE standard_changes ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_standard_changes ON standard_changes;
CREATE POLICY tenant_isolation_standard_changes ON standard_changes
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

-- Known Errors
ALTER TABLE known_errors ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_known_errors ON known_errors;
CREATE POLICY tenant_isolation_known_errors ON known_errors
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

-- SLA Alert Rules
ALTER TABLE sla_alert_rules ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_sla_alert_rules ON sla_alert_rules;
CREATE POLICY tenant_isolation_sla_alert_rules ON sla_alert_rules
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

-- Tags
ALTER TABLE tags ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_tags ON tags;
CREATE POLICY tenant_isolation_tags ON tags
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

-- Departments
ALTER TABLE departments ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_departments ON departments;
CREATE POLICY tenant_isolation_departments ON departments
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

-- Ticket Categories
ALTER TABLE ticket_categories ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_ticket_categories ON ticket_categories;
CREATE POLICY tenant_isolation_ticket_categories ON ticket_categories
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

-- Process Bindings
ALTER TABLE process_bindings ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_process_bindings ON process_bindings;
CREATE POLICY tenant_isolation_process_bindings ON process_bindings
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

-- Process Definitions
ALTER TABLE process_definitions ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_process_definitions ON process_definitions;
CREATE POLICY tenant_isolation_process_definitions ON process_definitions
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

-- Process Deployments
ALTER TABLE process_deployments ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_process_deployments ON process_deployments;
CREATE POLICY tenant_isolation_process_deployments ON process_deployments
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

-- Approval Workflows
ALTER TABLE approval_workflows ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_approval_workflows ON approval_workflows;
CREATE POLICY tenant_isolation_approval_workflows ON approval_workflows
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

-- Ticket Views
ALTER TABLE ticket_views ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_ticket_views ON ticket_views;
CREATE POLICY tenant_isolation_ticket_views ON ticket_views
    USING (tenant_id = get_current_tenant_id())
    WITH CHECK (tenant_id = get_current_tenant_id());

-- Force role to bypass RLS for system operations (e.g., itsm-backend service account)
-- The itsm_backend_role is the service account used by the application
-- Uncomment if you need a superuser role to bypass RLS:
-- ALTER TABLE teams FORCE ROW LEVEL SECURITY;
-- ALTER TABLE roles FORCE ROW LEVEL SECURITY;
-- etc. for other tables

COMMENT ON FUNCTION get_current_tenant_id() IS
    'Returns the current tenant ID from session settings, used by RLS policies for tenant isolation';
`
	case "010_add_ticket_types":
		return `
CREATE TABLE IF NOT EXISTS ticket_types (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    icon VARCHAR(50),
    color VARCHAR(20),
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'archived')),
    custom_fields JSONB DEFAULT '{}',
    approval_enabled BOOLEAN NOT NULL DEFAULT false,
    approval_workflow_id BIGINT,
    approval_chain JSONB DEFAULT '[]',
    sla_enabled BOOLEAN NOT NULL DEFAULT false,
    default_sla_id BIGINT,
    auto_assign_enabled BOOLEAN NOT NULL DEFAULT false,
    assignment_rules JSONB DEFAULT '[]',
    notification_config JSONB DEFAULT '{}',
    permission_config JSONB DEFAULT '{}',
    created_by BIGINT NOT NULL REFERENCES users(id),
    tenant_id BIGINT NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by BIGINT REFERENCES users(id),
    usage_count INTEGER NOT NULL DEFAULT 0,
    UNIQUE(code, tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_ticket_types_tenant ON ticket_types(tenant_id);
CREATE INDEX IF NOT EXISTS idx_ticket_types_code ON ticket_types(code);
CREATE INDEX IF NOT EXISTS idx_ticket_types_status ON ticket_types(status);
`
	default:
		return ""
	}
}
