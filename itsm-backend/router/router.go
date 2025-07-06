package router

import (
	"itsm-backend/controller"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupRoutes 设置路由
func SetupRoutes(r *gin.Engine, ticketController *controller.TicketController) {
	// API路由组
	api := r.Group("/api")
	{
		// 工单相关路由（需要认证）
		tickets := api.Group("/tickets")
		tickets.Use(middleware.AuthMiddleware())
		{
			tickets.POST("", ticketController.CreateTicket)           // POST /api/tickets
			tickets.GET("/:id", ticketController.GetTicket)           // GET /api/tickets/:id
			tickets.PATCH("/:id", ticketController.UpdateTicket)      // PATCH /api/tickets/:id
			tickets.POST("/:id/approve", ticketController.ApproveTicket) // POST /api/tickets/:id/approve
			tickets.POST("/:id/comment", ticketController.AddComment)   // POST /api/tickets/:id/comment
		}
	}
}