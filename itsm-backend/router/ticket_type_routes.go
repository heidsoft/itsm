package router

import (
	"github.com/gin-gonic/gin"

	ticketTypeHandler "itsm-backend/handlers/ticket_type"
	"itsm-backend/middleware"
)

// SetupTicketTypeRoutes 注册工单类型域路由（扁平注册于 tenant 根组）。
// 从 router.go 集中注册块抽取，行为与原内联等价。
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
	tenant.GET("/ticket-type-presets", middleware.RequirePermission("ticket_type", "manage"), h.ListPresets)
	tenant.POST("/ticket-type-presets/:presetId/install", middleware.RequirePermission("ticket_type", "install_preset"), h.InstallPreset)
}
