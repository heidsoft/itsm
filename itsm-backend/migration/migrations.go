package migration

// RegisteredMigrations contains all available migrations
// Each migration has a version, description, SQL to apply, and SQL to rollback
var RegisteredMigrations = []Migration{
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
	default:
		return ""
	}
}
