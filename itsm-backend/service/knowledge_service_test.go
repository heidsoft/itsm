package service

import (
	"context"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestKnowledgeService_CreateArticle(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	knowledgeService := NewKnowledgeService(client, logger)

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建测试用户
	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetPasswordHash("hashedpassword").
		SetName("Test User").
		SetDepartment("IT").
		SetPhone("1234567890").
		SetActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name          string
		request       *dto.CreateKnowledgeArticleRequest
		authorID      int
		tenantID      int
		expectedError bool
	}{
		{
			name: "成功创建知识文章",
			request: &dto.CreateKnowledgeArticleRequest{
				Title:    "如何重置密码",
				Content:  "详细的密码重置步骤说明...",
				Category: "用户指南",
				Tags:     []string{"密码", "重置", "用户"},
				Status:   "published",
			},
			authorID:      testUser.ID,
			tenantID:      testTenant.ID,
			expectedError: false,
		},
		{
			name: "创建技术文档",
			request: &dto.CreateKnowledgeArticleRequest{
				Title:    "服务器维护指南",
				Content:  "服务器日常维护和故障排除指南...",
				Category: "技术文档",
				Tags:     []string{"服务器", "维护", "故障排除"},
				Status:   "draft",
			},
			authorID:      testUser.ID,
			tenantID:      testTenant.ID,
			expectedError: false,
		},
		{
			name: "标题为空",
			request: &dto.CreateKnowledgeArticleRequest{
				Title:    "",
				Content:  "内容",
				Category: "分类",
				Status:   "draft",
			},
			authorID:      testUser.ID,
			tenantID:      testTenant.ID,
			expectedError: true,
		},
		{
			name: "内容为空",
			request: &dto.CreateKnowledgeArticleRequest{
				Title:    "标题",
				Content:  "",
				Category: "分类",
				Status:   "draft",
			},
			authorID:      testUser.ID,
			tenantID:      testTenant.ID,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			article, err := knowledgeService.CreateArticle(ctx, tt.request, tt.authorID, tt.tenantID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, article)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, article)
				assert.Equal(t, tt.request.Title, article.Title)
				assert.Equal(t, tt.request.Content, article.Content)
				assert.Equal(t, tt.request.Category, article.Category)
				assert.Equal(t, tt.request.Status, article.Status)
				assert.Equal(t, tt.authorID, article.AuthorID)
				assert.Equal(t, tt.tenantID, article.TenantID)
				assert.Equal(t, 0, article.ViewCount)
				assert.Equal(t, 0, article.LikeCount)
			}
		})
	}
}

func TestKnowledgeService_GetArticle(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	knowledgeService := NewKnowledgeService(client, logger)

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建测试用户
	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetPasswordHash("hashedpassword").
		SetName("Test User").
		SetDepartment("IT").
		SetPhone("1234567890").
		SetActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 创建测试知识文章
	testArticle, err := client.KnowledgeArticle.Create().
		SetTitle("测试文章").
		SetContent("测试文章内容").
		SetCategory("测试分类").
		SetTags([]string{"测试", "文章"}).
		SetStatus("published").
		SetAuthorID(testUser.ID).
		SetTenantID(testTenant.ID).
		SetViewCount(10).
		SetLikeCount(5).
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name          string
		articleID     int
		tenantID      int
		expectedError bool
	}{
		{
			name:          "成功获取文章",
			articleID:     testArticle.ID,
			tenantID:      testTenant.ID,
			expectedError: false,
		},
		{
			name:          "文章不存在",
			articleID:     99999,
			tenantID:      testTenant.ID,
			expectedError: true,
		},
		{
			name:          "租户不匹配",
			articleID:     testArticle.ID,
			tenantID:      99999,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			article, err := knowledgeService.GetArticle(ctx, tt.articleID, tt.tenantID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, article)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, article)
				assert.Equal(t, testArticle.ID, article.ID)
				assert.Equal(t, testArticle.Title, article.Title)
				assert.Equal(t, testArticle.Content, article.Content)
				assert.Equal(t, testArticle.Category, article.Category)
				assert.Equal(t, testArticle.Status, article.Status)
				// 获取文章时应该增加浏览次数
				assert.Equal(t, testArticle.ViewCount+1, article.ViewCount)
			}
		})
	}
}

func TestKnowledgeService_UpdateArticle(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	knowledgeService := NewKnowledgeService(client, logger)

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建测试用户
	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetPasswordHash("hashedpassword").
		SetName("Test User").
		SetDepartment("IT").
		SetPhone("1234567890").
		SetActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 创建测试知识文章
	testArticle, err := client.KnowledgeArticle.Create().
		SetTitle("原始标题").
		SetContent("原始内容").
		SetCategory("原始分类").
		SetTags([]string{"原始", "标签"}).
		SetStatus("draft").
		SetAuthorID(testUser.ID).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name          string
		articleID     int
		request       *dto.UpdateKnowledgeArticleRequest
		tenantID      int
		expectedError bool
	}{
		{
			name:      "成功更新文章",
			articleID: testArticle.ID,
			request: &dto.UpdateKnowledgeArticleRequest{
				Title:    "更新后的标题",
				Content:  "更新后的内容",
				Category: "更新后的分类",
				Tags:     []string{"更新", "标签"},
				Status:   "published",
			},
			tenantID:      testTenant.ID,
			expectedError: false,
		},
		{
			name:      "部分更新",
			articleID: testArticle.ID,
			request: &dto.UpdateKnowledgeArticleRequest{
				Status: "archived",
			},
			tenantID:      testTenant.ID,
			expectedError: false,
		},
		{
			name:      "文章不存在",
			articleID: 99999,
			request: &dto.UpdateKnowledgeArticleRequest{
				Title: "不存在的文章",
			},
			tenantID:      testTenant.ID,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			article, err := knowledgeService.UpdateArticle(ctx, tt.articleID, tt.request, tt.tenantID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, article)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, article)
				assert.Equal(t, tt.articleID, article.ID)
				if tt.request.Title != "" {
					assert.Equal(t, tt.request.Title, article.Title)
				}
				if tt.request.Content != "" {
					assert.Equal(t, tt.request.Content, article.Content)
				}
				if tt.request.Category != "" {
					assert.Equal(t, tt.request.Category, article.Category)
				}
				if tt.request.Status != "" {
					assert.Equal(t, tt.request.Status, article.Status)
				}
			}
		})
	}
}

func TestKnowledgeService_ListArticles(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	knowledgeService := NewKnowledgeService(client, logger)

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建测试用户
	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetPasswordHash("hashedpassword").
		SetName("Test User").
		SetDepartment("IT").
		SetPhone("1234567890").
		SetActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 创建测试知识文章
	articles := []struct {
		title    string
		category string
		status   string
		tags     []string
	}{
		{"密码重置指南", "用户指南", "published", []string{"密码", "重置"}},
		{"服务器维护", "技术文档", "published", []string{"服务器", "维护"}},
		{"网络故障排除", "故障排除", "draft", []string{"网络", "故障"}},
		{"安全策略", "安全文档", "published", []string{"安全", "策略"}},
		{"备份恢复", "技术文档", "archived", []string{"备份", "恢复"}},
	}

	for _, art := range articles {
		_, err := client.KnowledgeArticle.Create().
			SetTitle(art.title).
			SetContent("测试文章内容").
			SetCategory(art.category).
			SetTags(art.tags).
			SetStatus(art.status).
			SetAuthorID(testUser.ID).
			SetTenantID(testTenant.ID).
			Save(ctx)
		require.NoError(t, err)
	}

	tests := []struct {
		name          string
		request       *dto.ListKnowledgeArticlesRequest
		expectedCount int
		expectedError bool
	}{
		{
			name: "获取所有文章",
			request: &dto.ListKnowledgeArticlesRequest{
				Page:     1,
				PageSize: 10,
			},
			expectedCount: 5,
			expectedError: false,
		},
		{
			name: "按分类筛选",
			request: &dto.ListKnowledgeArticlesRequest{
				Page:     1,
				PageSize: 10,
				Category: "技术文档",
			},
			expectedCount: 2,
			expectedError: false,
		},
		{
			name: "按状态筛选",
			request: &dto.ListKnowledgeArticlesRequest{
				Page:     1,
				PageSize: 10,
				Status:   "published",
			},
			expectedCount: 3,
			expectedError: false,
		},
		{
			name: "关键词搜索",
			request: &dto.ListKnowledgeArticlesRequest{
				Page:     1,
				PageSize: 10,
				Keyword:  "密码",
			},
			expectedCount: 1,
			expectedError: false,
		},
		{
			name: "按标签筛选",
			request: &dto.ListKnowledgeArticlesRequest{
				Page:     1,
				PageSize: 10,
				Tag:      "服务器",
			},
			expectedCount: 1,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := knowledgeService.ListArticles(ctx, tt.request, testTenant.ID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.Len(t, response.Articles, tt.expectedCount)
				assert.Equal(t, tt.request.Page, response.Page)
				assert.Equal(t, tt.request.PageSize, response.PageSize)
			}
		})
	}
}

func TestKnowledgeService_DeleteArticle(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	knowledgeService := NewKnowledgeService(client, logger)

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建测试用户
	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetPasswordHash("hashedpassword").
		SetName("Test User").
		SetDepartment("IT").
		SetPhone("1234567890").
		SetActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 创建测试知识文章
	testArticle, err := client.KnowledgeArticle.Create().
		SetTitle("待删除文章").
		SetContent("这篇文章将被删除").
		SetCategory("测试分类").
		SetTags([]string{"测试", "删除"}).
		SetStatus("draft").
		SetAuthorID(testUser.ID).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name          string
		articleID     int
		tenantID      int
		expectedError bool
	}{
		{
			name:          "成功删除文章",
			articleID:     testArticle.ID,
			tenantID:      testTenant.ID,
			expectedError: false,
		},
		{
			name:          "文章不存在",
			articleID:     99999,
			tenantID:      testTenant.ID,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := knowledgeService.DeleteArticle(ctx, tt.articleID, tt.tenantID)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// 验证文章已被删除
				_, err := client.KnowledgeArticle.Get(ctx, tt.articleID)
				assert.Error(t, err)
			}
		})
	}
}

func TestKnowledgeService_LikeArticle(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	knowledgeService := NewKnowledgeService(client, logger)

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建测试用户
	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetPasswordHash("hashedpassword").
		SetName("Test User").
		SetDepartment("IT").
		SetPhone("1234567890").
		SetActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 创建测试知识文章
	testArticle, err := client.KnowledgeArticle.Create().
		SetTitle("测试文章").
		SetContent("测试文章内容").
		SetCategory("测试分类").
		SetTags([]string{"测试"}).
		SetStatus("published").
		SetAuthorID(testUser.ID).
		SetTenantID(testTenant.ID).
		SetLikeCount(0).
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name          string
		articleID     int
		userID        int
		tenantID      int
		expectedError bool
	}{
		{
			name:          "成功点赞文章",
			articleID:     testArticle.ID,
			userID:        testUser.ID,
			tenantID:      testTenant.ID,
			expectedError: false,
		},
		{
			name:          "文章不存在",
			articleID:     99999,
			userID:        testUser.ID,
			tenantID:      testTenant.ID,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			article, err := knowledgeService.LikeArticle(ctx, tt.articleID, tt.userID, tt.tenantID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, article)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, article)
				assert.Equal(t, testArticle.LikeCount+1, article.LikeCount)
			}
		})
	}
}

func TestKnowledgeService_SearchArticles(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	knowledgeService := NewKnowledgeService(client, logger)

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建测试用户
	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetPasswordHash("hashedpassword").
		SetName("Test User").
		SetDepartment("IT").
		SetPhone("1234567890").
		SetActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 创建测试知识文章
	searchArticles := []struct {
		title   string
		content string
		tags    []string
	}{
		{"密码重置指南", "如何重置用户密码的详细步骤", []string{"密码", "重置"}},
		{"网络故障排除", "网络连接问题的诊断和解决方法", []string{"网络", "故障"}},
		{"服务器维护手册", "服务器日常维护和监控指南", []string{"服务器", "维护"}},
	}

	for _, art := range searchArticles {
		_, err := client.KnowledgeArticle.Create().
			SetTitle(art.title).
			SetContent(art.content).
			SetCategory("技术文档").
			SetTags(art.tags).
			SetStatus("published").
			SetAuthorID(testUser.ID).
			SetTenantID(testTenant.ID).
			Save(ctx)
		require.NoError(t, err)
	}

	tests := []struct {
		name          string
		keyword       string
		expectedCount int
		expectedError bool
	}{
		{
			name:          "搜索密码相关文章",
			keyword:       "密码",
			expectedCount: 1,
			expectedError: false,
		},
		{
			name:          "搜索网络相关文章",
			keyword:       "网络",
			expectedCount: 1,
			expectedError: false,
		},
		{
			name:          "搜索服务器相关文章",
			keyword:       "服务器",
			expectedCount: 1,
			expectedError: false,
		},
		{
			name:          "搜索不存在的关键词",
			keyword:       "不存在的关键词",
			expectedCount: 0,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			articles, err := knowledgeService.SearchArticles(ctx, tt.keyword, testTenant.ID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, articles)
			} else {
				assert.NoError(t, err)
				assert.Len(t, articles, tt.expectedCount)
			}
		})
	}
}

// 基准测试
func BenchmarkKnowledgeService_CreateArticle(b *testing.B) {
	client := enttest.Open(b, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(b).Sugar()
	knowledgeService := NewKnowledgeService(client, logger)

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(b, err)

	// 创建测试用户
	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetPasswordHash("hashedpassword").
		SetName("Test User").
		SetDepartment("IT").
		SetPhone("1234567890").
		SetActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := &dto.CreateKnowledgeArticleRequest{
			Title:    "基准测试文章",
			Content:  "这是一个基准测试文章的内容",
			Category: "测试分类",
			Tags:     []string{"基准", "测试"},
			Status:   "draft",
		}
		_, _ = knowledgeService.CreateArticle(ctx, req, testUser.ID, testTenant.ID)
	}
}

func BenchmarkKnowledgeService_ListArticles(b *testing.B) {
	client := enttest.Open(b, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(b).Sugar()
	knowledgeService := NewKnowledgeService(client, logger)

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(b, err)

	// 创建测试用户
	testUser, err := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetPasswordHash("hashedpassword").
		SetName("Test User").
		SetDepartment("IT").
		SetPhone("1234567890").
		SetActive(true).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(b, err)

	// 预创建一些文章
	for i := 0; i < 100; i++ {
		_, err := client.KnowledgeArticle.Create().
			SetTitle("基准测试文章").
			SetContent("基准测试文章内容").
			SetCategory("测试分类").
			SetTags([]string{"基准", "测试"}).
			SetStatus("published").
			SetAuthorID(testUser.ID).
			SetTenantID(testTenant.ID).
			Save(ctx)
		require.NoError(b, err)
	}

	req := &dto.ListKnowledgeArticlesRequest{
		Page:     1,
		PageSize: 10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = knowledgeService.ListArticles(ctx, req, testTenant.ID)
	}
}