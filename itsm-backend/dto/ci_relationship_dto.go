package dto

import "time"

// CIRelationshipType CI关系类型
type CIRelationshipType string

const (
	// DependsOn 依赖关系
	DependsOn CIRelationshipType = "depends_on"
	// Hosts 托管关系
	Hosts CIRelationshipType = "hosts"
	// HostedOn 所属关系
	HostedOn CIRelationshipType = "hosted_on"
	// ConnectsTo 连接关系
	ConnectsTo CIRelationshipType = "connects_to"
	// RunsOn 运行关系
	RunsOn CIRelationshipType = "runs_on"
	// Contains 包含关系
	Contains CIRelationshipType = "contains"
	// PartOf 组成部分
	PartOf CIRelationshipType = "part_of"
	// Impacts 影响关系
	Impacts CIRelationshipType = "impacts"
	// ImpactedBy 受影响于
	ImpactedBy CIRelationshipType = "impacted_by"
	// Owns 拥有关系
	Owns CIRelationshipType = "owns"
	// OwnedBy 所属关系
	OwnedBy CIRelationshipType = "owned_by"
	// Uses 使用关系
	Uses CIRelationshipType = "uses"
	// UsedBy 被使用关系
	UsedBy CIRelationshipType = "used_by"
)

// RelationshipStrength 关系强度
type RelationshipStrength string

const (
	StrengthCritical RelationshipStrength = "critical"
	StrengthHigh     RelationshipStrength = "high"
	StrengthMedium   RelationshipStrength = "medium"
	StrengthLow      RelationshipStrength = "low"
)

// ImpactLevel 影响程度
type ImpactLevel string

const (
	ImpactCritical ImpactLevel = "critical"
	ImpactHigh     ImpactLevel = "high"
	ImpactMedium   ImpactLevel = "medium"
	ImpactLow      ImpactLevel = "low"
)

// CreateCIRelationshipRequest 创建CI关系请求
type CreateCIRelationshipRequest struct {
	SourceCIID       int                    `json:"source_ci_id" binding:"required"`
	TargetCIID       int                    `json:"target_ci_id" binding:"required"`
	RelationshipType CIRelationshipType     `json:"relationship_type" binding:"required"`
	Strength         RelationshipStrength   `json:"strength"`
	ImpactLevel      ImpactLevel            `json:"impact_level"`
	Description      string                 `json:"description"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// UpdateCIRelationshipRequest 更新CI关系请求
type UpdateCIRelationshipRequest struct {
	RelationshipType *CIRelationshipType     `json:"relationship_type"`
	Strength         *RelationshipStrength   `json:"strength"`
	ImpactLevel      *ImpactLevel            `json:"impact_level"`
	Description      *string                 `json:"description"`
	Metadata         *map[string]interface{} `json:"metadata"`
	IsActive         *bool                   `json:"is_active"`
}

// CIRelationshipResponse CI关系响应
type CIRelationshipResponse struct {
	ID                   int                    `json:"id"`
	SourceCIID           int                    `json:"source_ci_id"`
	SourceCIName         string                 `json:"source_ci_name"`
	SourceCIType         string                 `json:"source_ci_type"`
	TargetCIID           int                    `json:"target_ci_id"`
	TargetCIName         string                 `json:"target_ci_name"`
	TargetCIType         string                 `json:"target_ci_type"`
	RelationshipType     CIRelationshipType     `json:"relationship_type"`
	RelationshipTypeName string                 `json:"relationship_type_name"`
	Strength             RelationshipStrength   `json:"strength"`
	ImpactLevel          ImpactLevel            `json:"impact_level"`
	IsActive             bool                   `json:"is_active"`
	IsDiscovered         bool                   `json:"is_discovered"`
	Description          string                 `json:"description"`
	Metadata             map[string]interface{} `json:"metadata"`
	CreatedAt            time.Time              `json:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at"`
}

// TopologyNode 拓扑图节点
type TopologyNode struct {
	ID          int                    `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	TypeName    string                 `json:"type_name"`
	Status      string                 `json:"status"`
	Criticality string                 `json:"criticality"`
	Icon        string                 `json:"icon"`
	Attributes  map[string]interface{} `json:"attributes"`
}

// TopologyEdge 拓扑图边
type TopologyEdge struct {
	ID                int    `json:"id"`
	Source            int    `json:"source"`
	Target            int    `json:"target"`
	RelationshipType  string `json:"relationship_type"`
	RelationshipLabel string `json:"relationship_label"`
	Strength          string `json:"strength"`
	ImpactLevel       string `json:"impact_level"`
}

// TopologyGraph 拓扑图数据
type TopologyGraph struct {
	Nodes      []TopologyNode `json:"nodes"`
	Edges      []TopologyEdge `json:"edges"`
	RootCIID   int            `json:"root_ci_id"`
	Depth      int            `json:"depth"`
	TotalNodes int            `json:"total_nodes"`
	TotalEdges int            `json:"total_edges"`
}

// ImpactAnalysisRequest 影响分析请求
type ImpactAnalysisRequest struct {
	CIID              int                  `json:"ci_id" binding:"required"`
	IncludeUpstream   bool                 `json:"include_upstream"`
	IncludeDownstream bool                 `json:"include_downstream"`
	MaxDepth          int                  `json:"max_depth"`
	RelationshipTypes []CIRelationshipType `json:"relationship_types"`
}

// ImpactAnalysisItem 影响分析项
type ImpactAnalysisItem struct {
	CIID             int                `json:"ci_id"`
	CIName           string             `json:"ci_name"`
	CIT              string             `json:"ci_type"`
	Relationship     string             `json:"relationship"`
	RelationshipType CIRelationshipType `json:"relationship_type"`
	ImpactLevel      ImpactLevel        `json:"impact_level"`
	Distance         int                `json:"distance"`
	Direction        string             `json:"direction"`      // upstream / downstream
	AffectedCount    int                `json:"affected_count"` // 受影响的工单/事件数量
}

// ImpactAnalysisResponse 影响分析响应
type ImpactAnalysisResponse struct {
	TargetCI             *TopologyNode        `json:"target_ci"`
	UpstreamImpact       []ImpactAnalysisItem `json:"upstream_impact"`
	DownstreamImpact     []ImpactAnalysisItem `json:"downstream_impact"`
	CriticalDependencies []ImpactAnalysisItem `json:"critical_dependencies"`
	AffectedTickets      []AffectedTicket     `json:"affected_tickets"`
	AffectedIncidents    []AffectedIncident   `json:"affected_incidents"`
	RiskLevel            string               `json:"risk_level"`
	Summary              string               `json:"summary"`
}

// AffectedTicket 受影响的工单
type AffectedTicket struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	Priority  string `json:"priority"`
	CreatedAt string `json:"created_at"`
}

// AffectedIncident 受影响的事件
type AffectedIncident struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	Severity  string `json:"severity"`
	CreatedAt string `json:"created_at"`
}

// CIImpactRequest CI影响范围请求（用于变更/工单）
type CIImpactRequest struct {
	CIIDs          []int `json:"ci_ids" binding:"required"`
	ShowDownstream bool  `json:"show_downstream"`
	ShowUpstream   bool  `json:"show_upstream"`
	MaxDepth       int   `json:"max_depth"`
}

// CIImpactResponse CI影响范围响应
type CIImpactResponse struct {
	SelectedCIs    []TopologyNode `json:"selected_cis"`
	ImpactedCIs    []TopologyNode `json:"impacted_cis"`
	Relationships  []TopologyEdge `json:"relationships"`
	TotalImpacted  int            `json:"total_impacted"`
	CriticalPaths  [][]int        `json:"critical_paths"`
	RiskAssessment string         `json:"risk_assessment"`
}

// GetCIRelationshipsRequest 获取CI关系请求
type GetCIRelationshipsRequest struct {
	CIID             int                `json:"ci_id" binding:"required"`
	IncludeOutgoing  bool               `json:"include_outgoing"`
	IncludeIncoming  bool               `json:"include_incoming"`
	RelationshipType CIRelationshipType `json:"relationship_type"`
	ActiveOnly       bool               `json:"active_only"`
}

// GetCIRelationshipsResponse 获取CI关系响应
type GetCIRelationshipsResponse struct {
	OutgoingRelations []CIRelationshipResponse `json:"outgoing_relations"`
	IncomingRelations []CIRelationshipResponse `json:"incoming_relations"`
	TotalOutgoing     int                      `json:"total_outgoing"`
	TotalIncoming     int                      `json:"total_incoming"`
}

// GetCIWithRelationsRequest 获取CI及其所有关系请求
type GetCIWithRelationsRequest struct {
	CIID  int `json:"ci_id" binding:"required"`
	Depth int `json:"depth"`
}

// GetCIWithRelationsResponse 获取CI及其所有关系响应
type GetCIWithRelationsResponse struct {
	CI              *ConfigurationItemResponse  `json:"ci"`
	DirectRelations *GetCIRelationshipsResponse `json:"direct_relations"`
	TopologyGraph   *TopologyGraph              `json:"topology_graph"`
}

// BatchCreateCIRelationshipsRequest 批量创建CI关系请求
type BatchCreateCIRelationshipRequest struct {
	Relationships []CreateCIRelationshipRequest `json:"relationships" binding:"required,dive"`
}

// BatchCreateCIRelationshipsResponse 批量创建CI关系响应
type BatchCreateCIRelationshipResponse struct {
	CreatedCount int      `json:"created_count"`
	FailedCount  int      `json:"failed_count"`
	Errors       []string `json:"errors"`
}

// RelationshipTypeInfo 关系类型信息
type RelationshipTypeInfo struct {
	Type        CIRelationshipType `json:"type"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Direction   string             `json:"direction"` // uni-directional / bi-directional
	Icon        string             `json:"icon"`
}

// GetRelationshipTypesResponse 获取所有关系类型响应
type GetRelationshipTypesResponse struct {
	Types []RelationshipTypeInfo `json:"types"`
}
