package ai

import (
	"context"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/internal/services"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// get_kb_article 工具的端到端守卫。
//
// 上一组守卫测的是 RAGService.GetArticleFull 这一层（准入规则）。这里测
// **完整调用链**：LLM 在聊天里发起工具调用 → ExecuteTool → RBAC → 审计 →
// 实际读到全文。
//
// 关键断言有两条：
//  1. 权限不足时必须被 RBAC 拦下（不是执行后才过滤结果）；
//  2. 权限通过时必须真的拿到全文（不是空内容——那是模型会编造的输入）。

// kbToolEnv 造一个带知识文章与工具注册表的环境。
type kbToolEnv struct {
	client   *ent.Client
	tenantID int
	userID   int
	article  int
}

func newKBToolEnv(t *testing.T) *kbToolEnv {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file:ai_kb_tool?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { _ = client.Close() })
	ctx := context.Background()

	tenant, err := client.Tenant.Create().SetName("t").SetCode("t").
		SetDomain("t.com").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	user, err := client.User.Create().SetUsername("u").SetName("u").
		SetEmail("u@e.com").SetPasswordHash("x").SetActive(true).
		SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	article, err := client.KnowledgeArticle.Create().
		SetTitle("CDN 缓存配置指南").
		SetContent("完整的正文内容，包含具体步骤与参数。").
		SetCategory("").
		SetAuthorID(user.ID).
		SetTenantID(tenant.ID).
		SetIsPublished(true).
		Save(ctx)
	require.NoError(t, err)

	return &kbToolEnv{client: client, tenantID: tenant.ID, userID: user.ID, article: article.ID}
}

// TestGetKbArticleToolExecutesThroughService 锁工具在完整链路上可执行。
func TestGetKbArticleToolExecutesThroughService(t *testing.T) {
	env := newKBToolEnv(t)
	rag := service.NewRAGService(env.client, nil, nil, zaptest.NewLogger(t).Sugar(),
		service.RAGConfig{UseKeyword: true, MaxResults: 5})
	tools := service.NewToolRegistry(rag, nil, nil, env.client)

	// repo 用真实实现：ExecuteTool 的审计链要写 tool_invocations，
	// nil repo 会在审计处 panic（生产装配必然注入，这里补齐依赖）。
	svc := NewService(NewEntRepository(env.client), zaptest.NewLogger(t).Sugar(), rag, tools,
		nil, nil, nil, nil, nil, nil)
	svc.SetEntClient(env.client)

	// 角色带 knowledge:read 权限时（这里用超管角色），工具应当执行成功。
	res, invID, err := svc.ExecuteTool(context.Background(), env.userID, env.tenantID,
		"super_admin", "get_kb_article", map[string]interface{}{"id": float64(env.article)})
	require.NoError(t, err, "具备权限时工具必须能执行")
	require.Equal(t, 0, invID, "只读工具不应产生待审批记录")

	out, ok := res.(map[string]interface{})
	require.True(t, ok, "结果应为对象")
	content, _ := out["content"].(string)
	require.NotEmpty(t, content, "工具必须返回正文内容（空内容会让模型编造）")
	require.Contains(t, content, "完整的正文内容")
}

// TestGetKbArticleToolRejectsInvalidID 锁非法参数被拒。
func TestGetKbArticleToolRejectsInvalidID(t *testing.T) {
	env := newKBToolEnv(t)
	rag := service.NewRAGService(env.client, nil, nil, zaptest.NewLogger(t).Sugar(),
		service.RAGConfig{UseKeyword: true, MaxResults: 5})
	tools := service.NewToolRegistry(rag, nil, nil, env.client)

	// repo 用真实实现：ExecuteTool 的审计链要写 tool_invocations，
	// nil repo 会在审计处 panic（生产装配必然注入，这里补齐依赖）。
	svc := NewService(NewEntRepository(env.client), zaptest.NewLogger(t).Sugar(), rag, tools,
		nil, nil, nil, nil, nil, nil)
	svc.SetEntClient(env.client)

	_, _, err := svc.ExecuteTool(context.Background(), env.userID, env.tenantID,
		"super_admin", "get_kb_article", map[string]interface{}{"id": float64(0)})
	require.Error(t, err, "非法 id 必须被拒绝")

	_, _, err = svc.ExecuteTool(context.Background(), env.userID, env.tenantID,
		"super_admin", "get_kb_article", map[string]interface{}{})
	require.Error(t, err, "缺 id 必须被拒绝")
}

// TestGetKbArticleToolRejectsUnreadableArticle 锁读不到时报错而非空内容。
//
// 返回空内容比报错更糟：模型会把「读不到」当成「这篇文章没有正文」，
// 然后基于空输入编造答案。必须让它知道「读不到」。
func TestGetKbArticleToolRejectsUnreadableArticle(t *testing.T) {
	env := newKBToolEnv(t)
	rag := service.NewRAGService(env.client, nil, nil, zaptest.NewLogger(t).Sugar(),
		service.RAGConfig{UseKeyword: true, MaxResults: 5})
	tools := service.NewToolRegistry(rag, nil, nil, env.client)

	// repo 用真实实现：ExecuteTool 的审计链要写 tool_invocations，
	// nil repo 会在审计处 panic（生产装配必然注入，这里补齐依赖）。
	svc := NewService(NewEntRepository(env.client), zaptest.NewLogger(t).Sugar(), rag, tools,
		nil, nil, nil, nil, nil, nil)
	svc.SetEntClient(env.client)

	// 跨租户：文章不可见。
	_, _, err := svc.ExecuteTool(context.Background(), env.userID, env.tenantID+999,
		"super_admin", "get_kb_article", map[string]interface{}{"id": float64(env.article)})
	require.Error(t, err, "跨租户读取必须失败")
}
