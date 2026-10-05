package bpmn

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 台账 E4-48：BPMN 审计读取历史上同时存在
// /bpmn/dashboard/audit-logs（直接序列化 Ent 模型：snake_case 键 + 泄漏 tenant_id）
// 与 /bpmn/monitoring/audit-logs（把行降维成弱 AuditLogEntry 形状）两套只读表面。
// 现在唯一所有者是 BPMNAuditService.QueryAuditLogs → dashboard 单入口，
// 挂 middleware.RequirePermission("bpmn","read")，响应统一为 camelCase DTO
// dto.ProcessAuditLogResponse（不含 tenantId）。

// doAuditGet 发一次 GET 并解析统一响应壳，返回 HTTP status + 业务 code + data。
// deny 分支的 data 为 nil，所以业务 code 必须从外层壳取。
func doAuditGet(t *testing.T, r http.Handler, path string, tenant string) (int, int, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("X-Test-Tenant", tenant)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var body struct {
		Code int         `json:"code"`
		Data interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body), w.Body.String())
	data, _ := body.Data.(map[string]interface{})
	return w.Code, body.Code, data
}

// TestAuditLogs_MonitoringDuplicateSurfaceGone 锁重复表面已消失：
// /bpmn/monitoring/audit-logs 必须 404（Gin 未注册路由），而不是任何降级 200。
func TestAuditLogs_MonitoringDuplicateSurfaceGone(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_audit_48_gone?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	seedAuditLog(t, client, 1, 11, "gone-a", time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC))
	r := newEnvelopeTestRouter(t, client)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bpmn/monitoring/audit-logs?page=1&pageSize=20", nil)
	req.Header.Set("X-Test-Tenant", "1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code, "monitoring/audit-logs 重复表面必须已从 Router 消失")
}

// TestDashboardAuditLogs_CamelCaseDTOContract 锁响应形状：items 行必须恰好是
// dto.ProcessAuditLogResponse 的 21 个 camelCase 键；snake_case 旧键与 tenantId
// 一律不得出现（旧实现直接序列化 Ent 模型时两者都在）。
func TestDashboardAuditLogs_CamelCaseDTOContract(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_audit_48_dto?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	seedAuditLog(t, client, 1, 11, "dto-a", time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC))
	r := newEnvelopeTestRouter(t, client)

	data := decodeEnvelope(t, r, "/api/v1/bpmn/dashboard/audit-logs?page=1&pageSize=20", "1")
	items, ok := data["items"].([]interface{})
	require.True(t, ok)
	require.Len(t, items, 1)

	row, ok := items[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, []string{
		"action", "activityId", "activityName", "activityType",
		"assigneeId", "assigneeName", "comment", "durationMs",
		"id", "ipAddress", "metadata", "processDefinitionId",
		"processDefinitionKey", "processInstanceId", "processInstanceKey",
		"timestamp", "userAgent", "userId", "userName",
		"variablesAfter", "variablesBefore",
	}, sortedKeys(row), "响应键集合必须恰好是 camelCase DTO，不得混入 snake_case 或 tenantId")

	assert.Equal(t, "dto-a", row["processInstanceKey"])
	assert.NotContains(t, row, "tenantId")
	assert.NotContains(t, row, "tenant_id")
}

// TestDashboardAuditLogs_PermissionGate 锁权限门三态：
// admin（兜底词表持 bpmn:read）200；msp_viewer（不持）403/2003；
// 未注入认证上下文 fail-closed 401/2001。
func TestDashboardAuditLogs_PermissionGate(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_audit_48_gate?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	seedAuditLog(t, client, 1, 11, "gate-a", time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC))
	path := "/api/v1/bpmn/dashboard/audit-logs?page=1&pageSize=20"

	status, _, _ := doAuditGet(t, newEnvelopeTestRouterWithRole(t, client, "admin"), path, "1")
	assert.Equal(t, http.StatusOK, status, "admin 持 bpmn:read 应放行")

	status, code, _ := doAuditGet(t, newEnvelopeTestRouterWithRole(t, client, "msp_viewer"), path, "1")
	assert.Equal(t, http.StatusForbidden, status)
	assert.Equal(t, common.ForbiddenCode, code, "msp_viewer 无 bpmn:read 必须 403/2003")

	status, code, _ = doAuditGet(t, newEnvelopeTestRouterWithRole(t, client, ""), path, "1")
	assert.Equal(t, http.StatusUnauthorized, status)
	assert.Equal(t, common.AuthFailedCode, code, "缺 role/client 上下文必须 fail-closed 401/2001")
}

// TestUserActivityRoute_PermissionGate 覆盖同家族的 user/:userId 表面：
// 同样挂 bpmn:read 门，deny 侧不得泄漏任何行。
func TestUserActivityRoute_PermissionGate(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_audit_48_user_gate?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	seedAuditLog(t, client, 1, 11, "usergate-a", time.Now().UTC())
	path := "/api/v1/bpmn/dashboard/audit-logs/user/1"

	status, _, _ := doAuditGet(t, newEnvelopeTestRouterWithRole(t, client, "admin"), path, "1")
	assert.Equal(t, http.StatusOK, status)

	status, _, _ = doAuditGet(t, newEnvelopeTestRouterWithRole(t, client, "msp_viewer"), path, "1")
	assert.Equal(t, http.StatusForbidden, status)
}
