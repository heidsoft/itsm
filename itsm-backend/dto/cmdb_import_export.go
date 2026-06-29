package dto

import "time"

// ImportCIRequest 导入CI请求
type ImportCIRequest struct {
	FileURL    string `json:"file_url" binding:"required,url"`                           // 导入文件地址
	FileType   string `json:"file_type" binding:"required,oneof=xlsx csv"`               // 文件类型：xlsx/csv
	UpdateMode string `json:"update_mode" binding:"required,oneof=skip overwrite merge"` // 更新模式：skip=跳过已存在 overwrite=覆盖 merge=合并更新
	SheetName  string `json:"sheet_name,omitempty"`                                      // Excel的Sheet名称，默认第一个
}

// ImportCIResult 导入结果
type ImportCIResult struct {
	TaskID       string          `json:"task_id"`          // 导入任务ID
	Status       string          `json:"status"`           // 任务状态：pending processing completed failed
	TotalCount   int             `json:"total_count"`      // 总行数
	SuccessCount int             `json:"success_count"`    // 成功行数
	FailedCount  int             `json:"failed_count"`     // 失败行数
	Errors       []CIImportError `json:"errors,omitempty"` // 错误详情
	CreatedAt    time.Time       `json:"created_at"`
	CompletedAt  *time.Time      `json:"completed_at,omitempty"`
}

// CIImportError 导入错误详情
type CIImportError struct {
	RowNumber int    `json:"row_number"` // 行号
	FieldName string `json:"field_name"` // 字段名
	Message   string `json:"message"`    // 错误信息
}

// ExportCIRequest 导出CI请求
type ExportCIRequest struct {
	Filters      *CISearchFilter `json:"filters,omitempty"`                             // 导出过滤条件
	ExportType   string          `json:"export_type" binding:"required,oneof=xlsx csv"` // 导出类型：xlsx/csv
	ExportFields []string        `json:"export_fields,omitempty"`                       // 指定导出字段，默认导出所有
}

// ExportCIResult 导出结果
type ExportCIResult struct {
	TaskID      string     `json:"task_id"`             // 导出任务ID
	Status      string     `json:"status"`              // 任务状态：pending processing completed failed
	FileURL     string     `json:"file_url,omitempty"`  // 导出文件下载地址
	TotalCount  int        `json:"total_count"`         // 导出的CI数量
	FileSize    int64      `json:"file_size,omitempty"` // 文件大小（字节）
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	ExpiresAt   time.Time  `json:"expires_at"` // 下载链接过期时间
}

// CISearchFilter CI搜索过滤条件
type CISearchFilter struct {
	Keyword         string                 `json:"keyword,omitempty"`           // 全文搜索关键词
	CITypeID        int                    `json:"ci_type_id,omitempty"`        // CI类型ID
	Status          string                 `json:"status,omitempty"`            // 状态
	Environment     string                 `json:"environment,omitempty"`       // 环境
	Criticality     string                 `json:"criticality,omitempty"`       // 重要级别
	AssetTag        string                 `json:"asset_tag,omitempty"`         // 资产标签
	SerialNumber    string                 `json:"serial_number,omitempty"`     // 序列号
	Vendor          string                 `json:"vendor,omitempty"`            // 厂商
	Location        string                 `json:"location,omitempty"`          // 位置
	AssignedTo      string                 `json:"assigned_to,omitempty"`       // 负责人
	OwnedBy         string                 `json:"owned_by,omitempty"`          // 归属人
	TagIDs          []int                  `json:"tag_ids,omitempty"`           // 标签ID列表
	Attributes      map[string]interface{} `json:"attributes,omitempty"`        // 自定义属性过滤
	DateFrom        *time.Time             `json:"date_from,omitempty"`         // 创建时间起始
	DateTo          *time.Time             `json:"date_to,omitempty"`           // 创建时间截止
	CloudProvider   string                 `json:"cloud_provider,omitempty"`    // 云厂商
	CloudRegion     string                 `json:"cloud_region,omitempty"`      // 云区域
	CloudResourceID string                 `json:"cloud_resource_id,omitempty"` // 云资源ID
}

// CISearchRequest CI搜索请求
type CISearchRequest struct {
	Filters   CISearchFilter `json:"filters"`                                                 // 搜索过滤条件
	Page      int            `json:"page" binding:"min=1"`                                    // 页码
	PageSize  int            `json:"page_size" binding:"min=1,max=100"`                       // 每页数量
	SortBy    string         `json:"sort_by,omitempty"`                                       // 排序字段
	SortOrder string         `json:"sort_order,omitempty" binding:"omitempty,oneof=asc desc"` // 排序方向
}

// CISavedView 保存的搜索视图
type CISavedView struct {
	ID          int            `json:"id"`
	Name        string         `json:"name"`                  // 视图名称
	Description string         `json:"description,omitempty"` // 视图描述
	Filters     CISearchFilter `json:"filters"`               // 保存的过滤条件
	SortBy      string         `json:"sort_by,omitempty"`
	SortOrder   string         `json:"sort_order,omitempty"`
	IsPublic    bool           `json:"is_public"`              // 是否公开
	CreatorID   int            `json:"creator_id"`             // 创建人ID
	CreatorName string         `json:"creator_name,omitempty"` // 创建人名称
	TenantID    int            `json:"tenant_id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// CreateCISavedViewRequest 创建保存视图请求
type CreateCISavedViewRequest struct {
	Name        string         `json:"name" binding:"required,max=100"`
	Description string         `json:"description,omitempty" max:"500"`
	Filters     CISearchFilter `json:"filters" binding:"required"`
	SortBy      string         `json:"sort_by,omitempty"`
	SortOrder   string         `json:"sort_order,omitempty"`
	IsPublic    bool           `json:"is_public"`
}

// UpdateCISavedViewRequest 更新保存视图请求
type UpdateCISavedViewRequest struct {
	Name        *string         `json:"name,omitempty" binding:"omitempty,max=100"`
	Description *string         `json:"description,omitempty" binding:"omitempty,max=500"`
	Filters     *CISearchFilter `json:"filters,omitempty"`
	SortBy      *string         `json:"sort_by,omitempty"`
	SortOrder   *string         `json:"sort_order,omitempty"`
	IsPublic    *bool           `json:"is_public,omitempty"`
}
