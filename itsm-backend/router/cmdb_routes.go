package router

import (
	"itsm-backend/controller"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupCMDBRoutes 设置CMDB相关路由
func SetupCMDBRoutes(
	auth *gin.RouterGroup,
	cmdbController *controller.CMDBController,
	config *RouterConfig,
) {
	configurationItems := auth.Group("/configuration-items")
	configurationItems.Use(middleware.RequirePermission("cmdb_ci", "read"))
	{
		configurationItems.POST("/search", cmdbController.SearchCI)
		configurationItems.GET("/stats", cmdbController.GetCIStats)
		configurationItems.GET("/types", middleware.RequirePermission("cmdb_ci_type", "read"), cmdbController.ListCITypes)
		configurationItems.POST("/types", middleware.RequirePermission("cmdb_ci_type", "write"), cmdbController.CreateCIType)
		configurationItems.GET("/types/:id", middleware.RequirePermission("cmdb_ci_type", "read"), cmdbController.GetCIType)
		configurationItems.PUT("/types/:id", middleware.RequirePermission("cmdb_ci_type", "write"), cmdbController.UpdateCIType)
		configurationItems.DELETE("/types/:id", middleware.RequirePermission("cmdb_ci_type", "delete"), cmdbController.DeleteCIType)
		configurationItems.GET("/relationships", middleware.RequirePermission("cmdb_relationship", "read"), cmdbController.ListCIRelationships)
		configurationItems.POST("/relationships", middleware.RequirePermission("cmdb_relationship", "write"), cmdbController.CreateCIRelationship)
		configurationItems.GET("/relationship-types", middleware.RequirePermission("cmdb_relationship", "read"), cmdbController.ListRelationshipTypes)
		configurationItems.GET("", cmdbController.ListCIs)
		configurationItems.POST("", middleware.RequirePermission("cmdb_ci", "write"), cmdbController.CreateCI)
		configurationItems.GET("/:id", cmdbController.GetCI)
		configurationItems.PUT("/:id", middleware.RequirePermission("cmdb_ci", "write"), cmdbController.UpdateCI)
		configurationItems.DELETE("/:id", middleware.RequirePermission("cmdb_ci", "delete"), cmdbController.DeleteCI)
		configurationItems.GET("/:id/relationships", cmdbController.ListCIRelationshipsByCIID)
		configurationItems.GET("/:id/impact-analysis", cmdbController.GetCIImpactAnalysis)
		configurationItems.GET("/:id/change-history", cmdbController.GetCIHistory)
		configurationItems.GET("/:id/history", cmdbController.GetCIHistory)
	}

	// CMDB管理路由
	cmdb := auth.Group("/cmdb")
	{
		// ------------------------------ CI类型相关路由 ------------------------------
		ciTypes := cmdb.Group("/ci-types")
		ciTypes.Use(middleware.RequirePermission("cmdb_ci_type", "read"))
		{
			ciTypes.GET("", cmdbController.ListCITypes)
			ciTypes.GET("/:id", cmdbController.GetCIType)

			// 写入权限
			ciTypes.POST("", middleware.RequirePermission("cmdb_ci_type", "write"), cmdbController.CreateCIType)
			ciTypes.PUT("/:id", middleware.RequirePermission("cmdb_ci_type", "write"), cmdbController.UpdateCIType)
			ciTypes.DELETE("/:id", middleware.RequirePermission("cmdb_ci_type", "delete"), cmdbController.DeleteCIType)

			// 属性定义
			ciTypes.GET("/:id/attributes", cmdbController.ListCIAttributeDefinitions)
		}

		// ------------------------------ CI属性定义相关路由 ------------------------------
		attributes := cmdb.Group("/attributes")
		attributes.Use(middleware.RequirePermission("cmdb_ci_attribute", "read"))
		{
			attributes.GET("/:id", cmdbController.GetCIAttributeDefinition)

			// 写入权限
			attributes.POST("", middleware.RequirePermission("cmdb_ci_attribute", "write"), cmdbController.CreateCIAttributeDefinition)
			attributes.PUT("/:id", middleware.RequirePermission("cmdb_ci_attribute", "write"), cmdbController.UpdateCIAttributeDefinition)
			attributes.DELETE("/:id", middleware.RequirePermission("cmdb_ci_attribute", "delete"), cmdbController.DeleteCIAttributeDefinition)
		}

		// ------------------------------ CI标签相关路由 ------------------------------
		tags := cmdb.Group("/tags")
		tags.Use(middleware.RequirePermission("cmdb_tag", "read"))
		{
			tags.GET("", cmdbController.ListCITags)
			tags.GET("/:id", cmdbController.GetCITag)

			// 写入权限
			tags.POST("", middleware.RequirePermission("cmdb_tag", "write"), cmdbController.CreateCITag)
			tags.PUT("/:id", middleware.RequirePermission("cmdb_tag", "write"), cmdbController.UpdateCITag)
			tags.DELETE("/:id", middleware.RequirePermission("cmdb_tag", "delete"), cmdbController.DeleteCITag)
		}

		// ------------------------------ 保存视图相关路由 ------------------------------
		views := cmdb.Group("/views")
		views.Use(middleware.RequirePermission("cmdb_view", "read"))
		{
			views.GET("", cmdbController.ListSavedViews)
			views.GET("/:id", cmdbController.GetSavedView)

			// 写入权限
			views.POST("", middleware.RequirePermission("cmdb_view", "write"), cmdbController.CreateSavedView)
			views.PUT("/:id", middleware.RequirePermission("cmdb_view", "write"), cmdbController.UpdateSavedView)
			views.DELETE("/:id", middleware.RequirePermission("cmdb_view", "delete"), cmdbController.DeleteSavedView)
		}

		// ------------------------------ 导入导出相关路由 ------------------------------
		importRoute := cmdb.Group("/import")
		importRoute.Use(middleware.RequirePermission("cmdb_import_export", "read"))
		{
			importRoute.GET("", cmdbController.ListImportTasks)
			importRoute.GET("/:task_id", cmdbController.GetImportTaskStatus)

			// 写入权限
			importRoute.POST("", middleware.RequirePermission("cmdb_import_export", "write"), cmdbController.CreateImportTask)
		}

		exportRoute := cmdb.Group("/export")
		exportRoute.Use(middleware.RequirePermission("cmdb_import_export", "read"))
		{
			exportRoute.GET("", cmdbController.ListExportTasks)
			exportRoute.GET("/:task_id", cmdbController.GetExportTaskStatus)

			// 写入权限
			exportRoute.POST("", middleware.RequirePermission("cmdb_import_export", "write"), cmdbController.CreateExportTask)
		}

		// ------------------------------ 配置项相关路由 ------------------------------
		cis := cmdb.Group("/cis")
		cis.Use(middleware.RequirePermission("cmdb_ci", "read"))
		{
			// 高级搜索
			cis.POST("/search", cmdbController.SearchCI)

			// 批量操作
			batch := cis.Group("/batch")
			{
				batch.POST("", middleware.RequirePermission("cmdb_ci", "write"), cmdbController.BatchCreateCI)
				batch.PUT("", middleware.RequirePermission("cmdb_ci", "write"), cmdbController.BatchUpdateCI)
				batch.DELETE("", middleware.RequirePermission("cmdb_ci", "delete"), cmdbController.BatchDeleteCI)
				batch.PUT("/lifecycle", middleware.RequirePermission("cmdb_ci", "write"), cmdbController.BatchUpdateLifecycleStatus)
			}

			// 统计接口必须放在 /:id 之前
			cis.GET("/stats", cmdbController.GetCIStats)
			// 兼容旧路径：/cmdb/cis/relationships -> /cmdb/relationships，必须放在 /:id 之前
			cis.POST("/relationships", middleware.RequirePermission("cmdb_relationship", "write"), cmdbController.CreateCIRelationship)
			cis.GET("/relationships", middleware.RequirePermission("cmdb_relationship", "read"), cmdbController.ListCIRelationships)

			// 基础CRUD
			cis.GET("", cmdbController.ListCIs)
			cis.GET("/:id", cmdbController.GetCI)

			// 写入权限
			cis.POST("", middleware.RequirePermission("cmdb_ci", "write"), cmdbController.CreateCI)
			cis.PUT("/:id", middleware.RequirePermission("cmdb_ci", "write"), cmdbController.UpdateCI)
			cis.DELETE("/:id", middleware.RequirePermission("cmdb_ci", "delete"), cmdbController.DeleteCI)

			// 关系查询
			cis.GET("/:id/relationships", cmdbController.ListCIRelationshipsByCIID)

			// 影响分析
			cis.GET("/:id/impact-analysis", cmdbController.GetCIImpactAnalysis)

			// 变更历史
			cis.GET("/:id/history", cmdbController.GetCIHistory)

			// 版本回滚
			cis.POST("/:id/revert", middleware.RequirePermission("cmdb_ci", "write"), cmdbController.RevertCIVersion)

			// 生命周期管理
			cis.GET("/:id/lifecycle/history", cmdbController.GetLifecycleHistory)
			cis.PUT("/:id/lifecycle", middleware.RequirePermission("cmdb_ci", "write"), cmdbController.UpdateLifecycleStatus)

			// 标签管理
			cis.POST("/:id/tags", middleware.RequirePermission("cmdb_ci", "write"), cmdbController.AddTagsToCI)
			cis.DELETE("/:id/tags", middleware.RequirePermission("cmdb_ci", "write"), cmdbController.RemoveTagsFromCI)
		}

		// ------------------------------ CI关系相关路由 ------------------------------
		relationships := cmdb.Group("/relationships")
		relationships.Use(middleware.RequirePermission("cmdb_relationship", "read"))
		{
			relationships.GET("", cmdbController.ListCIRelationships)
			relationships.GET("/:id", cmdbController.GetCIRelationship)

			// 写入权限
			relationships.POST("", middleware.RequirePermission("cmdb_relationship", "write"), cmdbController.CreateCIRelationship)
			relationships.PUT("/:id", middleware.RequirePermission("cmdb_relationship", "write"), cmdbController.UpdateCIRelationship)
			relationships.DELETE("/:id", middleware.RequirePermission("cmdb_relationship", "delete"), cmdbController.DeleteCIRelationship)
		}

		// ------------------------------ 云账号相关路由 ------------------------------
		cloudAccounts := cmdb.Group("/cloud-accounts")
		cloudAccounts.Use(middleware.RequirePermission("cmdb_cloud_account", "read"))
		{
			cloudAccounts.GET("", config.CMDBHandler.ListCloudAccounts)
			cloudAccounts.GET("/:id", config.CMDBHandler.GetCloudAccount)

			// 写入权限
			cloudAccounts.POST("", middleware.RequirePermission("cmdb_cloud_account", "write"), config.CMDBHandler.CreateCloudAccount)
			cloudAccounts.PUT("/:id", middleware.RequirePermission("cmdb_cloud_account", "write"), config.CMDBHandler.UpdateCloudAccount)
			cloudAccounts.DELETE("/:id", middleware.RequirePermission("cmdb_cloud_account", "delete"), config.CMDBHandler.DeleteCloudAccount)
		}

		// ------------------------------ 云服务类型相关路由 ------------------------------
		cloudServices := cmdb.Group("/cloud-services")
		cloudServices.Use(middleware.RequirePermission("cmdb_cloud_service", "read"))
		{
			cloudServices.GET("", config.CMDBHandler.ListCloudServices)
			cloudServices.GET("/:id", config.CMDBHandler.GetCloudService)

			// 写入权限
			cloudServices.POST("", middleware.RequirePermission("cmdb_cloud_service", "write"), config.CMDBHandler.CreateCloudService)
			cloudServices.PUT("/:id", middleware.RequirePermission("cmdb_cloud_service", "write"), config.CMDBHandler.UpdateCloudService)
			cloudServices.DELETE("/:id", middleware.RequirePermission("cmdb_cloud_service", "delete"), config.CMDBHandler.DeleteCloudService)
		}

		// ------------------------------ 云资源相关路由 ------------------------------
		cloudResources := cmdb.Group("/cloud-resources")
		cloudResources.Use(middleware.RequirePermission("cmdb_cloud_resource", "read"))
		{
			cloudResources.GET("", config.CMDBHandler.ListCloudResources)
			cloudResources.GET("/:id", config.CMDBHandler.GetCloudResource)

			// 写入权限
			cloudResources.POST("", middleware.RequirePermission("cmdb_cloud_resource", "write"), config.CMDBHandler.CreateCloudResource)
			cloudResources.PUT("/:id", middleware.RequirePermission("cmdb_cloud_resource", "write"), config.CMDBHandler.UpdateCloudResource)
			cloudResources.DELETE("/:id", middleware.RequirePermission("cmdb_cloud_resource", "delete"), config.CMDBHandler.DeleteCloudResource)
		}

		// ------------------------------ 对账相关路由 ------------------------------
		cmdb.GET("/reconciliation", middleware.RequirePermission("cmdb", "read"), config.CMDBHandler.GetReconciliation)
	}
}
