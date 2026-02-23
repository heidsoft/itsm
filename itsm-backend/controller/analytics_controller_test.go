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

func setupTestAnalyticsController(t *testing.T) (*gin.Engine, *ent.Client) {
	gin.SetMode(gin.TestMode)

	// 创建内存数据库
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")

	// 创建 logger
	logger := zaptest.NewLogger(t).Sugar()

	// 创建服务
	analyticsService := service.NewAnalyticsService(client, logger)

	// 创建控制器
	_ = NewAnalyticsController(analyticsService)

	// 创建路由
	r := gin.New()
	r.Use(gin.Recovery())

	return r, client
}

func createTestTenantForAnalytics(t *testing.T, client *ent.Client) *ent.Tenant {
	ctx := context.Background()

	// 创建测试租户
	tenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TESTANAYLTICS").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	return tenant
}

func TestAnalyticsController_GetDeepAnalytics(t *testing.T) {
	r, client := setupTestAnalyticsController(t)
	defer client.Close()

	tenant := createTestTenantForAnalytics(t, client)

	tests := []struct {
		name        string
		queryParams string
	}{
		{
			name:        "获取深度分析",
			queryParams: "type=tickets",
		},
		{
			name:        "默认分析",
			queryParams: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := "/api/v1/analytics/deep"
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

func TestAnalyticsController_ExportAnalytics(t *testing.T) {
	r, client := setupTestAnalyticsController(t)
	defer client.Close()

	tenant := createTestTenantForAnalytics(t, client)

	tests := []struct {
		name string
	}{
		{
			name: "导出分析数据",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/analytics/export", nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set("tenant_id", tenant.ID)

			r.ServeHTTP(w, req)
		})
	}
}
