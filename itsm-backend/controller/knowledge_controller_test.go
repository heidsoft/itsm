package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func setupTestKnowledgeController(t *testing.T) (*gin.Engine, *ent.Client, *KnowledgeController) {
	gin.SetMode(gin.TestMode)

	// 创建内存数据库
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")

	// 创建 logger
	logger := zaptest.NewLogger(t).Sugar()

	// 创建服务
	knowledgeService := service.NewKnowledgeService(client, logger)

	// 创建控制器
	knowledgeController := NewKnowledgeController(knowledgeService, logger)

	// 创建路由
	r := gin.New()
	r.Use(gin.Recovery())

	// 注册路由
	r.GET("/api/v1/knowledge-articles", knowledgeController.ListArticles)
	r.POST("/api/v1/knowledge-articles", knowledgeController.CreateArticle)
	r.GET("/api/v1/knowledge-articles/:id", knowledgeController.GetArticle)
	r.PUT("/api/v1/knowledge-articles/:id", knowledgeController.UpdateArticle)
	r.DELETE("/api/v1/knowledge-articles/:id", knowledgeController.DeleteArticle)
	r.GET("/api/v1/knowledge/categories", knowledgeController.GetCategories)

	return r, client, knowledgeController
}

func createTestTenantForKnowledge(t *testing.T, client *ent.Client) *ent.Tenant {
	ctx := context.Background()

	// 创建测试租户
	tenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TESTKB").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	return tenant
}

func TestKnowledgeController_ListArticles(t *testing.T) {
	r, client, _ := setupTestKnowledgeController(t)
	defer client.Close()

	tenant := createTestTenantForKnowledge(t, client)

	tests := []struct {
		name        string
		queryParams string
	}{
		{
			name:        "成功获取文章列表",
			queryParams: "",
		},
		{
			name:        "带分页参数",
			queryParams: "page=1&pageSize=10",
		},
		{
			name:        "按分类筛选",
			queryParams: "category_id=1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := "/api/v1/knowledge-articles"
			if tt.queryParams != "" {
				path += "?" + tt.queryParams
			}

			req, err := http.NewRequest("GET", path, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set("tenant_id", tenant.ID)

			r.ServeHTTP(w, req)
		})
	}
}

func TestKnowledgeController_GetArticle(t *testing.T) {
	r, client, _ := setupTestKnowledgeController(t)
	defer client.Close()

	tenant := createTestTenantForKnowledge(t, client)

	tests := []struct {
		name     string
		articleID string
	}{
		{
			name:      "成功获取文章",
			articleID: "1",
		},
		{
			name:      "无效的文章ID",
			articleID: "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/knowledge-articles/"+tt.articleID, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set("tenant_id", tenant.ID)

			r.ServeHTTP(w, req)
		})
	}
}

func TestKnowledgeController_GetCategories(t *testing.T) {
	r, client, _ := setupTestKnowledgeController(t)
	defer client.Close()

	tenant := createTestTenantForKnowledge(t, client)

	tests := []struct {
		name string
	}{
		{
			name: "成功获取分类列表",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/knowledge/categories", nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set("tenant_id", tenant.ID)

			r.ServeHTTP(w, req)
		})
	}
}
