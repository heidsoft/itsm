package dto

import (
	"time"

	"itsm-backend/ent"
)

// CreateTicketViewRequest 创建工单视图请求
type CreateTicketViewRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Description *string                `json:"description"`
	Filters     map[string]interface{} `json:"filters"`
	Columns     []string               `json:"columns"`
	SortConfig  map[string]interface{} `json:"sortConfig"`
	GroupConfig map[string]interface{} `json:"groupConfig"`
	IsShared    bool                   `json:"isShared"`
}

// UpdateTicketViewRequest 更新工单视图请求
type UpdateTicketViewRequest struct {
	Name        *string                `json:"name"`
	Description *string                `json:"description"`
	Filters     map[string]interface{} `json:"filters"`
	Columns     []string               `json:"columns"`
	SortConfig  map[string]interface{} `json:"sortConfig"`
	GroupConfig map[string]interface{} `json:"groupConfig"`
	IsShared    *bool                  `json:"isShared"`
}

// ShareTicketViewRequest 共享工单视图请求
type ShareTicketViewRequest struct {
	TeamIDs []int `json:"teamIds"` // 共享给哪些团队
}

// TicketViewResponse 工单视图响应
type TicketViewResponse struct {
	ID          int                    `json:"id"`
	Name        string                 `json:"name"`
	Description *string                `json:"description"`
	Filters     map[string]interface{} `json:"filters"`
	Columns     []string               `json:"columns"`
	SortConfig  map[string]interface{} `json:"sortConfig"`
	GroupConfig map[string]interface{} `json:"groupConfig"`
	IsShared    bool                   `json:"isShared"`
	CreatedBy   int                    `json:"createdBy"`
	Creator     *UserResponse          `json:"creator,omitempty"`
	TenantID    int                    `json:"tenantId"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

// ListTicketViewsResponse 工单视图列表响应
type ListTicketViewsResponse struct {
	Views []*TicketViewResponse `json:"views"`
	Total int                   `json:"total"`
}

// ToTicketViewResponse 转换为工单视图响应
func ToTicketViewResponse(view *ent.TicketView, creator *ent.User) *TicketViewResponse {
	var description *string
	if view.Description != "" {
		description = &view.Description
	}

	response := &TicketViewResponse{
		ID:          view.ID,
		Name:        view.Name,
		Description: description,
		Filters:     view.Filters,
		Columns:     view.Columns,
		SortConfig:  view.SortConfig,
		GroupConfig: view.GroupConfig,
		IsShared:    view.IsShared,
		CreatedBy:   view.CreatedBy,
		TenantID:    view.TenantID,
		CreatedAt:   view.CreatedAt,
		UpdatedAt:   view.UpdatedAt,
	}

	if creator != nil {
		response.Creator = &UserResponse{
			ID:       creator.ID,
			Username: creator.Username,
			Email:    creator.Email,
			Name:     creator.Name,
			Role:     string(creator.Role),
		}
	}

	return response
}

func (r *CreateTicketViewRequest) Normalize() {
}

func (r *UpdateTicketViewRequest) Normalize() {
}

func (r *ShareTicketViewRequest) Normalize() {
}
