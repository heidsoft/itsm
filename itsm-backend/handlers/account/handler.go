// Package account 提供登录用户对**自己账号**的自助操作端点。
//
// 与管理员用户管理面（handlers/users、PUT /users/:id/reset-password）的区别：
// 自助端点只作用于调用者本人，且必须校验旧密码；管理面由管理员重置他人密码，
// 不校验旧密码。两个权限面不可互相顶替。
package account

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

// Handler 自助账号操作 handler。
type Handler struct {
	userService *service.UserService
}

// NewHandler 创建自助账号 handler。
func NewHandler(userService *service.UserService) *Handler {
	return &Handler{userService: userService}
}

// ChangePassword 处理 PUT /api/v1/account/change-password。
//
// 权限边界：只挂 AuthMiddleware（与 /auth/me 同组），不进租户级 RBAC 预检 ——
// 这是"改自己的密码"，不涉及任何业务资源授权。user_id 一律取鉴权中间件
// 写入的上下文，**不信任请求体或路径参数**，否则就是越权重置他人密码的后门。
//
// 语义：必须校验旧密码（旧密码错 → 403），改密成功后吊销该用户全部存量
// access token（改密的常见动机正是"可能泄露了"）。
func (h *Handler) ChangePassword(c *gin.Context) {
	// 鉴权与参数校验必须**先于**依赖装配检查完成。
	// 顺序颠倒会让未认证请求拿到 503（泄露"服务未就绪"这一内部状态）、
	// 参数缺失也拿不到 400；两者都让排障时误判为部署问题。
	userID, ok := accountUserID(c)
	if !ok {
		return
	}

	var req struct {
		OldPassword string `json:"oldPassword" binding:"required"`
		NewPassword string `json:"newPassword" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误：oldPassword 与 newPassword 均为必填")
		return
	}

	if h.userService == nil {
		common.Fail(c, common.ServiceUnavailableCode, "账号服务未就绪")
		return
	}

	tenantID, _ := c.Get("tenant_id")
	tid, _ := tenantID.(int)

	if err := h.userService.ChangeOwnPassword(c.Request.Context(), userID, req.OldPassword, req.NewPassword, tid); err != nil {
		common.RespondError(c, err, "修改密码失败")
		return
	}

	common.Success(c, gin.H{
		"message":       "密码修改成功，请重新登录",
		"reLogin":       true,
		"tokensRevoked": true,
	})
}

// UpdateProfile 处理 PUT /api/v1/account/profile。
//
// 权限边界与 ChangePassword 同构：只挂 AuthMiddleware，user_id 只取鉴权上下文。
// 之所以不能用 PUT /users/:id（需 user:write）替代，是因为 user:write 是管理员语义，
// 端用户持有它才改得了自己资料 —— 那等于把整个用户目录的写权限发给端用户。
// 提权风险由 DTO 白名单挡在门外：请求体里的 role/username/active 不会被绑定。
func (h *Handler) UpdateProfile(c *gin.Context) {
	userID, ok := accountUserID(c)
	if !ok {
		return
	}

	var req dto.UpdateOwnProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误：name 与 email 均为必填且需合法")
		return
	}

	if h.userService == nil {
		common.Fail(c, common.ServiceUnavailableCode, "账号服务未就绪")
		return
	}

	tenantID, _ := c.Get("tenant_id")
	tid, _ := tenantID.(int)

	updated, err := h.userService.UpdateOwnProfile(c.Request.Context(), userID, tid, &req)
	if err != nil {
		common.RespondError(c, err, "更新个人资料失败")
		return
	}

	common.Success(c, dto.ToUserDetailResponse(updated))
}

// accountUserID 从鉴权上下文取当前用户 ID。
func accountUserID(c *gin.Context) (int, bool) {
	v, exists := c.Get("user_id")
	if !exists {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return 0, false
	}
	id, ok := v.(int)
	if !ok || id <= 0 {
		common.Fail(c, common.UnauthorizedCode, "未授权访问")
		return 0, false
	}
	return id, true
}

// ParseUserID 解析路径中的用户 ID（供未来管理面复用）。
func ParseUserID(s string) (int, bool) {
	id, err := strconv.Atoi(s)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}
