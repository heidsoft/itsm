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

func setupTestApprovalController(t *testing.T) *gin.Engine {
	gin.SetMode(gin.TestMode)

	// 创建内存数据库
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")

	// 创建 logger
	logger := zaptest.NewLogger(t).Sugar()

	// 创建服务
	approvalService := service.NewApprovalService(client, logger)

	// 创建控制器
	approvalController := NewApprovalController(approvalService)

	// 创建路由
	r := gin.New()
	r.Use(gin.Recovery())

	// 注册路由
	r.GET("/api/v1/approvals/workflows", approvalController.ListWorkflows)
	r.POST("/api/v1/approvals/workflows", approvalController.CreateWorkflow)

	return r
}

func TestApprovalController_ListWorkflows(t *testing.T) {
	r := setupTestApprovalController(t)

	tests := []struct {
		name string
	}{
		{
			name: "获取审批工作流列表",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/approvals/workflows", nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set("tenant_id", 1)

			r.ServeHTTP(w, req)
		})
	}
}

func TestApprovalController_CreateWorkflow(t *testing.T) {
	r := setupTestApprovalController(t)

	tests := []struct {
		name string
	}{
		{
			name: "创建审批工作流",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "/api/v1/approvals/workflows", nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set("tenant_id", 1)

			r.ServeHTTP(w, req)
		})
	}
}
