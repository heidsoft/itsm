package router

import (
	"github.com/gin-gonic/gin"

	ticketTagHandler "itsm-backend/handlers/ticket_tag"
	"itsm-backend/middleware"
)

// SetupTicketTagRoutes 注册工单标签域路由。
// 从 router.go 集中注册块抽取，行为与原内联等价。
func SetupTicketTagRoutes(tenant *gin.RouterGroup, h *ticketTagHandler.Handler) {
	tags := tenant.Group("/ticket-tags")
	{
		tags.GET("", middleware.RequirePermission("ticket_tag", "read"), h.ListTags)
		tags.POST("", middleware.RequirePermission("ticket_tag", "create"), h.CreateTag)
		tags.GET("/:id", middleware.RequirePermission("ticket_tag", "read"), h.GetTag)
		tags.PUT("/:id", middleware.RequirePermission("ticket_tag", "update"), h.UpdateTag)
		tags.DELETE("/:id", middleware.RequirePermission("ticket_tag", "delete"), h.DeleteTag)
	}
}
