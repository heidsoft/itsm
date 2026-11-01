package router

import (
	"github.com/gin-gonic/gin"

	"itsm-backend/handlers/cloudsignal"
	"itsm-backend/middleware"
)

// SetupCloudSignalRoutes 注册云信号路由（云监控告警与安全漏洞）。
// 读面与写面都要求租户上下文，权限沿用 cmdb 资源：云信号和 CMDB 共用同一批
// 云账号，单独开一套 permission 词表只会让运维配两遍。
func SetupCloudSignalRoutes(tenant *gin.RouterGroup, h *cloudsignal.Handler) {
	group := tenant.Group("/cloud-signals")
	{
		group.GET("/alerts", middleware.RequirePermission("cmdb", "read"), h.ListAlerts)
		group.POST("/alerts/:id/acknowledge", middleware.RequirePermission("cmdb", "write"), h.AcknowledgeAlert)
		group.POST("/alerts/:id/resolve", middleware.RequirePermission("cmdb", "write"), h.ResolveAlert)

		group.GET("/vulnerabilities", middleware.RequirePermission("cmdb", "read"), h.ListFindings)
		group.GET("/vulnerabilities/summary", middleware.RequirePermission("cmdb", "read"), h.FindingSummary)
		group.POST("/vulnerabilities/:id/ignore", middleware.RequirePermission("cmdb", "write"), h.IgnoreFinding)
		group.POST("/vulnerabilities/:id/reopen", middleware.RequirePermission("cmdb", "write"), h.ReopenFinding)

		group.GET("/sync-status", middleware.RequirePermission("cmdb", "read"), h.SyncStatus)
	}
}
