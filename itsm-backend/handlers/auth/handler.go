package auth

import (
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

// SetSSOService 注入 SSO 子系统。单独 setter 而不是改 NewHandler 签名：
// 既有装配与测试只构造账号自助面，不应被 SSO 的依赖（加密服务、state 存储）传染。
func (h *Handler) SetSSOService(sso *SSOService) { h.sso = sso }

// SSOEnabled 报告 SSO 路由是否可注册。
func (h *Handler) SSOEnabled() bool { return h.sso != nil }

func (h *Handler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "参数错误")
		return
	}
	response, err := h.service.Register(c.Request.Context(), &req)
	if err != nil {
		// 注册失败的原因（用户名已占用、自助注册未开放）是用户下一步动作的依据，
		// 统一压成 2001/「认证失败」会让新用户以为是自己填错了凭据。
		common.RespondError(c, err, "注册失败")
		return
	}
	common.Success(c, response)
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "参数错误")
		return
	}
	response, err := h.service.ForgotPassword(c.Request.Context(), &req)
	if err != nil {
		// service 明确用 5003 表达「邮件服务未配置」；压成 2001 会让用户以为是
		// 自己的邮箱有问题并反复重试。
		common.RespondError(c, err, "密码找回请求失败")
		return
	}
	common.Success(c, response)
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req dto.PasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "参数错误")
		return
	}
	response, err := h.service.ResetPassword(c.Request.Context(), &req)
	if err != nil {
		common.AuthFailedWithErr(c, err, "认证失败")
		return
	}
	common.Success(c, response)
}

func (h *Handler) ValidateResetToken(c *gin.Context) {
	var req dto.ValidateResetTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "参数错误")
		return
	}
	response, err := h.service.ValidateResetToken(c.Request.Context(), &req)
	if err != nil {
		common.AuthFailedWithErr(c, err, "认证失败")
		return
	}
	common.Success(c, response)
}

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
