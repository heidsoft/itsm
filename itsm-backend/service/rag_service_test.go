package service

import (
	"context"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// ==================== Mock Implementations ====================

// MockEmbedder implements Embedder interface
type MockEmbedder struct {
	MockEmbed func(text string) ([]float32, error)
}

func (m *MockEmbedder) Embed(text string) ([]float32, error) {
	if m.MockEmbed != nil {
		return m.MockEmbed(text)
	}
	return []float32{0.1, 0.2, 0.3}, nil
}

// ==================== Test Setup Helpers ====================

func setupRAGTestClient(t *testing.T) *ent.Client {
	return enttest.Open(t, "sqlite3", "file:rag_ent?mode=memory&cache=shared&_fk=1")
}

func createTestTenant(ctx context.Context, client *ent.Client) (*ent.Tenant, error) {
	return client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
}

func createTestUser(ctx context.Context, client *ent.Client, tenantID int) (*ent.User, error) {
	return client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetName("Test User").
		SetPasswordHash("hashedpassword").
		SetRole("agent").
		SetActive(true).
		SetTenantID(tenantID).
		Save(ctx)
}

func createKnowledgeArticle(ctx context.Context, client *ent.Client, tenantID, authorID int, title, content, category string) (*ent.KnowledgeArticle, error) {
	return client.KnowledgeArticle.Create().
		SetTitle(title).
		SetContent(content).
		SetCategory(category).
		SetAuthorID(authorID).
		SetTenantID(tenantID).
		SetIsPublished(true).
		Save(ctx)
}

// ==================== Test 5: snippet Truncation ====================

func TestRAG_snippet_Truncation(t *testing.T) {
	// Long text should be truncated with "..." suffix
	longText := strings.Repeat("a", 500)
	result := snippet(longText, 200)

	assert.LessOrEqual(t, len(result), 203) // 200 + "..."
	assert.True(t, strings.HasSuffix(result, "..."))
}

func TestRAG_snippet_Truncation_WithPeriod(t *testing.T) {
	// Text with sentence boundary should cut at period
	longText := "这是第一句话。这是第二句话，包含很多额外内容。"
	// Make it longer than 200 chars
	longText = strings.Repeat("测试", 100) + "。真正有用的内容在这里。"
	result := snippet(longText, 200)

	// Should end with ...
	assert.True(t, strings.HasSuffix(result, "..."))
}

func TestRAG_snippet_Truncation_WithNewline(t *testing.T) {
	// Text with newline should prefer newline boundary
	longText := "第一行内容\n第二行内容\n" + strings.Repeat("x", 200)
	result := snippet(longText, 100)

	assert.True(t, strings.HasSuffix(result, "..."))
}

// ==================== Test 6: snippet Short Text ====================

func TestRAG_snippet_ShortText(t *testing.T) {
	// Short text should not be truncated
	shortText := "这是一段短文本"
	result := snippet(shortText, 200)

	assert.Equal(t, shortText, result)
	assert.False(t, strings.HasSuffix(result, "..."))
}

func TestRAG_snippet_ExactlyLimit(t *testing.T) {
	// Text exactly at limit should not be truncated
	text := strings.Repeat("a", 100)
	result := snippet(text, 100)

	assert.Equal(t, text, result)
}

func TestRAG_snippet_ZeroLimit(t *testing.T) {
	// Zero limit should use default 160
	text := "这是一段测试文本"
	result := snippet(text, 0)

	assert.Equal(t, text, result) // len(text) < 160
}

func TestRAG_snippet_NegativeLimit(t *testing.T) {
	// Negative limit should use default 160
	text := "这是一段测试文本"
	result := snippet(text, -1)

	assert.Equal(t, text, result) // len(text) < 160
}

// ==================== Test 7: Empty Query Handling ====================

func TestRAG_Ask_EmptyQuery(t *testing.T) {
	client := setupRAGTestClient(t)
	defer client.Close()

	ctx := context.Background()
	tenant, err := createTestTenant(ctx, client)
	require.NoError(t, err)

	user, err := createTestUser(ctx, client, tenant.ID)
	require.NoError(t, err)

	// Create some articles
	_, err = createKnowledgeArticle(ctx, client, tenant.ID, user.ID, "标题1", "内容1", "分类1")
	require.NoError(t, err)
	_, err = createKnowledgeArticle(ctx, client, tenant.ID, user.ID, "标题2", "内容2", "分类2")
	require.NoError(t, err)

	logger := zaptest.NewLogger(t).Sugar()

	// Keyword-only config (no vector)
	cfg := RAGConfig{
		UseVector:    false,
		UseKeyword:   true,
		HybridSearch: false,
		MaxResults:   5,
	}
	svc := NewRAGService(client, nil, nil, logger, cfg)

	// Empty query should still return results (no filter applied)
	results, err := svc.Ask(ctx, tenant.ID, "", 5)
	require.NoError(t, err)
	// Empty query returns all articles due to TrimSpace check
	assert.Len(t, results, 2)
}

func TestRAG_Ask_WhitespaceQuery(t *testing.T) {
	client := setupRAGTestClient(t)
	defer client.Close()

	ctx := context.Background()
	tenant, err := createTestTenant(ctx, client)
	require.NoError(t, err)

	user, err := createTestUser(ctx, client, tenant.ID)
	require.NoError(t, err)

	_, err = createKnowledgeArticle(ctx, client, tenant.ID, user.ID, "标题1", "内容1", "分类1")
	require.NoError(t, err)

	logger := zaptest.NewLogger(t).Sugar()

	cfg := RAGConfig{
		UseVector:    false,
		UseKeyword:   true,
		HybridSearch: false,
		MaxResults:   5,
	}
	svc := NewRAGService(client, nil, nil, logger, cfg)

	// Whitespace-only query should be treated as empty
	results, err := svc.Ask(ctx, tenant.ID, "   ", 5)
	require.NoError(t, err)
	assert.Len(t, results, 1) // Returns all since query is trimmed to empty
}

// ==================== Test 8: No Results ====================

func TestRAG_Ask_NoResults(t *testing.T) {
	client := setupRAGTestClient(t)
	defer client.Close()

	ctx := context.Background()
	tenant, err := createTestTenant(ctx, client)
	require.NoError(t, err)

	logger := zaptest.NewLogger(t).Sugar()

	cfg := RAGConfig{
		UseVector:    false,
		UseKeyword:   true,
		HybridSearch: false,
		MaxResults:   5,
	}
	svc := NewRAGService(client, nil, nil, logger, cfg)

	// No articles exist, query for something that doesn't match
	results, err := svc.Ask(ctx, tenant.ID, "不存在的标题", 5)
	require.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestRAG_Ask_NoResults_NoVectorConfigured(t *testing.T) {
	client := setupRAGTestClient(t)
	defer client.Close()

	ctx := context.Background()
	tenant, err := createTestTenant(ctx, client)
	require.NoError(t, err)

	logger := zaptest.NewLogger(t).Sugar()

	// No vector store configured
	cfg := RAGConfig{
		UseVector:    true, // but vectors is nil
		UseKeyword:   true,
		HybridSearch: false,
		MaxResults:   5,
	}
	svc := NewRAGService(client, nil, nil, logger, cfg)

	// Should fall back to keyword search
	results, err := svc.Ask(ctx, tenant.ID, "test", 5)
	require.NoError(t, err)
	assert.Len(t, results, 0) // No articles created
}

// ==================== Test 4: Limit Parameter ====================

func TestRAG_Ask_LimitParameter(t *testing.T) {
	client := setupRAGTestClient(t)
	defer client.Close()

	ctx := context.Background()
	tenant, err := createTestTenant(ctx, client)
	require.NoError(t, err)

	user, err := createTestUser(ctx, client, tenant.ID)
	require.NoError(t, err)

	// Create 5 articles
	for i := 1; i <= 5; i++ {
		_, err = createKnowledgeArticle(ctx, client, tenant.ID, user.ID,
			"标题"+string(rune('0'+i)), "内容包含某些关键词", "分类")
		require.NoError(t, err)
	}

	logger := zaptest.NewLogger(t).Sugar()

	cfg := RAGConfig{
		UseVector:    false,
		UseKeyword:   true,
		HybridSearch: false,
		MaxResults:   3,
	}
	svc := NewRAGService(client, nil, nil, logger, cfg)

	// Request only 2 results
	results, err := svc.Ask(ctx, tenant.ID, "关键词", 2)
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestRAG_Ask_LimitParameter_Zero(t *testing.T) {
	client := setupRAGTestClient(t)
	defer client.Close()

	ctx := context.Background()
	tenant, err := createTestTenant(ctx, client)
	require.NoError(t, err)

	user, err := createTestUser(ctx, client, tenant.ID)
	require.NoError(t, err)

	// Create 3 articles
	for i := 1; i <= 3; i++ {
		_, err = createKnowledgeArticle(ctx, client, tenant.ID, user.ID,
			"标题", "内容", "分类")
		require.NoError(t, err)
	}

	logger := zaptest.NewLogger(t).Sugar()

	cfg := RAGConfig{
		UseVector:    false,
		UseKeyword:   true,
		HybridSearch: false,
		MaxResults:   5,
	}
	svc := NewRAGService(client, nil, nil, logger, cfg)

	// Zero limit should default to 5 (MaxResults)
	results, err := svc.Ask(ctx, tenant.ID, "内容", 0)
	require.NoError(t, err)
	assert.Len(t, results, 3) // All 3 since only 3 exist
}

// ==================== Test 3: Keyword Search Multiple Matches ====================

func TestRAG_Ask_KeywordSearch_MultipleMatches(t *testing.T) {
	client := setupRAGTestClient(t)
	defer client.Close()

	ctx := context.Background()
	tenant, err := createTestTenant(ctx, client)
	require.NoError(t, err)

	user, err := createTestUser(ctx, client, tenant.ID)
	require.NoError(t, err)

	// Create articles with matching content
	articles := []struct {
		title    string
		content  string
		category string
	}{
		{"服务器故障处理", "当服务器出现故障时，需要立即检查系统状态", "故障处理"},
		{"网络故障排查", "网络故障可能由多种原因引起，包括配置错误", "故障处理"},
		{"数据库维护", "数据库维护是日常工作中的重要部分", "运维"},
		{"如何重启服务", "服务重启可以帮助解决一些临时性问题", "运维"},
	}

	for _, a := range articles {
		_, err = createKnowledgeArticle(ctx, client, tenant.ID, user.ID, a.title, a.content, a.category)
		require.NoError(t, err)
	}

	logger := zaptest.NewLogger(t).Sugar()

	cfg := RAGConfig{
		UseVector:    false,
		UseKeyword:   true,
		HybridSearch: false,
		MaxResults:   10,
	}
	svc := NewRAGService(client, nil, nil, logger, cfg)

	// Search for "故障" - should match first two articles
	results, err := svc.Ask(ctx, tenant.ID, "故障", 10)
	require.NoError(t, err)
	assert.Len(t, results, 2)

	// Verify all results have search_type = "keyword"
	for _, r := range results {
		assert.Equal(t, "keyword", r["search_type"])
	}
}

func TestRAG_Ask_KeywordSearch_TitleMatch(t *testing.T) {
	client := setupRAGTestClient(t)
	defer client.Close()

	ctx := context.Background()
	tenant, err := createTestTenant(ctx, client)
	require.NoError(t, err)

	user, err := createTestUser(ctx, client, tenant.ID)
	require.NoError(t, err)

	_, err = createKnowledgeArticle(ctx, client, tenant.ID, user.ID,
		"密码重置流程", "用户可以通过邮箱重置密码", "账户管理")
	require.NoError(t, err)

	logger := zaptest.NewLogger(t).Sugar()

	cfg := RAGConfig{
		UseVector:    false,
		UseKeyword:   true,
		HybridSearch: false,
		MaxResults:   5,
	}
	svc := NewRAGService(client, nil, nil, logger, cfg)

	// Search by title keyword
	results, err := svc.Ask(ctx, tenant.ID, "密码", 5)
	require.NoError(t, err)
	require.Len(t, results, 1)

	// Title match should have higher score (0.9)
	assert.Equal(t, 0.9, results[0]["score"])
}

// ==================== Test 1: Keyword-Only Mode ====================

func TestRAG_Ask_KeywordOnly_NoVector(t *testing.T) {
	client := setupRAGTestClient(t)
	defer client.Close()

	ctx := context.Background()
	tenant, err := createTestTenant(ctx, client)
	require.NoError(t, err)

	user, err := createTestUser(ctx, client, tenant.ID)
	require.NoError(t, err)

	_, err = createKnowledgeArticle(ctx, client, tenant.ID, user.ID,
		"测试文章", "这是测试内容", "测试分类")
	require.NoError(t, err)

	logger := zaptest.NewLogger(t).Sugar()

	// Pure keyword-only configuration
	cfg := RAGConfig{
		UseVector:    false, // Vector disabled
		UseKeyword:   true,
		HybridSearch: false,
		MaxResults:   5,
	}
	svc := NewRAGService(client, nil, nil, logger, cfg)

	// Should use keyword search even if vector is technically "available"
	results, err := svc.Ask(ctx, tenant.ID, "测试", 5)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "keyword", results[0]["search_type"])
}

func TestRAG_Ask_KeywordOnly_VectorFallbackDisabled(t *testing.T) {
	client := setupRAGTestClient(t)
	defer client.Close()

	ctx := context.Background()
	tenant, err := createTestTenant(ctx, client)
	require.NoError(t, err)

	user, err := createTestUser(ctx, client, tenant.ID)
	require.NoError(t, err)

	_, err = createKnowledgeArticle(ctx, client, tenant.ID, user.ID,
		"网络问题", "网络连接失败的处理方法", "网络")
	require.NoError(t, err)

	logger := zaptest.NewLogger(t).Sugar()

	// Keyword only, hybrid disabled
	cfg := RAGConfig{
		UseVector:    false,
		UseKeyword:   true,
		HybridSearch: false,
		MaxResults:   5,
	}
	svc := NewRAGService(client, nil, nil, logger, cfg)

	results, err := svc.Ask(ctx, tenant.ID, "网络", 5)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "keyword", results[0]["search_type"])
	assert.Equal(t, "网络问题", results[0]["title"])
}

// ==================== Test 2: Hybrid Search Configuration ====================

func TestRAG_Ask_HybridSearch_ConfigEnabled(t *testing.T) {
	client := setupRAGTestClient(t)
	defer client.Close()

	ctx := context.Background()
	tenant, err := createTestTenant(ctx, client)
	require.NoError(t, err)

	user, err := createTestUser(ctx, client, tenant.ID)
	require.NoError(t, err)

	_, err = createKnowledgeArticle(ctx, client, tenant.ID, user.ID,
		"Docker容器部署", "Docker是常用的容器化平台", "DevOps")
	require.NoError(t, err)

	logger := zaptest.NewLogger(t).Sugar()

	// Hybrid search enabled but vector not available - should fall back to keyword
	cfg := RAGConfig{
		UseVector:    true, // enabled but vectors is nil -> will be set to false
		UseKeyword:   true,
		HybridSearch: true,
		MaxResults:   5,
	}
	svc := NewRAGService(client, nil, nil, logger, cfg)

	// Should work since vector falls back to keyword
	results, err := svc.Ask(ctx, tenant.ID, "Docker", 5)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "keyword", results[0]["search_type"])
}

// ==================== Test: GetStats ====================

func TestRAG_GetStats(t *testing.T) {
	client := setupRAGTestClient(t)
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()

	// Note: when vectors/embedder are nil, useVector becomes false regardless of cfg
	cfg := RAGConfig{
		UseVector:    false, // explicitly disabled since no vector store
		UseKeyword:   true,
		HybridSearch: false,
		MaxResults:   5,
	}
	svc := NewRAGService(client, nil, nil, logger, cfg)

	stats := svc.GetStats()
	assert.Equal(t, false, stats["use_vector"])
	assert.Equal(t, true, stats["use_keyword"])
	assert.Equal(t, false, stats["hybrid_search"])
}

// ==================== Test: CheckHealth ====================

func TestRAG_CheckHealth(t *testing.T) {
	client := setupRAGTestClient(t)
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()

	cfg := RAGConfig{
		UseVector:  false,
		UseKeyword: true,
	}
	svc := NewRAGService(client, nil, nil, logger, cfg)

	health := svc.CheckHealth(context.Background())
	assert.Equal(t, "healthy", health["status"])
	assert.Equal(t, "disabled", health["vector_store"])
	assert.Equal(t, "not configured", health["embedder"])
}

// ==================== Test: NewRAGServiceWithAutoConfig ====================

func TestRAG_NewRAGServiceWithAutoConfig_NilVectors(t *testing.T) {
	client := setupRAGTestClient(t)
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()

	// With nil vectors, should auto-disable vector search
	svc := NewRAGServiceWithAutoConfig(client, nil, nil, logger)
	assert.False(t, svc.useVector)
	assert.True(t, svc.useKeyword)
	assert.False(t, svc.hybridSearch)
}
