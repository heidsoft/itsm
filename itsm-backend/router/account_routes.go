package router

import (
	"itsm-backend/ent"
	accountHandler "itsm-backend/handlers/account"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupAccountRoutes 注册登录用户对自身账号的自助操作路由。
//
// 挂在 r 层 `/api/v1/account` 而不是 tenant 分组，原因与 /auth/me 相同：
// tenant 分组的 RBAC 中间件会拦端用户，而"改自己的密码/资料"必须对所有角色开放。
//
// 权限边界：只挂 AuthMiddleware，不挂 RequirePermission。
// 自助面不涉及业务资源授权（user:write 是管理用户目录的语义，
// 用在这里会让端用户无法维护自己的账号信息）；handler 内部 user_id 只取鉴权上下文，
// 不接受请求体/路径参数指定目标用户，且 DTO 白名单排除 role/username/active，
// 从根上排除越权改他人资料与自我提权的可能。
func SetupAccountRoutes(r *gin.Engine, jwtSecret string, h *accountHandler.Handler, client *ent.Client) {
	if h == nil {
		return
	}
	grp := r.Group("/api/v1/account")
	// 自助改密/改资料是身份面写操作，必须留痕：改密成功后旧会话被吊销，
	// 事后要能回答「谁在什么时候从哪里改了密码」。account 分组不在 tenant
	// 分组内，tenant 层的审计中间件覆盖不到它。
	// 审计放在分组最前面：鉴权失败（401）也要落一条，用于识别撞库/盗用尝试。
	grp.Use(middleware.AuditMiddleware(client))
	{
		grp.PUT("/change-password", middleware.AuthMiddleware(jwtSecret), h.ChangePassword)
		grp.PUT("/profile", middleware.AuthMiddleware(jwtSecret), h.UpdateProfile)
	}
}
