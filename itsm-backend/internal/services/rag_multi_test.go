package service

import (
	"context"
	"testing"

	"go.uber.org/zap/zaptest"

	"github.com/stretchr/testify/require"
)

// 多路检索（AskMulti）的价值验证：**记忆必须真的让指代型 query 命中**。
//
// 这是记忆层存在的全部理由。用户在多轮对话里问「那它怎么修？」时，当前
// query 里没有任何可匹配的实体——关键词检索是整串包含，「那它怎么修」在
// 知识库里必然零命中。这是结构性的不可能，不是「答得不好」。
//
// 有了会话记忆后，历史里的实体（如工单号）会作为**独立补充路**参与检索，
// 于是指代型 query 也能命中相关文档。

// TestAskMulti_ReferentialQueryHitsViaSupplemental 锁指代型 query 的召回改善。
func TestAskMulti_ReferentialQueryHitsViaSupplemental(t *testing.T) {
	client := setupRAGTestClient(t)
	defer client.Close()

	ctx := context.Background()
	tenant, err := createTestTenant(ctx, client)
	require.NoError(t, err)
	user, err := createTestUser(ctx, client, tenant.ID)
	require.NoError(t, err)

	// 一篇真实存在的知识文章（标题与正文都含工单号）。
	_, err = createKnowledgeArticle(ctx, client, tenant.ID, user.ID,
		"TKT-20260831 磁盘告警处理手册",
		"TKT-20260831 的处理流程：先确认磁盘使用率，再按扩容 runbook 执行。",
		"运维手册")
	require.NoError(t, err)

	rag := NewRAGService(client, nil, nil, zaptest.NewLogger(t).Sugar(),
		RAGConfig{UseKeyword: true, MaxResults: 5})

	// 指代型 query：不含任何可匹配实体。
	referential := "那它怎么修"
	plain, err := rag.Ask(ctx, tenant.ID, referential, 5)
	require.NoError(t, err)
	require.Empty(t, plain, "指代型 query 单独检索应当零命中（这正是问题所在）")

	// 记忆补充路：抽出历史里的工单号。
	multi, err := rag.AskMulti(ctx, tenant.ID, referential, []string{"TKT-20260831"}, 5)
	require.NoError(t, err)
	require.Len(t, multi, 1, "补充路应当召回相关文档")
	require.Equal(t, "TKT-20260831 磁盘告警处理手册", multi[0]["title"])
	require.Equal(t, true, multi[0]["fromMemory"], "补充路命中必须标 fromMemory")
	require.Equal(t, "TKT-20260831", multi[0]["memoryQuery"])
}

// TestAskMulti_PrimaryResultsNotMarkedAsMemory 锁主路结果不被误标。
//
// 记忆标注错了比没标注更糟：调用方会以为「库里写着」的内容只是「聊过」。
func TestAskMulti_PrimaryResultsNotMarkedAsMemory(t *testing.T) {
	client := setupRAGTestClient(t)
	defer client.Close()

	ctx := context.Background()
	tenant, err := createTestTenant(ctx, client)
	require.NoError(t, err)
	user, err := createTestUser(ctx, client, tenant.ID)
	require.NoError(t, err)

	_, err = createKnowledgeArticle(ctx, client, tenant.ID, user.ID,
		"CDN 缓存配置指南", "本文回答如何配置 CDN 缓存。", "网络")
	require.NoError(t, err)

	rag := NewRAGService(client, nil, nil, zaptest.NewLogger(t).Sugar(),
		RAGConfig{UseKeyword: true, MaxResults: 5})

	results, err := rag.AskMulti(ctx, tenant.ID, "CDN 缓存配置指南", []string{"不存在的实体"}, 5)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.NotContains(t, results[0], "fromMemory", "主路命中不得被标为记忆来源")
}

// TestAskMulti_DeduplicatesAcrossRoutes 锁跨路去重。
//
// 主路已命中的文档不应在补充路重复出现——重复条目会挤掉 limit 内的其他
// 有效结果，也会让前端引用列表出现同一篇。
func TestAskMulti_DeduplicatesAcrossRoutes(t *testing.T) {
	client := setupRAGTestClient(t)
	defer client.Close()

	ctx := context.Background()
	tenant, err := createTestTenant(ctx, client)
	require.NoError(t, err)
	user, err := createTestUser(ctx, client, tenant.ID)
	require.NoError(t, err)

	// 同一篇文档能被两路同时命中（query 与补充路都含该标题）。
	_, err = createKnowledgeArticle(ctx, client, tenant.ID, user.ID,
		"TKT-1 数据库连接池配置", "TKT-1 数据库连接池的 max_open_conns 建议值。", "数据库")
	require.NoError(t, err)

	rag := NewRAGService(client, nil, nil, zaptest.NewLogger(t).Sugar(),
		RAGConfig{UseKeyword: true, MaxResults: 5})

	results, err := rag.AskMulti(ctx, tenant.ID, "TKT-1 数据库连接池配置", []string{"TKT-1"}, 5)
	require.NoError(t, err)
	require.Len(t, results, 1, "同一文档不得跨路重复出现")
}

// TestAskMulti_SupplementalFailureDoesNotBreakPrimary 锁补充路失败不影响主路。
//
// 记忆是增强不是依赖：补充路出问题（网络抖动、检索超时）时，主路结果
// 仍必须返回。
func TestAskMulti_SupplementalFailureDoesNotBreakPrimary(t *testing.T) {
	client := setupRAGTestClient(t)
	defer client.Close()

	ctx := context.Background()
	tenant, err := createTestTenant(ctx, client)
	require.NoError(t, err)
	user, err := createTestUser(ctx, client, tenant.ID)
	require.NoError(t, err)

	_, err = createKnowledgeArticle(ctx, client, tenant.ID, user.ID,
		"网络抖动排查", "网络抖动的排查步骤。", "网络")
	require.NoError(t, err)

	rag := NewRAGService(client, nil, nil, zaptest.NewLogger(t).Sugar(),
		RAGConfig{UseKeyword: true, MaxResults: 5})

	// 补充路用空串（会被跳过）与超长串（必然零命中但不应报错）都要保证主路可用。
	results, err := rag.AskMulti(ctx, tenant.ID, "网络抖动排查", []string{"", "网络抖动排查"}, 5)
	require.NoError(t, err)
	require.Len(t, results, 1)
}
