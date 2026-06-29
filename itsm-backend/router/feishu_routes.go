package router

import (
	"itsm-backend/controller"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupFeishuRoutes 设置飞书相关路由
func SetupFeishuRoutes(
	auth *gin.RouterGroup,
	public *gin.RouterGroup,
	feishuController *controller.FeishuController,
) {
	// 飞书OAuth路由（部分需要公开访问）
	feishu := auth.Group("/feishu")
	{
		// OAuth授权URL获取（需要登录）
		feishu.GET("/oauth/auth-url", middleware.RequirePermission("feishu", "use"), feishuController.GetOAuthAuthURL)
		// OAuth回调（公开访问，因为是从飞书跳转回来）
		public.GET("/feishu/oauth/callback", feishuController.OAuthCallback)
		
		// 工单同步路由
		feishu.POST("/sync/ticket/:ticket_id", middleware.RequirePermission("ticket", "update"), feishuController.SyncTicketToFeishu)
		
		// Webhook路由（公开访问，飞书调用）
		public.POST("/feishu/webhook", feishuController.Webhook)
	}
}
