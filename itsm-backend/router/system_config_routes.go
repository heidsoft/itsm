package router

import (
	"runtime"
	"time"

	"github.com/gin-gonic/gin"

	systemConfigHandler "itsm-backend/handlers/systemconfig"
	tenantHandler "itsm-backend/handlers/tenant"
	vectorStoreHandler "itsm-backend/handlers/vector_store"
	"itsm-backend/middleware"
)

// SetupSystemConfigRoutes 注册系统配置域路由。
// 从 router.go 的集中注册块抽取而来，路由路径/方法/中间件与抽取前逐行一致。
// 含三条路径族：/system-configs（新）、/system 与 /configs（兼容别名）、
// 以及向量存储（RAG）自注册委托（/system/vector-store）。
func SetupSystemConfigRoutes(tenant *gin.RouterGroup, h *systemConfigHandler.Handler, tenantHandler *tenantHandler.Handler, vectorStoreHandler *vectorStoreHandler.Handler, appStartTime time.Time) {
	sysConfigs := tenant.Group("/system-configs")
	{
		// 配置管理
		sysConfigs.GET("", middleware.RequirePermission("config", "read"), h.ListConfigs)
		sysConfigs.GET("/init", middleware.RequirePermission("config", "read"), h.InitDefaultConfigs)
		sysConfigs.GET("/:id", middleware.RequirePermission("config", "read"), h.GetConfig)
		sysConfigs.GET("/key/:key", middleware.RequirePermission("config", "read"), h.GetConfigByKey)
		sysConfigs.PUT("/:id", middleware.RequirePermission("config", "update"), h.UpdateConfig)
		sysConfigs.PUT("/batch", middleware.RequirePermission("config", "update"), h.BatchUpdateConfigs)

		// 系统状态
		sysConfigs.GET("/status", middleware.RequirePermission("config", "read"), func(c *gin.Context) {
			systemStatusResponse(c, appStartTime)
		})
	}

	// P1-01 别名：/system/config → 系统状态（前端默认 fetch 路径）
	sysRoot := tenant.Group("/system")
	{
		// 兼容旧路径：/configs → /system-configs
		configs := tenant.Group("/configs")
		{
			configs.GET("", middleware.RequirePermission("config", "read"), h.ListConfigs)
			configs.GET("/init", middleware.RequirePermission("config", "read"), h.InitDefaultConfigs)
			configs.GET("/:id", middleware.RequirePermission("config", "read"), h.GetConfig)
			configs.GET("/key/:key", middleware.RequirePermission("config", "read"), h.GetConfigByKey)
			configs.PUT("/:id", middleware.RequirePermission("config", "update"), h.UpdateConfig)
			configs.PUT("/batch", middleware.RequirePermission("config", "update"), h.BatchUpdateConfigs)
			configs.GET("/status", middleware.RequirePermission("config", "read"), func(c *gin.Context) {
				systemStatusResponse(c, appStartTime)
			})
		}

		sysRoot.GET("/config", middleware.RequirePermission("config", "read"), func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status":    "ok",
				"version":   "1.6.8",
				"timestamp": time.Now(),
			})
		})

		// Tenant settings (current tenant — uses auth context)
		sysRoot.GET("/settings", middleware.RequirePermission("tenant", "read"), tenantHandler.GetTenantSettings)
		sysRoot.PUT("/settings", middleware.RequirePermission("tenant", "update"), tenantHandler.UpdateTenantSettings)
	}

	// 向量存储（RAG）状态与连通性诊断：/api/v1/system/vector-store
	if vectorStoreHandler != nil {
		vectorStoreHandler.RegisterRoutes(tenant)
	}
}

// systemStatusResponse 输出运行时状态（mem/goroutine/uptime），供 /system-configs/status
// 与 /configs/status 共用。抽取自 router.go 内联闭包，响应体与原实现逐字段一致。
func systemStatusResponse(c *gin.Context, appStartTime time.Time) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	var uptime string
	if !appStartTime.IsZero() {
		uptime = time.Since(appStartTime).Truncate(time.Second).String()
	}

	c.JSON(200, gin.H{
		"cpu": gin.H{
			"usage": 0,
			"cores": runtime.NumCPU(),
		},
		"memory": gin.H{
			"used":  m.Alloc / 1024 / 1024,
			"total": m.Sys / 1024 / 1024,
			"usage": float64(m.Alloc) / float64(m.Sys) * 100,
		},
		"goroutines": runtime.NumGoroutine(),
		"startTime":  appStartTime,
		"uptime":     uptime,
		"timestamp":  time.Now(),
	})
}
