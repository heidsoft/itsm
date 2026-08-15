package service

import (
	"context"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// ==================== RAG 可见性过滤端到端测试 ====================
// 覆盖 AGENTS.md 规则：RAG 在检索前必须尊重租户隔离、发布状态（可见性）与软删除。
// 说明：sqlite 不支持 pgvector，vector 路径在 enrichment 阶段做同样的
// IsPublished/DeletedAt 过滤（见 rag_service.go vectorSearch）；此处用
// keyword-only 配置走完整 Ask 链路验证过滤语义。

// createVisibilityArticle 创建指定发布状态的文章。
func createVisibilityArticle(t *testing.T, ctx context.Context, client *ent.Client, tenantID, authorID int, title string, published bool) *ent.KnowledgeArticle {
	t.Helper()
	a, err := client.KnowledgeArticle.Create().
		SetTitle(title).
		SetContent("内容：" + title).
		SetCategory("故障处理").
		SetAuthorID(authorID).
		SetTenantID(tenantID).
		SetIsPublished(published).
		Save(ctx)
	require.NoError(t, err)
	return a
}

// newKeywordOnlyRAG 构建 keyword-only 的 RAG 服务（生产环境 vector 不可用时的降级路径）。
func newKeywordOnlyRAG(t *testing.T, client *ent.Client) *RAGService {
	t.Helper()
	return NewRAGService(client, nil, nil, zaptest.NewLogger(t).Sugar(), RAGConfig{
		UseVector:    false,
		UseKeyword:   true,
		HybridSearch: false,
		MaxResults:   10,
	})
}

// 回归：草稿（未发布）文章不得出现在 RAG 检索结果中。
// 修复前 keywordSearch 只过滤租户和软删除，草稿会泄漏到 /ai/rag/search。
func TestRAG_Ask_ExcludesDraftArticles(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("Tenant A").SetCode("rag-vis-a").SetDomain("a.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	user, err := client.User.Create().
		SetUsername("rag-vis-user").SetEmail("rag-vis@test.com").SetName("User").
		SetPasswordHash("hashed").SetRole("agent").SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	published := createVisibilityArticle(t, ctx, client, tenant.ID, user.ID, "VPN 连接故障排查", true)
	createVisibilityArticle(t, ctx, client, tenant.ID, user.ID, "VPN 内部草稿（不应被检索）", false)

	svc := newKeywordOnlyRAG(t, client)
	results, err := svc.Ask(ctx, tenant.ID, "VPN", 10)
	require.NoError(t, err)
	require.Len(t, results, 1, "草稿文章不得进入 RAG 检索结果")
	assert.Equal(t, published.ID, results[0]["id"])
	assert.Equal(t, "VPN 连接故障排查", results[0]["title"])
}

// 回归：租户隔离——租户 B 检索时不得看到租户 A 的文章（即使标题完全匹配）。
func TestRAG_Ask_TenantIsolation(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	ctx := context.Background()

	tenantA, err := client.Tenant.Create().
		SetName("Tenant A").SetCode("rag-iso-a").SetDomain("a.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	tenantB, err := client.Tenant.Create().
		SetName("Tenant B").SetCode("rag-iso-b").SetDomain("b.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	userA, err := client.User.Create().
		SetUsername("rag-iso-a").SetEmail("iso-a@test.com").SetName("A").
		SetPasswordHash("hashed").SetRole("agent").SetActive(true).SetTenantID(tenantA.ID).
		Save(ctx)
	require.NoError(t, err)
	userB, err := client.User.Create().
		SetUsername("rag-iso-b").SetEmail("iso-b@test.com").SetName("B").
		SetPasswordHash("hashed").SetRole("agent").SetActive(true).SetTenantID(tenantB.ID).
		Save(ctx)
	require.NoError(t, err)

	// 两个租户各有一篇标题相同的已发布文章
	createVisibilityArticle(t, ctx, client, tenantA.ID, userA.ID, "邮件服务器故障", true)
	tenantBArticle := createVisibilityArticle(t, ctx, client, tenantB.ID, userB.ID, "邮件服务器故障", true)

	svc := newKeywordOnlyRAG(t, client)
	results, err := svc.Ask(ctx, tenantB.ID, "邮件服务器", 10)
	require.NoError(t, err)
	require.Len(t, results, 1, "跨租户文章不得被检索到")
	assert.Equal(t, tenantBArticle.ID, results[0]["id"], "只应返回本租户的文章")
}

// 回归：软删除文章不得出现在 RAG 检索结果中。
func TestRAG_Ask_ExcludesSoftDeleted(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("Tenant A").SetCode("rag-del-a").SetDomain("a.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	user, err := client.User.Create().
		SetUsername("rag-del-user").SetEmail("del@test.com").SetName("User").
		SetPasswordHash("hashed").SetRole("agent").SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	alive := createVisibilityArticle(t, ctx, client, tenant.ID, user.ID, "数据库备份指南", true)
	deleted := createVisibilityArticle(t, ctx, client, tenant.ID, user.ID, "数据库备份（已删除）", true)
	_, err = client.KnowledgeArticle.UpdateOneID(deleted.ID).SetDeletedAt(time.Now()).Save(ctx)
	require.NoError(t, err)

	svc := newKeywordOnlyRAG(t, client)
	results, err := svc.Ask(ctx, tenant.ID, "数据库", 10)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, alive.ID, results[0]["id"])
}

// 回归：混合（hybrid）配置下（vector 不可用时降级 keyword），过滤语义保持一致。
func TestRAG_Ask_Hybrid_ExcludesDraft(t *testing.T) {
	client := enttest.Open(t, "sqlite3", testDSN())
	defer client.Close()
	ctx := context.Background()

	tenant, err := client.Tenant.Create().
		SetName("Tenant A").SetCode("rag-hyb-a").SetDomain("a.test").SetStatus("active").
		Save(ctx)
	require.NoError(t, err)
	user, err := client.User.Create().
		SetUsername("rag-hyb-user").SetEmail("hyb@test.com").SetName("User").
		SetPasswordHash("hashed").SetRole("agent").SetActive(true).SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	createVisibilityArticle(t, ctx, client, tenant.ID, user.ID, "网络设备配置手册", true)
	createVisibilityArticle(t, ctx, client, tenant.ID, user.ID, "网络设备配置（草稿）", false)

	svc := NewRAGService(client, nil, nil, zaptest.NewLogger(t).Sugar(), RAGConfig{
		UseVector:    true, // 请求 vector，但 vectors 为 nil → NewRAGService 自动禁用
		UseKeyword:   true,
		HybridSearch: true,
		MaxResults:   10,
	})
	results, err := svc.Ask(ctx, tenant.ID, "网络设备", 10)
	require.NoError(t, err)
	require.Len(t, results, 1, "hybrid 降级 keyword 时同样必须过滤草稿")
	assert.Equal(t, "网络设备配置手册", results[0]["title"])
}
