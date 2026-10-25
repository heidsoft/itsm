package router

import (
	"github.com/gin-gonic/gin"

	ticketTypeHandler "itsm-backend/handlers/ticket_type"
	"itsm-backend/middleware"
)

// SetupTicketTypeRoutes 注册工单类型域路由（扁平注册于 tenant 根组）。
func SetupTicketTypeRoutes(tenant *gin.RouterGroup, h *ticketTypeHandler.Handler) {
	tenant.GET("/ticket-types", middleware.RequirePermission("ticket", "read"), h.ListTicketTypes)
	tenant.POST("/ticket-types", middleware.RequirePermission("ticket_type", "manage"), h.CreateTicketType)
	tenant.GET("/ticket-types/:id", middleware.RequirePermission("ticket", "read"), h.GetTicketType)
	tenant.PUT("/ticket-types/:id", middleware.RequirePermission("ticket_type", "manage"), h.UpdateTicketType)
	tenant.DELETE("/ticket-types/:id", middleware.RequirePermission("ticket_type", "archive"), h.DeleteTicketType)
	tenant.POST("/ticket-types/:id/enable", middleware.RequirePermission("ticket_type", "manage"), h.EnableTicketType)
	tenant.POST("/ticket-types/:id/disable", middleware.RequirePermission("ticket_type", "manage"), h.DisableTicketType)
	tenant.POST("/ticket-types/:id/clone", middleware.RequirePermission("ticket_type", "manage"), h.CloneTicketType)
	tenant.POST("/ticket-types/:id/restore", middleware.RequirePermission("ticket_type", "manage"), h.RestoreTicketType)
	// 预设库是产品内置模板，读取口径按 authz catalog 对 ticket_type:read 的定义
	// （「查看工单类型和预设」）；安装另按 install_preset 单独授权。
	// 之前挂 manage 时，只有 ticket_type:read 的坐席连预设列表都 403（本地 prod 栈实测 code 2003）。
	tenant.GET("/ticket-type-presets", middleware.RequirePermission("ticket_type", "read"), h.ListPresets)
	tenant.POST("/ticket-type-presets/:presetId/install", middleware.RequirePermission("ticket_type", "install_preset"), h.InstallPreset)
}
