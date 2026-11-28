package change

import (
	"context"
	"strconv"
	"testing"

	"itsm-backend/ent/enttest"
	"itsm-backend/ent/operationalcommand"
	"itsm-backend/internal/commandbus"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestEntRepositoryCreateWithWorkflowCommandIsAtomic(t *testing.T) {
	client := enttest.Open(t, dialect.SQLite, "file:change-outbox?mode=memory&cache=shared&_fk=1")
	ctx := context.Background()
	tenant, err := client.Tenant.Create().SetName("变更租户").SetCode("changes").SetDomain("changes.example.test").Save(ctx)
	require.NoError(t, err)
	dept, err := client.Department.Create().SetName("运维").SetCode("ops").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	actor, err := client.User.Create().SetUsername("creator").SetName("创建人").
		SetEmail("creator@example.test").SetPasswordHash("test-hash").SetTenantID(tenant.ID).
		SetDepartmentID(dept.ID).Save(ctx)
	require.NoError(t, err)
	repo := NewEntRepository(client, nil)
	created, err := repo.CreateWithWorkflowCommand(context.Background(), &Change{
		Title: "生产变更", Description: "升级", Type: "normal", Status: "draft",
		Priority: "medium", ImpactScope: "medium", RiskLevel: "medium",
		CreatedBy: actor.ID, TenantID: tenant.ID, ImplementationPlan: "灰度", RollbackPlan: "回滚",
	})
	require.NoError(t, err)
	require.Positive(t, created.ID)

	cmd, err := client.OperationalCommand.Query().
		Where(operationalcommand.TenantIDEQ(tenant.ID), operationalcommand.AggregateTypeEQ("change"), operationalcommand.AggregateIDEQ(created.ID)).
		Only(context.Background())
	require.NoError(t, err)
	require.Equal(t, "workflow.start", cmd.CommandType)
	require.Equal(t, "pending", cmd.Status)
	require.Equal(t, strconv.Itoa(dept.ID), cmd.Payload[commandbus.ChangeRoutingDepartmentKey])
}
