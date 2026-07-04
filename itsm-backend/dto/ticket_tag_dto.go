package dto

import "time"

// TicketTagResponse 标签响应
type TicketTagResponse struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Color       string    `json:"color"`
	Description string    `json:"description"`
	IsActive    bool      `json:"isActive"`
	TenantID    int       `json:"tenantId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// TicketCategoryResponse 分类响应
type TicketCategoryResponse struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	ParentID    *int      `json:"parentId,omitempty"`
	SortOrder   int       `json:"sortOrder"`
	IsActive    bool      `json:"isActive"`
	TenantID    int       `json:"tenantId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// TicketCategoryTreeResponse 分类树响应
type TicketCategoryTreeResponse struct {
	ID          int                           `json:"id"`
	Name        string                        `json:"name"`
	Code        string                        `json:"code"`
	Description string                        `json:"description"`
	SortOrder   int                           `json:"sortOrder"`
	Children    []*TicketCategoryTreeResponse `json:"children,omitempty"`
}
