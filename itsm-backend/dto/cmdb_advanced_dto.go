package dto

import (
	"time"
)

// CI类型相关DTO
type CreateCITypeRequest struct {
	Name            string                 `json:"name" binding:"required"`
	Description     string                 `json:"description"`
	Icon            string                 `json:"icon"`
	Color           string                 `json:"color"`
	AttributeSchema map[string]interface{} `json:"attribute_schema"`
	IsSystemDefined bool                   `json:"is_system_defined"`
	TenantID        int                    `json:"tenant_id"`
}

type CITypeResponse struct {
	ID              int                    `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Icon            string                 `json:"icon"`
	Color           string                 `json:"color"`
	AttributeSchema map[string]interface{} `json:"attribute_schema"`
	IsSystemDefined bool                   `json:"is_system_defined"`
	TenantID        int                    `json:"tenant_id"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// 生命周期管理DTO
type UpdateCILifecycleStateRequest struct {
	CIID      int       `json:"ci_id" binding:"required"`
	State     string    `json:"state" binding:"required"`
	Reason    string    `json:"reason"`
	ChangedBy string    `json:"changed_by" binding:"required"`
	ChangedAt time.Time `json:"changed_at"`
	TenantID  int       `json:"tenant_id"`
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
	TenantID int                  `json:"tenant_id"`
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
	IsRequired      bool                   `json:"is_required"`
	IsUnique        bool                   `json:"is_unique"`
	DefaultValue    string                 `json:"default_value"`
	ValidationRules map[string]interface{} `json:"validation_rules"`
	EnumValues      []string               `json:"enum_values"`
	ReferenceType   string                 `json:"reference_type"`
	DisplayOrder    int                    `json:"display_order"`
	IsSearchable    bool                   `json:"is_searchable"`
	CITypeID        int                    `json:"ci_type_id" binding:"required"`
}

type CIAttributeDefinitionResponse struct {
	ID              int                    `json:"id"`
	Name            string                 `json:"name"`
	DisplayName     string                 `json:"display_name"`
	Description     string                 `json:"description"`
	DataType        string                 `json:"data_type"`
	IsRequired      bool                   `json:"is_required"`
	IsUnique        bool                   `json:"is_unique"`
	DefaultValue    string                 `json:"default_value"`
	ValidationRules map[string]interface{} `json:"validation_rules"`
	EnumValues      []string               `json:"enum_values"`
	ReferenceType   string                 `json:"reference_type"`
	DisplayOrder    int                    `json:"display_order"`
	IsSearchable    bool                   `json:"is_searchable"`
	IsSystem        bool                   `json:"is_system"`
	IsActive        bool                   `json:"is_active"`
	CITypeID        int                    `json:"ci_type_id"`
	TenantID        int                    `json:"tenant_id"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

type CITypeWithAttributesResponse struct {
	ID                   int                             `json:"id"`
	Name                 string                          `json:"name"`
	DisplayName          string                          `json:"display_name"`
	Description          string                          `json:"description"`
	Category             string                          `json:"category"`
	Icon                 string                          `json:"icon"`
	IsSystem             bool                            `json:"is_system"`
	IsActive             bool                            `json:"is_active"`
	AttributeDefinitions []CIAttributeDefinitionResponse `json:"attribute_definitions"`
	CreatedAt            time.Time                       `json:"created_at"`
	UpdatedAt            time.Time                       `json:"updated_at"`
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
