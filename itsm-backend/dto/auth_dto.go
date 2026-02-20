package dto

import (
	"itsm-backend/ent"
	"time"
)

type LoginRequest struct {
	Username   string `json:"username" binding:"required"`
	Password   string `json:"password" binding:"required"`
	TenantCode string `json:"tenant_code,omitempty"` // 可选的租户代码
}

type LoginResponse struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	User         *ent.User   `json:"user"`
	Tenant       *ent.Tenant `json:"tenant"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type RefreshTokenResponse struct {
	AccessToken string `json:"access_token"`
}

type UserInfo struct {
	ID         int    `json:"id"`
	Username   string `json:"username"`
	Role       string `json:"role"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	Department string `json:"department"`
	TenantID   int    `json:"tenant_id"`
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
	TenantID int `json:"tenant_id" binding:"required"`
}

// 获取用户租户列表响应
type UserTenantsResponse struct {
	Tenants []TenantInfo `json:"tenants"`
}

// RegisterRequest 用户注册请求
type RegisterRequest struct {
	Username    string `json:"username" binding:"required,min=3,max=20,alphanum"`
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8"`
	FullName    string `json:"full_name" binding:"required"`
	Phone       string `json:"phone" binding:"omitempty"`
	Company     string `json:"company,omitempty"`
	Role        string `json:"role" binding:"omitempty"`
	TenantCode  string `json:"tenant_code,omitempty"`
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
	TenantCode string `json:"tenant_code,omitempty"`
}

// ForgotPasswordResponse 忘记密码响应
type ForgotPasswordResponse struct {
	Message string `json:"message"`
}

// PasswordResetRequest 密码重置请求（用于忘记密码流程）
type PasswordResetRequest struct {
	Token          string `json:"token" binding:"required"`
	Email          string `json:"email" binding:"required,email"`
	Password       string `json:"password" binding:"required,min=8"`
	PasswordConfirm string `json:"password_confirm" binding:"required"`
}

// PasswordResetResponse 密码重置响应
type PasswordResetResponse struct {
	Message string `json:"message"`
}

// ValidateResetTokenRequest 验证重置令牌请求
type ValidateResetTokenRequest struct {
	Token  string `json:"token" binding:"required"`
	Email  string `json:"email" binding:"required,email"`
}

// ValidateResetTokenResponse 验证重置令牌响应
type ValidateResetTokenResponse struct {
	Valid bool   `json:"valid"`
	Email string `json:"email"`
}

// PasswordResetToken 密码重置令牌记录
type PasswordResetToken struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Email     string    `json:"email"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	Used      bool      `json:"used"`
	CreatedAt time.Time `json:"created_at"`
}
