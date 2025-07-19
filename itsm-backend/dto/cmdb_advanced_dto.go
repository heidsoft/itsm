package dto

import (
	"itsm-backend/ent"
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

// CI关系相关DTO
type CreateCIRelationshipRequest struct {
	SourceCIID         int                    `json:"source_ci_id" binding:"required"`
	TargetCIID         int                    `json:"target_ci_id" binding:"required"`
	RelationshipTypeID int                    `json:"relationship_type_id" binding:"required"`
	Properties         map[string]interface{} `json:"properties"`
	EffectiveFrom      time.Time              `json:"effective_from"`
	EffectiveTo        *time.Time             `json:"effective_to"`
	TenantID           int                    `json:"tenant_id"`
}

type CIRelationshipResponse struct {
	ID                 int                    `json:"id"`
	SourceCIID         int                    `json:"source_ci_id"`
	TargetCIID         int                    `json:"target_ci_id"`
	RelationshipTypeID int                    `json:"relationship_type_id"`
	Properties         map[string]interface{} `json:"properties"`
	EffectiveFrom      time.Time              `json:"effective_from"`
	EffectiveTo        *time.Time             `json:"effective_to"`
	TenantID           int                    `json:"tenant_id"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

// 服务图谱相关DTO
type GetServiceMapRequest struct {
	RootCIID int `json:"root_ci_id" binding:"required"`
	Depth    int `json:"depth" binding:"min=1,max=10"`
	TenantID int `json:"tenant_id"`
}

type ServiceMapNode struct {
	ID       int                    `json:"id"`
	Name     string                 `json:"name"`
	Type     string                 `json:"type"`
	Level    int                    `json:"level"`
	Metadata map[string]interface{} `json:"metadata"`
}

type ServiceMapEdge struct {
	ID       int                    `json:"id"`
	SourceID int                    `json:"source_id"`
	TargetID int                    `json:"target_id"`
	Type     string                 `json:"type"`
	Metadata map[string]interface{} `json:"metadata"`
}

type ServiceMapData struct {
	RootCI   *ServiceMapNode        `json:"root_ci"`
	Nodes    []*ServiceMapNode      `json:"nodes"`
	Edges    []*ServiceMapEdge      `json:"edges"`
	Metadata map[string]interface{} `json:"metadata"`
}

type ServiceMapResponse struct {
	RootCI   *ServiceMapNode        `json:"root_ci"`
	Nodes    []*ServiceMapNode      `json:"nodes"`
	Edges    []*ServiceMapEdge      `json:"edges"`
	Metadata map[string]interface{} `json:"metadata"`
}

// 影响分析相关DTO
type AnalyzeImpactRequest struct {
	SourceCIID int `json:"source_ci_id" binding:"required"`
	Depth      int `json:"depth" binding:"min=1,max=10"`
	TenantID   int `json:"tenant_id"`
}

type ImpactedCI struct {
	CI           *ent.ConfigurationItem `json:"ci"`
	ImpactLevel  int                    `json:"impact_level"`
	Relationship *ent.CIRelationship    `json:"relationship"`
}

type ImpactAnalysisResponse struct {
	SourceCI          *ent.ConfigurationItem `json:"source_ci"`
	UpstreamImpacts   []*ImpactedCI          `json:"upstream_impacts"`
	DownstreamImpacts []*ImpactedCI          `json:"downstream_impacts"`
	ImpactLevel       string                 `json:"impact_level"`
	AnalysisTime      time.Time              `json:"analysis_time"`
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
	IsSystem        bool                   `json:"is_system"` // 添加这个字段
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
