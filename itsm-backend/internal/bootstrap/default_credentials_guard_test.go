package bootstrap

import (
	"context"
	"testing"

	"itsm-backend/config"
	"itsm-backend/ent/enttest"
	_ "itsm-backend/ent/runtime"
	"itsm-backend/middleware"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGuardDefaultCredentialsRejectsProductionDefaults(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD", "admin123")
	t.Setenv("JWT_SECRET", "your-jwt-secret")
	t.Setenv("DB_PASSWORD", "postgres")

	risks := GuardDefaultCredentials("saas")

	assertRisk := func(code, severity string) {
		t.Helper()
		for _, risk := range risks {
			if risk.Code == code {
				assert.Equal(t, severity, risk.Severity)
				return
			}
		}
		t.Errorf("expected risk %s", code)
	}
	assertRisk("DEFAULT_ADMIN_PASSWORD", "fatal")
	assertRisk("DEFAULT_JWT_SECRET", "fatal")
	assertRisk("WEAK_DB_PASSWORD", "warning")
}

func TestGuardDefaultCredentialsRejectsPrivateProductionDefaults(t *testing.T) {
	t.Setenv("ENV", "")
	t.Setenv("ADMIN_PASSWORD", "admin123")
	t.Setenv("JWT_SECRET", "your-jwt-secret")
	t.Setenv("DB_PASSWORD", "postgres")

	assert.NotEmpty(t, GuardDefaultCredentials("private"))
}

func TestGuardDefaultCredentialsAllowsExplicitDevelopment(t *testing.T) {
	t.Setenv("ENV", "development")
	t.Setenv("ADMIN_PASSWORD", "admin123")
	t.Setenv("JWT_SECRET", "your-jwt-secret")
	t.Setenv("DB_PASSWORD", "postgres")

	assert.Empty(t, GuardDefaultCredentials("private"))
}

func TestGuardDefaultCredentialsAcceptsStrongProductionCredentials(t *testing.T) {
	t.Setenv("ENV", "")
	t.Setenv("ADMIN_PASSWORD", "A-unique-bootstrap-password-2026!")
	t.Setenv("JWT_SECRET", "86eab260e28f4e13b9121cbf6c916cf8fba27c1907dd4d84")
	t.Setenv("DB_PASSWORD", "database-password-from-secret-manager")

	assert.Empty(t, GuardDefaultCredentials("saas_msp"))
}

func TestGuardRuntimeCredentialsDoesNotRequireConsumedAdminSecret(t *testing.T) {
	t.Setenv("ENV", "production")
	t.Setenv("ADMIN_PASSWORD", "")
	t.Setenv("JWT_SECRET", "86eab260e28f4e13b9121cbf6c916cf8fba27c1907dd4d84")
	t.Setenv("DB_PASSWORD", "database-password-from-secret-manager")

	assert.Empty(t, GuardRuntimeCredentials(
		"private",
		"86eab260e28f4e13b9121cbf6c916cf8fba27c1907dd4d84",
		"database-password-from-secret-manager",
	))
	assert.NotEmpty(t, GuardBootstrapAdminCredentials("private", ""))
}

func TestGuardRuntimeCredentialsUsesEffectiveConfiguration(t *testing.T) {
	t.Setenv("ENV", "production")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("DB_PASSWORD", "")

	strong := GuardRuntimeCredentials(
		"private",
		"86eab260e28f4e13b9121cbf6c916cf8fba27c1907dd4d84",
		"database-password-from-config-file",
	)
	assert.Empty(t, strong)

	weak := GuardRuntimeCredentials("private", "your-jwt-secret", "postgres")
	assert.Len(t, weak, 2)
}

func TestNeedsBootstrapAdminTracksPersistedAdministrator(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bootstrap-admin?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })
	ctx := context.Background()

	needsAdmin, err := needsBootstrapAdmin(ctx, client)
	require.NoError(t, err)
	assert.True(t, needsAdmin)

	rootTenant, err := client.Tenant.Create().
		SetName("Platform").
		SetCode("default").
		Save(ctx)
	require.NoError(t, err)

	needsAdmin, err = needsBootstrapAdmin(ctx, client)
	require.NoError(t, err)
	assert.True(t, needsAdmin)

	_, err = client.User.Create().
		SetUsername("admin").
		SetEmail("admin@example.com").
		SetName("Platform Admin").
		SetRole("super_admin").
		SetPasswordHash("already-hashed").
		SetTenantID(rootTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	needsAdmin, err = needsBootstrapAdmin(ctx, client)
	require.NoError(t, err)
	assert.False(t, needsAdmin)
}

func TestValidateWebStartupConfigRejectsMutationFlags(t *testing.T) {
	for _, tc := range []struct {
		name        string
		autoMigrate bool
		autoSeed    bool
	}{
		{name: "auto migrate", autoMigrate: true},
		{name: "auto seed", autoSeed: true},
		{name: "both", autoMigrate: true, autoSeed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{Deployment: config.DeploymentConfig{
				AutoMigrate: tc.autoMigrate,
				AutoSeed:    tc.autoSeed,
			}}
			assert.Error(t, ValidateWebStartupConfig(cfg))
		})
	}

	assert.NoError(t, ValidateWebStartupConfig(&config.Config{}))
}

func TestConfigurePermissionModeFailsClosedByDefault(t *testing.T) {
	original := middleware.PermissionConfig.Mode
	t.Cleanup(func() { middleware.PermissionConfig.Mode = original })

	configurePermissionMode("")
	require.Equal(t, middleware.PermissionConfigModeDBOnly, middleware.PermissionConfig.Mode)

	configurePermissionMode("production")
	require.Equal(t, middleware.PermissionConfigModeDBOnly, middleware.PermissionConfig.Mode)

	configurePermissionMode("development")
	require.Equal(t, middleware.PermissionConfigModeFallback, middleware.PermissionConfig.Mode)
}
