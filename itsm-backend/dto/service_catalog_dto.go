package dto

import "time"

// ServiceCatalogItemResponse 服务目录项响应
type ServiceCatalogItemResponse struct {
	ID               int                    `json:"id"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	Details          string                 `json:"details"`
	Category         string                 `json:"category"`
	Icon             string                 `json:"icon"`
	FormSchema       map[string]interface{} `json:"formSchema"`
	SlaID            int                    `json:"slaId"`
	ApprovalChainID  int                    `json:"approvalChainId"`
	IsActive         bool                   `json:"isActive"`
	RequiresApproval bool                   `json:"requiresApproval"`
	EstimatedDays    int                    `json:"estimatedDays"`
	TenantID         int                    `json:"tenantId"`
	CreatedAt        time.Time              `json:"createdAt"`
	UpdatedAt        time.Time              `json:"updatedAt"`
}

// CreateServiceCatalogItemRequest 创建服务目录项请求
type CreateServiceCatalogItemRequest struct {
	CatalogID        int                    `json:"catalogId" binding:"required"`
	Name             string                 `json:"name" binding:"required"`
	Description      string                 `json:"description"`
	Details          string                 `json:"details"`
	Category         string                 `json:"category"`
	Icon             string                 `json:"icon"`
	FormSchema       map[string]interface{} `json:"formSchema"`
	SlaID            int                    `json:"slaId"`
	ApprovalChainID  int                    `json:"approvalChainId"`
	RequiresApproval bool                   `json:"requiresApproval"`
	EstimatedDays    int                    `json:"estimatedDays"`
}

// UpdateServiceCatalogItemRequest 更新服务目录项请求
type UpdateServiceCatalogItemRequest struct {
	Name             *string                 `json:"name"`
	Description      *string                 `json:"description"`
	Details          *string                 `json:"details"`
	Category         *string                 `json:"category"`
	Icon             *string                 `json:"icon"`
	FormSchema       *map[string]interface{} `json:"formSchema"`
	SlaID            *int                    `json:"slaId"`
	ApprovalChainID  *int                    `json:"approvalChainId"`
	IsActive         *bool                   `json:"isActive"`
	RequiresApproval *bool                   `json:"requiresApproval"`
	EstimatedDays    *int                    `json:"estimatedDays"`
}
