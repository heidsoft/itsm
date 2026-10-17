package sla

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"itsm-backend/common"
	"itsm-backend/dto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// slaTemplate shaped config 与 service/sla_template_service.go 写入模板目录的键名一致。
func slaTemplateCreateBody(name string) dto.CreateSLADefinitionRequest {
	return dto.CreateSLADefinitionRequest{
		Name:           name,
		ServiceType:    "incident",
		Priority:       "critical",
		ResponseTime:   15,
		ResolutionTime: 240,
		IsActive:       true,
		BusinessHours: map[string]interface{}{
			"time_zone":  "Asia/Shanghai",
			"work_days":  []int{1, 2, 3, 4, 5},
			"start_time": "00:00",
			"end_time":   "23:59",
			"is_24_7":    true,
		},
		EscalationRules: map[string]interface{}{
			"levels": []map[string]interface{}{
				{"level": 1, "threshold_minutes": 15, "notify_roles": []string{"manager"}},
				{"level": 2, "threshold_minutes": 30, "notify_roles": []string{"it_admin"}},
			},
		},
	}
}

// doSLAReqRawBody 返回原始响应体，用于断言响应 JSON 里没有 snake_case 配置键。
func doSLAReqRawBody(t *testing.T, r http.Handler, method, path string, body interface{}) string {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, path, reader)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	return w.Body.String()
}

// TestSLAHandler_DefinitionConfigProjectionRoundTrip 走真实 handler：写入 snake_case 配置后，
// 详情与列表都必须返回 camelCase 投影，且不再透出原始 escalationRules map。
func TestSLAHandler_DefinitionConfigProjectionRoundTrip(t *testing.T) {
	r, _, _ := setupSLAHandler(t)

	raw := doSLAReqRawBody(t, r, "POST", "/api/v1/sla/definitions", slaTemplateCreateBody("事件 P1 紧急"))
	for _, leaked := range []string{"threshold_minutes", "notify_roles", "work_days", "time_zone", "is_24_7", "start_time", "end_time"} {
		assert.NotContains(t, raw, leaked, "响应不得泄漏存储层 snake_case 键: %s", raw)
	}

	var resp common.Response
	require.NoError(t, json.Unmarshal([]byte(raw), &resp))
	require.Equal(t, common.SuccessCode, resp.Code)
	data := resp.Data.(map[string]interface{})

	bh, ok := data["businessHours"].(map[string]interface{})
	require.True(t, ok, "businessHours 必须是对象，got=%T", data["businessHours"])
	assert.Equal(t, "Asia/Shanghai", bh["timeZone"])
	assert.Equal(t, true, bh["alwaysOn"])
	assert.Equal(t, "00:00", bh["startTime"])
	assert.Equal(t, "23:59", bh["endTime"])
	assert.Len(t, bh["workDays"], 5)

	levels, ok := data["escalationLevels"].([]interface{})
	require.True(t, ok, "escalationLevels 必须是数组，got=%T", data["escalationLevels"])
	require.Len(t, levels, 2)
	first := levels[0].(map[string]interface{})
	assert.Equal(t, float64(1), first["level"])
	assert.Equal(t, float64(15), first["thresholdMinutes"])
	assert.Equal(t, []interface{}{"manager"}, first["notifyRoles"])

	assert.NotContains(t, data, "escalationRules", "原始 escalationRules map 已从响应契约移除")

	// 列表路径共用同一个投影函数，不能只在详情里正确。
	id := int(data["id"].(float64))
	listResp := doSLAReq(t, r, "GET", "/api/v1/sla/definitions", nil, false)
	require.Equal(t, common.SuccessCode, listResp.Code, "body=%s", slaStr(listResp))
	items := listResp.Data.(map[string]interface{})["items"].([]interface{})
	require.Len(t, items, 1)
	listed := items[0].(map[string]interface{})
	assert.Equal(t, float64(id), listed["id"])
	assert.Equal(t, true, listed["businessHours"].(map[string]interface{})["alwaysOn"])
	assert.Len(t, listed["escalationLevels"].([]interface{}), 2)
}

// TestSLAHandler_DefinitionWithoutBusinessHours 空配置必须缺省而不是返回零值对象，
// 否则前端无法区分「未配置」与「配置成非 7×24」。
func TestSLAHandler_DefinitionWithoutBusinessHours(t *testing.T) {
	r, _, _ := setupSLAHandler(t)

	resp := doSLAReq(t, r, "POST", "/api/v1/sla/definitions", dto.CreateSLADefinitionRequest{
		Name:           "无窗口 SLA",
		Priority:       "low",
		ResponseTime:   60,
		ResolutionTime: 480,
		IsActive:       true,
	}, false)
	require.Equal(t, common.SuccessCode, resp.Code, "body=%s", slaStr(resp))
	data := resp.Data.(map[string]interface{})

	assert.NotContains(t, data, "businessHours")
	assert.NotContains(t, data, "escalationLevels")
	assert.Contains(t, data, "conditions")
	assert.Equal(t, true, data["isActive"])
}
