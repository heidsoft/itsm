package router

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"itsm-backend/handlers/cloudsignal"
)

// 云信号路由与 CMDB 共用 /api/v1 租户分组；本测试只验证注册不冲突、
// 路径与权限守卫扫描的一致口径（权限本身由 route_permission_guard_test 覆盖）。
func TestCloudSignalRoutesRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	SetupCloudSignalRoutes(r.Group("/api/v1"), &cloudsignal.Handler{})

	registered := make(map[string]bool, len(r.Routes()))
	for _, route := range r.Routes() {
		registered[route.Method+" "+route.Path] = true
	}

	want := []string{
		"GET /api/v1/cloud-signals/alerts",
		"POST /api/v1/cloud-signals/alerts/:id/acknowledge",
		"POST /api/v1/cloud-signals/alerts/:id/resolve",
		"GET /api/v1/cloud-signals/vulnerabilities",
		"GET /api/v1/cloud-signals/vulnerabilities/summary",
		"POST /api/v1/cloud-signals/vulnerabilities/:id/ignore",
		"POST /api/v1/cloud-signals/vulnerabilities/:id/reopen",
		"GET /api/v1/cloud-signals/sync-status",
	}
	for _, route := range want {
		assert.True(t, registered[route], "missing route %s", route)
	}
}
