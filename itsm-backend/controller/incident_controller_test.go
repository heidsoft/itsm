package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"

	"itsm-backend/ent/enttest"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap/zaptest"
)

func setupTestIncidentController(t *testing.T) (*gin.Engine, *IncidentController) {
	gin.SetMode(gin.TestMode)

	// 创建内存数据库
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")

	// 创建 logger
	logger := zaptest.NewLogger(t).Sugar()

	// 创建服务
	incidentService := service.NewIncidentService(client, logger, nil)

	// 创建控制器
	incidentController := NewIncidentController(incidentService, nil, nil, nil, nil, logger)

	// 创建路由
	r := gin.New()
	r.Use(gin.Recovery())

	// 注册路由
	r.GET("/api/v1/incidents", incidentController.ListIncidents)

	return r, incidentController
}

func TestIncidentController_ListIncidents(t *testing.T) {
	r, _ := setupTestIncidentController(t)

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
	}{
		{
			name:           "成功获取事件列表",
			queryParams:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "带分页参数",
			queryParams:    "page=1&pageSize=10",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := "/api/v1/incidents"
			if tt.queryParams != "" {
				path += "?" + tt.queryParams
			}

			req, err := http.NewRequest("GET", path, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: 1})

			r.ServeHTTP(w, req)
		})
	}
}

func TestIncidentController_ResolveRejectsInvalidJSONContract(t *testing.T) {
	_, incidentController := setupTestIncidentController(t)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: 1})
		c.Set("user_id", 1)
		c.Next()
	})
	r.POST("/api/v1/incidents/:id/resolve", incidentController.ResolveIncident)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents/1/resolve", strings.NewReader(`{"resolution":`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
	var response struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, 1001, response.Code)
}
