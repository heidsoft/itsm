package dto

import "time"

// CI类型相关DTO
type CreateCITypeRequest struct {
	Name            string                 `json:"name" binding:"required"`
	DisplayName     string                 `json:"display_name" binding:"required"`
	Description     string                 `json:"description"`
	Category        string                 `json:"category" binding:"required"`
	Icon            string                 `json:"icon"`
	AttributeSchema map[string]interface{} `json:"attribute_schema"`
	ValidationRules map[string]interface{} `json:"validation_rules"`
	TenantID        int                    `json:"tenant_id"`
}

type CITypeResponse struct {
	ID              int                    `json:"id"`
	Name            string                 `json:"name"`
	DisplayName     string                 `json:"display_name"`
	Description     string                 `json:"description"`
	Category        string                 `json:"category"`
	Icon            string                 `json:"icon"`
	AttributeSchema map[string]interface{} `json:"attribute_schema"`
	ValidationRules map[string]interface{} `json:"validation_rules"`
	IsSystem        bool                   `json:"is_system"`
	IsActive        bool                   `json:"is_active"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// CI关系相关DTO
type CreateCIRelationshipRequest struct {
	SourceCIID         int                    `json:"source_ci_id" binding:"required"`
	TargetCIID         int                    `json:"target_ci_id" binding:"required"`
	RelationshipTypeID int                    `json:"relationship_type_id" binding:"required"`
	Properties         map[string]interface{} `json:"properties"`
	TenantID           int                    `json:"tenant_id"`
}

type CIRelationshipResponse struct {
	ID                 int                    `json:"id"`
	SourceCIID         int                    `json:"source_ci_id"`
	TargetCIID         int                    `json:"target_ci_id"`
	RelationshipTypeID int                    `json:"relationship_type_id"`
	Properties         map[string]interface{} `json:"properties"`
	Status             string                 `json:"status"`
	EffectiveFrom      time.Time              `json:"effective_from"`
	EffectiveTo        *time.Time             `json:"effective_to"`
	CreatedAt          time.Time              `json:"created_at"`
}

// 服务图谱相关DTO
type ServiceMapNode struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Status   string `json:"status"`
	Category string `json:"category"`
}

type ServiceMapEdge struct {
	SourceID         int                    `json:"source_id"`
	TargetID         int                    `json:"target_id"`
	RelationshipType string                 `json:"relationship_type"`
	Properties       map[string]interface{} `json:"properties"`
}

type ServiceMapResponse struct {
	Nodes []*ServiceMapNode `json:"nodes"`
	Edges []*ServiceMapEdge `json:"edges"`
}

// 影响分析相关DTO
type AffectedCI struct {
	CIID        int    `json:"ci_id"`
	CIName      string `json:"ci_name"`
	CIType      string `json:"ci_type"`
	ImpactLevel string `json:"impact_level"`
	Direction   string `json:"direction"`
}

type ImpactAnalysisResponse struct {
	SourceCIID  int           `json:"source_ci_id"`
	ChangeType  string        `json:"change_type"`
	AffectedCIs []*AffectedCI `json:"affected_cis"`
	TotalCount  int           `json:"total_count"`
	AnalyzedAt  time.Time     `json:"analyzed_at"`
}

// 生命周期管理DTO
type UpdateCILifecycleRequest struct {
	CIID      int                    `json:"ci_id" binding:"required"`
	NewState  string                 `json:"new_state" binding:"required"`
	SubState  string                 `json:"sub_state"`
	Reason    string                 `json:"reason"`
	ChangedBy string                 `json:"changed_by" binding:"required"`
	Metadata  map[string]interface{} `json:"metadata"`
	TenantID  int                    `json:"tenant_id"`
}

// 批量导入DTO
type BatchImportCIData struct {
	Name            string                 `json:"name"`
	DisplayName     string                 `json:"display_name"`
	Description     string                 `json:"description"`
	CITypeID        int                    `json:"ci_type_id"`
	SerialNumber    string                 `json:"serial_number"`
	AssetTag        string                 `json:"asset_tag"`
	Status          string                 `json:"status"`
	LifecycleState  string                 `json:"lifecycle_state"`
	BusinessService string                 `json:"business_service"`
	Owner           string                 `json:"owner"`
	Environment     string                 `json:"environment"`
	Location        string                 `json:"location"`
	Attributes      map[string]interface{} `json:"attributes"`
}

type BatchImportCIRequest struct {
	BatchID  string               `json:"batch_id"`
	TenantID int                  `json:"tenant_id"`
	CIs      []*BatchImportCIData `json:"cis"`
}

type BatchImportResponse struct {
	Total     int      `json:"total"`
	Succeeded int      `json:"succeeded"`
	Failed    int      `json:"failed"`
	Errors    []string `json:"errors"`
}
