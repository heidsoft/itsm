package knowledge

import (
	"context"
	"testing"

	"itsm-backend/ent/enttest"
	"itsm-backend/ent/operationalcommand"
	"itsm-backend/internal/commandbus"
	"itsm-backend/service"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestCreateArticle_EnqueuesVectorSyncInSameTransaction(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:knowledge_outbox?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	svc := NewService(NewEntRepository(client), zaptest.NewLogger(t).Sugar())
	svc.SetEntClient(client)
	// A non-nil RAG service activates the production outbox path. Its vector
	// dependencies are intentionally absent: this test verifies persistence,
	// not a provider call.
	svc.SetRAG(service.NewRAGService(nil, nil, nil, zaptest.NewLogger(t).Sugar(), service.RAGConfig{}))

	created, err := svc.CreateArticle(ctx, &Article{
		Title: "runbook", Content: "restart safely", Category: "操作指南",
		AuthorID: 1, TenantID: 7, IsPublished: true,
	})
	require.NoError(t, err)

	cmd, err := client.OperationalCommand.Query().Where(
		operationalcommand.TenantIDEQ(7),
		operationalcommand.CommandTypeEQ(commandbus.CommandSyncKnowledgeVector),
		operationalcommand.AggregateIDEQ(created.ID),
	).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "knowledge_article", cmd.AggregateType)
	require.Equal(t, vectorIndexSync, cmd.Payload["action"])
}
