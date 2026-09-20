package scenarios

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"itsm-backend/ent"
	entuser "itsm-backend/ent/user"

	"github.com/stretchr/testify/require"
)

var scenarioDBCounter int64

func scenarioDSN(prefix string) string {
	return fmt.Sprintf("file:scenario_%s_%d?mode=memory&cache=shared&_fk=1", prefix, atomic.AddInt64(&scenarioDBCounter, 1))
}

func mustCreateTenant(ctx context.Context, t *testing.T, client *ent.Client, name, code, domain string) *ent.Tenant {
	t.Helper()
	tn, err := client.Tenant.Create().
		SetName(name).
		SetCode(code).
		SetDomain(domain).
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	return tn
}

func mustCreateUser(ctx context.Context, t *testing.T, client *ent.Client, tenantID int, username, email, role string) *ent.User {
	t.Helper()
	u, err := client.User.Create().
		SetUsername(username).
		SetEmail(email).
		SetName(username).
		SetPasswordHash("hashed").
		SetRole(entuser.Role(role)).
		SetActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return u
}
