package dto

// CreateCIAttributeDefinitionRequest 创建CI属性定义请求。
type CreateCIAttributeDefinitionRequest struct {
	Name            string `json:"name" binding:"required,max=100"`
	DisplayName     string `json:"displayName" binding:"required,max=100"`
	Type            string `json:"type" binding:"required,oneof=string int integer float bool boolean date datetime json enum reference list map"`
	Required        bool   `json:"required,omitempty"`
	Unique          bool   `json:"unique,omitempty"`
	DefaultValue    string `json:"defaultValue,omitempty"`
	ValidationRules string `json:"validationRules,omitempty"`
	CiTypeID        int    `json:"ciTypeId" binding:"required"`
}

// UpdateCIAttributeDefinitionRequest 更新CI属性定义请求。
type UpdateCIAttributeDefinitionRequest struct {
	DisplayName     *string `json:"displayName,omitempty" binding:"omitempty,max=100"`
	Type            *string `json:"type,omitempty" binding:"omitempty,oneof=string int integer float bool boolean date datetime json enum reference list map"`
	Required        *bool   `json:"required,omitempty"`
	Unique          *bool   `json:"unique,omitempty"`
	DefaultValue    *string `json:"defaultValue,omitempty"`
	ValidationRules *string `json:"validationRules,omitempty"`
	IsActive        *bool   `json:"isActive,omitempty"`
}

// CITypeListResponse CI类型列表响应。
type CITypeListResponse struct {
	Items []*CITypeResponse `json:"items"`
	Total int               `json:"total"`
	Page  int               `json:"page"`
	Size  int               `json:"size"`
}

// ListCIRequest 获取配置项列表请求。
type ListCIRequest struct {
	Page           int    `form:"page,default=1" binding:"omitempty,min=1"`
	Size           int    `form:"size,default=20" binding:"omitempty,min=1,max=200"`
	CITypeID       int    `form:"ciTypeId"`
	Status         string `form:"status"`
	Environment    string `form:"environment"`
	Criticality    string `form:"criticality"`
	CloudProvider  string `form:"cloudProvider"`
	CloudAccountID string `form:"cloudAccountId"`
	CloudRegion    string `form:"cloudRegion"`
	AssignedTo     string `form:"assignedTo"`
	OwnedBy        string `form:"ownedBy"`
	Search         string `form:"search"`
}

// CIListResponse 配置项列表响应。
type CIListResponse struct {
	Items []*CIResponse `json:"items"`
	Total int           `json:"total"`
	Page  int           `json:"page"`
	Size  int           `json:"size"`
}

// CIStatsResponse 配置项统计响应。
type CIStatsResponse struct {
	TotalCount              int            `json:"totalCount"`
	StatusDistribution      map[string]int `json:"statusDistribution"`
	TypeDistribution        map[string]int `json:"typeDistribution"`
	EnvironmentDistribution map[string]int `json:"environmentDistribution"`
	CriticalityDistribution map[string]int `json:"criticalityDistribution"`
}

// ImpactedCI 受影响的配置项。
type ImpactedCI struct {
	CI    *CIResponse `json:"ci"`
	Depth int         `json:"depth"`
	Path  []int       `json:"path"`
}

// CIImpactAnalysisResponse 配置项影响分析响应。
type CIImpactAnalysisResponse struct {
	SourceCIID    int           `json:"sourceCiId"`
	ImpactedCIs   []*ImpactedCI `json:"impactedCis"`
	TotalImpacted int           `json:"totalImpacted"`
}

// CIRelationshipListResponse CI关系列表响应。
type CIRelationshipListResponse struct {
	Items []*CIRelationshipResponse `json:"items"`
	Total int                       `json:"total"`
	Page  int                       `json:"page"`
	Size  int                       `json:"size"`
}
