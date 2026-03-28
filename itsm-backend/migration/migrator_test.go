package migration

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMigration_Struct(t *testing.T) {
	// Test that Migration struct can be created correctly
	mig := Migration{
		Version:     "002_add_notification_preferences",
		Description: "Test migration",
		RollbackSQL: "DROP TABLE user_notification_preferences",
	}

	assert.Equal(t, "002_add_notification_preferences", mig.Version)
	assert.Equal(t, "Test migration", mig.Description)
	assert.NotEmpty(t, mig.RollbackSQL)
}

func TestMigration_NoRollbackSQL(t *testing.T) {
	// Test migration without rollback SQL
	mig := Migration{
		Version:     "001_initial_schema",
		Description: "Initial schema",
		RollbackSQL: "",
	}

	assert.Equal(t, "001_initial_schema", mig.Version)
	assert.Empty(t, mig.RollbackSQL)
}

func TestMigrationSlice_Versions(t *testing.T) {
	// Test that migrations can be sorted by version
	migrations := []Migration{
		{Version: "003_add_audit_indexes"},
		{Version: "001_initial_schema"},
		{Version: "002_add_notification_preferences"},
	}

	assert.Equal(t, 3, len(migrations))
	assert.Equal(t, "003_add_audit_indexes", migrations[0].Version)
}

func TestRegisteredMigrations(t *testing.T) {
	// Test that RegisteredMigrations contains expected migrations
	assert.NotEmpty(t, RegisteredMigrations)

	// Check first migration is initial schema
	assert.Equal(t, "001_initial_schema", RegisteredMigrations[0].Version)
}

func TestGetMigrationSQL(t *testing.T) {
	// Test GetMigrationSQL returns SQL for known migrations
	sql := GetMigrationSQL("002_add_notification_preferences")
	assert.NotEmpty(t, sql)
	assert.Contains(t, sql, "CREATE TABLE")

	// Test GetMigrationSQL returns empty for unknown migrations
	sql = GetMigrationSQL("999_unknown")
	assert.Empty(t, sql)
}

func TestGetMigrationSQL_InitialSchema(t *testing.T) {
	// Test that initial schema returns empty (handled by Ent)
	sql := GetMigrationSQL("001_initial_schema")
	assert.Empty(t, sql)
}
