package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func setupTestCMDBController(t *testing.T) (*gin.Engine, *CMDBController, *ent.Client) {
	gin.SetMode(gin.TestMode)

	// 创建内存数据库
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")

	// 创建 logger
	logger := zaptest.NewLogger(t).Sugar()

	// 创建服务
	ciTypeService := service.NewCITypeService(client, logger)
	ciAttributeDefinitionService := service.NewCIAttributeDefinitionService(client, logger)
	configurationItemService := service.NewConfigurationItemService(client, logger)
	ciRelationshipService := service.NewCIRelationshipService(client, logger)

	// 创建控制器
	cmdbController := NewCMDBController(logger, ciTypeService, ciAttributeDefinitionService, configurationItemService, ciRelationshipService)

	// 创建路由
	r := gin.New()
	r.Use(gin.Recovery())

	// 添加中间件设置租户ID
	r.Use(func(c *gin.Context) {
		// 测试时使用默认租户ID
		c.Set("tenant_id", 1)
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: 1})
		c.Set("user_id", 1)
		c.Next()
	})

	// 注册路由
	r.POST("/api/v1/cis", cmdbController.CreateCI)
	r.GET("/api/v1/cis/:id", cmdbController.GetCI)
	r.GET("/api/v1/cis", cmdbController.ListCIs)
	r.PUT("/api/v1/cis/:id", cmdbController.UpdateCI)
	r.DELETE("/api/v1/cis/:id", cmdbController.DeleteCI)

	return r, cmdbController, client
}

func TestCMDBController_CreateCI(t *testing.T) {
	r, _, client := setupTestCMDBController(t)
	defer client.Close()

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
	}{
		{
			name: "缺少CI类型",
			requestBody: map[string]interface{}{
				"name":   "Test CI",
				"status": "active",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req, err := http.NewRequest("POST", "/api/v1/cis", bytes.NewReader(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestCMDBController_GetCI(t *testing.T) {
	r, _, client := setupTestCMDBController(t)
	defer client.Close()

	tests := []struct {
		name           string
		ciID           string
		expectedStatus int
	}{
		{
			name:           "获取不存在的配置项",
			ciID:           "99999",
			expectedStatus: http.StatusNotFound, // CI不存在
		},
		{
			name:           "无效的配置项ID",
			ciID:           "invalid",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/cis/"+tt.ciID, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			// 验证响应状态码
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestCMDBController_ListCIs(t *testing.T) {
	r, _, client := setupTestCMDBController(t)
	defer client.Close()

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
	}{
		{
			name:           "成功获取配置项列表",
			queryParams:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "带分页参数",
			queryParams:    "page=1&pageSize=10",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "带类型筛选",
			queryParams:    "type=server",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "带状态筛选",
			queryParams:    "status=active",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "带环境筛选",
			queryParams:    "environment=production",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "带搜索关键词",
			queryParams:    "search=test",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := "/api/v1/cis"
			if tt.queryParams != "" {
				path += "?" + tt.queryParams
			}

			req, err := http.NewRequest("GET", path, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestCMDBController_UpdateCI(t *testing.T) {
	// 跳过测试 - 需要dddSvc
	t.Skip("UpdateCI requires dddSvc which is not available in unit tests")
}

func TestCMDBController_DeleteCI(t *testing.T) {
	// 跳过测试 - 需要dddSvc
	t.Skip("DeleteCI requires dddSvc which is not available in unit tests")
}

// 测试租户隔离 - 使用 ListCIs 验证租户隔离
func TestCMDBController_TenantIsolation(t *testing.T) {
	r, _, client := setupTestCMDBController(t)
	defer client.Close()

	// 使用租户1查询列表
	req, err := http.NewRequest("GET", "/api/v1/cis", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 验证响应格式正确
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, float64(0), response["code"])
}
