// AST 扫描器夹具：只被 route_scan_scanner_fixture_test.go 解析，不参与编译
// （testdata 目录被 go 工具忽略）。
//
// 每条注册形态对应扫描器的一条受支持约定，改动 route_scan.go 时必须先在这里加/改形态。
package fixture

import (
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

// FixtureHandler 占位。
type FixtureHandler struct{}

func (h *FixtureHandler) List(c *gin.Context)   {}
func (h *FixtureHandler) Create(c *gin.Context) {}
func (h *FixtureHandler) Delete(c *gin.Context) {}

// 形态 1：以 *gin.RouterGroup 形参为组根（router/ticket_type_routes.go 的真实形态）。
// 2026-10-04 P0 修复前，这三条声明解析不出前缀、在 /api/v1 过滤处被静默丢弃。
func SetupFixtureParamGroupRoutes(tenant *gin.RouterGroup, h *FixtureHandler) {
	tenant.GET("/fixture-things", middleware.RequirePermission("fixture", "read"), h.List)
	tenant.POST("/fixture-things", middleware.RequirePermission("fixture", "write"), h.Create)
	tenant.DELETE("/fixture-things/:id", middleware.RequirePermission("fixture", "delete"), h.Delete)
}

// 形态 2：`x := y.Use(...)` 别名继承父组前缀（router.go 里 tenant := auth.Use(...)）。
func SetupFixtureUseAliasRoutes(auth *gin.RouterGroup, h *FixtureHandler) {
	tenant := auth.Use(middleware.TenantIsolationMiddleware())
	tenant.GET("/fixture-alias", middleware.RequirePermission("fixture_alias", "read"), h.List)
}

// 形态 3：直接收 *gin.Engine 的注册函数用绝对路径段派生分组，根前缀为空。
func SetupFixtureEngineRoutes(r *gin.Engine, h *FixtureHandler) {
	api := r.Group("/api/v1/fixture-absolute")
	api.GET("/items", middleware.RequirePermission("fixture_absolute", "read"), h.List)
}

// 形态 4：组变量既不是形参、也没有 `:= x.Group(...)` 可追——扫描器不许猜，
// 必须进 unresolved 让生成层硬报错。
func SetupFixtureUnresolvedRoutes(mystery *customRouter, h *FixtureHandler) {
	mystery.GET("/fixture-mystery", middleware.RequirePermission("fixture_mystery", "read"), h.List)
}

type customRouter struct{}

func (c *customRouter) GET(path string, mw ...gin.HandlerFunc) {}
