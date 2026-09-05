package router

import (
	"github.com/gin-gonic/gin"

	connectorHandler "itsm-backend/handlers/connector"
	"itsm-backend/middleware"
)

// SetupConnectorRoutes 注册连接器管理域路由（连接器/插件/技能市场）。
// 从 router.go 集中注册块抽取，行为与原内联等价。
func SetupConnectorRoutes(tenant *gin.RouterGroup, h *connectorHandler.Handler) {
	conns := tenant.Group("/connectors")
	{
		conns.GET("", middleware.RequirePermission("connector", "read"), h.ListMarket)
		conns.GET("/configs", middleware.RequirePermission("connector", "read"), h.ListConfigs)
		conns.GET("/lifecycle", middleware.RequirePermission("connector", "read"), h.Lifecycle)
		conns.POST("/configs", middleware.RequirePermission("connector", "write"), h.Provision)
		conns.DELETE("/configs/:name", middleware.RequirePermission("connector", "write"), h.Revoke)
		conns.POST("/:name/send", middleware.RequirePermission("connector", "write"), h.Send)
		conns.POST("/:name/test", middleware.RequirePermission("connector", "write"), h.Test)
		conns.GET("/health", middleware.RequirePermission("connector", "read"), h.Health)
	}
}
