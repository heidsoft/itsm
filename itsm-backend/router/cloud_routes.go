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
			cloudAccounts.GET("", middleware.RequirePermission("cloud_account", "read"), h.ListCloudAccounts)
			cloudAccounts.POST("", middleware.RequirePermission("cloud_account", "write"), h.CreateCloudAccount)
			cloudAccounts.GET("/:id", middleware.RequirePermission("cloud_account", "read"), h.GetCloudAccount)
			cloudAccounts.PUT("/:id", middleware.RequirePermission("cloud_account", "write"), h.UpdateCloudAccount)
			cloudAccounts.DELETE("/:id", middleware.RequirePermission("cloud_account", "delete"), h.DeleteCloudAccount)
		}

		// Cloud Services (云服务)
		cloudServices := cloudGrp.Group("/services")
		{
			cloudServices.GET("", middleware.RequirePermission("cloud_service", "read"), h.ListCloudServices)
			cloudServices.POST("", middleware.RequirePermission("cloud_service", "write"), h.CreateCloudService)
			cloudServices.GET("/:id", middleware.RequirePermission("cloud_service", "read"), h.GetCloudService)
			cloudServices.PUT("/:id", middleware.RequirePermission("cloud_service", "write"), h.UpdateCloudService)
			cloudServices.DELETE("/:id", middleware.RequirePermission("cloud_service", "delete"), h.DeleteCloudService)
		}

		// Cloud Resources (云资源)
		cloudResources := cloudGrp.Group("/resources")
		{
			cloudResources.GET("", middleware.RequirePermission("cloud_resource", "read"), h.ListCloudResources)
			cloudResources.POST("", middleware.RequirePermission("cloud_resource", "write"), h.CreateCloudResource)
			cloudResources.GET("/:id", middleware.RequirePermission("cloud_resource", "read"), h.GetCloudResource)
			cloudResources.PUT("/:id", middleware.RequirePermission("cloud_resource", "write"), h.UpdateCloudResource)
			cloudResources.DELETE("/:id", middleware.RequirePermission("cloud_resource", "delete"), h.DeleteCloudResource)
		}
	}
}
