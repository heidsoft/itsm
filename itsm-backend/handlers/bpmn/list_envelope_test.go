package bpmn

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// 列表接口信封契约：/api/v1/bpmn/dashboard/audit-logs、
// /api/v1/bpmn/monitoring/instances/status 历史上分别用 list / instances / logs
// 作为数组键，前端只能各自猜一套结构。
// AGENTS.md 规定列表响应唯一形态是 data:{items,total,page,pageSize,totalPages}。
// E4-48：/bpmn/monitoring/audit-logs 重复表面已删除，收敛为 dashboard 单入口。

func newEnvelopeTestRouter(t *testing.T, client *ent.Client) *gin.Engine {
	return newEnvelopeTestRouterWithRole(t, client, "admin")
}

// newEnvelopeTestRouterWithRole 允许按用例注入认证角色：审计路由挂了
// middleware.RequirePermission("bpmn","read")（E4-48），该门要求 role+client
// 上下文键，缺失即 fail-closed 401——测试栈必须像真实 auth 中间件一样注入。
// 空库（无 Role 行）走 unconfigured 兜底词表：admin 持 bpmn:read、
// msp_viewer 不持，用作 allow/deny 两侧。role 传空串模拟"未认证"：
// 不注入任何上下文键，让 RequirePermission 走缺键 401 分支。
func newEnvelopeTestRouterWithRole(t *testing.T, client *ent.Client, role string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	logger := zaptest.NewLogger(t).Sugar()

	auditService := service.NewBPMNAuditService(client, logger)
	dashboard := NewDashboardHandler(
		service.NewBPMNMetricsService(client, logger),
		auditService,
		service.NewBPMNTenantService(client, logger),
		service.NewBPMNSLAService(client, logger),
	)
	monitoring := NewMonitoringHandler(service.NewBPMNMonitoringService(client, auditService, logger))

	r := gin.New()
	r.Use(func(c *gin.Context) {
		tenantID := 1
		if c.GetHeader("X-Test-Tenant") == "2" {
			tenantID = 2
		}
		if role != "" {
			c.Set("role", role)
			c.Set("client", client)
		}
		c.Set("tenant_id", tenantID)
		c.Next()
	})
	group := r.Group("/api/v1")
	dashboard.RegisterRoutes(group)
	monitoring.RegisterRoutes(group)
	return r
}

func seedAuditLog(t *testing.T, client *ent.Client, tenantID, instanceID int, key string, ts time.Time) {
	t.Helper()
	_, err := client.ProcessAuditLog.Create().
		SetProcessInstanceID(instanceID).
		SetProcessInstanceKey(key).
		SetProcessDefinitionKey("envelope-process").
		SetProcessDefinitionID(1).
		SetActivityID("activity-" + key).
		SetActivityType("userTask").
		SetAction("completed").
		SetTenantID(tenantID).
		SetTimestamp(ts).
		Save(context.Background())
	require.NoError(t, err)
}

func seedProcessInstance(t *testing.T, client *ent.Client, definition *ent.ProcessDefinition, tenantID int, key string, start time.Time) {
	t.Helper()
	_, err := client.ProcessInstance.Create().
		SetProcessInstanceID(key).
		SetProcessDefinitionKey(definition.Key).
		SetProcessDefinitionID(definition.ID).
		SetTenantID(tenantID).
		SetStatus("running").
		SetStartTime(start).
		Save(context.Background())
	require.NoError(t, err)
}

func seedProcessDefinition(t *testing.T, client *ent.Client, tenantID int) *ent.ProcessDefinition {
	t.Helper()
	ctx := context.Background()
	deployment, err := client.ProcessDeployment.Create().
		SetDeploymentID("envelope-deploy").
		SetDeploymentName("envelope").
		SetDeploymentTime(time.Now()).
		SetDeployedBy("test").
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	definition, err := client.ProcessDefinition.Create().
		SetKey("envelope-process").
		SetName("envelope").
		SetVersion("1").
		SetBpmnXML([]byte("<definitions/>")).
		SetDeploymentID(deployment.ID).
		SetDeployedAt(time.Now()).
		SetTenantID(tenantID).
		Save(ctx)
	require.NoError(t, err)
	return definition
}

func decodeEnvelope(t *testing.T, r *gin.Engine, path string, tenant string) map[string]interface{} {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("X-Test-Tenant", tenant)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body struct {
		Code    int                    `json:"code"`
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body), w.Body.String())
	assert.Equal(t, 0, body.Code, body.Message)
	return body.Data
}

// assertListEnvelope 断言 data 恰好是标准列表信封：只有 items + 四个分页字段，
// 且 total/pageSize/totalPages 来自后端真实计数而不是 len(items)。
// 可选传入本次请求的查询串，失败时能一眼看出是哪组分页参数。
func assertListEnvelope(t *testing.T, data map[string]interface{}, wantTotal, wantPage, wantPageSize, wantTotalPages, wantItems int, query ...string) {
	t.Helper()
	tag := ""
	if len(query) > 0 {
		tag = " [查询: " + query[0] + "]"
	}
	assert.Equal(t, []string{"items", "page", "pageSize", "total", "totalPages"}, sortedKeys(data),
		"data 键集合必须等于标准列表信封，实际 %v%s", data, tag)
	items, ok := data["items"].([]interface{})
	require.True(t, ok, "items 必须是数组，实际 %v%s", data["items"], tag)
	assert.Len(t, items, wantItems, tag)
	assert.Equal(t, float64(wantTotal), data["total"], "total%s", tag)
	assert.Equal(t, float64(wantPage), data["page"], "page%s", tag)
	assert.Equal(t, float64(wantPageSize), data["pageSize"], "pageSize%s", tag)
	assert.Equal(t, float64(wantTotalPages), data["totalPages"], "totalPages%s", tag)
}

func sortedKeys(data map[string]interface{}) []string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}

func TestDashboardAuditLogs_UsesListEnvelope(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_envelope_dashboard?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	base := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	seedAuditLog(t, client, 1, 11, "t1-a", base)
	seedAuditLog(t, client, 1, 12, "t1-b", base.Add(time.Minute))
	seedAuditLog(t, client, 1, 13, "t1-c", base.Add(2*time.Minute))
	seedAuditLog(t, client, 2, 21, "t2-a", base)

	r := newEnvelopeTestRouter(t, client)

	// 分页必须真实生效：total 是全量计数，items 是当前页。
	assertListEnvelope(t, decodeEnvelope(t, r, "/api/v1/bpmn/dashboard/audit-logs?page=1&pageSize=2", "1"), 3, 1, 2, 2, 2)
	assertListEnvelope(t, decodeEnvelope(t, r, "/api/v1/bpmn/dashboard/audit-logs?page=2&pageSize=2", "1"), 3, 2, 2, 2, 1)
	// 租户 2 只能看到自己的 1 条。
	assertListEnvelope(t, decodeEnvelope(t, r, "/api/v1/bpmn/dashboard/audit-logs?page=1&pageSize=20", "2"), 1, 1, 20, 1, 1)
}

func TestMonitoringInstancesStatus_UsesListEnvelope(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_envelope_instances?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	base := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	definition := seedProcessDefinition(t, client, 1)
	seedProcessInstance(t, client, definition, 1, "pi-1", base)
	seedProcessInstance(t, client, definition, 1, "pi-2", base.Add(time.Minute))
	seedProcessInstance(t, client, definition, 2, "pi-3", base.Add(2*time.Minute))

	r := newEnvelopeTestRouter(t, client)

	data := decodeEnvelope(t, r, "/api/v1/bpmn/monitoring/instances/status?page=1&pageSize=20", "1")
	assertListEnvelope(t, data, 2, 1, 20, 1, 2)

	// 数组元素字段必须与后端 DTO 一致；历史前端类型凭猜测声明了 instanceId /
	// processDefinitionKey / endTime / currentActivities 等后端从未返回的字段。
	first, ok := data["items"].([]interface{})[0].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, first, "processInstanceId")
	assert.Contains(t, first, "riskLevel")
}

// E4-48：原 TestMonitoringAuditLogs_UsesListEnvelope 打的是重复表面
// /bpmn/monitoring/audit-logs，该路由已删除；同用例信封契约合并进
// TestDashboardAuditLogs_UsesListEnvelope，重复表面消失由
// TestAuditLogs_MonitoringDuplicateSurfaceGone 锁住。
