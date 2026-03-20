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
	Department   string    `json:"department"`
	DepartmentID int       `json:"department_id"`
	Phone        string    `json:"phone"`
	Active       bool      `json:"active"`
	TenantID     int       `json:"tenant_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Department represents a spatial or organizational unit
type Department struct {
	ID          int           `json:"id"`
	Name        string        `json:"name"`
	Code        string        `json:"code"`
	Description string        `json:"description"`
	ManagerID   int           `json:"manager_id"`
	ParentID    int           `json:"parent_id"`
	TenantID    int           `json:"tenant_id"`
	Children    []*Department `json:"children,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// Team represents a group of users
type Team struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	ManagerID   int       `json:"manager_id"`
	TenantID    int       `json:"tenant_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Tag represents a metadata label
type Tag struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	Color       string    `json:"color"`
	TenantID    int       `json:"tenant_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AuditLog represents a system activity record
type AuditLog struct {
	ID          int       `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	TenantID    int       `json:"tenant_id"`
	UserID      int       `json:"user_id"`
	RequestID   string    `json:"request_id"`
	IP          string    `json:"ip"`
	Resource    string    `json:"resource"`
	Action      string    `json:"action"`
	Path        string    `json:"path"`
	Method      string    `json:"method"`
	StatusCode  int       `json:"status_code"`
	RequestBody string    `json:"request_body"`
}

// AuthResult contains tokens and user context
type AuthResult struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         *User  `json:"user"`
}
