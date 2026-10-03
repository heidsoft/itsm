package bpmn

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent/enttest"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// /bpmn/monitoring/* 的 startTime/endTime 按 time.RFC3339 解析。
// 历史实现写成 `if parsed, err := ...; err == nil`：无法解析的过滤条件被静默丢弃，
// 请求照常返回 200 和"全量"数据，调用方无法区分"没过滤"和"过滤条件被我忽略了"。

func doBPMNGet(t *testing.T, r *gin.Engine, path string, tenant string) (int, int, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("X-Test-Tenant", tenant)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var body struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body), w.Body.String())
	return w.Code, body.Code, body.Message
}

func TestMonitoringTimeFilter_RejectsDateOnly(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_time_filter_reject?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	r := newEnvelopeTestRouter(t, client)

	cases := []struct {
		path      string
		wantParam string
	}{
		{"/api/v1/bpmn/monitoring/metrics?startTime=2026-01-01&endTime=2026-01-31", "startTime"},
		{"/api/v1/bpmn/monitoring/metrics/some-process?startTime=2026-01-01", "startTime"},
		{"/api/v1/bpmn/monitoring/instances/status?endTime=2026-01-31", "endTime"},
		{"/api/v1/bpmn/monitoring/audit-logs?startTime=2026-01-01", "startTime"},
	}
	for _, tc := range cases {
		status, code, msg := doBPMNGet(t, r, tc.path, "1")
		assert.Equal(t, http.StatusBadRequest, status, tc.path)
		assert.Equal(t, common.ParamErrorCode, code, tc.path)
		assert.Contains(t, msg, tc.wantParam, "错误消息要指名被拒绝的参数,实际 %q", msg)
	}
}

// TestMonitoringTimeFilter_AppliesValidBounds 证明合法 RFC3339 边界真的进了查询：
// 三条审计日志分布在两个时间点，只取后一半必须得到 1 条而不是 3 条。
func TestMonitoringTimeFilter_AppliesValidBounds(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_time_filter_apply?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	base := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	seedAuditLog(t, client, 1, 11, "tf-a", base)
	seedAuditLog(t, client, 1, 12, "tf-b", base.Add(time.Hour))
	seedAuditLog(t, client, 1, 13, "tf-c", base.Add(2*time.Hour))

	r := newEnvelopeTestRouter(t, client)

	all := decodeEnvelope(t, r, "/api/v1/bpmn/monitoring/audit-logs?page=1&pageSize=20", "1")
	assertListEnvelope(t, all, 3, 1, 20, 1, 3)

	// 边界值本身要包含在结果里，所以用 11:00 作为下界应命中后两条。
	filtered := decodeEnvelope(t, r,
		"/api/v1/bpmn/monitoring/audit-logs?page=1&pageSize=20&startTime=2026-01-01T11%3A00%3A00Z&endTime=2026-01-01T13%3A00%3A00Z",
		"1")
	assertListEnvelope(t, filtered, 2, 1, 20, 1, 2)
}

// TestDashboardTimeFilter_KeepsDateOnlyContract 锁住另一套契约：
// /bpmn/dashboard/* 用 "2006-01-02" 解析日期，/workflow/dashboard、/workflow/sla、
// /workflow/audit 三个页面正是按日期发送的。统一成 RFC3339 会让这些页面直接 400。
func TestDashboardTimeFilter_KeepsDateOnlyContract(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:bpmn_time_filter_dashboard?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	base := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	seedAuditLog(t, client, 1, 11, "db-a", base)
	seedAuditLog(t, client, 1, 12, "db-b", base.AddDate(0, 0, 10))

	r := newEnvelopeTestRouter(t, client)

	status, code, msg := doBPMNGet(t, r, "/api/v1/bpmn/dashboard/audit-logs?startTime=2026-01-10&endTime=2026-01-31", "1")
	require.Equal(t, http.StatusOK, status, msg)
	require.Equal(t, 0, code, msg)
}
