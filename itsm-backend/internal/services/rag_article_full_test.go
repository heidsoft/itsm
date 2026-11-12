package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/internal/knowledgeaccess"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 知识全文下钻（渐进式披露）的守卫测试。
//
// 渐进式披露的核心承诺：**读与检索走同一套准入规则**。
// 如果两者口径不同，就会出现两种坏结果：
//   - 检索命中了但工具读不到 → 模型困惑、反复重试；
//   - 工具能读到但检索永远命中不了 → 那是一条绕过权限的读通道。
//
// 所以这些断言刻意用**同一篇文章**分别走检索与全文两条路，比对结果是否
// 一致。

// TestGetArticleFull_MatchesRetrievalVisibility 锁「检索能看到的，全文也能读到」。
func TestGetArticleFull_MatchesRetrievalVisibility(t *testing.T) {
	client := setupRAGTestClient(t)
	defer client.Close()

	ctx := context.Background()
	tenant, err := createTestTenant(ctx, client)
	require.NoError(t, err)
	user, err := createTestUser(ctx, client, tenant.ID)
	require.NoError(t, err)

	article, err := createKnowledgeArticle(ctx, client, tenant.ID, user.ID,
		"CDN 缓存配置指南", strings.Repeat("缓存配置步骤。", 100), "网络")
	require.NoError(t, err)

	rag := NewRAGService(client, nil, nil, zaptest.NewLogger(t).Sugar(),
		RAGConfig{UseKeyword: true, MaxResults: 5})

	// 检索命中。
	hits, err := rag.Ask(ctx, tenant.ID, "CDN 缓存配置指南", 5)
	require.NoError(t, err)
	require.Len(t, hits, 1, "检索应当命中")

	// 全文读得到，且是完整正文（不是节选）。
	full, err := rag.GetArticleFull(ctx, tenant.ID, article.ID)
	require.NoError(t, err)
	content, _ := full["content"].(string)
	require.Equal(t, article.Content, content, "全文必须返回完整正文")
	require.Equal(t, article.Title, full["title"])
}

// TestGetArticleFull_RejectsUnpublishedAndDeleted 锁草稿与软删不可读。
//
// 草稿不进 RAG 结果，草稿也不该被 AI 读到——否则「检索看不到但工具读得到」
// 就是一条绕过评审的读通道。
func TestGetArticleFull_RejectsUnpublishedAndDeleted(t *testing.T) {
	client := setupRAGTestClient(t)
	defer client.Close()

	ctx := context.Background()
	tenant, err := createTestTenant(ctx, client)
	require.NoError(t, err)
	user, err := createTestUser(ctx, client, tenant.ID)
	require.NoError(t, err)

	// 草稿。
	draft, err := createKnowledgeArticle(ctx, client, tenant.ID, user.ID,
		"草稿文章", "内容", "网络")
	require.NoError(t, err)
	_, err = client.KnowledgeArticle.UpdateOneID(draft.ID).SetIsPublished(false).Save(ctx)
	require.NoError(t, err)

	// 已发布但软删。
	deleted, err := createKnowledgeArticle(ctx, client, tenant.ID, user.ID,
		"已删除文章", "内容", "网络")
	require.NoError(t, err)
	_, err = client.KnowledgeArticle.UpdateOneID(deleted.ID).SetDeletedAt(nowTime()).Save(ctx)
	require.NoError(t, err)

	rag := NewRAGService(client, nil, nil, zaptest.NewLogger(t).Sugar(),
		RAGConfig{UseKeyword: true, MaxResults: 5})

	_, err = rag.GetArticleFull(ctx, tenant.ID, draft.ID)
	require.Error(t, err, "草稿不得被全文工具读到")

	_, err = rag.GetArticleFull(ctx, tenant.ID, deleted.ID)
	require.Error(t, err, "软删文章不得被全文工具读到")
}

// TestGetArticleFull_RejectsCrossTenant 锁跨租户不可读。
func TestGetArticleFull_RejectsCrossTenant(t *testing.T) {
	client := setupRAGTestClient(t)
	defer client.Close()

	ctx := context.Background()
	tenant, err := createTestTenant(ctx, client)
	require.NoError(t, err)
	user, err := createTestUser(ctx, client, tenant.ID)
	require.NoError(t, err)

	article, err := createKnowledgeArticle(ctx, client, tenant.ID, user.ID,
		"租户私有文章", "内容", "网络")
	require.NoError(t, err)

	rag := NewRAGService(client, nil, nil, zaptest.NewLogger(t).Sugar(),
		RAGConfig{UseKeyword: true, MaxResults: 5})

	_, err = rag.GetArticleFull(ctx, tenant.ID+999, article.ID)
	require.Error(t, err, "跨租户不得读到全文")
}

// TestGetArticleFull_InvalidInput 锁非法参数不 panic。
func TestGetArticleFull_InvalidInput(t *testing.T) {
	client := setupRAGTestClient(t)
	defer client.Close()

	rag := NewRAGService(client, nil, nil, zaptest.NewLogger(t).Sugar(),
		RAGConfig{UseKeyword: true})

	_, err := rag.GetArticleFull(context.Background(), 0, 1)
	require.Error(t, err)
	_, err = rag.GetArticleFull(context.Background(), 1, 0)
	require.Error(t, err)
	_, err = rag.GetArticleFull(context.Background(), 1, -5)
	require.Error(t, err)
}

// TestSearchResultsCarryTruncationSignal 锁渐进式披露的信号位。
//
// 没有 truncated/contentLength，模型无从判断片段是不是完整的——它会拿着
// 半截内容当作全文来答（而且不会说自己不确定）。
func TestSearchResultsCarryTruncationSignal(t *testing.T) {
	client := setupRAGTestClient(t)
	defer client.Close()

	ctx := context.Background()
	tenant, err := createTestTenant(ctx, client)
	require.NoError(t, err)
	user, err := createTestUser(ctx, client, tenant.ID)
	require.NoError(t, err)

	long := strings.Repeat("很长的正文内容。", 200) // 远超 snippetLimit
	_, err = createKnowledgeArticle(ctx, client, tenant.ID, user.ID, "长文标题", long, "网络")
	require.NoError(t, err)
	short := "短正文。"
	_, err = createKnowledgeArticle(ctx, client, tenant.ID, user.ID, "短文标题", short, "网络")
	require.NoError(t, err)

	rag := NewRAGService(client, nil, nil, zaptest.NewLogger(t).Sugar(),
		RAGConfig{UseKeyword: true, MaxResults: 5})

	hits, err := rag.Ask(ctx, tenant.ID, "长文标题", 5)
	require.NoError(t, err)
	require.Len(t, hits, 1)
	truncated, ok := hits[0]["truncated"].(bool)
	require.True(t, ok, "检索结果必须带 truncated 标记")
	require.True(t, truncated, "长文必须标为已截断")
	length, ok := hits[0]["contentLength"].(int)
	require.True(t, ok, "检索结果必须带 contentLength")
	require.Equal(t, len([]rune(long)), length)

	hits2, err := rag.Ask(ctx, tenant.ID, "短文标题", 5)
	require.NoError(t, err)
	require.Len(t, hits2, 1)
	truncated2, _ := hits2[0]["truncated"].(bool)
	require.False(t, truncated2, "短文不应标为已截断")
}

// TestGetKbArticleToolRegistered 锁工具已注册且是只读。
//
// 只读很重要：ChatStream 只放行只读工具，写工具要走审批流。一个能读
// 全文的工具若被误标成非只读，会让 AI 在聊天链路里直接调它。
func TestGetKbArticleToolRegistered(t *testing.T) {
	registry := NewToolRegistry(nil, nil, nil, nil)
	td := registry.GetTool("get_kb_article")
	require.NotNil(t, td, "get_kb_article 工具必须注册")
	require.True(t, td.ReadOnly, "get_kb_article 必须是只读工具")
	require.Equal(t, "knowledge", td.Resource)
	require.Equal(t, "read", td.Action)

	props, ok := td.ArgsSchema["properties"].(map[string]interface{})
	require.True(t, ok, "参数 schema 必须有 properties")
	_, ok = props["id"]
	require.True(t, ok, "必须接受文章 id 参数")
}

// TestGetArticleFullMatchesKnowledgeGuard 锁分类可见性一致性。
//
// 有 knowledgeGuard 时，无权分类的文章：检索读不到，全文也必须读不到。
func TestGetArticleFullMatchesKnowledgeGuard(t *testing.T) {
	client := setupRAGTestClient(t)
	defer client.Close()

	ctx := context.Background()
	tenant, err := createTestTenant(ctx, client)
	require.NoError(t, err)
	user, err := createTestUser(ctx, client, tenant.ID)
	require.NoError(t, err)

	article, err := createKnowledgeArticle(ctx, client, tenant.ID, user.ID,
		"受限分类文章", "内容", "受限分类")
	require.NoError(t, err)

	rag := NewRAGService(client, nil, nil, zaptest.NewLogger(t).Sugar(),
		RAGConfig{UseKeyword: true, MaxResults: 5})
	// 未装配 guard 时（默认放行），能读到——这是基线行为。
	_, err = rag.GetArticleFull(ctx, tenant.ID, article.ID)
	require.NoError(t, err)

	// 装配一个「全部受限」的 guard：无权分类应一律读不到。
	rag.SetKnowledgeGuard(newRestrictedCategoryGuard(t, client, tenant.ID))
	_, err = rag.GetArticleFull(ctx, tenant.ID, article.ID)
	require.Error(t, err, "受限分类在无权时不得被全文工具读到")

	// 检索侧同样读不到——两条路口径一致。
	hits, err := rag.Ask(ctx, tenant.ID, "受限分类文章", 5)
	require.NoError(t, err)
	require.Empty(t, hits, "受限分类在无权时检索也不得命中")
}

// TestViewerInjectionIsOptional 锁无 viewer 时的行为。
//
// 无身份上下文按匿名处理：受限分类一律不可见，普通分类可见。
func TestViewerInjectionIsOptional(t *testing.T) {
	client := setupRAGTestClient(t)
	defer client.Close()

	ctx := context.Background()
	tenant, err := createTestTenant(ctx, client)
	require.NoError(t, err)
	user, err := createTestUser(ctx, client, tenant.ID)
	require.NoError(t, err)

	article, err := createKnowledgeArticle(ctx, client, tenant.ID, user.ID,
		"普通文章", "内容", "")
	require.NoError(t, err)

	rag := NewRAGService(client, nil, nil, zaptest.NewLogger(t).Sugar(),
		RAGConfig{UseKeyword: true, MaxResults: 5})
	rag.SetKnowledgeGuard(newRestrictedCategoryGuard(t, client, tenant.ID))

	// 空分类在「全部受限」策略下仍可见（与检索的 CategoryEQ("") 分支一致）。
	_, err = rag.GetArticleFull(ctx, tenant.ID, article.ID)
	require.NoError(t, err, "空分类文章在受限策略下仍可读")

	// 带上 viewer 也不改变空分类的可见性。
	viewerCtx := knowledgeaccess.WithViewer(ctx, knowledgeaccess.Viewer{UserID: user.ID, Role: "agent"})
	_, err = rag.GetArticleFull(viewerCtx, tenant.ID, article.ID)
	require.NoError(t, err)
}

// newRestrictedCategoryGuard 造一个把指定分类纳管的可见性守卫。
//
// 受限分类来自 permissions 表（resource=knowledge_category、action=read:<cat>），
// 所以这里造的是真实数据而不是替身——判据是「真实 SQL 下两条路口径一致」。
func newRestrictedCategoryGuard(t *testing.T, client *ent.Client, tenantID int) *knowledgeaccess.Guard {
	t.Helper()
	ctx := context.Background()
	_, err := client.Permission.Create().
		SetTenantID(tenantID).
		SetResource("knowledge_category").
		SetAction("read:受限分类").
		SetCode("knowledge_category:read:受限分类").
		SetName("受限分类读取").
		Save(ctx)
	require.NoError(t, err)
	return knowledgeaccess.NewGuard(client, zaptest.NewLogger(t).Sugar())
}

// nowTime 返回当前时间（软删标记用）。
func nowTime() time.Time { return time.Now() }
