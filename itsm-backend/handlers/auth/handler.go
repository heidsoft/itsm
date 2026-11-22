package auth

import (
	"context"
	"net/http"
	"strings"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
)

// Handler exposes the account self-service endpoints that are not owned by
// handlers/common's session handler.
type Handler struct {
	service *Service
	// sso 为 SSO（OIDC）子系统；装配层未注入时为 nil，SSO 路由不注册。
	sso *SSOService
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

// 公开自助端点的路由路径。刻意写成常量并在 router 侧同一字符串复用：
// RecordLoginAudit 把 path 硬编码成 /api/v1/auth/login，导致 SSO 回调也被
// 记成登录路径——逐端点审计的前提是每条记录都能指向自己的真实入口。
const (
	pathRegister       = "/api/v1/auth/register"
	pathForgotPassword = "/api/v1/auth/forgot-password"
	pathResetPassword  = "/api/v1/auth/reset-password"
)

// recordSelfServiceAudit 落一条自助动作审计，状态码取**写完响应后**的真实值。
//
// 为什么在 handler 而不是 service 里调用：只有 handler 知道这一笔 HTTP 请求
// 最终返回 200 还是 400/401/503，而「失败的注册尝试」正是撞库与批量注册
// 最需要的证据。service 侧只能拿到 error，无法区分参数错误与业务拒绝。
//
// 因此本函数必须在所有 return 分支之后调用——包括参数绑定失败的分支：
// 那是批量脚本最容易命中的路径，恰恰最需要留痕。
func (h *Handler) recordSelfServiceAudit(c *gin.Context, auditCtx context.Context, path, action, subject, reason string) {
	h.recordSelfServiceAuditWithTenant(c, auditCtx, path, action, subject, reason, 0)
}

func (h *Handler) recordSelfServiceAuditWithTenant(c *gin.Context, auditCtx context.Context, path, action, subject, reason string, tenantID int) {
	middleware.RecordSelfServiceAudit(auditCtx, h.service.client, middleware.SelfServiceAuditEvent{
		Path:       path,
		Action:     action,
		StatusCode: c.Writer.Status(),
		// public 端点没有 JWT；失败时租户往往尚未确定，留 0 是如实记录。
		// 成功后若已解析出唯一活跃租户则传入真实值，让记录可归属。
		// 两种情况都刻意不填「默认租户」：那会让某个客户租户的审计视图
		// 看到不属于它的记录。
		TenantID: tenantID,
		UserID:   0,
		Subject:  subject,
		Reason:   reason,
	})
}

// SetSSOService 注入 SSO 子系统。单独 setter 而不是改 NewHandler 签名：
// 既有装配与测试只构造账号自助面，不应被 SSO 的依赖（加密服务、state 存储）传染。
func (h *Handler) SetSSOService(sso *SSOService) { h.sso = sso }

// SSOEnabled 报告 SSO 路由是否可注册。
func (h *Handler) SSOEnabled() bool { return h.sso != nil }

// @Summary 用户自助注册
// @Description 注册新账号。用户名被占用/自助注册未开放等业务拒绝按分类码返回，不统一压成认证失败。公开端点，无需 Bearer Token。
// @Tags 认证管理
// @Accept json
// @Produce json
// @Param body body dto.RegisterRequest true "注册请求"
// @Success 200 {object} common.Response
// @Failure 400 "参数错误"
// @Failure 409 "用户名已占用"
// @Failure 500 "后端故障"
// @Router /api/v1/auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	auditCtx := middleware.WithLoginAuditRequest(c.Request.Context(), c.ClientIP(), c.Request.UserAgent())
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "参数错误")
		h.recordSelfServiceAudit(c, auditCtx, pathRegister, "register", "", "参数错误")
		return
	}
	response, err := h.service.Register(c.Request.Context(), &req)
	if err != nil {
		// 注册失败的原因（用户名已占用、自助注册未开放）是用户下一步动作的依据，
		// 统一压成 2001/「认证失败」会让新用户以为是自己填错了凭据。
		common.RespondError(c, err, "注册失败")
		// 审计只记用户名，不记邮箱与密码：用户名是「谁想注册」的标识，
		// 邮箱与密码属于凭据面，一旦落库无法回收。
		h.recordSelfServiceAudit(c, auditCtx, pathRegister, "register", req.Username, err.Error())
		return
	}
	common.Success(c, response)
	// 成功分支的租户归属取自 service 解析出的真实租户（自助注册只允许在
	// 「恰好一个活跃租户」时成功），而不是像失败分支那样留 0——
	// 注册成功是一条需要归属到具体租户才能查的合规记录。
	h.recordSelfServiceAuditWithTenant(c, auditCtx, pathRegister, "register", req.Username, "", response.TenantID)
}

// @Summary 申请密码重置
// @Description 发起密码找回。service 以 5003 明确表达「邮件服务未配置」，不会让调用方误以为邮箱填错而反复重试。公开端点。
// @Tags 认证管理
// @Accept json
// @Produce json
// @Param body body dto.ForgotPasswordRequest true "找回请求"
// @Success 200 {object} common.Response
// @Failure 400 "参数错误"
// @Failure 503 "邮件服务未配置(5003)"
// @Router /api/v1/auth/forgot-password [post]
func (h *Handler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	auditCtx := middleware.WithLoginAuditRequest(c.Request.Context(), c.ClientIP(), c.Request.UserAgent())
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "参数错误")
		h.recordSelfServiceAudit(c, auditCtx, pathForgotPassword, "forgot_password", "", "参数错误")
		return
	}
	response, err := h.service.ForgotPassword(c.Request.Context(), &req)
	if err != nil {
		// service 明确用 5003 表达「邮件服务未配置」；压成 2001 会让用户以为是
		// 自己的邮箱有问题并反复重试。
		common.RespondError(c, err, "密码找回请求失败")
		h.recordSelfServiceAudit(c, auditCtx, pathForgotPassword, "forgot_password", "", err.Error())
		return
	}
	common.Success(c, response)
	// 成功分支**刻意不记邮箱**：本端点对「邮箱已注册/未注册」返回同一个通用
	// 响应以防账号枚举，审计正文若存邮箱就等于把这份防护挪进了另一张表——
	// 有 audit:read 权限的人仍可批量比对「谁真的收到了重置邮件」。
	h.recordSelfServiceAudit(c, auditCtx, pathForgotPassword, "forgot_password", "", "")
}

// @Summary 重置密码
// @Description 凭重置令牌设置新密码；令牌无效/过期返回认证失败。公开端点。
// @Tags 认证管理
// @Accept json
// @Produce json
// @Param body body dto.PasswordResetRequest true "重置请求"
// @Success 200 {object} common.Response
// @Failure 400 "参数错误"
// @Failure 401 "令牌无效或过期"
// @Router /api/v1/auth/reset-password [post]
func (h *Handler) ResetPassword(c *gin.Context) {
	var req dto.PasswordResetRequest
	auditCtx := middleware.WithLoginAuditRequest(c.Request.Context(), c.ClientIP(), c.Request.UserAgent())
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "参数错误")
		h.recordSelfServiceAudit(c, auditCtx, pathResetPassword, "reset_password", "", "参数错误")
		return
	}
	response, err := h.service.ResetPassword(c.Request.Context(), &req)
	if err != nil {
		common.AuthFailedWithErr(c, err, "认证失败")
		h.recordSelfServiceAudit(c, auditCtx, pathResetPassword, "reset_password", "", err.Error())
		return
	}
	common.Success(c, response)
	// 成功分支只记「有人完成了重置」，不记邮箱、不记令牌：
	// 重置令牌是一次性消费凭据，写进 append-only 审计表等于永久留了一份
	// 曾经可用的凭据副本；邮箱则会让审计读面变成账号枚举的第二条通道。
	h.recordSelfServiceAudit(c, auditCtx, pathResetPassword, "reset_password", "", "")
}

// @Summary 校验重置令牌
// @Description 校验密码重置令牌是否有效，供前端在展示重置表单前预检。公开端点。
// @Tags 认证管理
// @Accept json
// @Produce json
// @Param body body dto.ValidateResetTokenRequest true "令牌校验请求"
// @Success 200 {object} common.Response
// @Failure 400 "参数错误"
// @Failure 401 "令牌无效或过期"
// @Router /api/v1/auth/validate-reset-token [post]
func (h *Handler) ValidateResetToken(c *gin.Context) {
	var req dto.ValidateResetTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "参数错误")
		return
	}
	// 刻意**不审计**本端点：它只读、无副作用，且前端在展示重置表单前会先
	// 预检一次——每一次正常的密码重置都会附赠一条预检记录，审计表里
	// 「重置动作」的数量会凭空翻倍，反而看不出真实的重置次数。
	// 真正需要留痕的是 reset-password（状态变更）与 forgot-password（发信）。
	response, err := h.service.ValidateResetToken(c.Request.Context(), &req)
	if err != nil {
		common.AuthFailedWithErr(c, err, "认证失败")
		return
	}
	common.Success(c, response)
}

// @Summary 切换租户
// @Description 为已认证用户切换当前租户，校验其属于目标租户后签发新的 access/refresh token 并写入会话 cookie；越权返回 403。
// @Tags 认证管理
// @Accept json
// @Produce json
// @Param body body dto.SwitchTenantRequest true "目标租户"
// @Success 200 {object} common.Response
// @Failure 400 "参数错误"
// @Failure 401 "未认证"
// @Failure 403 "不属于目标租户"
// @Security BearerAuth
// @Router /api/v1/auth/switch-tenant [post]
func (h *Handler) SwitchTenant(c *gin.Context) {
	var req dto.SwitchTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "参数错误")
		return
	}
	userID := c.GetInt("user_id")
	if userID == 0 {
		common.AuthFailed(c, "用户未认证")
		return
	}
	response, err := h.service.SwitchTenant(c.Request.Context(), userID, req.TenantID)
	if err != nil {
		common.ForbiddenWithErr(c, err, "权限不足")
		return
	}
	setAuthCookies(c, response.AccessToken, response.RefreshToken)
	common.Success(c, response)
}

// authCookieAttrs 返回会话 cookie 的属性。名称与生命周期必须与 handlers/common
// 设置凭证时同源，否则同一个浏览器会被两条链路写入不同 Max-Age 的同名 cookie。
func authCookieAttrs(c *gin.Context) (domain string, secure bool) {
	return "", c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
}

func setAuthCookies(c *gin.Context, accessToken, refreshToken string) {
	domain, secure := authCookieAttrs(c)
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(middleware.AccessTokenCookie, accessToken, int(middleware.AccessTokenTTL.Seconds()), "/", domain, secure, true)
	if refreshToken != "" {
		c.SetCookie(middleware.RefreshTokenCookie, refreshToken, int(middleware.RefreshTokenTTL.Seconds()), "/", domain, secure, true)
	}
}
