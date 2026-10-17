package router

import (
	"time"

	"github.com/gin-gonic/gin"

	"itsm-backend/handlers"
	systemConfigHandler "itsm-backend/handlers/systemconfig"
	tenantHandler "itsm-backend/handlers/tenant"
	vectorStoreHandler "itsm-backend/handlers/vector_store"
	"itsm-backend/middleware"
)

// SetupSystemConfigRoutes 注册系统配置域路由。
// 含三条路径族：/system-configs（新）、/system 与 /configs（兼容别名）、
// 以及向量存储（RAG）自注册委托（/system/vector-store）。
//
// dashboard 为 nil 时不注册状态端点：运行时指标只有一个采集口径，
// 复制一份假数据或第二套采集实现都属于回归。
func SetupSystemConfigRoutes(tenant *gin.RouterGroup, h *systemConfigHandler.Handler, tenantHandler *tenantHandler.Handler, vectorStoreHandler *vectorStoreHandler.Handler, dashboard *handlers.DashboardHandler) {
	sysConfigs := tenant.Group("/system-configs")
	{
		// 配置管理。默认配置由 ListConfigs 在总数为 0 时懒初始化，
		// 不提供 GET /init：该入口按读权限守卫却整表写入安全策略，最终用户一次 GET 即可落地。
		sysConfigs.GET("", middleware.RequirePermission("system_config", "read"), h.ListConfigs)
		sysConfigs.GET("/:id", middleware.RequirePermission("system_config", "read"), h.GetConfig)
		sysConfigs.GET("/key/:key", middleware.RequirePermission("system_config", "read"), h.GetConfigByKey)
		sysConfigs.PUT("/:id", middleware.RequirePermission("system_config", "write"), h.UpdateConfig)
		sysConfigs.PUT("/batch", middleware.RequirePermission("system_config", "write"), h.BatchUpdateConfigs)

		// 系统状态
		if dashboard != nil {
			sysConfigs.GET("/status", middleware.RequirePermission("system_config", "read"), dashboard.GetSystemStats)
		}
	}

	// 别名：/system/config → 系统状态（前端默认 fetch 路径）
	sysRoot := tenant.Group("/system")
	{
		// 兼容旧路径：/configs → /system-configs
		configs := tenant.Group("/configs")
		{
			configs.GET("", middleware.RequirePermission("system_config", "read"), h.ListConfigs)
			configs.GET("/:id", middleware.RequirePermission("system_config", "read"), h.GetConfig)
			configs.GET("/key/:key", middleware.RequirePermission("system_config", "read"), h.GetConfigByKey)
			configs.PUT("/:id", middleware.RequirePermission("system_config", "write"), h.UpdateConfig)
			configs.PUT("/batch", middleware.RequirePermission("system_config", "write"), h.BatchUpdateConfigs)
			if dashboard != nil {
				configs.GET("/status", middleware.RequirePermission("system_config", "read"), dashboard.GetSystemStats)
			}
		}

		sysRoot.GET("/config", middleware.RequirePermission("system_config", "read"), func(c *gin.Context) {
			// 版本号走 appVersion() 单一事实源，不再各写一份字面量。
			c.JSON(200, gin.H{
				"status":    "ok",
				"version":   appVersion(),
				"timestamp": time.Now(),
			})
		})

		// Tenant settings (current tenant — uses auth context)
		sysRoot.GET("/settings", middleware.RequirePermission("tenant", "read"), tenantHandler.GetTenantSettings)
		sysRoot.PUT("/settings", middleware.RequirePermission("tenant", "write"), tenantHandler.UpdateTenantSettings)
	}

	// 向量存储（RAG）状态与连通性诊断：/api/v1/system/vector-store
	if vectorStoreHandler != nil {
		vectorStoreHandler.RegisterRoutes(tenant)
	}
}
