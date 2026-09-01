package approval

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/ent/enttest"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func setupTestHandler(t *testing.T) *gin.Engine {
	gin.SetMode(gin.TestMode)

	// 创建内存数据库
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")

	// 创建 logger
	logger := zaptest.NewLogger(t).Sugar()

	// 创建服务与 handler
	approvalService := service.NewApprovalService(client, logger)
	h := NewHandler(approvalService)

	// 创建路由
	r := gin.New()
	r.Use(gin.Recovery())

	// Tenant context injection: every request gets tenant_id=1 unless
	// X-Test-Tenant header overrides it. Required because handlers read
	// tenant_id via context, and ServeHTTP builds its own context
	// (c.Set on a separate context is not visible).
	r.Use(func(c *gin.Context) {
		tenantID := 1
		if h := c.GetHeader("X-Test-Tenant"); h != "" {
			if v, err := strconv.Atoi(h); err == nil {
				tenantID = v
			}
		}
		userID := 1
		if h := c.GetHeader("X-Test-User"); h != "" {
			if v, err := strconv.Atoi(h); err == nil {
				userID = v
			}
		}
		c.Set("tenant_id", tenantID)
		c.Set("user_id", userID)
		c.Next()
	})

	// 注册路由 - mirror router.go canonical contract /approval-workflows.
	r.GET("/api/v1/approval-workflows", h.ListWorkflows)
	r.POST("/api/v1/approval-workflows", h.CreateWorkflow)
	r.GET("/api/v1/approval-workflows/:id", h.GetWorkflow)
	r.PUT("/api/v1/approval-workflows/:id", h.UpdateWorkflow)
	r.PATCH("/api/v1/approval-workflows/:id", h.PatchWorkflow)
	r.DELETE("/api/v1/approval-workflows/:id", h.DeleteWorkflow)
	r.POST("/api/v1/approval-workflows/:id/migrate-to-bpmn", h.MigrateWorkflowToBPMN)

	return r
}

func TestHandler_ListWorkflows(t *testing.T) {
	r := setupTestHandler(t)

	req, err := http.NewRequest("GET", "/api/v1/approval-workflows", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	var resp struct {
		Code int         `json:"code"`
		Data interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code, "expected success code, got %d", resp.Code)
}

func TestHandler_CreateWorkflow(t *testing.T) {
	r := setupTestHandler(t)

	body := []byte(`{"name":"test","ticketType":"incident","priority":"high","isActive":true,"nodes":[{"level":1,"name":"主管审批","approverType":"role","approverIds":[1],"approvalMode":"any","allowReject":true,"allowDelegate":false,"rejectAction":"end"}]}`)
	req, err := http.NewRequest("POST", "/api/v1/approval-workflows", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	var resp struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

func TestMigrateWorkflowToBPMN_DryRun(t *testing.T) {
	r := setupTestHandler(t)

	// 先创建一个工作流
	body := []byte(`{"name":"migrate-me","ticketType":"incident","priority":"high","isActive":true,"nodes":[{"level":1,"name":"主管审批","approverType":"role","approverIds":[1],"approvalMode":"any","allowReject":true,"allowDelegate":false,"rejectAction":"end"}]}`)
	req, err := http.NewRequest("POST", "/api/v1/approval-workflows", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	var created struct {
		Code int `json:"code"`
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	require.NotZero(t, created.Data.ID)

	// dryRun 迁移
	url := fmt.Sprintf("/api/v1/approval-workflows/%d/migrate-to-bpmn?dryRun=true", created.Data.ID)
	req2, err := http.NewRequest("POST", url, nil)
	require.NoError(t, err)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code, "body=%s", w2.Body.String())
	var resp struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

func TestMigrateWorkflowToBPMN_NotFound(t *testing.T) {
	r := setupTestHandler(t)

	req, err := http.NewRequest("POST", "/api/v1/approval-workflows/999999/migrate-to-bpmn?dryRun=true", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Service returns "legacy approval workflow not found: ..." wrapped in fmt.Errorf,
	// handler maps that to InternalError → HTTP 500 with code 5001（与旧 controller 契约一致）.
	require.Equal(t, http.StatusInternalServerError, w.Code, "body=%s", w.Body.String())
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(5001), resp["code"])
	assert.Contains(t, resp["message"], "not found")
}

func TestMigrateWorkflowToBPMN_BadID(t *testing.T) {
	r := setupTestHandler(t)

	req, err := http.NewRequest("POST", "/api/v1/approval-workflows/abc/migrate-to-bpmn", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code, "non-numeric id must return 400, body=%s", w.Body.String())
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(1001), resp["code"], "param error code must be 1001")
}

func TestGetWorkflow_NotFound(t *testing.T) {
	r := setupTestHandler(t)

	req, err := http.NewRequest("GET", "/api/v1/approval-workflows/999999", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
}
