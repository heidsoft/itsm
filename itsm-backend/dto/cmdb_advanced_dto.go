package dto

import (
	"time"
)

// CI类型相关DTO
type CreateCITypeRequest struct {
	Name            string `json:"name" binding:"required"`
	Description     string `json:"description"`
	Icon            string `json:"icon"`
	Color           string `json:"color"`
	AttributeSchema string `json:"attributeSchema"`
	IsActive        *bool  `json:"isActive"`
	TenantID        int    `json:"tenantId"`
}

type UpdateCITypeRequest struct {
	Name            string `json:"name" binding:"required"`
	Description     string `json:"description"`
	Icon            string `json:"icon"`
	Color           string `json:"color"`
	AttributeSchema string `json:"attributeSchema"`
	IsActive        *bool  `json:"isActive"`
}

type CITypeResponse struct {
	ID              int       `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Icon            string    `json:"icon"`
	Color           string    `json:"color"`
	AttributeSchema string    `json:"attributeSchema"`
	IsActive        bool      `json:"isActive"`
	TenantID        int       `json:"tenantId"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// 生命周期管理DTO
type UpdateCILifecycleStateRequest struct {
	CIID      int       `json:"ci_id" binding:"required"`
	State     string    `json:"state" binding:"required"`
	Reason    string    `json:"reason"`
	ChangedBy string    `json:"changed_by" binding:"required"`
	ChangedAt time.Time `json:"changed_at"`
	TenantID  int       `json:"tenantId"`
}

// 批量导入DTO
type BatchImportCIData struct {
	Name       string                 `json:"name"`
	CITypeID   int                    `json:"ci_type_id"`
	Attributes map[string]interface{} `json:"attributes"`
	Status     string                 `json:"status"`
}

type BatchImportCIsRequest struct {
	CIs      []*BatchImportCIData `json:"cis" binding:"required"`
	TenantID int                  `json:"tenantId"`
}

type BatchImportCIsResponse struct {
	TotalCount   int      `json:"total_count"`
	SuccessCount int      `json:"success_count"`
	FailureCount int      `json:"failure_count"`
	Errors       []string `json:"errors"`
}

// CI属性定义相关DTO
type CIAttributeDefinitionRequest struct {
	Name            string                 `json:"name" binding:"required"`
	DisplayName     string                 `json:"display_name" binding:"required"`
	Description     string                 `json:"description"`
	DataType        string                 `json:"data_type" binding:"required,oneof=string integer float boolean date datetime json enum reference"`
	IsRequired      bool                   `json:"isRequired"`
	IsUnique        bool                   `json:"isUnique"`
	DefaultValue    string                 `json:"defaultValue"`
	ValidationRules map[string]interface{} `json:"validationRules"`
	EnumValues      []string               `json:"enumValues"`
	ReferenceType   string                 `json:"referenceType"`
	DisplayOrder    int                    `json:"displayOrder"`
	IsSearchable    bool                   `json:"isSearchable"`
	CITypeID        int                    `json:"ci_type_id" binding:"required"`
}

type CIAttributeDefinitionResponse struct {
	ID              int                    `json:"id"`
	Name            string                 `json:"name"`
	DisplayName     string                 `json:"displayName"`
	Description     string                 `json:"description"`
	DataType        string                 `json:"dataType"`
	IsRequired      bool                   `json:"isRequired"`
	IsUnique        bool                   `json:"isUnique"`
	DefaultValue    string                 `json:"defaultValue"`
	ValidationRules map[string]interface{} `json:"validationRules"`
	EnumValues      []string               `json:"enumValues"`
	ReferenceType   string                 `json:"referenceType"`
	DisplayOrder    int                    `json:"displayOrder"`
	IsSearchable    bool                   `json:"isSearchable"`
	IsSystem        bool                   `json:"isSystem"`
	IsActive        bool                   `json:"isActive"`
	CITypeID        int                    `json:"ci_type_id"`
	TenantID        int                    `json:"tenantId"`
	CreatedAt       time.Time              `json:"createdAt"`
	UpdatedAt       time.Time              `json:"updatedAt"`
}

type CITypeWithAttributesResponse struct {
	ID                   int                             `json:"id"`
	Name                 string                          `json:"name"`
	DisplayName          string                          `json:"displayName"`
	Description          string                          `json:"description"`
	Category             string                          `json:"category"`
	Icon                 string                          `json:"icon"`
	IsSystem             bool                            `json:"isSystem"`
	IsActive             bool                            `json:"isActive"`
	AttributeDefinitions []CIAttributeDefinitionResponse `json:"attribute_definitions"`
	CreatedAt            time.Time                       `json:"createdAt"`
	UpdatedAt            time.Time                       `json:"updatedAt"`
}

type ValidateCIAttributesRequest struct {
	CITypeID   int                    `json:"ci_type_id" binding:"required"`
	Attributes map[string]interface{} `json:"attributes" binding:"required"`
}

type ValidateCIAttributesResponse struct {
	IsValid              bool                   `json:"is_valid"`
	Errors               map[string]string      `json:"errors"`
	Warnings             map[string]string      `json:"warnings"`
	NormalizedAttributes map[string]interface{} `json:"normalized_attributes"`
}

type CIAttributeSearchRequest struct {
	CITypeID   int                    `json:"ci_type_id,omitempty"`
	Attributes map[string]interface{} `json:"attributes" binding:"required"`
	Limit      int                    `json:"limit,omitempty"`
	Offset     int                    `json:"offset,omitempty"`
}

// BatchCreateCIRequest 批量创建CI请求
type BatchCreateCIRequest struct {
	Items []CreateCIRequest `json:"items" binding:"required,dive,min=1,max=100"`
}

// BatchUpdateCIRequest 批量更新CI请求
type BatchUpdateCIRequest struct {
	IDs     []int            `json:"ids" binding:"required,min=1"`
	Updates *UpdateCIRequest `json:"updates" binding:"required"`
}

// BatchDeleteCIRequest 批量删除CI请求
type BatchDeleteCIRequest struct {
	IDs []int `json:"ids" binding:"required,min=1"`
}

// BatchOperationResponse 批量操作响应
type BatchOperationResponse struct {
	SuccessCount int      `json:"successCount"`
	FailedCount  int      `json:"failedCount"`
	FailedIDs    []int    `json:"failedIds,omitempty"`
	Errors       []string `json:"errors,omitempty"`
}

// BatchUpdateLifecycleRequest 批量更新CI生命周期状态请求
type BatchUpdateLifecycleRequest struct {
	IDs    []int  `json:"ids" binding:"required,min=1"`
	Status string `json:"status" binding:"required,oneof=draft online maintenance offline scrapped"`
	Remark string `json:"remark,omitempty" max:"500"`
}

// ListResponse 通用列表响应
type ListResponse[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
	Page  int `json:"page"`
	Size  int `json:"size"`
}
