package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"itsm-backend/ent"
	"itsm-backend/handlers/common/knowledgeaccess"
)

// ==================== 知识可引用性 L1（时效性）+ L2（权威性）集成测试 ====================
//
// 走真实 sqlite 内存库，覆盖 keywordSearch 的 SQL 过滤、后置过滤与权威性重排，
// 以及向量路径共用的 articleCitable 判定。

// createTimedArticle 创建带时效字段的测试文章。
func createTimedArticle(ctx context.Context, client *ent.Client, tenantID, authorID int,
	title, content, category string, published bool,
	validFrom, validUntil, lastReviewed *time.Time, intervalDays, authority int) (*ent.KnowledgeArticle, error) {
	b := client.KnowledgeArticle.Create().
		SetTitle(title).
		SetContent(content).
		SetCategory(category).
		SetAuthorID(authorID).
		SetTenantID(tenantID).
		SetIsPublished(published).
		SetReviewIntervalDays(intervalDays).
		SetAuthorityLevel(authority)
	if validFrom != nil {
		b = b.SetValidFrom(*validFrom)
	}
	if validUntil != nil {
		b = b.SetValidUntil(*validUntil)
	}
	if lastReviewed != nil {
		b = b.SetLastReviewedAt(*lastReviewed)
	}
	return b.Save(ctx)
}

func TestRAG_KeywordSearch_FiltersByFreshness(t *testing.T) {
	client := setupRAGTestClient(t)
	defer client.Close()

	ctx := context.Background()
	tenant, err := createTestTenant(ctx, client)
	require.NoError(t, err)
	user, err := createTestUser(ctx, client, tenant.ID)
	require.NoError(t, err)

	now := time.Now()
	articles := []struct {
		title    string
		content  string
		validTo  *time.Time
	}{
		// 永久有效：应被检索到
		{"VPN 配置指南", "VPN 连接配置步骤 vpn", nil},
		// 已失效：标题也含 VPN，必须被过滤
		{"旧版 VPN 手册", "旧版 vpn 配置已废弃", ptrTime(now.Add(-time.Hour))},
	}
	for _, a := range articles {
		_, err = createTimedArticle(ctx, client, tenant.ID, user.ID, a.title, a.content, "网络", true, nil, a.validTo, nil, 0, 0)
		require.NoError(t, err)
	}

	logger := zaptest.NewLogger(t).Sugar()
	svc := NewRAGService(client, nil, nil, logger, RAGConfig{UseKeyword: true, UseVector: false})

	results, err := svc.keywordSearch(ctx, tenant.ID, "VPN", 10)
	require.NoError(t, err)

	titles := make([]string, 0, len(results))
	for _, r := range results {
		titles = append(titles, r["title"].(string))
	}
	assert.NotContains(t, titles, "旧版 VPN 手册", "已失效文章不得进入检索结果")
	assert.Contains(t, titles, "VPN 配置指南")
}

func TestRAG_KeywordSearch_FiltersReviewOverdue(t *testing.T) {
	client := setupRAGTestClient(t)
	defer client.Close()

	ctx := context.Background()
	tenant, err := createTestTenant(ctx, client)
	require.NoError(t, err)
	user, err := createTestUser(ctx, client, tenant.ID)
	require.NoError(t, err)

	now := time.Now()
	// 逾期未复核：复核周期 30 天，上次复核 90 天前
	_, err = createTimedArticle(ctx, client, tenant.ID, user.ID,
		"薪酬制度", "薪酬 vpn 相关制度", "HR", true, nil, nil, ptrTime(now.AddDate(0, 0, -90)), 30, 0)
	require.NoError(t, err)
	// 按时复核：复核周期 30 天，上次复核 5 天前
	_, err = createTimedArticle(ctx, client, tenant.ID, user.ID,
		"报销制度", "报销 vpn 流程", "财务", true, nil, nil, ptrTime(now.AddDate(0, 0, -5)), 30, 0)
	require.NoError(t, err)

	logger := zaptest.NewLogger(t).Sugar()
	svc := NewRAGService(client, nil, nil, logger, RAGConfig{UseKeyword: true, UseVector: false})

	results, err := svc.keywordSearch(ctx, tenant.ID, "vpn", 10)
	require.NoError(t, err)

	titles := make([]string, 0, len(results))
	for _, r := range results {
		titles = append(titles, r["title"].(string))
	}
	assert.NotContains(t, titles, "薪酬制度", "逾期未复核文章必须被后置过滤剔除")
	assert.Contains(t, titles, "报销制度")
}

func TestRAG_KeywordSearch_OverdueFilterPreservesLimit(t *testing.T) {
	client := setupRAGTestClient(t)
	defer client.Close()

	ctx := context.Background()
	tenant, err := createTestTenant(ctx, client)
	require.NoError(t, err)
	user, err := createTestUser(ctx, client, tenant.ID)
	require.NoError(t, err)

	now := time.Now()
	// 3 篇逾期 + 3 篇正常，全部命中关键词，limit=3
	// over-fetch 必须保证过滤后仍凑得满 3 条，否则视为 limit 语义被破坏。
	for i := 0; i < 3; i++ {
		_, err = createTimedArticle(ctx, client, tenant.ID, user.ID,
			"逾期手册", "vpn 逾期内容", "归档", true, nil, nil, ptrTime(now.AddDate(0, 0, -300)), 30, 0)
		require.NoError(t, err)
	}
	for i := 0; i < 3; i++ {
		_, err = createTimedArticle(ctx, client, tenant.ID, user.ID,
			"现行手册", "vpn 现行内容", "现行", true, nil, nil, ptrTime(now.AddDate(0, 0, -1)), 30, 0)
		require.NoError(t, err)
	}

	logger := zaptest.NewLogger(t).Sugar()
	svc := NewRAGService(client, nil, nil, logger, RAGConfig{UseKeyword: true, UseVector: false})

	results, err := svc.keywordSearch(ctx, tenant.ID, "vpn", 3)
	require.NoError(t, err)
	assert.Len(t, results, 3, "后置过滤后召回数不得低于 limit")
	for _, r := range results {
		assert.Equal(t, "现行手册", r["title"], "过滤掉的必须是逾期文章")
	}
}

func TestRAG_RankByAuthority_OfficialBeatsNormal(t *testing.T) {
	client := setupRAGTestClient(t)
	defer client.Close()

	ctx := context.Background()
	tenant, err := createTestTenant(ctx, client)
	require.NoError(t, err)
	user, err := createTestUser(ctx, client, tenant.ID)
	require.NoError(t, err)

	// 两篇标题都含关键词（keyword 路径里标题命中=0.9 分）：
	// 一篇官方标准（authority=20），一篇普通（authority=0）。
	// 融合分：0.9+0.133 vs 0.9，官方标准应排第一。
	_, err = createTimedArticle(ctx, client, tenant.ID, user.ID,
		"普通 VPN 指南", "vpn 配置", "网络", true, nil, nil, nil, 0, knowledgeaccess.AuthorityNormal)
	require.NoError(t, err)
	_, err = createTimedArticle(ctx, client, tenant.ID, user.ID,
		"官方 VPN 标准", "vpn 配置", "网络", true, nil, nil, nil, 0, knowledgeaccess.AuthorityOfficial)
	require.NoError(t, err)

	logger := zaptest.NewLogger(t).Sugar()
	svc := NewRAGService(client, nil, nil, logger, RAGConfig{UseKeyword: true, UseVector: false})

	results, err := svc.Ask(ctx, tenant.ID, "VPN", 10)
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "官方 VPN 标准", results[0]["title"],
		"同相关性下权威等级高的文章应排第一")
	assert.Equal(t, "普通 VPN 指南", results[1]["title"])
}

func TestRAG_RankByAuthority_RelevanceStillDominates(t *testing.T) {
	client := setupRAGTestClient(t)
	defer client.Close()

	ctx := context.Background()
	tenant, err := createTestTenant(ctx, client)
	require.NoError(t, err)
	user, err := createTestUser(ctx, client, tenant.ID)
	require.NoError(t, err)

	// 标题命中（0.9 分）的普通文章 vs 仅正文命中（0.5 分）的唯一真相源：
	// 0.9 vs 0.5+0.2=0.7，相关性差距必须胜出。
	_, err = createTimedArticle(ctx, client, tenant.ID, user.ID,
		"VPN 标题指南", "vpn 配置说明", "网络", true, nil, nil, nil, 0, knowledgeaccess.AuthorityNormal)
	require.NoError(t, err)
	_, err = createTimedArticle(ctx, client, tenant.ID, user.ID,
		"网络总纲", "vpn 与网络架构总纲", "架构", true, nil, nil, nil, 0, knowledgeaccess.AuthorityAuthoritative)
	require.NoError(t, err)

	logger := zaptest.NewLogger(t).Sugar()
	svc := NewRAGService(client, nil, nil, logger, RAGConfig{UseKeyword: true, UseVector: false})

	results, err := svc.Ask(ctx, tenant.ID, "VPN", 10)
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "VPN 标题指南", results[0]["title"],
		"权威加成不得压过显著的相关性差距")
}

func TestRAG_ArticleCitable_CombinesL0AndL1(t *testing.T) {
	client := setupRAGTestClient(t)
	defer client.Close()

	ctx := context.Background()
	tenant, err := createTestTenant(ctx, client)
	require.NoError(t, err)
	user, err := createTestUser(ctx, client, tenant.ID)
	require.NoError(t, err)

	now := time.Now()
	logger := zaptest.NewLogger(t).Sugar()
	svc := NewRAGService(client, nil, nil, logger, RAGConfig{})

	// 空指针：fail-closed
	assert.False(t, svc.articleCitable(ctx, tenant.ID, nil))

	// 正常文章：可引用
	ok, err := createTimedArticle(ctx, client, tenant.ID, user.ID,
		"正常文章", "内容", "通用", true, nil, nil, nil, 0, 0)
	require.NoError(t, err)
	assert.True(t, svc.articleCitable(ctx, tenant.ID, ok))

	// 已失效文章：不可引用（即便它已发布且未删除）
	expired, err := createTimedArticle(ctx, client, tenant.ID, user.ID,
		"失效文章", "内容", "通用", true, nil, ptrTime(now.Add(-time.Second)), nil, 0, 0)
	require.NoError(t, err)
	assert.False(t, svc.articleCitable(ctx, tenant.ID, expired), "已失效文章不可被引用")

	// 未生效文章：不可引用
	future, err := createTimedArticle(ctx, client, tenant.ID, user.ID,
		"预定文章", "内容", "通用", true, ptrTime(now.Add(time.Hour)), nil, nil, 0, 0)
	require.NoError(t, err)
	assert.False(t, svc.articleCitable(ctx, tenant.ID, future), "未生效文章不可被引用")

	// 逾期未复核文章：不可引用
	overdue, err := createTimedArticle(ctx, client, tenant.ID, user.ID,
		"逾期文章", "内容", "通用", true, nil, nil, ptrTime(now.AddDate(0, 0, -100)), 30, 0)
	require.NoError(t, err)
	assert.False(t, svc.articleCitable(ctx, tenant.ID, overdue), "逾期未复核文章不可被引用")
}

func TestRAG_FreshnessDisabled_KeepsLegacyBehavior(t *testing.T) {
	client := setupRAGTestClient(t)
	defer client.Close()

	ctx := context.Background()
	tenant, err := createTestTenant(ctx, client)
	require.NoError(t, err)
	user, err := createTestUser(ctx, client, tenant.ID)
	require.NoError(t, err)

	now := time.Now()
	_, err = createTimedArticle(ctx, client, tenant.ID, user.ID,
		"失效文章", "vpn 内容", "通用", true, nil, ptrTime(now.Add(-time.Hour)), nil, 0, 0)
	require.NoError(t, err)

	logger := zaptest.NewLogger(t).Sugar()
	svc := NewRAGService(client, nil, nil, logger, RAGConfig{UseKeyword: true, UseVector: false})
	// 显式关闭时效过滤：宽松租户的降级开关
	svc.SetFreshnessJudger(nil)

	results, err := svc.keywordSearch(ctx, tenant.ID, "vpn", 10)
	require.NoError(t, err)
	assert.Len(t, results, 1, "关闭时效过滤后应回到旧行为（不过滤）")
}

func ptrTime(t time.Time) *time.Time { return &t }
