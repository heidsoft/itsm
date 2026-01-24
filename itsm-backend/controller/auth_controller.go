package controller

import (
	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

type AuthController struct {
	authService *service.AuthService
}

func NewAuthController(authService *service.AuthService) *AuthController {
	return &AuthController{
		authService: authService,
	}
}

// Login 登录接口
func (ac *AuthController) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}

	ctx := c.Request.Context()
	response, err := ac.authService.Login(ctx, &req)
	if err != nil {
		common.AuthFailed(c, err.Error())
		return
	}

	common.Success(c, response)
}

// RefreshToken 刷新token接口
func (ac *AuthController) RefreshToken(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}

	ctx := c.Request.Context()
	response, err := ac.authService.RefreshToken(ctx, &req)
	if err != nil {
		common.AuthFailed(c, err.Error())
		return
	}

	common.Success(c, response)
}

// GetUserTenants 获取用户可访问的租户列表
func (ac *AuthController) GetUserTenants(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.AuthFailed(c, "用户未认证")
		return
	}

	ctx := c.Request.Context()
	response, err := ac.authService.GetUserTenants(ctx, userID.(int))
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}

	common.Success(c, response)
}

// SwitchTenant 切换租户
func (ac *AuthController) SwitchTenant(c *gin.Context) {
	var req dto.SwitchTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		common.AuthFailed(c, "用户未认证")
		return
	}

	ctx := c.Request.Context()
	response, err := ac.authService.SwitchTenant(ctx, userID.(int), req.TenantID)
	if err != nil {
		common.Forbidden(c, err.Error())
		return
	}

	common.Success(c, response)
}

// GetUserInfo 获取当前用户信息
func (ac *AuthController) GetUserInfo(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.AuthFailed(c, "用户未认证")
		return
	}

	ctx := c.Request.Context()
	response, err := ac.authService.GetUserInfo(ctx, userID.(int))
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}

	common.Success(c, response)
}

// Logout 登出接口
func (ac *AuthController) Logout(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.AuthFailed(c, "用户未认证")
		return
	}

	ctx := c.Request.Context()
	err := ac.authService.Logout(ctx, userID.(int))
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}

	common.Success(c, nil)
}
