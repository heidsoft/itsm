package dto

import (
	"time"

	"itsm-backend/ent"
)

type LoginRequest struct {
	Username   string `json:"username" binding:"required"`
	Password   string `json:"password" binding:"required"`
	TenantCode string `json:"tenantCode,omitempty"` // 可选的租户代码
}

type LoginResponse struct {
	AccessToken  string             `json:"accessToken"`
	RefreshToken string             `json:"refreshToken"`
	User         *LoginUserResponse `json:"user"`
	Tenant       *ent.Tenant        `json:"tenant"`
}

// LoginUserResponse 登录返回的用户信息（包含权限列表）
type LoginUserResponse struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	Role         string    `json:"role"`
	MSPRole      *string   `json:"mspRole,omitempty"`
	Department   string    `json:"department"`
	DepartmentID int       `json:"departmentId"`
	Phone        string    `json:"phone"`
	Active       bool      `json:"active"`
	TenantID     int       `json:"tenantId"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	Permissions  []string  `json:"permissions"` // 用户权限列表
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

type RefreshTokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken,omitempty"`
}

type UserInfo struct {
	ID         int    `json:"id"`
	Username   string `json:"username"`
	Role       string `json:"role"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	Department string `json:"department"`
	TenantID   int    `json:"tenantId"`
}

type TenantInfo struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Code   string `json:"code"`
	Domain string `json:"domain"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

// 租户切换请求
type SwitchTenantRequest struct {
	TenantID int `json:"tenantId" binding:"required"`
}

// 获取用户租户列表响应
type UserTenantsResponse struct {
	Tenants []TenantInfo `json:"tenants"`
}

// RegisterRequest 用户注册请求
type RegisterRequest struct {
	Username   string `json:"username" binding:"required,min=3,max=20,alphanum"`
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required,min=8"`
	FullName   string `json:"fullName" binding:"required"`
	Phone      string `json:"phone" binding:"omitempty"`
	Company    string `json:"company,omitempty"`
	Role       string `json:"role" binding:"omitempty"`
	TenantCode string `json:"tenantCode,omitempty"`
}

// RegisterResponse 用户注册响应
type RegisterResponse struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Message  string `json:"message"`
}

// ForgotPasswordRequest 忘记密码请求
type ForgotPasswordRequest struct {
	Email      string `json:"email" binding:"required,email"`
	TenantCode string `json:"tenantCode,omitempty"`
}

// ForgotPasswordResponse 忘记密码响应
type ForgotPasswordResponse struct {
	Message string `json:"message"`
}

// PasswordResetRequest 密码重置请求（用于忘记密码流程）
type PasswordResetRequest struct {
	Token           string `json:"token" binding:"required"`
	Email           string `json:"email" binding:"required,email"`
	Password        string `json:"password" binding:"required,min=8"`
	PasswordConfirm string `json:"passwordConfirm" binding:"required"`
}

// PasswordResetResponse 密码重置响应
type PasswordResetResponse struct {
	Message string `json:"message"`
}

// ValidateResetTokenRequest 验证重置令牌请求
type ValidateResetTokenRequest struct {
	Token string `json:"token" binding:"required"`
	Email string `json:"email" binding:"required,email"`
}

// ValidateResetTokenResponse 验证重置令牌响应
type ValidateResetTokenResponse struct {
	Valid bool   `json:"valid"`
	Email string `json:"email"`
}

// PasswordResetToken 密码重置令牌记录
type PasswordResetToken struct {
	ID        int       `json:"id"`
	UserID    int       `json:"userId"`
	Email     string    `json:"email"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	Used      bool      `json:"used"`
	CreatedAt time.Time `json:"createdAt"`
}
