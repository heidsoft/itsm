
package dto

import "time"

// ServiceCatalogItemResponse 服务目录项响应
type ServiceCatalogItemResponse struct {
	ID                int                    `json:"id"`
	Name              string                 `json:"name"`
	Description       string                 `json:"description"`
	Details           string                 `json:"details"`
	Category          string                 `json:"category"`
	Icon              string                 `json:"icon"`
	FormSchema        map[string]interface{} `json:"form_schema"`
	SlaID             int                    `json:"sla_id"`
	ApprovalChainID   int                    `json:"approval_chain_id"`
	IsActive          bool                   `json:"is_active"`
	RequiresApproval  bool                   `json:"requires_approval"`
	EstimatedDays     int                    `json:"estimated_days"`
	TenantID          int                    `json:"tenant_id"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

// CreateServiceCatalogItemRequest 创建服务目录项请求
type CreateServiceCatalogItemRequest struct {
	CatalogID         int                    `json:"catalog_id" binding:"required"`
	Name              string                 `json:"name" binding:"required"`
	Description       string                 `json:"description"`
	Details           string                 `json:"details"`
	Category          string                 `json:"category"`
	Icon              string                 `json:"icon"`
	FormSchema        map[string]interface{} `json:"form_schema"`
	SlaID             int                    `json:"sla_id"`
	ApprovalChainID   int                    `json:"approval_chain_id"`
	RequiresApproval  bool                   `json:"requires_approval"`
	EstimatedDays     int                    `json:"estimated_days"`
}

// UpdateServiceCatalogItemRequest 更新服务目录项请求
type UpdateServiceCatalogItemRequest struct {
	Name              *string                 `json:"name"`
	Description       *string                 `json:"description"`
	Details           *string                 `json:"details"`
	Category          *string                 `json:"category"`
	Icon              *string                 `json:"icon"`
	FormSchema        *map[string]interface{} `json:"form_schema"`
	SlaID             *int                    `json:"sla_id"`
	ApprovalChainID   *int                    `json:"approval_chain_id"`
	IsActive          *bool                   `json:"is_active"`
	RequiresApproval  *bool                   `json:"requires_approval"`
	EstimatedDays     *int                    `json:"estimated_days"`
}
