package router

import (
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupCMDBRoutes 设置CMDB相关路由
//
// 规范前缀：/api/v1/cmdb/*（前端已全部切换）。
// /api/v1/configuration-items/* 为兼容别名，仅保留给尚未升级的旧客户端，
// 回归稳定后评估摘除（见 CHANGELOG）。新增端点只允许注册到 /cmdb 下。
func SetupCMDBRoutes(
	auth *gin.RouterGroup,
	config *RouterConfig,
) {
	// 兼容别名：/configuration-items（弃用，勿新增端点）
	configurationItems := auth.Group("/configuration-items")
	configurationItems.Use(middleware.RequirePermission("cmdb_ci", "read"))
	{
		configurationItems.POST("/search", middleware.RequirePermission("cmdb_ci", "read"), config.CMDBHandler.SearchCI)
		configurationItems.GET("/stats", middleware.RequirePermission("cmdb_ci", "read"), config.CMDBHandler.GetCIStats)
		configurationItems.GET("/types", middleware.RequirePermission("cmdb_ci_type", "read"), config.CMDBHandler.ListCITypes)
		configurationItems.POST("/types", middleware.RequirePermission("cmdb_ci_type", "write"), config.CMDBHandler.CreateCIType)
		configurationItems.GET("/types/:id", middleware.RequirePermission("cmdb_ci_type", "read"), config.CMDBHandler.GetCIType)
		configurationItems.PUT("/types/:id", middleware.RequirePermission("cmdb_ci_type", "write"), config.CMDBHandler.UpdateCIType)
		configurationItems.DELETE("/types/:id", middleware.RequirePermission("cmdb_ci_type", "delete"), config.CMDBHandler.DeleteCIType)
		configurationItems.GET("/relationships", middleware.RequirePermission("cmdb_relationship", "read"), config.CMDBHandler.ListCIRelationships)
		configurationItems.POST("/relationships", middleware.RequirePermission("cmdb_relationship", "write"), config.CMDBHandler.CreateCIRelationship)
		configurationItems.GET("/relationships/:id", middleware.RequirePermission("cmdb_relationship", "read"), config.CMDBHandler.GetCIRelationship)
		configurationItems.PUT("/relationships/:id", middleware.RequirePermission("cmdb_relationship", "write"), config.CMDBHandler.UpdateCIRelationship)
		configurationItems.DELETE("/relationships/:id", middleware.RequirePermission("cmdb_relationship", "delete"), config.CMDBHandler.DeleteCIRelationship)
		configurationItems.GET("/relationship-types", middleware.RequirePermission("cmdb_relationship", "read"), config.CMDBHandler.ListRelationshipTypes)
		configurationItems.GET("", middleware.RequirePermission("cmdb_ci", "read"), config.CMDBHandler.ListCIs)
		configurationItems.POST("", middleware.RequirePermission("cmdb_ci", "write"), config.CMDBHandler.CreateCI)
		configurationItems.GET("/:id", middleware.RequirePermission("cmdb_ci", "read"), config.CMDBHandler.GetCI)
		configurationItems.PUT("/:id", middleware.RequirePermission("cmdb_ci", "write"), config.CMDBHandler.UpdateCI)
		configurationItems.DELETE("/:id", middleware.RequirePermission("cmdb_ci", "delete"), config.CMDBHandler.DeleteCI)
		configurationItems.GET("/:id/relationships", middleware.RequirePermission("cmdb_ci", "read"), config.CMDBHandler.ListCIRelationshipsByCIID)
		configurationItems.GET("/:id/topology", middleware.RequirePermission("cmdb_ci", "read"), config.CMDBHandler.GetCITopology)
		configurationItems.GET("/:id/impact-analysis", middleware.RequirePermission("cmdb_ci", "read"), config.CMDBHandler.GetCIImpactAnalysis)
		configurationItems.GET("/:id/change-history", middleware.RequirePermission("cmdb_ci", "read"), config.CMDBHandler.GetCIHistory)
		configurationItems.GET("/:id/history", middleware.RequirePermission("cmdb_ci", "read"), config.CMDBHandler.GetCIHistory)
	}

	// CMDB管理路由（规范前缀）
	cmdb := auth.Group("/cmdb")
	{
		cmdb.GET("/capabilities", middleware.RequirePermission("cmdb", "read"), config.CMDBHandler.GetCapabilities)
		cmdb.GET("/discovery/health", middleware.RequirePermission("cmdb", "read"), config.CMDBHandler.GetDiscoveryHealth)

		// AI-Native 本体自描述端点：CI 类型 × 属性定义 × 关系词表 × 枚举值域 × AI 工具
		cmdb.GET("/ontology", middleware.RequirePermission("cmdb_ci", "read"), config.CMDBHandler.GetOntology)

		// 关系类型元数据
		cmdb.GET("/relationship-types", middleware.RequirePermission("cmdb_relationship", "read"), config.CMDBHandler.ListRelationshipTypes)

		// ------------------------------ CI类型相关路由 ------------------------------
		ciTypes := cmdb.Group("/ci-types")
		ciTypes.Use(middleware.RequirePermission("cmdb_ci_type", "read"))
		{
			ciTypes.GET("", config.CMDBHandler.ListCITypes)
			ciTypes.GET("/:id", config.CMDBHandler.GetCIType)

			// 写入权限
			ciTypes.POST("", middleware.RequirePermission("cmdb_ci_type", "write"), config.CMDBHandler.CreateCIType)
			ciTypes.PUT("/:id", middleware.RequirePermission("cmdb_ci_type", "write"), config.CMDBHandler.UpdateCIType)
			ciTypes.DELETE("/:id", middleware.RequirePermission("cmdb_ci_type", "delete"), config.CMDBHandler.DeleteCIType)

			// 属性定义
			ciTypes.GET("/:id/attributes", config.CMDBHandler.ListCIAttributeDefinitions)
		}

		// ------------------------------ CI属性定义相关路由 ------------------------------
		attributes := cmdb.Group("/attributes")
		attributes.Use(middleware.RequirePermission("cmdb_ci_attribute", "read"))
		{
			attributes.GET("/:id", config.CMDBHandler.GetCIAttributeDefinition)

			// 写入权限
			attributes.POST("", middleware.RequirePermission("cmdb_ci_attribute", "write"), config.CMDBHandler.CreateCIAttributeDefinition)
			attributes.PUT("/:id", middleware.RequirePermission("cmdb_ci_attribute", "write"), config.CMDBHandler.UpdateCIAttributeDefinition)
			attributes.DELETE("/:id", middleware.RequirePermission("cmdb_ci_attribute", "delete"), config.CMDBHandler.DeleteCIAttributeDefinition)
		}

		// ------------------------------ CI标签相关路由 ------------------------------
		tags := cmdb.Group("/tags")
		tags.Use(middleware.RequirePermission("cmdb_tag", "read"))
		{
			tags.GET("", config.CMDBHandler.ListCITags)
			tags.GET("/:id", config.CMDBHandler.GetCITag)

			// 写入权限
			tags.POST("", middleware.RequirePermission("cmdb_tag", "write"), config.CMDBHandler.CreateCITag)
			tags.PUT("/:id", middleware.RequirePermission("cmdb_tag", "write"), config.CMDBHandler.UpdateCITag)
			tags.DELETE("/:id", middleware.RequirePermission("cmdb_tag", "delete"), config.CMDBHandler.DeleteCITag)
		}

		// ------------------------------ 保存视图相关路由 ------------------------------
		views := cmdb.Group("/views")
		views.Use(middleware.RequirePermission("cmdb_view", "read"))
		{
			views.GET("", config.CMDBHandler.ListSavedViews)
			views.GET("/:id", config.CMDBHandler.GetSavedView)

			// 写入权限
			views.POST("", middleware.RequirePermission("cmdb_view", "write"), config.CMDBHandler.CreateSavedView)
			views.PUT("/:id", middleware.RequirePermission("cmdb_view", "write"), config.CMDBHandler.UpdateSavedView)
			views.DELETE("/:id", middleware.RequirePermission("cmdb_view", "delete"), config.CMDBHandler.DeleteSavedView)
		}

		// ------------------------------ 导入导出相关路由 ------------------------------
		importRoute := cmdb.Group("/import")
		importRoute.Use(middleware.RequirePermission("cmdb_import_export", "read"))
		{
			importRoute.GET("", config.CMDBHandler.ListImportTasks)
			importRoute.GET("/:task_id", config.CMDBHandler.GetImportTaskStatus)

			// 写入权限
			importRoute.POST("", middleware.RequirePermission("cmdb_import_export", "write"), config.CMDBHandler.CreateImportTask)
		}

		exportRoute := cmdb.Group("/export")
		exportRoute.Use(middleware.RequirePermission("cmdb_import_export", "read"))
		{
			exportRoute.GET("", config.CMDBHandler.ListExportTasks)
			exportRoute.GET("/:task_id", config.CMDBHandler.GetExportTaskStatus)

			// 写入权限
			exportRoute.POST("", middleware.RequirePermission("cmdb_import_export", "write"), config.CMDBHandler.CreateExportTask)
		}

		// ------------------------------ 配置项相关路由 ------------------------------
		cis := cmdb.Group("/cis")
		cis.Use(middleware.RequirePermission("cmdb_ci", "read"))
		{
			// 高级搜索
			cis.POST("/search", config.CMDBHandler.SearchCI)

			// 批量操作
			batch := cis.Group("/batch")
			{
				batch.POST("", middleware.RequirePermission("cmdb_ci", "write"), config.CMDBHandler.BatchCreateCI)
				batch.PUT("", middleware.RequirePermission("cmdb_ci", "write"), config.CMDBHandler.BatchUpdateCI)
				batch.DELETE("", middleware.RequirePermission("cmdb_ci", "delete"), config.CMDBHandler.BatchDeleteCI)
				batch.PUT("/lifecycle", middleware.RequirePermission("cmdb_ci", "write"), config.CMDBHandler.BatchUpdateLifecycleStatus)
			}

			// 统计接口必须放在 /:id 之前
			cis.GET("/stats", middleware.RequirePermission("cmdb_ci", "read"), config.CMDBHandler.GetCIStats)
			// 兼容旧路径：/cmdb/cis/relationships -> /cmdb/relationships，必须放在 /:id 之前
			cis.POST("/relationships", middleware.RequirePermission("cmdb_relationship", "write"), config.CMDBHandler.CreateCIRelationship)
			cis.GET("/relationships", middleware.RequirePermission("cmdb_relationship", "read"), config.CMDBHandler.ListCIRelationships)

			// 基础CRUD
			cis.GET("", middleware.RequirePermission("cmdb_ci", "read"), config.CMDBHandler.ListCIs)
			cis.GET("/:id", middleware.RequirePermission("cmdb_ci", "read"), config.CMDBHandler.GetCI)

			// 写入权限
			cis.POST("", middleware.RequirePermission("cmdb_ci", "write"), config.CMDBHandler.CreateCI)
			cis.PUT("/:id", middleware.RequirePermission("cmdb_ci", "write"), config.CMDBHandler.UpdateCI)
			cis.DELETE("/:id", middleware.RequirePermission("cmdb_ci", "delete"), config.CMDBHandler.DeleteCI)

			// 关系查询
			cis.GET("/:id/relationships", middleware.RequirePermission("cmdb_ci", "read"), config.CMDBHandler.ListCIRelationshipsByCIID)
			cis.GET("/:id/topology", middleware.RequirePermission("cmdb_ci", "read"), config.CMDBHandler.GetCITopology)

			// 影响分析
			cis.GET("/:id/impact-analysis", middleware.RequirePermission("cmdb_ci", "read"), config.CMDBHandler.GetCIImpactAnalysis)

			// 变更历史
			cis.GET("/:id/history", middleware.RequirePermission("cmdb_ci", "read"), config.CMDBHandler.GetCIHistory)

			// 版本回滚
			cis.POST("/:id/revert", middleware.RequirePermission("cmdb_ci", "write"), config.CMDBHandler.RevertCIVersion)

			// 生命周期管理
			cis.GET("/:id/lifecycle/history", config.CMDBHandler.GetLifecycleHistory)
			cis.PUT("/:id/lifecycle", middleware.RequirePermission("cmdb_ci", "write"), config.CMDBHandler.UpdateLifecycleStatus)

			// 标签管理
			cis.POST("/:id/tags", middleware.RequirePermission("cmdb_ci", "write"), config.CMDBHandler.AddTagsToCI)
			cis.DELETE("/:id/tags", middleware.RequirePermission("cmdb_ci", "write"), config.CMDBHandler.RemoveTagsFromCI)
		}

		// ------------------------------ CI关系相关路由 ------------------------------
		relationships := cmdb.Group("/relationships")
		relationships.Use(middleware.RequirePermission("cmdb_relationship", "read"))
		{
			relationships.GET("", config.CMDBHandler.ListCIRelationships)
			relationships.GET("/:id", config.CMDBHandler.GetCIRelationship)

			// 写入权限
			relationships.POST("", middleware.RequirePermission("cmdb_relationship", "write"), config.CMDBHandler.CreateCIRelationship)
			relationships.PUT("/:id", middleware.RequirePermission("cmdb_relationship", "write"), config.CMDBHandler.UpdateCIRelationship)
			relationships.DELETE("/:id", middleware.RequirePermission("cmdb_relationship", "delete"), config.CMDBHandler.DeleteCIRelationship)
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

		// ------------------------------ 发现相关路由 ------------------------------
		discovery := cmdb.Group("/discovery")
		discovery.Use(middleware.RequirePermission("cmdb", "read"))
		{
			discovery.GET("/sources", config.CMDBHandler.ListDiscoverySources)
			discovery.POST("/sources", middleware.RequirePermission("cmdb", "write"), config.CMDBHandler.CreateDiscoverySource)
			discovery.POST("/jobs", middleware.RequirePermission("cmdb", "write"), config.CMDBHandler.CreateDiscoveryJob)
			discovery.GET("/results", middleware.RequirePermission("cmdb", "read"), config.CMDBHandler.ListDiscoveryResults)
		}

		// ------------------------------ 对账相关路由 ------------------------------
		cmdb.GET("/reconciliation", middleware.RequirePermission("cmdb", "read"), config.CMDBHandler.GetReconciliation)
	}
}
