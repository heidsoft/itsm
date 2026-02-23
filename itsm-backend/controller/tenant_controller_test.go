package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/ent/enttest"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func setupTestTenantControllerForTest(t *testing.T) (*gin.Engine, *service.TenantService) {
	gin.SetMode(gin.TestMode)

	// 创建内存数据库
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")

	// 创建 logger
	logger := zaptest.NewLogger(t).Sugar()

	// 创建服务
	tenantService := service.NewTenantService(client, logger)

	// 创建控制器
	_ = NewTenantController(tenantService, logger)

	// 创建路由
	r := gin.New()
	r.Use(gin.Recovery())

	return r, tenantService
}

func TestTenantController_ListTenants(t *testing.T) {
	r, _ := setupTestTenantControllerForTest(t)

	tests := []struct {
		name string
	}{
		{
			name: "获取租户列表",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/tenants", nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			r.ServeHTTP(w, req)
		})
	}
}
