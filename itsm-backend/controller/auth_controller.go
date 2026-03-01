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
// @Summary 用户登录
// @Description 使用用户名/邮箱和密码登录，返回访问令牌和刷新令牌
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "登录请求（username/email + password）"
// @Success 200 {object} dto.LoginResponse "登录成功，返回 tokens 和用户信息"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "认证失败（用户名或密码错误）"
// @Router /api/v1/auth/login [post]
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

// RefreshToken 刷新 token 接口
// @Summary 刷新访问令牌
// @Description 使用刷新令牌获取新的访问令牌
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body dto.RefreshTokenRequest true "刷新令牌请求"
// @Success 200 {object} dto.RefreshTokenResponse "刷新成功，返回新的访问令牌"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "刷新令牌无效或已过期"
// @Router /api/v1/auth/refresh [post]
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
// @Summary 获取用户可访问的租户列表
// @Description 获取当前用户可以访问的所有租户信息
// @Tags 认证
// @Accept json
// @Produce json
// @Success 200 {object} dto.GetUserTenantsResponse "租户列表"
// @Failure 401 {object} map[string]interface{} "用户未认证"
// @Router /api/v1/auth/tenants [get]
// @Security BearerAuth
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
// @Summary 切换当前租户
// @Description 切换用户当前操作的租户上下文
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body dto.SwitchTenantRequest true "切换租户请求（tenant_id）"
// @Success 200 {object} dto.SwitchTenantResponse "切换成功，返回新的租户信息"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "用户未认证"
// @Failure 403 {object} map[string]interface{} "无权访问该租户"
// @Router /api/v1/auth/switch-tenant [post]
// @Security BearerAuth
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
// @Summary 获取当前登录用户信息
// @Description 获取当前登录用户的详细信息
// @Tags 认证
// @Accept json
// @Produce json
// @Success 200 {object} dto.UserInfo "用户详细信息"
// @Failure 401 {object} map[string]interface{} "用户未认证"
// @Router /api/v1/auth/me [get]
// @Security BearerAuth
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
// @Summary 用户登出
// @Description 使当前用户的令牌失效，退出登录
// @Tags 认证
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "登出成功"
// @Failure 401 {object} map[string]interface{} "用户未认证"
// @Router /api/v1/auth/logout [post]
// @Security BearerAuth
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

// Register 用户注册接口
// @Summary 用户注册
// @Description 创建新用户账号
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body dto.RegisterRequest true "注册请求（username, email, password 等）"
// @Success 200 {object} dto.RegisterResponse "注册成功，返回用户信息"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 409 {object} map[string]interface{} "用户名或邮箱已存在"
// @Router /api/v1/auth/register [post]
func (ac *AuthController) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}

	ctx := c.Request.Context()
	response, err := ac.authService.Register(ctx, &req)
	if err != nil {
		common.AuthFailed(c, err.Error())
		return
	}

	common.Success(c, response)
}

// ForgotPassword 忘记密码接口
// @Summary 忘记密码 - 申请重置
// @Description 发送密码重置邮件到用户邮箱
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body dto.ForgotPasswordRequest true "忘记密码请求（email）"
// @Success 200 {object} dto.ForgotPasswordResponse "重置邮件发送成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 404 {object} map[string]interface{} "用户不存在"
// @Router /api/v1/auth/forgot-password [post]
func (ac *AuthController) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}

	ctx := c.Request.Context()
	response, err := ac.authService.ForgotPassword(ctx, &req)
	if err != nil {
		common.AuthFailed(c, err.Error())
		return
	}

	common.Success(c, response)
}

// ResetPassword 重置密码接口
// @Summary 重置密码
// @Description 使用重置令牌设置新密码
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body dto.PasswordResetRequest true "重置密码请求（token + new_password）"
// @Success 200 {object} dto.PasswordResetResponse "密码重置成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "重置令牌无效或已过期"
// @Router /api/v1/auth/reset-password [post]
func (ac *AuthController) ResetPassword(c *gin.Context) {
	var req dto.PasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}

	ctx := c.Request.Context()
	response, err := ac.authService.ResetPassword(ctx, &req)
	if err != nil {
		common.AuthFailed(c, err.Error())
		return
	}

	common.Success(c, response)
}

// ValidateResetToken 验证重置令牌接口
// @Summary 验证重置令牌
// @Description 验证密码重置令牌是否有效
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body dto.ValidateResetTokenRequest true "验证令牌请求（token）"
// @Success 200 {object} dto.ValidateResetTokenResponse "令牌有效"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "令牌无效或已过期"
// @Router /api/v1/auth/validate-reset-token [post]
func (ac *AuthController) ValidateResetToken(c *gin.Context) {
	var req dto.ValidateResetTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamError(c, "参数错误: "+err.Error())
		return
	}

	ctx := c.Request.Context()
	response, err := ac.authService.ValidateResetToken(ctx, &req)
	if err != nil {
		common.AuthFailed(c, err.Error())
		return
	}

	common.Success(c, response)
}
