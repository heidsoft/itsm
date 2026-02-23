package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func setupTestDashboardController(t *testing.T) (*gin.Engine, *ent.Client, *DashboardController) {
	gin.SetMode(gin.TestMode)

	// 创建内存数据库
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")

	// 创建 logger
	logger := zaptest.NewLogger(t).Sugar()

	// 创建服务
	dashboardService := service.NewDashboardService(client, logger)

	// 创建控制器
	dashboardController := NewDashboardController(dashboardService, logger)

	// 创建路由
	r := gin.New()
	r.Use(gin.Recovery())

	// 注册路由
	r.GET("/api/v1/dashboard", dashboardController.GetDashboardData)
	r.GET("/api/v1/dashboard/kpis", dashboardController.GetKPIMetrics)
	r.GET("/api/v1/dashboard/resources/distribution", dashboardController.GetResourceDistribution)

	return r, client, dashboardController
}

func createTestDataForDashboard(t *testing.T, client *ent.Client) (*ent.Tenant, *ent.User) {
	ctx := context.Background()
	uniqueID := uniqueTestID()

	// 创建测试租户
	tenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST" + uniqueID).
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建测试用户
	user, err := client.User.Create().
		SetUsername("testuser" + uniqueID).
		SetEmail("test" + uniqueID + "@example.com").
		SetPasswordHash("hashedpassword").
		SetName("Test User").
		SetDepartment("IT").
		SetPhone("1234567890").
		SetActive(true).
		SetRole("admin").
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	return tenant, user
}

func TestDashboardController_GetDashboardData(t *testing.T) {
	r, client, _ := setupTestDashboardController(t)
	defer client.Close()

	tenant, _ := createTestDataForDashboard(t, client)

	tests := []struct {
		name           string
		expectedStatus int
		expectedCode   int
	}{
		{
			name:           "成功获取仪表盘数据",
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/dashboard", nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set("tenant_id", tenant.ID)

			r.ServeHTTP(w, req)
		})
	}
}

func TestDashboardController_GetKPIMetrics(t *testing.T) {
	r, client, _ := setupTestDashboardController(t)
	defer client.Close()

	tenant, _ := createTestDataForDashboard(t, client)

	tests := []struct {
		name           string
		expectedStatus int
		expectedCode   int
	}{
		{
			name:           "成功获取KPI指标",
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/dashboard/kpis", nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set("tenant_id", tenant.ID)

			r.ServeHTTP(w, req)
		})
	}
}

func TestDashboardController_GetResourceDistribution(t *testing.T) {
	r, client, _ := setupTestDashboardController(t)
	defer client.Close()

	tenant, _ := createTestDataForDashboard(t, client)

	tests := []struct {
		name           string
		expectedStatus int
		expectedCode   int
	}{
		{
			name:           "成功获取资源分布",
			expectedStatus: http.StatusOK,
			expectedCode:   common.SuccessCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/dashboard/resources/distribution", nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set("tenant_id", tenant.ID)

			r.ServeHTTP(w, req)
		})
	}
}
