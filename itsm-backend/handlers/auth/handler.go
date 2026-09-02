package auth

import (
	"net/http"
	"strings"

	"itsm-backend/common"
	"itsm-backend/dto"

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

func setAuthCookies(c *gin.Context, accessToken, refreshToken string) {
	c.SetSameSite(http.SameSiteLaxMode)
	secure := c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
	c.SetCookie("access_token", accessToken, 900, "/", "", secure, true)
	if refreshToken != "" {
		c.SetCookie("refresh_token", refreshToken, 604800, "/", "", secure, true)
	}
}
