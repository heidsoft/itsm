package dto

import "time"

// AssetStatus 资产状态
type AssetStatus string

const (
	AssetStatusAvailable   AssetStatus = "available"   // 可用
	AssetStatusInUse       AssetStatus = "in-use"      // 使用中
	AssetStatusMaintenance AssetStatus = "maintenance" // 维护中
	AssetStatusRetired     AssetStatus = "retired"     // 已退役
	AssetStatusDisposed    AssetStatus = "disposed"    // 已处置
)

// AssetType 资产类型
type AssetType string

const (
	AssetTypeHardware AssetType = "hardware" // 硬件
	AssetTypeSoftware AssetType = "software" // 软件
	AssetTypeCloud    AssetType = "cloud"    // 云资源
	AssetTypeLicense  AssetType = "license"  // 许可证
)

// CreateAssetRequest 创建资产请求
type CreateAssetRequest struct {
	AssetNumber    string            `json:"assetNumber" binding:"required"` // 资产编号
	Name           string            `json:"name" binding:"required"`        // 资产名称
	Description    string            `json:"description"`                    // 资产描述
	Type           string            `json:"type"`                           // 资产类型
	Category       string            `json:"category"`                       // 资产分类
	Subcategory    string            `json:"subcategory"`                    // 资产子分类
	CIID           *int              `json:"ciId"`                           // 关联CMDB配置项ID
	AssignedTo     *int              `json:"assignedTo"`                     // 分配给的用户ID
	LocationID     *int              `json:"locationId"`                     // 位置ID
	SerialNumber   string            `json:"serialNumber"`                   // 序列号
	Model          string            `json:"model"`                          // 型号
	Manufacturer   string            `json:"manufacturer"`                   // 制造商
	Vendor         string            `json:"vendor"`                         // 供应商
	PurchaseDate   string            `json:"purchaseDate"`                   // 采购日期
	PurchasePrice  *float64          `json:"purchasePrice"`                  // 采购价格
	WarrantyExpiry string            `json:"warrantyExpiry"`                 // 保修期到期
	SupportExpiry  string            `json:"supportExpiry"`                  // 支持期到期
	Location       string            `json:"location"`                       // 物理位置
	Department     string            `json:"department"`                     // 所属部门
	ParentAssetID  *int              `json:"parentAssetId"`                  // 父资产ID
	Specifications map[string]string `json:"specifications"`                 // 规格参数
	CustomFields   map[string]string `json:"customFields"`                   // 自定义字段
	Tags           []string          `json:"tags"`                           // 标签
}

// UpdateAssetRequest 更新资产请求
type UpdateAssetRequest struct {
	Name           *string           `json:"name"`           // 资产名称
	Description    *string           `json:"description"`    // 资产描述
	Type           *string           `json:"type"`           // 资产类型
	Category       *string           `json:"category"`       // 资产分类
	Subcategory    *string           `json:"subcategory"`    // 资产子分类
	CIID           *int              `json:"ciId"`           // 关联CMDB配置项ID
	AssignedTo     *int              `json:"assignedTo"`     // 分配给的用户ID
	LocationID     *int              `json:"locationId"`     // 位置ID
	SerialNumber   *string           `json:"serialNumber"`   // 序列号
	Model          *string           `json:"model"`          // 型号
	Manufacturer   *string           `json:"manufacturer"`   // 制造商
	Vendor         *string           `json:"vendor"`         // 供应商
	PurchaseDate   *string           `json:"purchaseDate"`   // 采购日期
	PurchasePrice  *float64          `json:"purchasePrice"`  // 采购价格
	WarrantyExpiry *string           `json:"warrantyExpiry"` // 保修期到期
	SupportExpiry  *string           `json:"supportExpiry"`  // 支持期到期
	Location       *string           `json:"location"`       // 物理位置
	Department     *string           `json:"department"`     // 所属部门
	ParentAssetID  *int              `json:"parentAssetId"`  // 父资产ID
	Specifications map[string]string `json:"specifications"` // 规格参数
	CustomFields   map[string]string `json:"customFields"`   // 自定义字段
	Tags           []string          `json:"tags"`           // 标签
}

// AssetResponse 资产响应
type AssetResponse struct {
	ID              int               `json:"id"`              // 资产ID
	AssetNumber     string            `json:"assetNumber"`     // 资产编号
	Name            string            `json:"name"`            // 资产名称
	Description     string            `json:"description"`     // 资产描述
	Type            string            `json:"type"`            // 资产类型
	Status          string            `json:"status"`          // 状态
	Category        string            `json:"category"`        // 资产分类
	Subcategory     string            `json:"subcategory"`     // 资产子分类
	TenantID        int               `json:"tenantId"`        // 租户ID
	CIID            *int              `json:"ciId"`            // 关联CMDB配置项ID
	CIName          *string           `json:"ciName"`          // 关联CMDB配置项名称
	AssignedTo      *int              `json:"assignedTo"`      // 分配给的用户ID
	AssignedToName  *string           `json:"assignedToName"`  // 分配给的用户姓名
	LocationID      *int              `json:"locationId"`      // 位置ID
	SerialNumber    string            `json:"serialNumber"`    // 序列号
	Model           string            `json:"model"`           // 型号
	Manufacturer    string            `json:"manufacturer"`    // 制造商
	Vendor          string            `json:"vendor"`          // 供应商
	PurchaseDate    string            `json:"purchaseDate"`    // 采购日期
	PurchasePrice   *float64          `json:"purchasePrice"`   // 采购价格
	WarrantyExpiry  string            `json:"warrantyExpiry"`  // 保修期到期
	SupportExpiry   string            `json:"supportExpiry"`   // 支持期到期
	Location        string            `json:"location"`        // 物理位置
	Department      string            `json:"department"`      // 所属部门
	ParentAssetID   *int              `json:"parentAssetId"`   // 父资产ID
	ParentAssetName *string           `json:"parentAssetName"` // 父资产名称
	Specifications  map[string]string `json:"specifications"`  // 规格参数
	CustomFields    map[string]string `json:"customFields"`    // 自定义字段
	Tags            []string          `json:"tags"`            // 标签
	CreatedAt       time.Time         `json:"createdAt"`       // 创建时间
	UpdatedAt       time.Time         `json:"updatedAt"`       // 更新时间
}

// AssetListResponse 资产列表响应
type AssetListResponse struct {
	Total      int             `json:"total"`      // 总数
	Assets     []AssetResponse `json:"assets"`     // 资产列表
	Page       int             `json:"page"`       // 当前页
	PageSize   int             `json:"pageSize"`   // 每页数量
	TotalPages int             `json:"totalPages"` // 总页数
}

// AssetStatsResponse 资产统计响应
type AssetStatsResponse struct {
	Total       int `json:"total"`       // 总数
	Available   int `json:"available"`   // 可用
	InUse       int `json:"inUse"`       // 使用中
	Maintenance int `json:"maintenance"` // 维护中
	Retired     int `json:"retired"`     // 已退役
	Disposed    int `json:"disposed"`    // 已处置
}

// AssetStatusUpdateRequest 资产状态更新请求
type AssetStatusUpdateRequest struct {
	Status     AssetStatus `json:"status" binding:"required"` // 新状态
	AssignedTo *int        `json:"assignedTo"`                // 分配给的用户ID
	Comment    *string     `json:"comment"`                   // 状态变更说明
}

// AssetAssignRequest 资产分配请求
type AssetAssignRequest struct {
	AssignedTo int     `json:"assignedTo" binding:"required"` // 分配给的用户ID
	Comment    *string `json:"comment"`                       // 分配说明
}

// AssetTransferRequest 资产转移请求
type AssetTransferRequest struct {
	AssignedTo int     `json:"assignedTo" binding:"required"` // 新所有者ID
	Location   *string `json:"location"`                      // 新位置
	Comment    *string `json:"comment"`                       // 转移说明
}

// AssetMaintenanceRequest 资产维护请求
type AssetMaintenanceRequest struct {
	MaintenanceType string     `json:"maintenanceType" binding:"required"` // 维护类型
	Description     string     `json:"description" binding:"required"`     // 维护描述
	StartDate       *time.Time `json:"startDate"`                          // 开始日期
	EndDate         *time.Time `json:"endDate"`                            // 结束日期
	Cost            *float64   `json:"cost"`                               // 维护成本
	Vendor          *string    `json:"vendor"`                             // 供应商
	VendorContact   *string    `json:"vendorContact"`                      // 供应商联系方式
	Result          *string    `json:"result"`                             // 维护结果
}

// AssetRetireRequest 资产退役请求
type AssetRetireRequest struct {
	RetireDate     string   `json:"retireDate" binding:"required"`   // 退役日期
	RetireReason   string   `json:"retireReason" binding:"required"` // 退役原因
	DisposalMethod string   `json:"disposalMethod"`                  // 处置方式
	DisposalValue  *float64 `json:"disposalValue"`                   // 处置价值
	Comment        *string  `json:"comment"`                         // 备注
}
