package router

import (
	"net/http"
	"testing"

	mspHandler "itsm-backend/handlers/msp"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

// 回归（2026-10-02 边缘功能收口 E3-1）：前端 msp-api.ts 调用的
// GET /api/v1/msp/allocations/history 从未在 router/msp_routes.go 注册，
// 分配历史页签只能靠能力开关藏着。这里用真实注册函数锁定路由契约本身，
// 方法/完整前缀/静态段都必须与前端一致，否则同类漂移会再次静默发生。
func TestSetupMSPRoutes_AllocationHistoryIsRegistered(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	gin.SetMode(gin.TestMode)

	r := gin.New()
	// handler 的业务依赖在注册阶段不会被调用，这里只需非空 handler。
	SetupMSPRoutes(r, &RouterConfig{
		JWTSecret:  "msp-route-secret",
		Logger:     logger,
		MSPHandler: mspHandler.NewHandler(nil, nil, logger),
	})

	registered := make(map[string]bool)
	for _, route := range r.Routes() {
		registered[route.Method+" "+route.Path] = true
	}

	assert.True(t, registered[http.MethodGet+" /api/v1/msp/allocations/history"],
		"分配历史必须由真实 Router 注册，实际路由 %v", registered)
	// 同一族里已有的分配入口必须保持原路径，防止注册时被挪走或改名。
	assert.True(t, registered[http.MethodGet+" /api/v1/msp/allocations"])
	assert.True(t, registered[http.MethodPost+" /api/v1/msp/allocations"])
	assert.True(t, registered[http.MethodPost+" /api/v1/msp/allocations/deallocate"])
}
