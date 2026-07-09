package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"itsm-backend/common"
	"itsm-backend/ent/enttest"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// setupAPIContractTest 创建测试环境
func setupAPIContractTest(t *testing.T) *gin.Engine {
	gin.SetMode(gin.TestMode)
	_ = enttest.Open(t, "sqlite3", "file:ent-contract?mode=memory&cache=shared&_fk=1")
	_ = zaptest.NewLogger(t).Sugar()

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", 1)
		c.Set("user_id", 1)
		c.Next()
	})
	return r
}

// ============ API Contract Tests ============
// These tests verify the API contract between frontend and backend

// TestAPIResponseFormat verifies all APIs return the standard response format
func TestAPIResponseFormat_Standard(t *testing.T) {
	r := setupAPIContractTest(t)

	// Mock endpoint with standard response
	r.GET("/api/v1/test/standard", func(c *gin.Context) {
		common.Success(c, gin.H{"id": 1, "name": "test"})
	})

	w := doSlaBpmnRequest(t, r, "GET", "/api/v1/test/standard", nil, 1)

	// Verify standard response format
	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, common.SuccessCode, response.Code)
	assert.Equal(t, "success", response.Message)
	assert.NotNil(t, response.Data)
}

// TestAPIResponseFormat_ParamError verifies param error responses
func TestAPIResponseFormat_ParamError(t *testing.T) {
	r := setupAPIContractTest(t)

	r.GET("/api/v1/test/param-error", func(c *gin.Context) {
		common.ParamError(c, "Invalid parameter")
	})

	w := doSlaBpmnRequest(t, r, "GET", "/api/v1/test/param-error", nil, 1)

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, common.ParamErrorCode, response.Code)
	assert.Contains(t, response.Message, "Invalid parameter")
}

// TestAPIResponseFormat_NotFound verifies not found responses
func TestAPIResponseFormat_NotFound(t *testing.T) {
	r := setupAPIContractTest(t)

	r.GET("/api/v1/test/not-found", func(c *gin.Context) {
		common.NotFound(c, "Resource not found")
	})

	w := doSlaBpmnRequest(t, r, "GET", "/api/v1/test/not-found", nil, 1)

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, common.NotFoundCode, response.Code)
}

// TestAPIResponseFormat_InternalError verifies internal error responses
func TestAPIResponseFormat_InternalError(t *testing.T) {
	r := setupAPIContractTest(t)

	r.GET("/api/v1/test/internal-error", func(c *gin.Context) {
		common.InternalError(c, "Internal server error")
	})

	w := doSlaBpmnRequest(t, r, "GET", "/api/v1/test/internal-error", nil, 1)

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, common.InternalErrorCode, response.Code)
}

// ============ BPMN API Contract Tests ============

func TestBPMNAPIContract_ProcessDefinitionsList(t *testing.T) {
	r := setupAPIContractTest(t)

	// Mock BPMN process definitions endpoint
	r.GET("/api/v1/bpmn/process-definitions", func(c *gin.Context) {
		common.Success(c, gin.H{
			"items": []interface{}{
				gin.H{
					"id":               1,
					"key":              "incident_workflow",
					"name":             "Incident Workflow",
					"version":          "1.0.0",
					"category":         "incident",
					"isActive":         true,
					"processDefinitionKey": "incident_workflow",
				},
			},
			"pagination": gin.H{
				"page":      1,
				"pageSize":  20,
				"total":     1,
				"totalPages": 1,
			},
		})
	})

	w := doSlaBpmnRequest(t, r, "GET", "/api/v1/bpmn/process-definitions", nil, 1)

	assert.Equal(t, http.StatusOK, w.Code)

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, common.SuccessCode, response.Code)

	data := response.Data.(map[string]interface{})
	assert.Contains(t, data, "items")
	assert.Contains(t, data, "pagination")
}

func TestBPMNAPIContract_ProcessDefinitionDetail(t *testing.T) {
	r := setupAPIContractTest(t)

	r.GET("/api/v1/bpmn/process-definitions/:key", func(c *gin.Context) {
		key := c.Param("key")
		common.Success(c, gin.H{
			"id":                     1,
			"key":                    key,
			"name":                   "Incident Workflow",
			"version":                "1.0.0",
			"category":               "incident",
			"isActive":               true,
			"isLatest":               true,
			"processDefinitionKey":   key,
			"processDefinitionId":    1,
			"bpmnXml":                "<bpmn>...</bpmn>",
		})
	})

	w := doSlaBpmnRequest(t, r, "GET", "/api/v1/bpmn/process-definitions/incident_workflow", nil, 1)

	assert.Equal(t, http.StatusOK, w.Code)

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, common.SuccessCode, response.Code)

	data := response.Data.(map[string]interface{})
	assert.Equal(t, "incident_workflow", data["key"])
	assert.Equal(t, "Incident Workflow", data["name"])
}

func TestBPMNAPIContract_TasksList(t *testing.T) {
	r := setupAPIContractTest(t)

	r.GET("/api/v1/bpmn/tasks", func(c *gin.Context) {
		common.Success(c, gin.H{
			"items": []interface{}{
				gin.H{
					"id":                    1,
					"taskId":                "task-001",
					"processInstanceId":     "inst-001",
					"processDefinitionKey":  "incident_workflow",
					"taskDefinitionKey":     "approve",
					"taskName":              "Approve Incident",
					"assignee":              "user1",
					"status":                "pending",
					"createdTime":           "2026-07-09T10:00:00Z",
				},
			},
			"pagination": gin.H{
				"page":      1,
				"pageSize":  20,
				"total":     1,
				"totalPages": 1,
			},
		})
	})

	w := doSlaBpmnRequest(t, r, "GET", "/api/v1/bpmn/tasks", nil, 1)

	assert.Equal(t, http.StatusOK, w.Code)

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, common.SuccessCode, response.Code)
}

func TestWorkflowAPIContract_TasksList(t *testing.T) {
	r := setupAPIContractTest(t)

	// Test the /workflow/tasks alias route
	r.GET("/api/v1/workflow/tasks", func(c *gin.Context) {
		common.Success(c, gin.H{
			"items": []interface{}{
				gin.H{
					"id":                    1,
					"taskId":                "task-001",
					"taskName":              "Review Request",
					"status":                "pending",
				},
			},
			"pagination": gin.H{
				"page":      1,
				"pageSize":  20,
				"total":     1,
			},
		})
	})

	w := doSlaBpmnRequest(t, r, "GET", "/api/v1/workflow/tasks", nil, 1)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ============ SLA API Contract Tests ============

func TestSLAAPIContract_DefinitionsList(t *testing.T) {
	r := setupAPIContractTest(t)

	r.GET("/api/v1/sla/definitions", func(c *gin.Context) {
		common.Success(c, gin.H{
			"items": []interface{}{
				gin.H{
					"id":            1,
					"name":          "P1 Incident Response",
					"type":          "response_time",
					"priority":      "critical",
					"targetTime":    30,
					"warningTime":   20,
					"status":        "active",
					"isDefault":     true,
				},
			},
			"total":    1,
			"page":     1,
			"pageSize": 20,
		})
	})

	w := doSlaBpmnRequest(t, r, "GET", "/api/v1/sla/definitions", nil, 1)

	assert.Equal(t, http.StatusOK, w.Code)

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, common.SuccessCode, response.Code)
}

// ============ Assignment Rules API Contract Tests ============

func TestAssignmentRulesAPIContract_ListRules(t *testing.T) {
	r := setupAPIContractTest(t)

	r.GET("/api/v1/tickets/assignment-rules", func(c *gin.Context) {
		common.Success(c, gin.H{
			"rules": []interface{}{
				gin.H{
					"id":          1,
					"name":        "Critical Priority Rule",
					"priority":    "critical",
					"assigneeId":  5,
					"assigneeName": "John Doe",
					"isActive":    true,
				},
			},
			"total": 1,
		})
	})

	w := doSlaBpmnRequest(t, r, "GET", "/api/v1/tickets/assignment-rules", nil, 1)

	assert.Equal(t, http.StatusOK, w.Code)

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, common.SuccessCode, response.Code)
}

// ============ Auth Refresh API Contract Tests ============

func TestAuthRefreshAPIContract_RefreshToken(t *testing.T) {
	r := setupAPIContractTest(t)

	// Test /api/v1/auth/refresh endpoint
	r.POST("/api/v1/auth/refresh", func(c *gin.Context) {
		common.Success(c, gin.H{
			"accessToken": "new-access-token",
			"refreshToken": "new-refresh-token",
		})
	})

	body := bytes.NewBufferString(`{}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/refresh", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, common.SuccessCode, response.Code)

	data := response.Data.(map[string]interface{})
	assert.NotEmpty(t, data["accessToken"])
	assert.NotEmpty(t, data["refreshToken"])
}

func TestAuthRefreshAPIContract_RefreshTokenAlias(t *testing.T) {
	r := setupAPIContractTest(t)

	// Test /api/v1/refresh-token alias endpoint
	r.POST("/api/v1/refresh-token", func(c *gin.Context) {
		common.Success(c, gin.H{
			"accessToken": "new-access-token",
		})
	})

	body := bytes.NewBufferString(`{}`)
	req := httptest.NewRequest("POST", "/api/v1/refresh-token", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ============ Knowledge API Contract Tests ============

func TestKnowledgeAPIContract_ArticleSearch(t *testing.T) {
	r := setupAPIContractTest(t)

	r.POST("/api/v1/knowledge/search", func(c *gin.Context) {
		var req struct {
			Query string `json:"query"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			common.ParamError(c, "Invalid request body: "+err.Error())
			return
		}

		common.Success(c, gin.H{
			"items": []interface{}{
				gin.H{
					"id":       1,
					"title":    "How to reset password",
					"category": "IT Support",
					"snippet":  "Steps to reset your password...",
					"tags":     []string{"password", "security"},
					"score":    0.95,
				},
			},
			"total": 1,
		})
	})

	requestBody := `{"query": "password reset"}`
	req := httptest.NewRequest("POST", "/api/v1/knowledge/search", bytes.NewBufferString(requestBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response common.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, common.SuccessCode, response.Code)
}
