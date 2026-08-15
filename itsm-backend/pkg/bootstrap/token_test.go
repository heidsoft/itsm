package bootstrap

import (
	"context"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/auditlog"
	"itsm-backend/ent/bootstraptoken"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/user"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestBootstrapTokenLifecycleIsSingleUseAndAudited(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bootstrap-token-lifecycle?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	rootTenant, err := client.Tenant.Create().
		SetName("Default").SetCode("default").SetDomain("default.local").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	manager := NewBootstrapTokenManager(client, zaptest.NewLogger(t).Sugar())

	oldToken, err := manager.GenerateToken(ctx, rootTenant.ID)
	require.NoError(t, err)
	newToken, err := manager.GenerateToken(ctx, rootTenant.ID)
	require.NoError(t, err)
	require.NotEqual(t, oldToken, newToken)
	require.Equal(t, 1, mustCountBootstrapTokens(t, client, true))
	require.Equal(t, 1, mustCountBootstrapTokens(t, client, false))

	_, err = manager.ConsumeToken(ctx, oldToken, rootTenant.ID, "A-secure-admin-password-2026!")
	require.Error(t, err)
	adminID, err := manager.ConsumeToken(ctx, newToken, rootTenant.ID, "A-secure-admin-password-2026!")
	require.NoError(t, err)
	require.Positive(t, adminID)
	require.Equal(t, 1, mustCountUsers(t, client))
	require.Equal(t, 1, mustCountBootstrapAudits(t, client))

	required, available, expiresAt, err := manager.Status(ctx, rootTenant.ID)
	require.NoError(t, err)
	require.False(t, required)
	require.False(t, available)
	require.Nil(t, expiresAt)
	_, err = manager.ConsumeToken(ctx, newToken, rootTenant.ID, "A-secure-admin-password-2026!")
	require.Error(t, err)
}

func mustCountBootstrapTokens(t *testing.T, client *ent.Client, used bool) int {
	t.Helper()
	count, err := client.BootstrapToken.Query().Where(bootstraptoken.UsedEQ(used)).Count(context.Background())
	require.NoError(t, err)
	return count
}

func mustCountUsers(t *testing.T, client *ent.Client) int {
	t.Helper()
	count, err := client.User.Query().Where(user.UsernameEQ("admin")).Count(context.Background())
	require.NoError(t, err)
	return count
}

func mustCountBootstrapAudits(t *testing.T, client *ent.Client) int {
	t.Helper()
	count, err := client.AuditLog.Query().Where(auditlog.ActionEQ("BOOTSTRAP_ADMIN_CREATED")).Count(context.Background())
	require.NoError(t, err)
	return count
}
