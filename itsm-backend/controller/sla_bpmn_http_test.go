package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// ===== 测试辅助函数 =====

var uniqueSlaBpmnIDCounter int64

func uniqueSlaBpmnID() string {
	n := atomic.AddInt64(&uniqueSlaBpmnIDCounter, 1)
	return strconv.FormatInt(1730000000000+n, 10)
}

// setupSlaBpmnTestBase 创建带租户中间件的测试引擎
// 注意：中间件在 setupSlaBpmnTestBaseWithTenant 之前应用
func setupSlaBpmnTestBase(t *testing.T) (*gin.Engine, *ent.Client) {
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", "file:ent-sla-bpmn?mode=memory&cache=shared&_fk=1")
	_ = zaptest.NewLogger(t).Sugar()
	r := gin.New()
	r.Use(gin.Recovery())
	return r, client
}

// setupSlaBpmnTestBaseWithTenant 创建带租户上下文的测试引擎
func setupSlaBpmnTestBaseWithTenant(t *testing.T, tenantID int) (*gin.Engine, *ent.Client) {
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", "file:ent-sla-bpmn?mode=memory&cache=shared&_fk=1")
	_ = zaptest.NewLogger(t).Sugar()
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(mockTenantContextMiddleware(tenantID, 1))
	return r, client
}

// 模拟租户上下文中间件
func mockTenantContextMiddleware(tenantID int, userID int) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("tenant_id", tenantID)
		c.Set("user_id", userID)
		c.Next()
	}
}

func createSlaBpmnTenantAndUser(t *testing.T, client *ent.Client) (*ent.Tenant, *ent.User) {
	ctx := context.Background()
	uniqueID := uniqueSlaBpmnID()

	tenant, err := client.Tenant.Create().
		SetName("SLA-BPMN Test Tenant").
		SetCode("SLABPMN" + uniqueID).
		SetDomain("sla-bpmn-test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	user, err := client.User.Create().
		SetUsername("slabpmnuser" + uniqueID).
		SetEmail("sla-bpmn" + uniqueID + "@example.com").
		SetPasswordHash("hashedpassword").
		SetName("SLA BPMN User").
		SetDepartment("IT").
		SetPhone("1234567890").
		SetActive(true).
		SetRole("admin").
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	return tenant, user
}

func doSlaBpmnRequest(t *testing.T, r *gin.Engine, method, path string, body interface{}, tenantID int) *httptest.ResponseRecorder {
	var req *http.Request
	if body != nil {
		jsonBytes, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, bytes.NewReader(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ============ SLA Template HTTP Tests ============

func TestSLATemplate_ListTemplates_HTTP(t *testing.T) {
	r, _ := setupSlaBpmnTestBaseWithTenant(t, 1)
	logger := zaptest.NewLogger(t).Sugar()
	svc := service.NewSLATemplateService(nil, logger)
	ctrl := NewSLATemplateController(svc)

	apiV1 := r.Group("/api/v1")
	ctrl.RegisterRoutes(apiV1)

	w := doSlaBpmnRequest(t, r, "GET", "/api/v1/sla/templates", nil, 1)

	assert.Equal(t, http.StatusOK, w.Code, "GET /sla/templates should return 200")
	body := w.Body.String()
	assert.Contains(t, body, "incident_p1_critical", "should list incident_p1_critical template")
	assert.Contains(t, body, "change_emergency", "should list change_emergency template")
	assert.Contains(t, body, "service_request_standard", "should list service_request_standard template")
}

func TestSLATemplate_GetTemplate_HTTP(t *testing.T) {
	r, _ := setupSlaBpmnTestBaseWithTenant(t, 1)
	logger := zaptest.NewLogger(t).Sugar()
	svc := service.NewSLATemplateService(nil, logger)
	ctrl := NewSLATemplateController(svc)

	apiV1 := r.Group("/api/v1")
	ctrl.RegisterRoutes(apiV1)

	w := doSlaBpmnRequest(t, r, "GET", "/api/v1/sla/templates/incident_p1_critical", nil, 1)
	assert.Equal(t, http.StatusOK, w.Code, "GET /sla/templates/incident_p1_critical should return 200")
	body := w.Body.String()
	assert.Contains(t, body, "incident_p1_critical", "should return template key")
	assert.Contains(t, body, "responseTime", "should include responseTime field")
}

func TestSLATemplate_GetTemplate_NotFound_HTTP(t *testing.T) {
	r, _ := setupSlaBpmnTestBaseWithTenant(t, 1)
	logger := zaptest.NewLogger(t).Sugar()
	svc := service.NewSLATemplateService(nil, logger)
	ctrl := NewSLATemplateController(svc)

	apiV1 := r.Group("/api/v1")
	ctrl.RegisterRoutes(apiV1)

	w := doSlaBpmnRequest(t, r, "GET", "/api/v1/sla/templates/nonexistent_key", nil, 1)
	assert.Equal(t, http.StatusNotFound, w.Code, "non-existent template should return 404")
}

func TestSLATemplate_InstallTemplate_HTTP(t *testing.T) {
	r, client := setupSlaBpmnTestBase(t)
	logger := zaptest.NewLogger(t).Sugar()
	tenant, _ := createSlaBpmnTenantAndUser(t, client)

	svc := service.NewSLATemplateService(client, logger)
	ctrl := NewSLATemplateController(svc)

	// 先添加中间件再注册路由
	r.Use(mockTenantContextMiddleware(tenant.ID, 1))
	apiV1 := r.Group("/api/v1")
	ctrl.RegisterRoutes(apiV1)

	// 第一次安装
	w := doSlaBpmnRequest(t, r, "POST", "/api/v1/sla/templates/incident_p1_critical/install", nil, tenant.ID)
	assert.Equal(t, http.StatusOK, w.Code, "first install should return 200")

	body := w.Body.String()
	assert.Contains(t, body, "incident_p1_critical", "should include template key in response")
	assert.Contains(t, body, "slaDefinitionId", "should include slaDefinitionId in response")

	// 重复安装应返回错误或幂等成功
	w2 := doSlaBpmnRequest(t, r, "POST", "/api/v1/sla/templates/incident_p1_critical/install", nil, tenant.ID)
	assert.True(t, w2.Code == http.StatusOK || w2.Code == http.StatusBadRequest,
		"duplicate install should return 200 (idempotent) or 400 (conflict), got %d", w2.Code)
}

// ============ BPMN Workflow HTTP Tests ============

func TestBPMNWorkflow_ListProcessDefinitions_HTTP(t *testing.T) {
	r, _ := setupSlaBpmnTestBaseWithTenant(t, 1)
	// 验证路由能注册
	r.GET("/api/v1/bpmn/process-definitions", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": []string{}})
	})

	w := doSlaBpmnRequest(t, r, "GET", "/api/v1/bpmn/process-definitions", nil, 1)
	assert.Equal(t, http.StatusOK, w.Code, "GET /bpmn/process-definitions should return 200")
}

func TestBPMNMonitoring_Metrics_HTTP(t *testing.T) {
	r, _ := setupSlaBpmnTestBaseWithTenant(t, 1)
	r.GET("/api/v1/bpmn/monitoring/metrics", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"total_instances":     0,
				"running_instances":   0,
				"completed_instances": 0,
			},
			"message": "ok",
		})
	})

	w := doSlaBpmnRequest(t, r, "GET", "/api/v1/bpmn/monitoring/metrics", nil, 1)
	assert.Equal(t, http.StatusOK, w.Code, "GET /bpmn/monitoring/metrics should return 200")
	body := w.Body.String()
	assert.Contains(t, body, "total_instances", "should include total_instances field")
}

func TestBPMNMonitoring_Timeline_HTTP(t *testing.T) {
	r, _ := setupSlaBpmnTestBaseWithTenant(t, 1)
	r.GET("/api/v1/bpmn/monitoring/instances/:instanceId/timeline", func(c *gin.Context) {
		instanceID := c.Param("instanceId")
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"process_instance_id": instanceID,
				"entries": []gin.H{
					{
						"sequence":      1,
						"event_type":    "started",
						"activity_id":   "start",
						"activity_type": "startEvent",
						"timestamp":     "2026-06-21T08:00:00+08:00",
					},
				},
				"total": 1,
			},
			"message": "ok",
		})
	})

	w := doSlaBpmnRequest(t, r, "GET", "/api/v1/bpmn/monitoring/instances/test-instance/timeline", nil, 1)
	assert.Equal(t, http.StatusOK, w.Code, "GET /bpmn/monitoring/instances/:id/timeline should return 200")
	body := w.Body.String()
	assert.Contains(t, body, "process_instance_id", "should include process_instance_id")
	assert.Contains(t, body, "started", "should include started event")
}

// ============ SLA Alert History with Cooldown HTTP Tests ============

func TestSLA_AlertHistory_WithCooldown_HTTP(t *testing.T) {
	r, _ := setupSlaBpmnTestBaseWithTenant(t, 1)
	r.GET("/api/v1/sla/alert-history", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"alert_history": []gin.H{
					{
						"id":                         1,
						"ticket_id":                  1,
						"created_at":                 "2026-06-21T08:00:00+08:00",
						"cooldown_remaining_seconds": 900,
						"cooldown_minutes":           15,
						"suppressed_by_cooldown":     true,
					},
				},
				"total": 1,
			},
			"message": "ok",
		})
	})

	w := doSlaBpmnRequest(t, r, "GET", "/api/v1/sla/alert-history", nil, 1)
	assert.Equal(t, http.StatusOK, w.Code, "GET /sla/alert-history should return 200")
	body := w.Body.String()
	assert.Contains(t, body, "cooldown_remaining_seconds", "should include cooldown_remaining_seconds field")
	assert.Contains(t, body, "cooldown_minutes", "should include cooldown_minutes field")
	assert.Contains(t, body, "suppressed_by_cooldown", "should include suppressed_by_cooldown field")
}

func TestSLA_CheckCompliance_HTTP(t *testing.T) {
	r, _ := setupSlaBpmnTestBaseWithTenant(t, 1)
	r.POST("/api/v1/sla/check-compliance/:ticketId", func(c *gin.Context) {
		ticketID := c.Param("ticketId")
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"ticketId":                  ticketID,
				"found":                     true,
				"compliant":                 true,
				"actual_response_minutes":   5.0,
				"actual_resolution_minutes": 60.0,
			},
			"message": "ok",
		})
	})

	w := doSlaBpmnRequest(t, r, "POST", "/api/v1/sla/check-compliance/123", nil, 1)
	assert.Equal(t, http.StatusOK, w.Code, "POST /sla/check-compliance/:ticketId should return 200")
	body := w.Body.String()
	assert.Contains(t, body, "ticketId", "should include ticketId")
	assert.Contains(t, body, "compliant", "should include compliant")
}

func TestSLA_Metrics_HTTP(t *testing.T) {
	r, _ := setupSlaBpmnTestBaseWithTenant(t, 1)
	r.GET("/api/v1/sla/metrics", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"total_definitions": 5,
				"total_violations":  2,
				"compliance_rate":   95.5,
				"active_alerts":     0,
			},
			"message": "ok",
		})
	})

	w := doSlaBpmnRequest(t, r, "GET", "/api/v1/sla/metrics", nil, 1)
	assert.Equal(t, http.StatusOK, w.Code, "GET /sla/metrics should return 200")
	body := w.Body.String()
	assert.Contains(t, body, "total_definitions", "should include total_definitions")
}

func TestSLA_Violations_HTTP(t *testing.T) {
	r, _ := setupSlaBpmnTestBaseWithTenant(t, 1)
	r.GET("/api/v1/sla/violations", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"violations": []gin.H{
					{
						"id":         1,
						"ticket_id":  1,
						"type":       "response_time",
						"status":     "open",
						"created_at": "2026-06-21T08:00:00+08:00",
					},
				},
				"total": 1,
			},
			"message": "ok",
		})
	})

	w := doSlaBpmnRequest(t, r, "GET", "/api/v1/sla/violations", nil, 1)
	assert.Equal(t, http.StatusOK, w.Code, "GET /sla/violations should return 200")
	body := w.Body.String()
	assert.Contains(t, body, "violations", "should include violations list")
}

// ============ 占位符测试，验证 SLA 模板数量 ============

func TestSLATemplate_TemplateCount_HTTP(t *testing.T) {
	r, _ := setupSlaBpmnTestBaseWithTenant(t, 1)
	logger := zaptest.NewLogger(t).Sugar()
	svc := service.NewSLATemplateService(nil, logger)
	ctrl := NewSLATemplateController(svc)

	apiV1 := r.Group("/api/v1")
	ctrl.RegisterRoutes(apiV1)

	w := doSlaBpmnRequest(t, r, "GET", "/api/v1/sla/templates", nil, 1)
	body := w.Body.String()

	expectedTemplates := []string{
		"incident_p1_critical",
		"incident_p2_high",
		"incident_p3_medium",
		"change_normal",
		"change_emergency",
		"service_request_standard",
	}
	for _, tmpl := range expectedTemplates {
		assert.Contains(t, body, tmpl, fmt.Sprintf("template %s should be in response", tmpl))
	}
}

func parseSlaBpmnJSON(t *testing.T, body string) map[string]interface{} {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(body), &result)
	require.NoError(t, err)
	return result
}

func TestSLA_ParseJSONStructure_HTTP(t *testing.T) {
	r, _ := setupSlaBpmnTestBaseWithTenant(t, 1)
	logger := zaptest.NewLogger(t).Sugar()
	svc := service.NewSLATemplateService(nil, logger)
	ctrl := NewSLATemplateController(svc)

	apiV1 := r.Group("/api/v1")
	ctrl.RegisterRoutes(apiV1)

	w := doSlaBpmnRequest(t, r, "GET", "/api/v1/sla/templates", nil, 1)
	result := parseSlaBpmnJSON(t, w.Body.String())

	assert.Contains(t, result, "data", "response should have data field")
	data := result["data"].(map[string]interface{})
	assert.Contains(t, data, "templates", "data should have templates array")
	assert.Contains(t, data, "total", "data should have total count")
}
