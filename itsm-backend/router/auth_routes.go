package router

import (
	"itsm-backend/controller"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupAuthRoutes 设置认证相关路由
func SetupAuthRoutes(r *gin.RouterGroup, authController *controller.AuthController, jwtSecret string) {
	// 公共路由（无需认证）
	public := r.Group("/auth")
	{
		public.POST("/login", authController.Login)
		public.POST("/refresh-token", authController.RefreshToken)
	}

	// 需要认证的路由
	protected := r.Group("/auth").Use(middleware.AuthMiddleware(jwtSecret))
	{
		protected.GET("/user-info", authController.GetUserInfo)
		protected.GET("/tenants", authController.GetUserTenants)
		protected.POST("/switch-tenant", authController.SwitchTenant)
	}
}