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
		}

		// ------------------------------ CI属性定义相关路由 ------------------------------
		attributes := cmdb.Group("/attributes")
		attributes.Use(middleware.RequirePermission("cmdb_ci_attribute", "read"))
		{
			// 按CI类型获取属性定义
			ciTypes.GET("/:ciTypeId/attributes", cmdbController.ListCIAttributeDefinitions)
			attributes.GET("/:id", cmdbController.GetCIAttributeDefinition)
			
			// 写入权限
			attributes.POST("", middleware.RequirePermission("cmdb_ci_attribute", "write"), cmdbController.CreateCIAttributeDefinition)
			attributes.PUT("/:id", middleware.RequirePermission("cmdb_ci_attribute", "write"), cmdbController.UpdateCIAttributeDefinition)
			attributes.DELETE("/:id", middleware.RequirePermission("cmdb_ci_attribute", "delete"), cmdbController.DeleteCIAttributeDefinition)
		}

		// ------------------------------ 配置项相关路由 ------------------------------
		cis := cmdb.Group("/cis")
		cis.Use(middleware.RequirePermission("cmdb_ci", "read"))
		{
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
			cis.GET("/:ciId/relationships", cmdbController.ListCIRelationshipsByCIID)
			
			// 影响分析
			cis.GET("/:ciId/impact-analysis", cmdbController.GetCIImpactAnalysis)
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
