package router

import (
	"github.com/gin-gonic/gin"

	"itsm-backend/handlers/cloud"
	"itsm-backend/middleware"
)

// SetupCloudRoutes 注册云资源管理域路由（云账号/云服务/云资源）。
// 从 router.go 集中注册块抽取，行为与原内联等价。
func SetupCloudRoutes(tenant *gin.RouterGroup, h *cloud.Handler) {
	cloudGrp := tenant.Group("/cloud")
	{
		// Cloud Accounts (云账号)
		cloudAccounts := cloudGrp.Group("/accounts")
		{
			cloudAccounts.GET("", middleware.RequirePermission("cmdb", "read"), h.ListCloudAccounts)
			cloudAccounts.POST("", middleware.RequirePermission("cmdb", "write"), h.CreateCloudAccount)
			cloudAccounts.GET("/:id", middleware.RequirePermission("cmdb", "read"), h.GetCloudAccount)
			cloudAccounts.PUT("/:id", middleware.RequirePermission("cmdb", "write"), h.UpdateCloudAccount)
			cloudAccounts.DELETE("/:id", middleware.RequirePermission("cmdb", "delete"), h.DeleteCloudAccount)
		}

		// Cloud Services (云服务)
		cloudServices := cloudGrp.Group("/services")
		{
			cloudServices.GET("", middleware.RequirePermission("cmdb", "read"), h.ListCloudServices)
			cloudServices.POST("", middleware.RequirePermission("cmdb", "write"), h.CreateCloudService)
			cloudServices.GET("/:id", middleware.RequirePermission("cmdb", "read"), h.GetCloudService)
			cloudServices.PUT("/:id", middleware.RequirePermission("cmdb", "write"), h.UpdateCloudService)
			cloudServices.DELETE("/:id", middleware.RequirePermission("cmdb", "delete"), h.DeleteCloudService)
		}

		// Cloud Resources (云资源)
		cloudResources := cloudGrp.Group("/resources")
		{
			cloudResources.GET("", middleware.RequirePermission("cmdb", "read"), h.ListCloudResources)
			cloudResources.POST("", middleware.RequirePermission("cmdb", "write"), h.CreateCloudResource)
			cloudResources.GET("/:id", middleware.RequirePermission("cmdb", "read"), h.GetCloudResource)
			cloudResources.PUT("/:id", middleware.RequirePermission("cmdb", "write"), h.UpdateCloudResource)
			cloudResources.DELETE("/:id", middleware.RequirePermission("cmdb", "delete"), h.DeleteCloudResource)
		}
	}
}
