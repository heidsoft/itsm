package connector

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"itsm-backend/ent/enttest"
)

func TestPersistentConfigStoreEncryptsAndReloadsCredentials(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:connector-config-store?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	store, err := NewPersistentConfigStore(client, "test-connector-encryption-key")
	require.NoError(t, err)
	cfg := Config{TenantID: 1, Name: "email", Provider: "standard", Type: TypeEmail, Enabled: true, Credentials: map[string]string{"username": "noc@example.com", "password": "secret-app-password"}, Settings: map[string]interface{}{"imapHost": "imap.example.com"}}
	require.NoError(t, store.Save(context.Background(), cfg))

	stored := client.ConnectorConfig.Query().OnlyX(context.Background())
	require.NotContains(t, stored.EncryptedCredentials, "secret-app-password")
	loaded, err := store.LoadAll(context.Background())
	require.NoError(t, err)
	require.Len(t, loaded, 1)
	require.Equal(t, "secret-app-password", loaded[0].Credentials["password"])
}
