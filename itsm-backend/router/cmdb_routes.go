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
) {
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
			// 兼容旧路径：/cmdb/cis/relationships → /cmdb/relationships
			cis.POST("/relationships", middleware.RequirePermission("cmdb_relationship", "write"), cmdbController.CreateCIRelationship)
			cis.GET("/relationships", middleware.RequirePermission("cmdb_relationship", "read"), cmdbController.ListCIRelationships)


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
	}
}
