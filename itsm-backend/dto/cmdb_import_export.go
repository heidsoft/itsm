package dto

import "time"

// ImportCIRequest 导入CI请求
type ImportCIRequest struct {
	FileURL    string `json:"fileUrl" binding:"required,url"`                           // 导入文件地址
	FileType   string `json:"fileType" binding:"required,oneof=xlsx csv"`               // 文件类型：xlsx/csv
	UpdateMode string `json:"updateMode" binding:"required,oneof=skip overwrite merge"` // 更新模式：skip=跳过已存在 overwrite=覆盖 merge=合并更新
	SheetName  string `json:"sheetName,omitempty"`                                      // Excel的Sheet名称，默认第一个
}

// ImportCIResult 导入结果
type ImportCIResult struct {
	TaskID       string          `json:"taskId"`           // 导入任务ID
	Status       string          `json:"status"`           // 任务状态：pending processing completed failed
	TotalCount   int             `json:"totalCount"`       // 总行数
	SuccessCount int             `json:"successCount"`     // 成功行数
	FailedCount  int             `json:"failedCount"`      // 失败行数
	Errors       []CIImportError `json:"errors,omitempty"` // 错误详情
	CreatedAt    time.Time       `json:"createdAt"`
	CompletedAt  *time.Time      `json:"completedAt,omitempty"`
}

// CIImportError 导入错误详情
type CIImportError struct {
	RowNumber int    `json:"rowNumber"` // 行号
	FieldName string `json:"fieldName"` // 字段名
	Message   string `json:"message"`   // 错误信息
}

// ExportCIRequest 导出CI请求
type ExportCIRequest struct {
	Filters      *CISearchFilter `json:"filters,omitempty"`                            // 导出过滤条件
	ExportType   string          `json:"exportType" binding:"required,oneof=xlsx csv"` // 导出类型：xlsx/csv
	ExportFields []string        `json:"exportFields,omitempty"`                       // 指定导出字段，默认导出所有
}

// ExportCIResult 导出结果
type ExportCIResult struct {
	TaskID      string     `json:"taskId"`             // 导出任务ID
	Status      string     `json:"status"`             // 任务状态：pending processing completed failed
	FileURL     string     `json:"fileUrl,omitempty"`  // 导出文件下载地址
	TotalCount  int        `json:"totalCount"`         // 导出的CI数量
	FileSize    int64      `json:"fileSize,omitempty"` // 文件大小（字节）
	CreatedAt   time.Time  `json:"createdAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	ExpiresAt   time.Time  `json:"expiresAt"` // 下载链接过期时间
}

// CISearchFilter CI搜索过滤条件
type CISearchFilter struct {
	Keyword         string                 `json:"keyword,omitempty"`         // 全文搜索关键词
	CITypeID        int                    `json:"ciTypeId,omitempty"`        // CI类型ID
	Status          string                 `json:"status,omitempty"`          // 状态
	Environment     string                 `json:"environment,omitempty"`     // 环境
	Criticality     string                 `json:"criticality,omitempty"`     // 重要级别
	AssetTag        string                 `json:"assetTag,omitempty"`        // 资产标签
	SerialNumber    string                 `json:"serialNumber,omitempty"`    // 序列号
	Vendor          string                 `json:"vendor,omitempty"`          // 厂商
	Location        string                 `json:"location,omitempty"`        // 位置
	AssignedTo      string                 `json:"assignedTo,omitempty"`      // 负责人
	OwnedBy         string                 `json:"ownedBy,omitempty"`         // 归属人
	TagIDs          []int                  `json:"tagIds,omitempty"`          // 标签ID列表
	Attributes      map[string]interface{} `json:"attributes,omitempty"`      // 自定义属性过滤
	DateFrom        *time.Time             `json:"dateFrom,omitempty"`        // 创建时间起始
	DateTo          *time.Time             `json:"dateTo,omitempty"`          // 创建时间截止
	CloudProvider   string                 `json:"cloudProvider,omitempty"`   // 云厂商
	CloudRegion     string                 `json:"cloudRegion,omitempty"`     // 云区域
	CloudResourceID string                 `json:"cloudResourceId,omitempty"` // 云资源ID
}

// CISearchRequest CI搜索请求
type CISearchRequest struct {
	Filters   CISearchFilter `json:"filters"`                                                // 搜索过滤条件
	Page      int            `json:"page" binding:"min=1"`                                   // 页码
	PageSize  int            `json:"pageSize" binding:"min=1,max=100"`                       // 每页数量
	SortBy    string         `json:"sortBy,omitempty"`                                       // 排序字段
	SortOrder string         `json:"sortOrder,omitempty" binding:"omitempty,oneof=asc desc"` // 排序方向
}

// CISavedView 保存的搜索视图
type CISavedView struct {
	ID          int            `json:"id"`
	Name        string         `json:"name"`                  // 视图名称
	Description string         `json:"description,omitempty"` // 视图描述
	Filters     CISearchFilter `json:"filters"`               // 保存的过滤条件
	SortBy      string         `json:"sortBy,omitempty"`
	SortOrder   string         `json:"sortOrder,omitempty"`
	IsPublic    bool           `json:"isPublic"`              // 是否公开
	CreatorID   int            `json:"creatorId"`             // 创建人ID
	CreatorName string         `json:"creatorName,omitempty"` // 创建人名称
	TenantID    int            `json:"tenantId"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}

// CreateCISavedViewRequest 创建保存视图请求
type CreateCISavedViewRequest struct {
	Name        string         `json:"name" binding:"required,max=100"`
	Description string         `json:"description,omitempty" max:"500"`
	Filters     CISearchFilter `json:"filters" binding:"required"`
	SortBy      string         `json:"sortBy,omitempty"`
	SortOrder   string         `json:"sortOrder,omitempty"`
	IsPublic    bool           `json:"isPublic"`
}

// UpdateCISavedViewRequest 更新保存视图请求
type UpdateCISavedViewRequest struct {
	Name        *string         `json:"name,omitempty" binding:"omitempty,max=100"`
	Description *string         `json:"description,omitempty" binding:"omitempty,max=500"`
	Filters     *CISearchFilter `json:"filters,omitempty"`
	SortBy      *string         `json:"sortBy,omitempty"`
	SortOrder   *string         `json:"sortOrder,omitempty"`
	IsPublic    *bool           `json:"isPublic,omitempty"`
}
