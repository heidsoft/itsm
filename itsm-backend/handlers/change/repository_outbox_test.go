package change

import (
	"context"
	"testing"

	"itsm-backend/ent/enttest"
	"itsm-backend/ent/operationalcommand"

	"entgo.io/ent/dialect"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestEntRepositoryCreateWithWorkflowCommandIsAtomic(t *testing.T) {
	client := enttest.Open(t, dialect.SQLite, "file:change-outbox?mode=memory&cache=shared&_fk=1")
	repo := NewEntRepository(client, nil)
	created, err := repo.CreateWithWorkflowCommand(context.Background(), &Change{
		Title: "生产变更", Description: "升级", Type: "normal", Status: "draft",
		Priority: "medium", ImpactScope: "medium", RiskLevel: "medium",
		CreatedBy: 1, TenantID: 7, ImplementationPlan: "灰度", RollbackPlan: "回滚",
	})
	require.NoError(t, err)
	require.Positive(t, created.ID)

	cmd, err := client.OperationalCommand.Query().
		Where(operationalcommand.TenantIDEQ(7), operationalcommand.AggregateTypeEQ("change"), operationalcommand.AggregateIDEQ(created.ID)).
		Only(context.Background())
	require.NoError(t, err)
	require.Equal(t, "workflow.start", cmd.CommandType)
	require.Equal(t, "pending", cmd.Status)
}
