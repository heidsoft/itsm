package dto

import "time"

// ProjectResponse 项目响应
type ProjectResponse struct {
	ID           int        `json:"id"`
	Name         string     `json:"name"`
	Code         string     `json:"code"`
	Description  string     `json:"description"`
	ManagerID    *int       `json:"managerId,omitempty"`
	DepartmentID *int       `json:"departmentId,omitempty"`
	StartDate    *time.Time `json:"startDate,omitempty"`
	EndDate      *time.Time `json:"endDate,omitempty"`
	Status       string     `json:"status"`
	TenantID     int        `json:"tenantId"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

// ProjectListResponse 项目列表响应
type ProjectListResponse struct {
	Projects []*ProjectResponse `json:"projects"`
	Total    int                `json:"total"`
}
