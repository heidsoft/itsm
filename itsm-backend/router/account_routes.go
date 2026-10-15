package router

import (
	accountHandler "itsm-backend/handlers/account"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupAccountRoutes 注册登录用户对自身账号的自助操作路由。
//
// 挂在 r 层 `/api/v1/account` 而不是 tenant 分组，原因与 /auth/me 相同：
// tenant 分组的 RBAC 中间件会拦端用户，而"改自己的密码"必须对所有角色开放。
//
// 权限边界：只挂 AuthMiddleware，不挂 RequirePermission。
// 改自己的密码不涉及任何业务资源授权（user:write 是管理面语义，
// 用在这里会让端用户无法改自己的密码）；handler 内部 user_id 只取鉴权上下文，
// 不接受请求体/路径参数指定目标用户，从根上排除越权重置他人密码的可能。
func SetupAccountRoutes(r *gin.Engine, jwtSecret string, h *accountHandler.Handler) {
	if h == nil {
		return
	}
	grp := r.Group("/api/v1/account")
	{
		grp.PUT("/change-password", middleware.AuthMiddleware(jwtSecret), h.ChangePassword)
	}
}
