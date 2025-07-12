package router

import (
	"itsm-backend/controller"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupRoutes 设置路由
func SetupRouter(ticketController *controller.TicketController, serviceController *controller.ServiceController, dashboardController *controller.DashboardController, jwtSecret string) *gin.Engine {
	r := gin.Default()

	// CORS 中间件
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// 创建认证控制器
	authController := controller.NewAuthController(jwtSecret)

	// 公开路由（不需要认证）
	public := r.Group("/api")
	{
		public.POST("/login", authController.Login)
	}

	// 需要认证的API路由组
	api := r.Group("/api")
	api.Use(middleware.AuthMiddleware(jwtSecret))
	{
		// 仪表盘相关路由
		api.GET("/dashboard", dashboardController.GetDashboardData)
		api.GET("/dashboard/kpis", dashboardController.GetKPIMetrics)
		api.GET("/dashboard/resources/distribution", dashboardController.GetResourceDistribution)
		api.GET("/dashboard/resources/health", dashboardController.GetResourceHealth)

		// 工单相关路由
		api.GET("/tickets", ticketController.GetTickets)
		api.POST("/tickets", ticketController.CreateTicket)
		api.GET("/tickets/:id", ticketController.GetTicket)
		api.PATCH("/tickets/:id", ticketController.UpdateTicket)
		api.PUT("/tickets/:id/status", ticketController.UpdateTicketStatus)
		api.POST("/tickets/:id/approve", ticketController.ApproveTicket)
		// 添加评论接口
		api.POST("/tickets/:id/comment", ticketController.AddComment)

		// 服务目录相关路由
		api.GET("/service-catalogs", serviceController.GetServiceCatalogs)
		api.POST("/service-catalogs", serviceController.CreateServiceCatalog)
		api.GET("/service-catalogs/:id", serviceController.GetServiceCatalogByID)
		api.PUT("/service-catalogs/:id", serviceController.UpdateServiceCatalog)
		api.DELETE("/service-catalogs/:id", serviceController.DeleteServiceCatalog)

		// 服务请求相关路由
		api.POST("/service-requests", serviceController.CreateServiceRequest)
		api.GET("/service-requests/me", serviceController.GetUserServiceRequests)
		api.GET("/service-requests/:id", serviceController.GetServiceRequestByID)
		api.PUT("/service-requests/:id/status", serviceController.UpdateServiceRequestStatus)
	}

	return r
}
