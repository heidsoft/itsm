package bpmn

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 这一组走 handlers/bpmn 真实注册的监控路由，锁住 2026-11-06 在本地生产实测到、
// 只在 service 层测不出来的 wire 契约：
//   - /monitoring/health 当时下发 active_processes / active_tasks / error_count，前端一个字段都读不到；
//   - /monitoring/performance/alerts 当时是手拼 map（count 字段），与 /performance 内嵌的同名概念形状不同；
//   - /monitoring/instances/{id}/status 当时把业务键按数据库主键解析，任何真实实例都是 400/1001；
//   - 实例状态的 variables 当时外泄 BPMN 引擎内部控制变量。

type wireEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func doGetWire(t *testing.T, r *gin.Engine, path, tenant string) (int, wireEnvelope, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("X-Test-Tenant", tenant)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var env wireEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), w.Body.String())
	return w.Code, env, w.Body.String()
}

func wireSeedInstance(t *testing.T, client *ent.Client, definition *ent.ProcessDefinition,
	tenantID int, key, status string, variables map[string]interface{}, start time.Time,
) *ent.ProcessInstance {
	t.Helper()
	instance, err := client.ProcessInstance.Create().
		SetProcessInstanceID(key).
		SetProcessDefinitionKey(definition.Key).
		SetProcessDefinitionID(definition.ID).
		SetTenantID(tenantID).
		SetStatus(status).
		SetVariables(variables).
		SetStartTime(start).
		Save(context.Background())
	require.NoError(t, err)
	return instance
}

func wireSeedTask(t *testing.T, client *ent.Client, instance *ent.ProcessInstance, taskID, status, assignee string) {
	t.Helper()
	_, err := client.ProcessTask.Create().
		SetTaskID(taskID).
		SetTaskDefinitionKey("user_task").
		SetTaskName("审批").
		SetTaskType("user_task").
		SetProcessDefinitionKey(instance.ProcessDefinitionKey).
		SetProcessInstanceID(instance.ID).
		SetTenantID(instance.TenantID).
		SetStatus(status).
		SetAssignee(assignee).
		SetTaskVariables(map[string]interface{}{}).
		Save(context.Background())
	require.NoError(t, err)
}

func decodeDataMap(t *testing.T, env wireEnvelope, raw string) map[string]interface{} {
	t.Helper()
	var data map[string]interface{}
	require.NoError(t, json.Unmarshal(env.Data, &data), raw)
	return data
}

func TestMonitoringHealthWireContract_CamelCaseDTOAndEngineStatusCounts(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_health_wire?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	definition := seedProcessDefinition(t, client, 1)
	base := time.Now()

	running := wireSeedInstance(t, client, definition, 1, "PI-HW-RUNNING", "running", map[string]interface{}{}, base)
	wireSeedInstance(t, client, definition, 1, "PI-HW-SUSPENDED", "suspended", map[string]interface{}{}, base)
	wireSeedInstance(t, client, definition, 1, "PI-HW-TERMINATED", "terminated", map[string]interface{}{}, base)
	wireSeedInstance(t, client, definition, 1, "PI-HW-COMPLETED", "completed", map[string]interface{}{}, base)
	// 另一个租户同样有活跃流程与待办，健康数不得跨租户相加。
	otherDefinition := seedProcessDefinition(t, client, 2)
	otherRunning := wireSeedInstance(t, client, otherDefinition, 2, "PI-HW-T2", "running", map[string]interface{}{}, base)

	wireSeedTask(t, client, running, "HT-1", "created", "")
	wireSeedTask(t, client, running, "HT-2", "assigned", "alice")
	wireSeedTask(t, client, running, "HT-3", "completed", "alice")
	wireSeedTask(t, client, otherRunning, "HT-4", "created", "")

	r := newEnvelopeTestRouter(t, client)
	status, env, raw := doGetWire(t, r, "/api/v1/bpmn/monitoring/health", "1")
	require.Equal(t, http.StatusOK, status, raw)
	assert.Equal(t, 0, env.Code, env.Message)

	data := decodeDataMap(t, env, raw)
	// 引擎只写 created/assigned 作为未结束任务，只写 running/suspended 作为在途实例，
	// 只写 terminated 作为异常结束；历史上这三处用的是 assigned+in_progress、running、failed。
	assert.EqualValues(t, 2, data["activeProcesses"], "活跃流程应含 running+suspended，且只算本租户")
	assert.EqualValues(t, 2, data["activeTasks"], "待办应含 created+assigned，且只算本租户")
	assert.EqualValues(t, 1, data["errorCount"], "异常数应取 terminated")
	assert.Equal(t, "connected", data["database"])
	assert.Equal(t, "healthy", data["status"])

	// 该端点修复前返回的是 map[string]any 手拼的 snake_case，前端读不到任何计数。
	for _, leaked := range []string{"active_processes", "active_tasks", "error_count", "tenant_id"} {
		assert.NotContains(t, raw, leaked, "健康响应不得出现 snake_case 键 %s", leaked)
	}
}

func TestMonitoringHealthWireContract_NoDataIsNotDatabaseDown(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_health_empty_wire?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	r := newEnvelopeTestRouter(t, client)
	status, env, raw := doGetWire(t, r, "/api/v1/bpmn/monitoring/health", "1")
	require.Equal(t, http.StatusOK, status, raw)
	require.Equal(t, 0, env.Code, env.Message)

	// 修复前用 Limit(1).First 探库：空表返回 ent.NotFound，于是「还没有流程实例」
	// 被报成 database=disconnected/status=unhealthy。
	data := decodeDataMap(t, env, raw)
	assert.Equal(t, "connected", data["database"], "没有实例不等于数据库断开")
	assert.Equal(t, "healthy", data["status"])
	assert.EqualValues(t, 0, data["activeProcesses"])
	assert.EqualValues(t, 0, data["activeTasks"])
	assert.EqualValues(t, 0, data["errorCount"])
}

func TestMonitoringPerformanceAlertsWireContract_ArrayShape(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_alerts_wire?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	definition := seedProcessDefinition(t, client, 1)
	wireSeedInstance(t, client, definition, 1, "PI-AW-TERMINATED", "terminated", map[string]interface{}{}, time.Now())

	r := newEnvelopeTestRouter(t, client)
	status, env, raw := doGetWire(t, r, "/api/v1/bpmn/monitoring/performance/alerts", "1")
	require.Equal(t, http.StatusOK, status, raw)
	require.Equal(t, 0, env.Code, env.Message)

	// data 必须是数组：修复前这里返回 {count, alerts, ...} 对象，与 /performance 内嵌的
	// PerformanceAlert[] 是两套形状，前端只能二选一猜。
	var alerts []map[string]interface{}
	require.NoError(t, json.Unmarshal(env.Data, &alerts), "performance/alerts 的 data 应为数组，实际: %s", raw)
	require.NotEmpty(t, alerts, "本租户 1 小时内有 terminated 实例，应下发 error 告警")

	var errorAlert map[string]interface{}
	for _, a := range alerts {
		if a["type"] == "error" {
			errorAlert = a
			break
		}
	}
	require.NotNil(t, errorAlert, "告警类型应含 error，实际 %v", alerts)
	assert.Equal(t, "error", errorAlert["severity"])
	assert.Equal(t, "active", errorAlert["status"])
	assert.EqualValues(t, 1, errorAlert["value"])
	assert.Equal(t, "process", errorAlert["resourceType"])
	assert.NotContains(t, errorAlert, "count", "旧形状独有字段不得回归")
	assert.Contains(t, raw, "resourceId", "应使用统一 PerformanceAlert 字段集")
}

func TestMonitoringPerformanceAlertsWireContract_EmptyIsArrayNotNull(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_alerts_empty_wire?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	seedProcessDefinition(t, client, 1)
	r := newEnvelopeTestRouter(t, client)

	status, env, raw := doGetWire(t, r, "/api/v1/bpmn/monitoring/performance/alerts", "2")
	require.Equal(t, http.StatusOK, status, raw)
	require.Equal(t, 0, env.Code, env.Message)
	assert.JSONEq(t, "[]", string(env.Data), "无告警必须是 []，不能是 null：%s", raw)
}

func TestMonitoringInstanceStatusWireContract_AddressesBusinessKey(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_instance_status_wire?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	definition := seedProcessDefinition(t, client, 1)
	instance := wireSeedInstance(t, client, definition, 1, "PI-IW-LIVE", "running", map[string]interface{}{
		"ticketId":        "T-100",
		"_done_":          true,
		"_commandId":      "cmd-1",
		"_idempotencyKey": "idem-1",
	}, time.Now().Add(-time.Hour))
	wireSeedTask(t, client, instance, "IW-1", "assigned", "alice")

	r := newEnvelopeTestRouter(t, client)

	// 业务键寻址：修复前 handler 对路径参数 strconv.Atoi，任何 PI-* 键都是 400/1001。
	status, env, raw := doGetWire(t, r, "/api/v1/bpmn/monitoring/instances/PI-IW-LIVE/status", "1")
	require.Equal(t, http.StatusOK, status, raw)
	require.Equal(t, 0, env.Code, env.Message)

	data := decodeDataMap(t, env, raw)
	assert.Equal(t, "PI-IW-LIVE", data["processInstanceId"])
	assert.Equal(t, "running", data["status"])
	assert.Equal(t, "alice", data["assignee"])
	assert.Equal(t, "user_task", data["currentTask"])

	variables, ok := data["variables"].(map[string]interface{})
	require.True(t, ok, "variables 应为对象，实际 %T", data["variables"])
	assert.Equal(t, "T-100", variables["ticketId"], "业务变量必须保留")
	for _, control := range []string{"_done_", "_commandId", "_idempotencyKey"} {
		assert.NotContains(t, variables, control, "引擎内部控制变量 %s 不得下发", control)
		assert.NotContains(t, raw, control, "引擎内部控制变量 %s 不得出现在 wire 响应", control)
	}

	// 数据库主键不是该端点的寻址口径：按主键请求只能得到 not-found。
	// 跨租户与不存在的键同样 404/4004，文案指明实例键，不泄漏其它租户内容。
	negativePaths := []struct {
		name   string
		path   string
		tenant string
	}{
		{"db_primary_key", "/api/v1/bpmn/monitoring/instances/" + strconv.Itoa(instance.ID) + "/status", "1"},
		{"cross_tenant", "/api/v1/bpmn/monitoring/instances/PI-IW-LIVE/status", "2"},
		{"missing_key", "/api/v1/bpmn/monitoring/instances/PI-IW-NOT-EXIST/status", "1"},
	}
	for _, tc := range negativePaths {
		gotStatus, gotEnv, gotRaw := doGetWire(t, r, tc.path, tc.tenant)
		assert.Equal(t, http.StatusNotFound, gotStatus, "%s → %s", tc.name, gotRaw)
		assert.Equal(t, 4004, gotEnv.Code, "%s → %s", tc.name, gotRaw)
	}
}
