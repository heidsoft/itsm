package dto

import "time"

// CreateCIAttributeDefinitionRequest 创建CI属性定义请求。
type CreateCIAttributeDefinitionRequest struct {
	Name            string   `json:"name" binding:"required,max=100"`
	DisplayName     string   `json:"displayName" binding:"required,max=100"`
	Description     string   `json:"description,omitempty" binding:"omitempty,max=500"`
	Type            string   `json:"type" binding:"required,oneof=string int integer float bool boolean date datetime json enum reference list map"`
	Required        bool     `json:"required,omitempty"`
	Unique          bool     `json:"unique,omitempty"`
	DefaultValue    string   `json:"defaultValue,omitempty"`
	ValidationRules string   `json:"validationRules,omitempty"`
	EnumValues      []string `json:"enumValues,omitempty"`
	ReferenceType   string   `json:"referenceType,omitempty" binding:"omitempty,max=100"`
	DisplayOrder    int      `json:"displayOrder,omitempty" binding:"omitempty,min=0"`
	GroupName       string   `json:"groupName,omitempty" binding:"omitempty,max=100"`
	Placeholder     string   `json:"placeholder,omitempty" binding:"omitempty,max=200"`
	HelpText        string   `json:"helpText,omitempty" binding:"omitempty,max=500"`
	IsSearchable    bool     `json:"isSearchable,omitempty"`
	CiTypeID        int      `json:"ciTypeId" binding:"required"`
}

// UpdateCIAttributeDefinitionRequest 更新CI属性定义请求。
type UpdateCIAttributeDefinitionRequest struct {
	DisplayName     *string   `json:"displayName,omitempty" binding:"omitempty,max=100"`
	Description     *string   `json:"description,omitempty" binding:"omitempty,max=500"`
	Type            *string   `json:"type,omitempty" binding:"omitempty,oneof=string int integer float bool boolean date datetime json enum reference list map"`
	Required        *bool     `json:"required,omitempty"`
	Unique          *bool     `json:"unique,omitempty"`
	DefaultValue    *string   `json:"defaultValue,omitempty"`
	ValidationRules *string   `json:"validationRules,omitempty"`
	EnumValues      *[]string `json:"enumValues,omitempty"`
	ReferenceType   *string   `json:"referenceType,omitempty" binding:"omitempty,max=100"`
	DisplayOrder    *int      `json:"displayOrder,omitempty" binding:"omitempty,min=0"`
	GroupName       *string   `json:"groupName,omitempty" binding:"omitempty,max=100"`
	Placeholder     *string   `json:"placeholder,omitempty" binding:"omitempty,max=200"`
	HelpText        *string   `json:"helpText,omitempty" binding:"omitempty,max=500"`
	IsSearchable    *bool     `json:"isSearchable,omitempty"`
	IsActive        *bool     `json:"isActive,omitempty"`
}

// CITypeListResponse CI类型列表响应。
//
// 信封与平台一致：{items,total,page,pageSize,totalPages}。此前只有 page+size，
// 调用方无法核对总页数，把当前页当成该租户的全部 CI 类型。
type CITypeListResponse struct {
	Items      []*CITypeResponse `json:"items"`
	Total      int               `json:"total"`
	Page       int               `json:"page"`
	PageSize   int               `json:"pageSize"`
	TotalPages int               `json:"totalPages"`
}

// ListCIRequest 获取配置项列表请求。
//
// 单一入口：覆盖原 ListCIRequest + CISearchRequest 合并需求
//   - 关键词模糊（search 字段，匹配：名称/资产标签/序列号/型号/厂商/云资源ID/位置/负责人/归属人）
//   - 枚举精确过滤（status/environment/criticality/ciType）
//   - 云字段（cloudProvider/cloudAccountId/cloudRegion）
//   - 责任人（assignedTo/ownedBy）
//   - 业务编号（ciNumber，AI 多轮定位自然键）
//   - 排序（sortBy/sortOrder）
//   - 时间范围（dateFrom/dateTo）
//   - 标签（tagIds，[]int）
//   - 分页（page/pageSize）
//
// 分页字段只由调用方显式赋值，不参与 query/form 绑定：HTTP 入口的生效页长由
// common.GetPaginationFromQuery 单点决定（缺省 1/20，只采纳 (0,100]），原先的
// `min=1,max=200` 是第二套真相——`size=0` 会被 validator 拦成业务码 1001，而平台语义是
// 「回落默认页长」；上限 200 也和平台 MaxPageSize=100 冲突。
// 非 HTTP 调用方（AI 工具注册表、导出任务）传 0/负数/超大值时由服务层
// common.ValidatePagination 兜住。
type ListCIRequest struct {
	Page           int    `json:"page"`
	PageSize       int    `json:"pageSize"`
	CITypeID       int    `form:"ciTypeId"`
	CIType         string `form:"ciType"`
	Status         string `form:"status"`
	Environment    string `form:"environment"`
	Criticality    string `form:"criticality"`
	CloudProvider  string `form:"cloudProvider"`
	CloudAccountID string `form:"cloudAccountId"`
	CloudRegion    string `form:"cloudRegion"`
	AssignedTo     string `form:"assignedTo"`
	OwnedBy        string `form:"ownedBy"`
	// Search 模糊搜索关键词（合并原 SearchCI.Keyword 宽语义）：
	// 名称 / 资产标签 / 序列号 / 型号 / 厂商 / 云资源ID / 位置 / 负责人 / 归属人
	Search string `form:"search"`
	// CINumber CI 唯一业务编号精确匹配（AI Agent 稳定定位实体的自然键）
	CINumber string `form:"ciNumber"`
	// 排序（P1-1 合并自 CISearchRequest）
	SortBy    string `form:"sortBy" binding:"omitempty,oneof=id name status environment criticality created_at updated_at"`
	SortOrder string `form:"sortOrder" binding:"omitempty,oneof=asc desc"`
	// 时间范围（P1-1 合并自 CISearchFilter.DateFrom/DateTo）
	DateFrom *time.Time `form:"dateFrom" time_format:"2006-01-02T15:04:05Z07:00"`
	DateTo   *time.Time `form:"dateTo" time_format:"2006-01-02T15:04:05Z07:00"`
	// 标签过滤（P1-1 合并自 CISearchFilter.TagIDs）：数组形如 ?tagIds=1&tagIds=2
	TagIDs []int `form:"tagIds"`
	// WithRelations 是否预加载关系（P1-1 与 SearchCI 对齐：预加载出/入边+标签+CITypeRef）
	WithRelations bool `form:"withRelations"`
}

// CIListResponse 配置项列表响应。
type CIListResponse struct {
	Items      []*CIResponse `json:"items"`
	Total      int           `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"pageSize"`
	TotalPages int           `json:"totalPages"`
}

// CIStatsResponse 配置项统计响应。
type CIStatsResponse struct {
	Total                   int            `json:"total"`
	StatusDistribution      map[string]int `json:"statusDistribution"`
	TypeDistribution        map[string]int `json:"typeDistribution"`
	EnvironmentDistribution map[string]int `json:"environmentDistribution"`
	CriticalityDistribution map[string]int `json:"criticalityDistribution"`
}

// CIRelationshipListResponse CI关系列表响应。
type CIRelationshipListResponse struct {
	Items      []*CIRelationshipResponse `json:"items"`
	Total      int                       `json:"total"`
	Page       int                       `json:"page"`
	PageSize   int                       `json:"pageSize"`
	TotalPages int                       `json:"totalPages"`
}

// CreateCITagRequest 创建CI标签请求
type CreateCITagRequest struct {
	Key         string `json:"key" binding:"required,max=50"`
	Value       string `json:"value,omitempty" max:"100"`
	Color       string `json:"color,omitempty" max:"7"`
	Description string `json:"description,omitempty" max:"200"`
}

// UpdateCITagRequest 更新CI标签请求
type UpdateCITagRequest struct {
	Key         *string `json:"key,omitempty" binding:"omitempty,max=50"`
	Value       *string `json:"value,omitempty" binding:"omitempty,max=100"`
	Color       *string `json:"color,omitempty" binding:"omitempty,max=7"`
	Description *string `json:"description,omitempty" binding:"omitempty,max=200"`
}

// CITagResponse CI标签响应
type CITagResponse struct {
	ID          int       `json:"id"`
	Key         string    `json:"key"`
	Value       string    `json:"value,omitempty"`
	Color       string    `json:"color,omitempty"`
	Description string    `json:"description,omitempty"`
	TenantID    int       `json:"tenantId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// CITagListResponse CI标签列表响应
type CITagListResponse struct {
	Items      []*CITagResponse `json:"items"`
	Total      int              `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"pageSize"`
	TotalPages int              `json:"totalPages"`
}

// AddCITagsRequest 给CI添加标签请求
type AddCITagsRequest struct {
	TagIDs []int `json:"tagIds" binding:"required,min=1"`
}

// RemoveCITagsRequest 给CI移除标签请求
type RemoveCITagsRequest struct {
	TagIDs []int `json:"tagIds" binding:"required,min=1"`
}
