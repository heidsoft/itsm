package router

import (
	"net/http"
	"testing"

	surveyHandler "itsm-backend/handlers/survey"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// 回归（2026-10-02 边缘功能收口 E3-3）：handlers/survey/handler.go 曾有一份从未被调用的
// RegisterRoutes，与 router.SetupSurveyRoutes 逐字重复 7 条注册。本域唯一的路由所有者是
// SetupSurveyRoutes（由 router.go 挂在 tenant 分组上），这里把它的表面契约锁住：
// 副本一旦复活并同时接线，gin 会因重复路由直接 panic，因此路由集合必须恰好等于这一份。
func TestSetupSurveyRoutes_RegisteredShape(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	tenant := r.Group("/api/v1")
	// 只验证注册表面，不触发 handler，因此 service 依赖保持为 nil。
	SetupSurveyRoutes(tenant, surveyHandler.NewHandler(nil, zap.NewNop().Sugar()))

	registered := make([]string, 0, len(r.Routes()))
	for _, route := range r.Routes() {
		registered = append(registered, route.Method+" "+route.Path)
	}
	require.ElementsMatch(t, []string{
		http.MethodGet + " /api/v1/surveys",
		http.MethodPost + " /api/v1/surveys",
		http.MethodGet + " /api/v1/surveys/:id",
		http.MethodPut + " /api/v1/surveys/:id",
		http.MethodGet + " /api/v1/surveys/:id/responses",
		http.MethodGet + " /api/v1/surveys/:id/analytics",
		http.MethodPost + " /api/v1/surveys/responses",
	}, registered, "问卷域只能有 SetupSurveyRoutes 这一份注册，且完整前缀必须与前端一致")
}
