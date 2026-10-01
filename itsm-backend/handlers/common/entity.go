package common

import (
	"time"
)

// User represents a system user
type User struct {
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
	Permissions  []string  `json:"permissions,omitempty"` // 用户权限列表
}

// Department represents a spatial or organizational unit
type Department struct {
	ID          int           `json:"id"`
	Name        string        `json:"name"`
	Code        string        `json:"code"`
	Description string        `json:"description"`
	ManagerID   int           `json:"managerId"`
	ParentID    int           `json:"parentId"`
	TenantID    int           `json:"tenantId"`
	Children    []*Department `json:"children,omitempty"`
	CreatedAt   time.Time     `json:"createdAt"`
	UpdatedAt   time.Time     `json:"updatedAt"`
}

// Team represents a group of users
type Team struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	ManagerID   int       `json:"managerId"`
	TenantID    int       `json:"tenantId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Tag represents a metadata label
type Tag struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	Color       string    `json:"color"`
	TenantID    int       `json:"tenantId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// AuditLog represents a system activity record
type AuditLog struct {
	ID          int       `json:"id"`
	CreatedAt   time.Time `json:"createdAt"`
	TenantID    int       `json:"tenantId"`
	UserID      int       `json:"userId"`
	RequestID   string    `json:"requestId"`
	IP          string    `json:"ip"`
	Resource    string    `json:"resource"`
	Action      string    `json:"action"`
	Path        string    `json:"path"`
	Method      string    `json:"method"`
	StatusCode  int       `json:"statusCode"`
	RequestBody string    `json:"requestBody"`
}

// AuthResult contains tokens and user context.
//
// AccessToken/RefreshToken 只能进 httpOnly cookie：json:"-" 是不许把凭证写回
// 浏览器可读响应体的硬边界，由 router/auth_handler_routes_test.go 断言。
type AuthResult struct {
	AccessToken  string `json:"-"`
	RefreshToken string `json:"-"`
	// ExpiresIn 是 access token 的剩余有效期（秒），由服务端时钟给出。
	// 前端据此安排续签，不再用浏览器时钟猜「15 分钟减一点」。
	ExpiresIn int   `json:"expiresIn"`
	User      *User `json:"user"`
}

// TenantBrief 是会话视图里的租户摘要（不含任何配置或凭证）。
type TenantBrief struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Code   string `json:"code"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

// SessionResponse 是「当前会话」的唯一后端真相：一次请求同时给出身份、
// 可访问租户范围，以及服务端权威的 access token 剩余有效期。
// 前端启动只依赖它，不再用 cookie 是否存在或本地持久化状态推断登录态。
type SessionResponse struct {
	User      *User         `json:"user"`
	Tenants   []TenantBrief `json:"tenants"`
	ExpiresIn int           `json:"expiresIn"`
}
