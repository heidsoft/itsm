package change

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/operationalcommand"
	"itsm-backend/internal/commandbus"

	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
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
			is_required BOOLEAN NOT NULL, approval_type TEXT NOT NULL DEFAULT 'serial', threshold INTEGER NOT NULL DEFAULT 1, created_at DATETIME
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
	require.NoError(t, repo.SubmitForApproval(ctx, changeEntity.ID, tenant.ID, []ApprovalLevelPlan{{Level: 1, ApprovalType: "serial", Threshold: 1, Required: true, ApproverIDs: []int{approver.ID}}}, "please review"))

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

func TestSubmitChangeCommitsBPMNAndBusinessStateAtomically(t *testing.T) {
	ctx := context.Background()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", strings.ReplaceAll(t.Name(), "/", "-"))
	db, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
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
			is_required BOOLEAN NOT NULL, approval_type TEXT NOT NULL DEFAULT 'serial', threshold INTEGER NOT NULL DEFAULT 1, created_at DATETIME
		);
	`)
	require.NoError(t, err)

	tenant, err := client.Tenant.Create().SetName("atomic tenant").SetCode("atomic-change").
		SetDomain("atomic-change.example.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	actor, err := client.User.Create().SetUsername("atomic-actor").SetEmail("atomic@example.com").
		SetName("Atomic Actor").SetPasswordHash("hash").SetRole("manager").SetActive(true).SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	newDraft := func(title string) *ent.Change {
		item, createErr := client.Change.Create().SetTitle(title).SetDescription("test").
			SetStatus("draft").SetCreatedBy(actor.ID).SetTenantID(tenant.ID).Save(ctx)
		require.NoError(t, createErr)
		return item
	}
	repo := NewEntRepository(client, db)
	svc := NewService(repo, client, zaptest.NewLogger(t).Sugar(), nil)

	t.Run("success commits both", func(t *testing.T) {
		item := newDraft("atomic success")
		taskID := createChangeBridgeProcessFixture(t, client, tenant.ID, "atomic-success", fmt.Sprintf("change:%d", item.ID), actor.ID)
		_, err := svc.SubmitChange(ctx, item.ID, tenant.ID, actor.ID, &dto.SubmitChangeRequest{ApproverIDs: []int{actor.ID}})
		require.NoError(t, err)
		stored, err := client.Change.Get(ctx, item.ID)
		require.NoError(t, err)
		require.Equal(t, "pending", stored.Status)
		task, err := client.ProcessTask.Get(ctx, taskID)
		require.NoError(t, err)
		require.Equal(t, "completed", task.Status)
	})

	t.Run("business update failure rolls back BPMN", func(t *testing.T) {
		item := newDraft("atomic rollback")
		taskID := createChangeBridgeProcessFixture(t, client, tenant.ID, "atomic-rollback", fmt.Sprintf("change:%d", item.ID), actor.ID)
		_, err := db.ExecContext(ctx, fmt.Sprintf(`
			CREATE TRIGGER fail_change_submit_%d BEFORE UPDATE OF status ON changes
			WHEN OLD.id = %d BEGIN SELECT RAISE(ABORT, 'forced status update failure'); END;
		`, item.ID, item.ID))
		require.NoError(t, err)
		_, err = svc.SubmitChange(ctx, item.ID, tenant.ID, actor.ID, &dto.SubmitChangeRequest{ApproverIDs: []int{actor.ID}})
		require.Error(t, err)
		stored, err := client.Change.Get(ctx, item.ID)
		require.NoError(t, err)
		require.Equal(t, "draft", stored.Status)
		task, err := client.ProcessTask.Get(ctx, taskID)
		require.NoError(t, err)
		require.Equal(t, "assigned", task.Status)
	})

	t.Run("BPMN failure leaves change draft", func(t *testing.T) {
		item := newDraft("BPMN failure rollback")
		taskID := createChangeBridgeProcessFixture(t, client, tenant.ID, "bpmn-failure", fmt.Sprintf("change:%d", item.ID), actor.ID+999)
		_, err := svc.SubmitChange(ctx, item.ID, tenant.ID, actor.ID, &dto.SubmitChangeRequest{ApproverIDs: []int{actor.ID}})
		require.Error(t, err)
		stored, err := client.Change.Get(ctx, item.ID)
		require.NoError(t, err)
		require.Equal(t, "draft", stored.Status)
		task, err := client.ProcessTask.Get(ctx, taskID)
		require.NoError(t, err)
		require.Equal(t, "assigned", task.Status)
	})
}
