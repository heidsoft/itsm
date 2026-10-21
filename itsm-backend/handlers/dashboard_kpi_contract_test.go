package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"itsm-backend/service"

	"github.com/gin-gonic/gin"
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
