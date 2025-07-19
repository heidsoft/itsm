package dto

import (
	"itsm-backend/ent"
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
	RefreshToken string `json:"refresh_token"`
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
