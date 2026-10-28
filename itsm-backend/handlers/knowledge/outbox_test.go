package knowledge

import (
	"context"
	"testing"

	connectorVector "itsm-backend/connector/vector"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/knowledgearticle"
	"itsm-backend/ent/operationalcommand"
	"itsm-backend/internal/commandbus"
	"itsm-backend/service"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type outboxTestEmbedder struct{}

func (outboxTestEmbedder) Embed(string) ([]float32, error) { return []float32{0.1, 0.2}, nil }

type outboxTestVectorStore struct{}

func (outboxTestVectorStore) Search(context.Context, connectorVector.SearchRequest) (connectorVector.SearchResponse, error) {
	return connectorVector.SearchResponse{}, nil
}

func (outboxTestVectorStore) Insert(context.Context, connectorVector.InsertRequest) error {
	return nil
}
func (outboxTestVectorStore) Delete(context.Context, []string) error { return nil }
func (outboxTestVectorStore) Ping(context.Context) error             { return nil }
func (outboxTestVectorStore) Close() error                           { return nil }

func newVectorOutboxService(t *testing.T, dsn string, embedder service.Embedder) (*Service, *ent.Client) {
	t.Helper()
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { _ = client.Close() })
	svc := NewService(NewEntRepository(client), zaptest.NewLogger(t).Sugar())
	svc.SetEntClient(client)
	rag := service.NewRAGService(nil, nil, embedder, zaptest.NewLogger(t).Sugar(), service.RAGConfig{UseVector: true})
	rag.SetVectorStore(outboxTestVectorStore{})
	svc.SetRAG(rag)
	return svc, client
}

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

func TestCreateArticle_EnqueuesSyncWhenVectorIndexReady(t *testing.T) {
	ctx := context.Background()
	svc, client := newVectorOutboxService(t,
		"file:knowledge_outbox_ready?mode=memory&cache=shared&_fk=1", outboxTestEmbedder{})

	created, err := svc.CreateArticle(ctx, &Article{
		Title: "runbook", Content: "restart safely", Category: "操作指南",
		AuthorID: 1, TenantID: 7, IsPublished: true,
	})
	require.NoError(t, err)

	cmd, err := client.OperationalCommand.Query().Where(
		operationalcommand.TenantIDEQ(7),
		operationalcommand.AggregateIDEQ(created.ID),
		operationalcommand.CommandTypeEQ(commandbus.CommandSyncKnowledgeVector),
	).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, vectorIndexSync, cmd.Payload["action"])
}

// 向量后端已装配、但 embedding 未配置时，发布不能制造注定失败的同步任务：
// worker 会重试到死信，而问题其实是部署配置缺失。
func TestCreateArticle_SkipsSyncWhenEmbedderUnconfigured(t *testing.T) {
	ctx := context.Background()
	svc, client := newVectorOutboxService(t,
		"file:knowledge_outbox_blocked?mode=memory&cache=shared&_fk=1", &service.OpenAIEmbedder{})

	created, err := svc.CreateArticle(ctx, &Article{
		Title: "runbook", Content: "restart safely", Category: "操作指南",
		AuthorID: 1, TenantID: 7, IsPublished: true,
	})
	require.NoError(t, err)

	n, err := client.OperationalCommand.Query().Where(
		operationalcommand.TenantIDEQ(7),
		operationalcommand.AggregateIDEQ(created.ID),
		operationalcommand.CommandTypeEQ(commandbus.CommandSyncKnowledgeVector),
	).Count(ctx)
	require.NoError(t, err)
	require.Zero(t, n)
}

// 已发布文章在 embedding 不可用时被修改：同步必然失败，旧索引却仍会被检索命中，
// 因此入删除兜底——宁可暂时搜不到，也不返回文章的上一个版本。
func TestUpdateArticle_PublishedWithoutEmbedderInvalidatesIndex(t *testing.T) {
	ctx := context.Background()
	svc, client := newVectorOutboxService(t,
		"file:knowledge_outbox_blocked_update?mode=memory&cache=shared&_fk=1", &service.OpenAIEmbedder{})

	created, err := svc.CreateArticle(ctx, &Article{
		Title: "runbook", Content: "restart safely", Category: "操作指南",
		AuthorID: 1, TenantID: 7, IsPublished: true,
	})
	require.NoError(t, err)

	// 模拟一篇已通过评审的发布文章：发布门禁要求 approved，
	// 这里直接置状态而非走 approve 流程，避免额外的同步任务干扰单命令断言。
	_, err = client.KnowledgeArticle.UpdateOneID(created.ID).
		SetReviewStatus(knowledgearticle.ReviewStatus(ReviewStatusApproved)).
		Save(ctx)
	require.NoError(t, err)

	created.Title = "runbook v2"
	_, err = svc.UpdateArticle(ctx, created)
	require.NoError(t, err)

	cmd, err := client.OperationalCommand.Query().Where(
		operationalcommand.TenantIDEQ(7),
		operationalcommand.AggregateIDEQ(created.ID),
		operationalcommand.CommandTypeEQ(commandbus.CommandSyncKnowledgeVector),
	).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, vectorIndexDelete, cmd.Payload["action"])
}
