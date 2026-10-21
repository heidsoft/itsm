package router

import (
	"net/http"
	"testing"

	"itsm-backend/handlers/on_call"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// 值班排班此前只在 /email-intake/on-call 下暴露，且缺少排班表改名/停用/删除。
// 本测试锁住规范入口的表面：一旦路由被删改，前端与邮件受理的兜底派单会静默失效。
func TestSetupOnCallRoutes_RegisteredShape(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	tenant := r.Group("/api/v1")
	// 只验证注册表面，不触发 handler，service 依赖保持为 nil。
	SetupOnCallRoutes(tenant, on_call.NewHandler(nil, zap.NewNop().Sugar()))

	registered := make([]string, 0, len(r.Routes()))
	for _, route := range r.Routes() {
		registered = append(registered, route.Method+" "+route.Path)
	}
	require.ElementsMatch(t, []string{
		http.MethodGet + " /api/v1/on-call/schedules",
		http.MethodPost + " /api/v1/on-call/schedules",
		http.MethodGet + " /api/v1/on-call/schedules/:id",
		http.MethodPut + " /api/v1/on-call/schedules/:id",
		http.MethodDelete + " /api/v1/on-call/schedules/:id",
		http.MethodGet + " /api/v1/on-call/shifts",
		http.MethodPost + " /api/v1/on-call/shifts",
		http.MethodPut + " /api/v1/on-call/shifts/:id",
		http.MethodDelete + " /api/v1/on-call/shifts/:id",
		http.MethodGet + " /api/v1/on-call/current",
	}, registered, "值班域规范入口必须完整，且前缀与前端 onCallBaseUrl 一致")
}
