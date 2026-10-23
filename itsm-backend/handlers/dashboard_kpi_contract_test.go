package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDashboardViewerFromContext KPI 的读取者身份只能来自认证上下文。
// 缺失 user_id 时 UserID 必须是 0（非管理角色会被收窄为空集），
// 不允许回退成"拿不到身份就按租户全量统计"。
func TestDashboardViewerFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("上下文提供身份", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", 42)
		c.Set("role", "agent")
		viewer := dashboardViewer(c)
		assert.Equal(t, 42, viewer.UserID)
		assert.Equal(t, "agent", viewer.Role)
		assert.False(t, viewer.ScopeAll)
	})

	t.Run("缺失 user_id 不得放大范围", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("role", "agent")
		viewer := dashboardViewer(c)
		assert.Zero(t, viewer.UserID)
		assert.Equal(t, "agent", viewer.Role)
	})
}

// TestConvertKPIMetricsCarriesContract 环比可信标记与统计范围必须原样出到响应 DTO，
// 否则前端无法区分"没有增长"和"没有可比基线"。
func TestConvertKPIMetricsCarriesContract(t *testing.T) {
	converted := convertKPIMetrics([]service.KPIMetricData{{
		ID:         "total-tickets",
		Value:      5,
		Change:     0,
		ChangeType: "stable",
		Comparable: false,
		Scope:      "own",
	}})
	require.Len(t, converted, 1)
	assert.False(t, converted[0].Comparable)
	assert.Equal(t, "own", converted[0].Scope)

	body, err := json.Marshal(converted[0])
	require.NoError(t, err)
	var fields map[string]any
	require.NoError(t, json.Unmarshal(body, &fields))
	assert.Contains(t, fields, "comparable")
	assert.Equal(t, "own", fields["scope"])
}

// TestParseQualityWindow 统一服务质量视图的窗口解析：默认 30、非法回退、上限钳制。
func TestParseQualityWindow(t *testing.T) {
	assert.Equal(t, 30, parseQualityWindow(""))
	assert.Equal(t, 30, parseQualityWindow("30"))
	assert.Equal(t, 30, parseQualityWindow("not-a-number"), "非法值回退默认")
	assert.Equal(t, 7, parseQualityWindow("7"))
	assert.Equal(t, service.MaxQualityWindowDays, parseQualityWindow("3650"), "超上限钳制")
}

// TestGetServiceQualityUnwiredIs503 聚合器未装配时显式 503，不得返回空报表假成功。
func TestGetServiceQualityUnwiredIs503(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewDashboardHandler(nil, nil, nil, zap.NewNop().Sugar())
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/reports/service-quality", nil)
	c.Set(middleware.TenantContextKey, &middleware.TenantContext{TenantID: 1})
	h.GetServiceQuality(c)
	assert.Equal(t, 503, w.Code, "未装配必须 503，body=%s", w.Body.String())
}
