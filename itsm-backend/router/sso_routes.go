package router

import (
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupSSORoutes 注册 SSO（OIDC）路由。
//
// 三组安全边界：
//   - public（/api/v1/auth/sso/initiate|callback|providers）：会话建立前调用，
//     不挂认证；initiate/callback 挂登录限流防爆破。
//   - r 层 auth 自助组（/api/v1/auth/sso/bind|unbind|identities）：与
//     SetupCommonSystemRoutes 的 /auth/me、/auth/session、/auth/switch-tenant
//     同一模式——只挂 AuthMiddleware，不进租户级 RBAC 预检（SmartCheckPermission
//     的白名单/ACL/预检三轨都没有这些自助路径，挂上去只会 403）。
//     user_id 一律取鉴权中间件写入的上下文，不信任请求体。
//   - tenant（/api/v1/admin/sso/providers）：提供方配置面，RequirePermission 逐动作授权。
//
// config.AuthHandler 未注入 SSO 子系统时整体不注册（装配层降级路径）。
func SetupSSORoutes(r *gin.Engine, auth, public, tenant *gin.RouterGroup, config *RouterConfig) {
	if config.AuthHandler == nil || !config.AuthHandler.SSOEnabled() {
		return
	}

	// 登录发起与回调：公开访问，但按登录链路限流。
	public.POST("/auth/sso/initiate", middleware.LoginRateLimiter(), config.AuthHandler.SSOInitiate)
	public.POST("/auth/sso/callback", middleware.LoginRateLimiter(), config.AuthHandler.SSOCallback)
	public.GET("/auth/sso/providers", config.AuthHandler.SSOListEnabledProviders)

	// 账号自助：绑定 / 解绑 / 我的绑定（仅认证，路径与 /auth/me 同组模式）。
	selfService := r.Group("/api/v1/auth")
	{
		selfService.POST("/sso/bind", middleware.AuthMiddleware(config.JWTSecret), config.AuthHandler.SSOBind)
		selfService.POST("/sso/unbind", middleware.AuthMiddleware(config.JWTSecret), config.AuthHandler.SSOUnbind)
		selfService.GET("/sso/identities", middleware.AuthMiddleware(config.JWTSecret), config.AuthHandler.SSOListIdentities)
	}

	// 提供方管理：租户内配置面。
	admin := tenant.Group("/admin/sso/providers")
	{
		admin.GET("", middleware.RequirePermission("sso", "read"), config.AuthHandler.SSOAdminListProviders)
		admin.POST("", middleware.RequirePermission("sso", "write"), config.AuthHandler.SSOAdminCreateProvider)
		admin.GET("/:id", middleware.RequirePermission("sso", "read"), config.AuthHandler.SSOAdminGetProvider)
		admin.PUT("/:id", middleware.RequirePermission("sso", "write"), config.AuthHandler.SSOAdminUpdateProvider)
		admin.DELETE("/:id", middleware.RequirePermission("sso", "write"), config.AuthHandler.SSOAdminDeleteProvider)
	}
}
