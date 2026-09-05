package router

import (
	"itsm-backend/handlers/service_catalog"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupServiceCatalogRoutes 注册服务目录相关路由。
// 从 router.go 的集中注册块抽取而来，路由路径/方法/中间件与抽取前逐行一致。
func SetupServiceCatalogRoutes(tenant *gin.RouterGroup, h *service_catalog.Handler) {
	tenant.GET("/service-catalog", middleware.RequirePermission("service_catalog", "read"), h.List)

	sc := tenant.Group("/service-catalogs")
	{
		sc.GET("", middleware.RequirePermission("service_catalog", "read"), h.List)
		sc.POST("", middleware.RequirePermission("service_catalog", "write"), h.Create)
		sc.GET("/search", middleware.RequirePermission("service_catalog", "read"), h.Search)
		sc.GET("/stats", middleware.RequirePermission("service_catalog", "read"), h.Stats)
		sc.GET("/:id", middleware.RequirePermission("service_catalog", "read"), h.Get)
		sc.PUT("/:id", middleware.RequirePermission("service_catalog", "write"), h.Update)
		sc.DELETE("/:id", middleware.RequirePermission("service_catalog", "delete"), h.Delete)
	}

	scServices := tenant.Group("/service-catalog-services")
	{
		scServices.GET("", middleware.RequirePermission("service_catalog", "read"), h.List)
		scServices.GET("/:id", middleware.RequirePermission("service_catalog", "read"), h.Get)
	}
}
