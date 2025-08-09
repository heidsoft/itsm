package controller

import (
	"context"
	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/tenant"
	"itsm-backend/ent/user"
	"itsm-backend/middleware"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthController struct {
	jwtSecret string
	client    *ent.Client
}

func NewAuthController(jwtSecret string, client *ent.Client) *AuthController {
	return &AuthController{
		jwtSecret: jwtSecret,
		client:    client,
	}
}

// Login 登录接口
func (ac *AuthController) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	ctx := c.Request.Context()

	// 查询用户
	userQuery := ac.client.User.Query().Where(user.UsernameEQ(req.Username))

	// 如果提供了租户代码，则按租户过滤
	if req.TenantCode != "" {
		// 先查询租户
		tenantEntity, err := ac.client.Tenant.Query().
			Where(tenant.CodeEQ(req.TenantCode)).
			First(ctx)
		if err != nil {
			common.Fail(c, 2001, "用户名或密码错误")
			return
		}
		userQuery = userQuery.Where(user.TenantIDEQ(tenantEntity.ID))
	}

	userEntity, err := userQuery.First(ctx)
	if err != nil {
		common.Fail(c, 2001, "用户名或密码错误")
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(userEntity.PasswordHash), []byte(req.Password)); err != nil {
		common.Fail(c, 2001, "用户名或密码错误")
		return
	}

	// 检查用户是否激活（简化处理，假设所有用户都是激活的）
	// if !userEntity.Active {
	// 	common.Fail(c, 2002, "用户账号已被禁用")
	// 	return
	// }

	// 检查租户状态
	tenantEntity, err := ac.client.Tenant.Get(ctx, userEntity.TenantID)
	if err != nil {
		common.Fail(c, 2003, "租户不存在")
		return
	}

	if tenantEntity.Status != "active" {
		common.Fail(c, 2003, "租户已被暂停或过期")
		return
	}

	// 生成access token（15分钟）
	accessToken, err := middleware.GenerateAccessToken(
		userEntity.ID,
		userEntity.Username,
		"user", // 暂时硬编码，因为User实体没有Role字段
		userEntity.TenantID,
		ac.jwtSecret,
		time.Duration(15)*time.Minute,
	)
	if err != nil {
		common.Fail(c, 5001, "生成访问令牌失败")
		return
	}

	// 生成refresh token（7天）
	refreshToken, err := middleware.GenerateRefreshToken(
		userEntity.ID,
		ac.jwtSecret,
		time.Duration(7*24)*time.Hour,
	)
	if err != nil {
		common.Fail(c, 5001, "生成刷新令牌失败")
		return
	}

	response := dto.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         userEntity,
		Tenant:       tenantEntity,
	}

	common.Success(c, response)
}

// RefreshToken 刷新token接口
func (ac *AuthController) RefreshToken(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	// 验证refresh token
	token, err := jwt.ParseWithClaims(req.RefreshToken, &middleware.Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(ac.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		common.Fail(c, 2001, "刷新令牌无效")
		return
	}

	claims, ok := token.Claims.(*middleware.Claims)
	if !ok || claims.TokenType != "refresh" {
		common.Fail(c, 2001, "令牌类型错误")
		return
	}

	// 获取用户信息
	userEntity, err := ac.client.User.Get(context.Background(), claims.UserID)
	if err != nil {
		common.Fail(c, 2001, "用户不存在")
		return
	}

	// 生成新的access token
	newAccessToken, err := middleware.GenerateAccessToken(
		userEntity.ID,
		userEntity.Username,
		"user", // 暂时硬编码
		userEntity.TenantID,
		ac.jwtSecret,
		time.Duration(15)*time.Minute,
	)
	if err != nil {
		common.Fail(c, 5001, "生成新令牌失败")
		return
	}

	response := dto.RefreshTokenResponse{
		AccessToken: newAccessToken,
	}

	common.Success(c, response)
}

// GetUserTenants 获取用户可访问的租户列表
func (ac *AuthController) GetUserTenants(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		common.Fail(c, 2001, "用户未认证")
		return
	}

	ctx := context.Background()

	// 查询用户的租户（这里简化为用户只属于一个租户）
	userEntity, err := ac.client.User.Query().
		Where(user.IDEQ(userID.(int))).
		First(ctx)
	if err != nil {
		common.Fail(c, 5001, "查询用户信息失败")
		return
	}

	// 查询租户信息
	tenantEntity, err := ac.client.Tenant.Get(ctx, userEntity.TenantID)
	if err != nil {
		common.Fail(c, 5001, "查询租户信息失败")
		return
	}

	tenants := []dto.TenantInfo{
		{
			ID:     tenantEntity.ID,
			Name:   tenantEntity.Name,
			Code:   tenantEntity.Code,
			Domain: tenantEntity.Domain,
			Type:   tenantEntity.Type,
			Status: tenantEntity.Status,
		},
	}

	response := dto.UserTenantsResponse{
		Tenants: tenants,
	}
	common.Success(c, response)
}

// SwitchTenant 切换租户
func (ac *AuthController) SwitchTenant(c *gin.Context) {
	var req dto.SwitchTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		common.Fail(c, 2001, "用户未认证")
		return
	}

	ctx := context.Background()

	// 验证用户是否有权限访问该租户
	userEntity, err := ac.client.User.Query().
		Where(
			user.IDEQ(userID.(int)),
			user.TenantIDEQ(req.TenantID),
		).
		First(ctx)
	if err != nil {
		common.Fail(c, 2004, "无权限访问该租户")
		return
	}

	// 查询租户信息
	tenantEntity, err := ac.client.Tenant.Get(ctx, req.TenantID)
	if err != nil {
		common.Fail(c, 2004, "租户不存在")
		return
	}

	// 生成新的access token
	accessToken, err := middleware.GenerateAccessToken(
		userEntity.ID,
		userEntity.Username,
		"user",
		req.TenantID,
		ac.jwtSecret,
		time.Duration(15)*time.Minute,
	)
	if err != nil {
		common.Fail(c, 5001, "生成token失败")
		return
	}

	// 生成新的refresh token
	refreshToken, err := middleware.GenerateRefreshToken(
		userEntity.ID,
		ac.jwtSecret,
		time.Duration(7*24)*time.Hour,
	)
	if err != nil {
		common.Fail(c, 5001, "生成刷新令牌失败")
		return
	}

	response := dto.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         userEntity,
		Tenant:       tenantEntity,
	}
	common.Success(c, response)
}
