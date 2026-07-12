package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
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

// ---------------------- MigrateWorkflowToBPMN tests ----------------------

// setupMigrateTestRouter builds a router wired to MigrateWorkflowToBPMN only,
// using an isolated SQLite database for each test so that ProcessDefinition
// rows from prior tests don't leak.
func setupMigrateTestRouter(t *testing.T) (*gin.Engine, *ent.Client) {
	gin.SetMode(gin.TestMode)
	dbName := fmt.Sprintf("file:migrate_workflow_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano())
	client := enttest.Open(t, "sqlite3", dbName)
	logger := zaptest.NewLogger(t).Sugar()
	approvalSvc := service.NewApprovalService(client, logger)
	approvalCtrl := NewApprovalController(approvalSvc)

	r := gin.New()
	r.Use(gin.Recovery())
	// Tenant middleware: X-Test-Tenant header overrides the default tenant id.
	r.Use(func(c *gin.Context) {
		tenantID := 1
		if h := c.GetHeader("X-Test-Tenant"); h != "" {
			if v, err := strconv.Atoi(h); err == nil {
				tenantID = v
			}
		}
		c.Set("tenant_id", tenantID)
		c.Next()
	})
	r.POST("/api/v1/approval-workflows/:id/migrate-to-bpmn", approvalCtrl.MigrateWorkflowToBPMN)
	return r, client
}

// seedLegacyWorkflow creates an ApprovalWorkflow with at least one node so the
// Migrate() builder has something to render into BPMN XML.
func seedLegacyWorkflow(t *testing.T, client *ent.Client, tenantID int, name string) *ent.ApprovalWorkflow {
	t.Helper()
	ctx := context.Background()
	nodes := []map[string]interface{}{
		{
			"name":          "Manager Review",
			"step_order":    1,
			"assignee_type": "user",
			"assignee_value": "alice",
		},
		{
			"name":          "Director Approval",
			"step_order":    2,
			"assignee_type": "role",
			"assignee_value": "director",
		},
	}
	wf, err := client.ApprovalWorkflow.Create().
		SetName(name).
		SetDescription("legacy workflow under test").
		SetTicketType("incident").
		SetPriority("high").
		SetNodes(nodes).
		SetIsActive(true).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return wf
}

func countProcessDefinitions(t *testing.T, client *ent.Client, key string, tenantID int) int {
	t.Helper()
	ctx := context.Background()
	n, err := client.ProcessDefinition.Query().All(ctx)
	require.NoError(t, err)
	count := 0
	for _, p := range n {
		if p.TenantID == tenantID && p.Key == key {
			count++
		}
	}
	return count
}

func TestMigrateWorkflowToBPMN_DryRun(t *testing.T) {
	r, client := setupMigrateTestRouter(t)
	defer client.Close()

	wf := seedLegacyWorkflow(t, client, 1, "dry-run-flow")
	expectedKey := fmt.Sprintf("legacy_approval_%d", wf.ID)

	body, _ := json.Marshal(map[string]string{})
	req, _ := http.NewRequest("POST", "/api/v1/approval-workflows/"+strconv.Itoa(wf.ID)+"/migrate-to-bpmn?dryRun=true", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	var resp struct {
		Code int                    `json:"code"`
		Data map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, float64(wf.ID), resp.Data["workflowId"])
	assert.Equal(t, expectedKey, resp.Data["processDefinitionKey"])
	// BPMN XML must be returned in the dry-run preview.
	bpmnXML, _ := resp.Data["bpmnXml"].(string)
	assert.NotEmpty(t, bpmnXML, "dry-run must return generated BPMN XML")
	// Critically: no ProcessDefinition row should be persisted during dry-run.
	assert.Equal(t, 0, countProcessDefinitions(t, client, expectedKey, 1),
		"dry-run must not persist any ProcessDefinition row")
}

func TestMigrateWorkflowToBPMN_Persists(t *testing.T) {
	r, client := setupMigrateTestRouter(t)
	defer client.Close()

	wf := seedLegacyWorkflow(t, client, 1, "persist-flow")
	expectedKey := fmt.Sprintf("legacy_approval_%d", wf.ID)

	// 1. dryRun=true must not persist.
	preCount := countProcessDefinitions(t, client, expectedKey, 1)

	body, _ := json.Marshal(map[string]string{})
	req, _ := http.NewRequest("POST", "/api/v1/approval-workflows/"+strconv.Itoa(wf.ID)+"/migrate-to-bpmn?dryRun=true", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, preCount, countProcessDefinitions(t, client, expectedKey, 1))

	// 2. Without dryRun, the ProcessDefinition row should appear.
	req2, _ := http.NewRequest("POST", "/api/v1/approval-workflows/"+strconv.Itoa(wf.ID)+"/migrate-to-bpmn", bytes.NewReader(body))
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	require.Equal(t, http.StatusOK, w2.Code, "body=%s", w2.Body.String())
	var resp struct {
		Code int                    `json:"code"`
		Data map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, expectedKey, resp.Data["processDefinitionKey"])
	assert.Equal(t, false, resp.Data["skipped"])

	// Verify a ProcessDefinition row was persisted for this tenant + key.
	assert.Equal(t, 1, countProcessDefinitions(t, client, expectedKey, 1),
		"persist path must create exactly one ProcessDefinition row")

	// Verify a ProcessBinding was created as well.
	bindingCount, err := client.ProcessBinding.Query().All(context.Background())
	require.NoError(t, err)
	foundBinding := false
	for _, b := range bindingCount {
		if b.ProcessDefinitionKey == expectedKey && b.TenantID == 1 {
			foundBinding = true
			break
		}
	}
	assert.True(t, foundBinding, "persist path must create a ProcessBinding for the migrated key")
}

func TestMigrateWorkflowToBPMN_NotFound(t *testing.T) {
	r, client := setupMigrateTestRouter(t)
	defer client.Close()

	body, _ := json.Marshal(map[string]string{})
	req, _ := http.NewRequest("POST", "/api/v1/approval-workflows/999999/migrate-to-bpmn?dryRun=true", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Service returns "legacy approval workflow not found: ..." wrapped in fmt.Errorf,
	// controller maps that to InternalError → HTTP 500 with code 5001.
	require.Equal(t, http.StatusInternalServerError, w.Code, "body=%s", w.Body.String())
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(5001), resp["code"])
	assert.Contains(t, resp["message"], "not found")
}

func TestMigrateWorkflowToBPMN_BadID(t *testing.T) {
	r, client := setupMigrateTestRouter(t)
	defer client.Close()

	body, _ := json.Marshal(map[string]string{})
	req, _ := http.NewRequest("POST", "/api/v1/approval-workflows/notanumber/migrate-to-bpmn", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code, "non-numeric id must return 400, body=%s", w.Body.String())
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(1001), resp["code"], "param error code must be 1001")
}
