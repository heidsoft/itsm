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

// AuthResult contains tokens and user context
type AuthResult struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	User         *User  `json:"user"`
}
