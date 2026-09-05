package router

import (
	"github.com/gin-gonic/gin"

	ticketCategoryHandler "itsm-backend/handlers/ticket_category"
	"itsm-backend/middleware"
)

// SetupTicketCategoryRoutes 注册工单分类域路由。
// 从 router.go 集中注册块抽取，行为与原内联等价。
func SetupTicketCategoryRoutes(tenant *gin.RouterGroup, h *ticketCategoryHandler.Handler) {
	categories := tenant.Group("/ticket-categories")
	{
		categories.GET("", middleware.RequirePermission("ticket_category", "read"), h.ListCategories)
		categories.POST("", middleware.RequirePermission("ticket_category", "create"), h.CreateCategory)
		categories.GET("/tree", middleware.RequirePermission("ticket_category", "read"), h.GetCategoryTree)
		categories.POST("/import/preview", middleware.RequirePermission("ticket_category", "create"), h.PreviewImport)
		categories.POST("/import", middleware.RequirePermission("ticket_category", "create"), h.ExecuteImport)
		categories.GET("/:id", middleware.RequirePermission("ticket_category", "read"), h.GetCategory)
		categories.PUT("/:id", middleware.RequirePermission("ticket_category", "update"), h.UpdateCategory)
		categories.PUT("/:id/move", middleware.RequirePermission("ticket_category", "update"), h.MoveCategory)
		categories.DELETE("/:id", middleware.RequirePermission("ticket_category", "delete"), h.DeleteCategory)
	}
}
