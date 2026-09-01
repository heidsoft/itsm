package ticketworkflow

import (
	"bytes"
	"database/sql"
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

	dbName := "file:tw_test_" + t.Name() + "?mode=memory&cache=shared&_fk=1"
	client := enttest.Open(t, "sqlite3", dbName)
	t.Cleanup(func() { client.Close() })

	db, err := sql.Open("sqlite3", dbName)
	require.NoError(t, err)

	logger := zaptest.NewLogger(t).Sugar()
	workflowService := service.NewTicketWorkflowService(client, logger)
	h := NewHandler(workflowService, db, logger)

	r := gin.New()
	r.Use(gin.Recovery())

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

	// 注册路由 - mirror router.go 契约 /tickets/workflow/*
	r.POST("/api/v1/tickets/workflow/accept", h.AcceptTicket)
	r.POST("/api/v1/tickets/workflow/approve", h.ApproveTicket)
	r.POST("/api/v1/tickets/workflow/resolve", h.ResolveTicket)
	r.GET("/api/v1/tickets/cc/my", h.ListMyCCRecords)
	r.GET("/api/v1/tickets/:id/cc", h.ListTicketCCRecords)
	r.GET("/api/v1/tickets/:id/workflow/state", h.GetTicketWorkflowState)
	r.GET("/api/v1/tickets/:id/workflow-history", h.GetTicketWorkflowHistory)

	return r
}

func TestHandler_AcceptTicket_EmptyBody(t *testing.T) {
	r := setupTestHandler(t)

	// AcceptTicketRequest 无 binding required → 空对象绑定成功，
	// TicketID=0 传给 service 报错 → 500（与旧 controller 契约一致）
	body := []byte(`{}`)
	req, err := http.NewRequest("POST", "/api/v1/tickets/workflow/accept", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code, "body=%s", w.Body.String())
}

func TestHandler_AcceptTicket_TicketNotFound(t *testing.T) {
	r := setupTestHandler(t)

	body := []byte(`{"ticketId":99999,"comment":"接单"}`)
	req, err := http.NewRequest("POST", "/api/v1/tickets/workflow/accept", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// service 层对不存在工单返回错误 → 500（与旧契约一致）
	assert.Equal(t, http.StatusInternalServerError, w.Code, "body=%s", w.Body.String())
}

func TestHandler_AcceptTicket_MissingTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dbName := "file:tw_notenant_" + t.Name() + "?mode=memory&_fk=1"
	client := enttest.Open(t, "sqlite3", dbName)
	t.Cleanup(func() { client.Close() })
	db, err := sql.Open("sqlite3", dbName)
	require.NoError(t, err)
	logger := zaptest.NewLogger(t).Sugar()
	workflowService := service.NewTicketWorkflowService(client, logger)
	h := NewHandler(workflowService, db, logger)

	r := gin.New()
	r.Use(gin.Recovery())
	// 不注入 tenant_id/user_id → getAuthContext 应 401
	r.POST("/api/v1/tickets/workflow/accept", h.AcceptTicket)

	body := []byte(`{"ticketId":1,"comment":"x"}`)
	req, err := http.NewRequest("POST", "/api/v1/tickets/workflow/accept", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code, "缺 tenant 上下文应 401")
}

func TestHandler_ListTicketCCRecords_InvalidID(t *testing.T) {
	r := setupTestHandler(t)

	req, err := http.NewRequest("GET", "/api/v1/tickets/invalid/cc", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetTicketWorkflowState_InvalidID(t *testing.T) {
	r := setupTestHandler(t)

	req, err := http.NewRequest("GET", "/api/v1/tickets/invalid/workflow/state", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetTicketWorkflowHistory_EmptyRecords(t *testing.T) {
	r := setupTestHandler(t)

	// ticket_workflow_records 表在 enttest 自动迁移中创建；查询不存在的工单 → 空列表
	req, err := http.NewRequest("GET", "/api/v1/tickets/99999/workflow-history", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// 空结果应 200 + 空数组（与旧契约一致：records == nil → []）
	assert.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
}

func TestHandler_ApproveTicket_BadRequest(t *testing.T) {
	r := setupTestHandler(t)

	// ApproveTicketRequest 的 Action 带 binding required,oneof
	body := []byte(`{"ticketId":1}`)
	req, err := http.NewRequest("POST", "/api/v1/tickets/workflow/approve", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
}
