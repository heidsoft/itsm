package change

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/operationalcommand"
	"itsm-backend/internal/commandbus"

	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestSubmitForApprovalPersistsApprovalAndNotificationAtomically(t *testing.T) {
	ctx := context.Background()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", strings.ReplaceAll(t.Name(), "/", "-"))
	db, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	client := ent.NewClient(ent.Driver(entsql.OpenDB("sqlite3", db)))
	t.Cleanup(func() { _ = client.Close() })
	require.NoError(t, client.Schema.Create(ctx))
	_, err = db.ExecContext(ctx, `
		CREATE TABLE change_approvals (
			id INTEGER PRIMARY KEY AUTOINCREMENT, change_id INTEGER NOT NULL, tenant_id INTEGER NOT NULL,
			approver_id INTEGER NOT NULL, status TEXT NOT NULL, comment TEXT, created_at DATETIME, updated_at DATETIME
		);
		CREATE TABLE change_approval_chains (
			id INTEGER PRIMARY KEY AUTOINCREMENT, change_id INTEGER NOT NULL, tenant_id INTEGER NOT NULL,
			level INTEGER NOT NULL, approver_id INTEGER NOT NULL, role TEXT NOT NULL, status TEXT NOT NULL,
			is_required BOOLEAN NOT NULL, created_at DATETIME
		);
	`)
	require.NoError(t, err)

	tenant, err := client.Tenant.Create().SetName("approval tenant").SetCode("approval-outbox").
		SetDomain("approval-outbox.example.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	creator, err := client.User.Create().SetUsername("change-creator").SetEmail("creator@example.com").
		SetName("Creator").SetPasswordHash("hash").SetRole("manager").SetActive(true).SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	approver, err := client.User.Create().SetUsername("change-approver").SetEmail("approver@example.com").
		SetName("Approver").SetPasswordHash("hash").SetRole("manager").SetActive(true).SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	changeEntity, err := client.Change.Create().SetTitle("transactional approval").SetDescription("test").
		SetStatus("draft").SetCreatedBy(creator.ID).SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	repo := NewEntRepository(client, db)
	require.NoError(t, repo.SubmitForApproval(ctx, changeEntity.ID, tenant.ID, []int{approver.ID}, "please review"))

	commands, err := client.OperationalCommand.Query().Where(
		operationalcommand.TenantIDEQ(tenant.ID),
		operationalcommand.AggregateTypeEQ("change"),
		operationalcommand.AggregateIDEQ(changeEntity.ID),
	).All(ctx)
	require.NoError(t, err)
	require.Len(t, commands, 1)
	require.Equal(t, commandbus.CommandDeliverNotification, commands[0].CommandType)
	require.Equal(t, "change_approval_required", commands[0].Payload["type"])
	require.Equal(t, "change", commands[0].Payload["resourceType"])
}
