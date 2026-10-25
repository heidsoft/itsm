package sla

// 回归：SLA 合规报表的日期过滤曾经只接受 RFC3339，而同语义的
// change.calendar 接受纯日期（2026-10-01）；并且同时接受 snake_case 别名
// （start_date/end_date），形成永久双轨契约。
//
// 现在：纯日期与 RFC3339 都接受（endDate 按当日 23:59:59.999 闭区间），
// 只认 camelCase 参数名；snake 别名被忽略后表现为「缺少必填参数」而不是
// 静默换一套口径。

import (
	"net/http"
	"testing"

	"itsm-backend/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComplianceReport_AcceptsPlainDateRange(t *testing.T) {
	r, _, _ := setupSLAHandler(t)

	resp := doSLAReq(t, r, http.MethodGet,
		"/api/v1/sla/compliance-report?startDate=2026-10-01&endDate=2026-10-31", nil, false)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok, "body=%s", slaStr(resp))
	period, ok := data["reportPeriod"].(map[string]interface{})
	require.True(t, ok, "reportPeriod 缺失: %s", slaStr(resp))
	assert.Contains(t, period, "startDate")
	assert.Contains(t, period, "endDate")
	// 纯日期表达的结束日期按闭区间处理，不能被截成当天 00:00:00。
	assert.Contains(t, period["endDate"], "2026-10-31")
}

func TestComplianceReport_AcceptsRFC3339Range(t *testing.T) {
	r, _, _ := setupSLAHandler(t)

	resp := doSLAReq(t, r, http.MethodGet,
		"/api/v1/sla/compliance-report?startDate=2026-10-01T08:00:00Z&endDate=2026-10-31T20:00:00Z", nil, false)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
}

func TestComplianceReport_IgnoresSnakeCaseAlias(t *testing.T) {
	r, _, _ := setupSLAHandler(t)

	status, resp := doSLAReqStatus(t, r, http.MethodGet,
		"/api/v1/sla/compliance-report?start_date=2026-10-01&end_date=2026-10-31")
	require.Equal(t, http.StatusBadRequest, status,
		"snake_case 别名不得生效，必须按缺少必填参数拒绝；body=%s", slaStr(resp))
	assert.Equal(t, common.ParamErrorCode, resp.Code)
}

func TestComplianceReport_RejectsInvalidDate(t *testing.T) {
	r, _, _ := setupSLAHandler(t)

	status, resp := doSLAReqStatus(t, r, http.MethodGet,
		"/api/v1/sla/compliance-report?startDate=2026-13-01&endDate=2026-10-31")
	require.Equal(t, http.StatusBadRequest, status, "body=%s", slaStr(resp))
	assert.Equal(t, common.ParamErrorCode, resp.Code)
}
