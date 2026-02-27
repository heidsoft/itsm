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
	AssetNumber    string            `json:"asset_number" binding:"required"` // 资产编号
	Name           string            `json:"name" binding:"required"`         // 资产名称
	Description    string            `json:"description"`                     // 资产描述
	Type           string            `json:"type"`                            // 资产类型
	Category       string            `json:"category"`                        // 资产分类
	Subcategory    string            `json:"subcategory"`                     // 资产子分类
	CIID           *int              `json:"ci_id"`                           // 关联CMDB配置项ID
	AssignedTo     *int              `json:"assigned_to"`                     // 分配给的用户ID
	LocationID     *int              `json:"location_id"`                     // 位置ID
	SerialNumber   string            `json:"serial_number"`                   // 序列号
	Model          string            `json:"model"`                           // 型号
	Manufacturer   string            `json:"manufacturer"`                    // 制造商
	Vendor         string            `json:"vendor"`                          // 供应商
	PurchaseDate   string            `json:"purchase_date"`                   // 采购日期
	PurchasePrice  *float64          `json:"purchase_price"`                  // 采购价格
	WarrantyExpiry string            `json:"warranty_expiry"`                 // 保修期到期
	SupportExpiry  string            `json:"support_expiry"`                  // 支持期到期
	Location       string            `json:"location"`                        // 物理位置
	Department     string            `json:"department"`                      // 所属部门
	ParentAssetID  *int              `json:"parent_asset_id"`                 // 父资产ID
	Specifications map[string]string `json:"specifications"`                  // 规格参数
	CustomFields   map[string]string `json:"custom_fields"`                   // 自定义字段
	Tags           []string          `json:"tags"`                            // 标签
}

// UpdateAssetRequest 更新资产请求
type UpdateAssetRequest struct {
	Name           *string           `json:"name"`            // 资产名称
	Description    *string           `json:"description"`     // 资产描述
	Type           *string           `json:"type"`            // 资产类型
	Category       *string           `json:"category"`        // 资产分类
	Subcategory    *string           `json:"subcategory"`     // 资产子分类
	CIID           *int              `json:"ci_id"`           // 关联CMDB配置项ID
	AssignedTo     *int              `json:"assigned_to"`     // 分配给的用户ID
	LocationID     *int              `json:"location_id"`     // 位置ID
	SerialNumber   *string           `json:"serial_number"`   // 序列号
	Model          *string           `json:"model"`           // 型号
	Manufacturer   *string           `json:"manufacturer"`    // 制造商
	Vendor         *string           `json:"vendor"`          // 供应商
	PurchaseDate   *string           `json:"purchase_date"`   // 采购日期
	PurchasePrice  *float64          `json:"purchase_price"`  // 采购价格
	WarrantyExpiry *string           `json:"warranty_expiry"` // 保修期到期
	SupportExpiry  *string           `json:"support_expiry"`  // 支持期到期
	Location       *string           `json:"location"`        // 物理位置
	Department     *string           `json:"department"`      // 所属部门
	ParentAssetID  *int              `json:"parent_asset_id"` // 父资产ID
	Specifications map[string]string `json:"specifications"`  // 规格参数
	CustomFields   map[string]string `json:"custom_fields"`   // 自定义字段
	Tags           []string          `json:"tags"`            // 标签
}

// AssetResponse 资产响应
type AssetResponse struct {
	ID              int               `json:"id"`                // 资产ID
	AssetNumber     string            `json:"asset_number"`      // 资产编号
	Name            string            `json:"name"`              // 资产名称
	Description     string            `json:"description"`       // 资产描述
	Type            string            `json:"type"`              // 资产类型
	Status          string            `json:"status"`            // 状态
	Category        string            `json:"category"`          // 资产分类
	Subcategory     string            `json:"subcategory"`       // 资产子分类
	TenantID        int               `json:"tenant_id"`         // 租户ID
	CIID            *int              `json:"ci_id"`             // 关联CMDB配置项ID
	CIName          *string           `json:"ci_name"`           // 关联CMDB配置项名称
	AssignedTo      *int              `json:"assigned_to"`       // 分配给的用户ID
	AssignedToName  *string           `json:"assigned_to_name"`  // 分配给的用户姓名
	LocationID      *int              `json:"location_id"`       // 位置ID
	SerialNumber    string            `json:"serial_number"`     // 序列号
	Model           string            `json:"model"`             // 型号
	Manufacturer    string            `json:"manufacturer"`      // 制造商
	Vendor          string            `json:"vendor"`            // 供应商
	PurchaseDate    string            `json:"purchase_date"`     // 采购日期
	PurchasePrice   *float64          `json:"purchase_price"`    // 采购价格
	WarrantyExpiry  string            `json:"warranty_expiry"`   // 保修期到期
	SupportExpiry   string            `json:"support_expiry"`    // 支持期到期
	Location        string            `json:"location"`          // 物理位置
	Department      string            `json:"department"`        // 所属部门
	ParentAssetID   *int              `json:"parent_asset_id"`   // 父资产ID
	ParentAssetName *string           `json:"parent_asset_name"` // 父资产名称
	Specifications  map[string]string `json:"specifications"`    // 规格参数
	CustomFields    map[string]string `json:"custom_fields"`     // 自定义字段
	Tags            []string          `json:"tags"`              // 标签
	CreatedAt       time.Time         `json:"created_at"`        // 创建时间
	UpdatedAt       time.Time         `json:"updated_at"`        // 更新时间
}

// AssetListResponse 资产列表响应
type AssetListResponse struct {
	Total  int             `json:"total"`  // 总数
	Assets []AssetResponse `json:"assets"` // 资产列表
}

// AssetStatsResponse 资产统计响应
type AssetStatsResponse struct {
	Total       int `json:"total"`       // 总数
	Available   int `json:"available"`   // 可用
	InUse       int `json:"in_use"`      // 使用中
	Maintenance int `json:"maintenance"` // 维护中
	Retired     int `json:"retired"`     // 已退役
	Disposed    int `json:"disposed"`    // 已处置
}

// AssetStatusUpdateRequest 资产状态更新请求
type AssetStatusUpdateRequest struct {
	Status     AssetStatus `json:"status" binding:"required"` // 新状态
	AssignedTo *int        `json:"assigned_to"`               // 分配给的用户ID
	Comment    *string     `json:"comment"`                   // 状态变更说明
}

// AssetAssignRequest 资产分配请求
type AssetAssignRequest struct {
	AssignedTo int     `json:"assigned_to" binding:"required"` // 分配给的用户ID
	Comment    *string `json:"comment"`                        // 分配说明
}

// AssetTransferRequest 资产转移请求
type AssetTransferRequest struct {
	AssignedTo int     `json:"assigned_to" binding:"required"` // 新所有者ID
	Location   *string `json:"location"`                       // 新位置
	Comment    *string `json:"comment"`                        // 转移说明
}

// AssetMaintenanceRequest 资产维护请求
type AssetMaintenanceRequest struct {
	MaintenanceType string     `json:"maintenance_type" binding:"required"` // 维护类型
	Description     string     `json:"description" binding:"required"`      // 维护描述
	StartDate       *time.Time `json:"start_date"`                          // 开始日期
	EndDate         *time.Time `json:"end_date"`                            // 结束日期
	Cost            *float64   `json:"cost"`                                // 维护成本
	Vendor          *string    `json:"vendor"`                              // 供应商
	VendorContact   *string    `json:"vendor_contact"`                      // 供应商联系方式
	Result          *string    `json:"result"`                              // 维护结果
}

// AssetRetireRequest 资产退役请求
type AssetRetireRequest struct {
	RetireDate     string   `json:"retire_date" binding:"required"`   // 退役日期
	RetireReason   string   `json:"retire_reason" binding:"required"` // 退役原因
	DisposalMethod string   `json:"disposal_method"`                  // 处置方式
	DisposalValue  *float64 `json:"disposal_value"`                   // 处置价值
	Comment        *string  `json:"comment"`                          // 备注
}
