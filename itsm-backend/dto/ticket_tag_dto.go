package dto

import "time"

// TicketTagResponse 标签响应
type TicketTagResponse struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Color       string    `json:"color"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	TenantID    int       `json:"tenant_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TicketCategoryResponse 分类响应
type TicketCategoryResponse struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	ParentID    *int     `json:"parent_id,omitempty"`
	SortOrder   int      `json:"sort_order"`
	IsActive    bool      `json:"is_active"`
	TenantID    int       `json:"tenant_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TicketCategoryTreeResponse 分类树响应
type TicketCategoryTreeResponse struct {
	ID          int                       `json:"id"`
	Name        string                    `json:"name"`
	Code        string                    `json:"code"`
	Description string                    `json:"description"`
	SortOrder   int                       `json:"sort_order"`
	Children    []*TicketCategoryTreeResponse `json:"children,omitempty"`
}
