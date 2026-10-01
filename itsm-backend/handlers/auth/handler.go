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
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}
	response, err := h.service.Register(c.Request.Context(), &req)
	if err != nil {
		common.AuthFailed(c, err.Error())
		return
	}
	common.Success(c, response)
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}
	response, err := h.service.ForgotPassword(c.Request.Context(), &req)
	if err != nil {
		common.AuthFailed(c, err.Error())
		return
	}
	common.Success(c, response)
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req dto.PasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}
	response, err := h.service.ResetPassword(c.Request.Context(), &req)
	if err != nil {
		common.AuthFailed(c, err.Error())
		return
	}
	common.Success(c, response)
}

func (h *Handler) ValidateResetToken(c *gin.Context) {
	var req dto.ValidateResetTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}
	response, err := h.service.ValidateResetToken(c.Request.Context(), &req)
	if err != nil {
		common.AuthFailed(c, err.Error())
		return
	}
	common.Success(c, response)
}

func (h *Handler) SwitchTenant(c *gin.Context) {
	var req dto.SwitchTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}
	userID := c.GetInt("user_id")
	if userID == 0 {
		common.AuthFailed(c, "用户未认证")
		return
	}
	response, err := h.service.SwitchTenant(c.Request.Context(), userID, req.TenantID)
	if err != nil {
		common.Forbidden(c, err.Error())
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
