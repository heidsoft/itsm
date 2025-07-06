package router

import (
	"itsm-backend/controller"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupRoutes 设置路由
func SetupRoutes(r *gin.Engine, ticketController *controller.TicketController) {
	// 配置CORS
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// API路由组
	api := r.Group("/api")
	{
		// 工单相关路由（需要认证）
		tickets := api.Group("/tickets")
		tickets.Use(middleware.AuthMiddleware())
		{
			tickets.GET("", ticketController.GetTickets)              // GET /api/tickets
			tickets.POST("", ticketController.CreateTicket)           // POST /api/tickets
			tickets.GET("/:id", ticketController.GetTicket)           // GET /api/tickets/:id
			tickets.PATCH("/:id", ticketController.UpdateTicket)      // PATCH /api/tickets/:id
			tickets.PUT("/:id/status", ticketController.UpdateTicketStatus) // PUT /api/tickets/:id/status
			tickets.POST("/:id/approve", ticketController.ApproveTicket) // POST /api/tickets/:id/approve
			tickets.POST("/:id/comment", ticketController.AddComment)   // POST /api/tickets/:id/comment
		}
	}
}