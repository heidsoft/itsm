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
	SourceCIID       int                    `json:"sourceCiId" binding:"required"`
	TargetCIID       int                    `json:"targetCiId" binding:"required"`
	RelationshipType CIRelationshipType     `json:"relationshipType" binding:"required"`
	Strength         RelationshipStrength   `json:"strength"`
	ImpactLevel      ImpactLevel            `json:"impactLevel"`
	Description      string                 `json:"description"`
	Metadata         map[string]interface{} `json:"metadata"`
	IsDiscovered     *bool                  `json:"isDiscovered,omitempty"`
}

// UpdateCIRelationshipRequest 更新CI关系请求
type UpdateCIRelationshipRequest struct {
	RelationshipType *CIRelationshipType     `json:"relationshipType"`
	Strength         *RelationshipStrength   `json:"strength"`
	ImpactLevel      *ImpactLevel            `json:"impactLevel"`
	Description      *string                 `json:"description"`
	Metadata         *map[string]interface{} `json:"metadata"`
	IsActive         *bool                   `json:"isActive"`
}

// CIRelationshipResponse CI关系响应
type CIRelationshipResponse struct {
	ID                   int                    `json:"id"`
	SourceCIID           int                    `json:"sourceCiId"`
	SourceCIName         string                 `json:"sourceCiName"`
	SourceCIType         string                 `json:"sourceCiType"`
	TargetCIID           int                    `json:"targetCiId"`
	TargetCIName         string                 `json:"targetCiName"`
	TargetCIType         string                 `json:"targetCiType"`
	RelationshipType     CIRelationshipType     `json:"relationshipType"`
	RelationshipTypeName string                 `json:"relationshipTypeName"`
	Strength             RelationshipStrength   `json:"strength"`
	ImpactLevel          ImpactLevel            `json:"impactLevel"`
	IsActive             bool                   `json:"isActive"`
	IsDiscovered         bool                   `json:"isDiscovered"`
	Description          string                 `json:"description"`
	Metadata             map[string]interface{} `json:"metadata"`
	CreatedAt            time.Time              `json:"createdAt"`
	UpdatedAt            time.Time              `json:"updatedAt"`
}

// TopologyNode 拓扑图节点
type TopologyNode struct {
	ID          int                    `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	TypeName    string                 `json:"typeName"`
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
	RelationshipType  string `json:"relationshipType"`
	RelationshipLabel string `json:"relationshipLabel"`
	Strength          string `json:"strength"`
	ImpactLevel       string `json:"impactLevel"`
}

// TopologyGraph 拓扑图数据
type TopologyGraph struct {
	Nodes      []TopologyNode `json:"nodes"`
	Edges      []TopologyEdge `json:"edges"`
	RootCIID   int            `json:"rootCiId"`
	Depth      int            `json:"depth"`
	TotalNodes int            `json:"totalNodes"`
	TotalEdges int            `json:"totalEdges"`
}

// ImpactAnalysisRequest 影响分析请求
type ImpactAnalysisRequest struct {
	CIID              int                  `json:"ciId" binding:"required"`
	IncludeUpstream   bool                 `json:"includeUpstream"`
	IncludeDownstream bool                 `json:"includeDownstream"`
	MaxDepth          int                  `json:"maxDepth"`
	RelationshipTypes []CIRelationshipType `json:"relationshipTypes"`
}

// ImpactAnalysisItem 影响分析项
type ImpactAnalysisItem struct {
	CIID             int                `json:"ciId"`
	CIName           string             `json:"ciName"`
	CIT              string             `json:"ciType"`
	Relationship     string             `json:"relationship"`
	RelationshipType CIRelationshipType `json:"relationshipType"`
	ImpactLevel      ImpactLevel        `json:"impactLevel"`
	Distance         int                `json:"distance"`
	Direction        string             `json:"direction"`     // upstream / downstream
	AffectedCount    int                `json:"affectedCount"` // 受影响的工单/事件数量
}

// ImpactAnalysisResponse 影响分析响应
type ImpactAnalysisResponse struct {
	TargetCI             *TopologyNode        `json:"targetCi"`
	UpstreamImpact       []ImpactAnalysisItem `json:"upstreamImpact"`
	DownstreamImpact     []ImpactAnalysisItem `json:"downstreamImpact"`
	CriticalDependencies []ImpactAnalysisItem `json:"criticalDependencies"`
	AffectedTickets      []AffectedTicket     `json:"affectedTickets"`
	AffectedIncidents    []AffectedIncident   `json:"affectedIncidents"`
	RiskLevel            string               `json:"riskLevel"`
	Summary              string               `json:"summary"`
}

// AffectedTicket 受影响的工单
type AffectedTicket struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	Priority  string `json:"priority"`
	CreatedAt string `json:"createdAt"`
}

// AffectedIncident 受影响的事件
type AffectedIncident struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	Severity  string `json:"severity"`
	CreatedAt string `json:"createdAt"`
}

// CIImpactRequest CI影响范围请求（用于变更/工单）
type CIImpactRequest struct {
	CIIDs          []int `json:"ciIds" binding:"required"`
	ShowDownstream bool  `json:"showDownstream"`
	ShowUpstream   bool  `json:"showUpstream"`
	MaxDepth       int   `json:"maxDepth"`
}

// CIImpactResponse CI影响范围响应
type CIImpactResponse struct {
	SelectedCIs    []TopologyNode `json:"selectedCis"`
	ImpactedCIs    []TopologyNode `json:"impactedCis"`
	Relationships  []TopologyEdge `json:"relationships"`
	TotalImpacted  int            `json:"totalImpacted"`
	CriticalPaths  [][]int        `json:"criticalPaths"`
	RiskAssessment string         `json:"riskAssessment"`
}

// GetCIRelationshipsRequest 获取CI关系请求
type GetCIRelationshipsRequest struct {
	CIID             int                `json:"ciId" binding:"required"`
	IncludeOutgoing  bool               `json:"includeOutgoing"`
	IncludeIncoming  bool               `json:"includeIncoming"`
	RelationshipType CIRelationshipType `json:"relationshipType"`
	ActiveOnly       bool               `json:"activeOnly"`
}

// GetCIRelationshipsResponse 获取CI关系响应
type GetCIRelationshipsResponse struct {
	OutgoingRelations []CIRelationshipResponse `json:"outgoingRelations"`
	IncomingRelations []CIRelationshipResponse `json:"incomingRelations"`
	TotalOutgoing     int                      `json:"totalOutgoing"`
	TotalIncoming     int                      `json:"totalIncoming"`
}

// GetCIWithRelationsRequest 获取CI及其所有关系请求
type GetCIWithRelationsRequest struct {
	CIID  int `json:"ciId" binding:"required"`
	Depth int `json:"depth"`
}

// GetCIWithRelationsResponse 获取CI及其所有关系响应
type GetCIWithRelationsResponse struct {
	CI              *ConfigurationItemResponse  `json:"ci"`
	DirectRelations *GetCIRelationshipsResponse `json:"directRelations"`
	TopologyGraph   *TopologyGraph              `json:"topologyGraph"`
}

// BatchCreateCIRelationshipsRequest 批量创建CI关系请求
type BatchCreateCIRelationshipRequest struct {
	Relationships []CreateCIRelationshipRequest `json:"relationships" binding:"required,dive"`
}

// BatchCreateCIRelationshipsResponse 批量创建CI关系响应
type BatchCreateCIRelationshipResponse struct {
	CreatedCount int      `json:"createdCount"`
	FailedCount  int      `json:"failedCount"`
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
