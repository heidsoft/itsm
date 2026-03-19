package dto

import "time"

// ProjectResponse 项目响应
type ProjectResponse struct {
	ID           int        `json:"id"`
	Name         string     `json:"name"`
	Code         string     `json:"code"`
	Description  string     `json:"description"`
	ManagerID    *int       `json:"manager_id,omitempty"`
	DepartmentID *int       `json:"department_id,omitempty"`
	StartDate    *time.Time `json:"start_date,omitempty"`
	EndDate      *time.Time `json:"end_date,omitempty"`
	Status       string     `json:"status"`
	TenantID     int        `json:"tenant_id"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// ProjectListResponse 项目列表响应
type ProjectListResponse struct {
	Projects []*ProjectResponse `json:"projects"`
	Total    int               `json:"total"`
}