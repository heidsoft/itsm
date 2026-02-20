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
		public.POST("/refresh", authController.RefreshToken) // 兼容前端 /api/v1/auth/refresh
		public.POST("/refresh-token", authController.RefreshToken)
		public.POST("/register", authController.Register)
		public.POST("/forgot-password", authController.ForgotPassword)
		public.POST("/reset-password", authController.ResetPassword)
		public.POST("/validate-reset-token", authController.ValidateResetToken)
	}

	// 需要认证的路由
	protected := r.Group("/auth").Use(middleware.AuthMiddleware(jwtSecret))
	{
		protected.GET("/user-info", authController.GetUserInfo)
		protected.GET("/profile", authController.GetUserInfo) // 兼容前端 /api/v1/auth/profile
		protected.GET("/tenants", authController.GetUserTenants)
		protected.POST("/switch-tenant", authController.SwitchTenant)
		protected.POST("/logout", authController.Logout)
	}
}