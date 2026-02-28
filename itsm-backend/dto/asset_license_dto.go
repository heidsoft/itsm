package dto

import "time"

// LicenseStatus 许可证状态
type LicenseStatus string

const (
	LicenseStatusActive       LicenseStatus = "active"        // 有效
	LicenseStatusExpired      LicenseStatus = "expired"       // 已过期
	LicenseStatusExpiringSoon LicenseStatus = "expiring-soon" // 即将过期
	LicenseStatusDepleted     LicenseStatus = "depleted"      // 已耗尽
)

// LicenseType 许可证类型
type LicenseType string

const (
	LicenseTypePerpetual    LicenseType = "perpetual"    // 永久
	LicenseTypeSubscription LicenseType = "subscription" // 订阅
	LicenseTypePerUser      LicenseType = "per-user"     // 按用户
	LicenseTypePerSeat      LicenseType = "per-seat"     // 按席位
	LicenseTypeSite         LicenseType = "site"         // 站点
)

// CreateLicenseRequest 创建许可证请求
type CreateLicenseRequest struct {
	Name           string   `json:"name" binding:"required"` // 许可证名称
	Description    string   `json:"description"`             // 许可证描述
	Vendor         string   `json:"vendor"`                  // 供应商
	LicenseType    string   `json:"license_type"`            // 许可证类型
	TotalQuantity  int      `json:"total_quantity"`          // 总数量
	AssetID        *int     `json:"asset_id"`                // 关联的资产ID
	PurchaseDate   string   `json:"purchase_date"`           // 采购日期
	PurchasePrice  *float64 `json:"purchase_price"`          // 采购价格
	ExpiryDate     string   `json:"expiry_date"`             // 到期日期
	SupportVendor  string   `json:"support_vendor"`          // 支持供应商
	SupportContact string   `json:"support_contact"`         // 支持联系方式
	RenewalCost    string   `json:"renewal_cost"`            // 续费成本
	Notes          string   `json:"notes"`                   // 备注
	Users          []int    `json:"users"`                   // 授权用户列表
	Tags           []string `json:"tags"`                    // 标签
}

// UpdateLicenseRequest 更新许可证请求
type UpdateLicenseRequest struct {
	Name           *string  `json:"name"`            // 许可证名称
	Description    *string  `json:"description"`     // 许可证描述
	Vendor         *string  `json:"vendor"`          // 供应商
	LicenseType    *string  `json:"license_type"`    // 许可证类型
	TotalQuantity  *int     `json:"total_quantity"`  // 总数量
	UsedQuantity   *int     `json:"used_quantity"`   // 已使用数量
	AssetID        *int     `json:"asset_id"`        // 关联的资产ID
	PurchaseDate   *string  `json:"purchase_date"`   // 采购日期
	PurchasePrice  *float64 `json:"purchase_price"`  // 采购价格
	ExpiryDate     *string  `json:"expiry_date"`     // 到期日期
	SupportVendor  *string  `json:"support_vendor"`  // 支持供应商
	SupportContact *string  `json:"support_contact"` // 支持联系方式
	RenewalCost    *string  `json:"renewal_cost"`    // 续费成本
	Notes          *string  `json:"notes"`           // 备注
	Users          []int    `json:"users"`           // 授权用户列表
	Tags           []string `json:"tags"`            // 标签
}

// LicenseResponse 许可证响应
type LicenseResponse struct {
	ID                int       `json:"id"`                 // 许可证ID
	LicenseKey        string    `json:"license_key"`        // 许可证密钥
	Name              string    `json:"name"`               // 许可证名称
	Description       string    `json:"description"`        // 许可证描述
	Vendor            string    `json:"vendor"`             // 供应商
	LicenseType       string    `json:"license_type"`       // 许可证类型
	TotalQuantity     int       `json:"total_quantity"`     // 总数量
	UsedQuantity      int       `json:"used_quantity"`      // 已使用数量
	AvailableQuantity int       `json:"available_quantity"` // 可用数量
	TenantID          int       `json:"tenant_id"`          // 租户ID
	AssetID           *int      `json:"asset_id"`           // 关联的资产ID
	AssetName         *string   `json:"asset_name"`         // 关联的资产名称
	PurchaseDate      string    `json:"purchase_date"`      // 采购日期
	PurchasePrice     *float64  `json:"purchase_price"`     // 采购价格
	ExpiryDate        string    `json:"expiry_date"`        // 到期日期
	SupportVendor     string    `json:"support_vendor"`     // 支持供应商
	SupportContact    string    `json:"support_contact"`    // 支持联系方式
	RenewalCost       string    `json:"renewal_cost"`       // 续费成本
	Status            string    `json:"status"`             // 状态
	Notes             string    `json:"notes"`              // 备注
	Users             []int     `json:"users"`              // 授权用户列表
	UserNames         []string  `json:"user_names"`         // 授权用户姓名列表
	Tags              []string  `json:"tags"`               // 标签
	CreatedAt         time.Time `json:"created_at"`         // 创建时间
	UpdatedAt         time.Time `json:"updated_at"`         // 更新时间
}

// LicenseListResponse 许可证列表响应
type LicenseListResponse struct {
	Total    int               `json:"total"`    // 总数
	Licenses []LicenseResponse `json:"licenses"` // 许可证列表
}

// LicenseStatsResponse 许可证统计响应
type LicenseStatsResponse struct {
	Total          int     `json:"total"`           // 总数
	Active         int     `json:"active"`          // 有效
	Expired        int     `json:"expired"`         // 已过期
	ExpiringSoon   int     `json:"expiring_soon"`   // 即将过期
	Depleted       int     `json:"depleted"`        // 已耗尽
	ComplianceRate float64 `json:"compliance_rate"` // 合规率
}

// LicenseAssignRequest 许可证分配请求
type LicenseAssignRequest struct {
	UserIDs []int   `json:"user_ids" binding:"required"` // 用户ID列表
	Comment *string `json:"comment"`                     // 分配说明
}

// LicenseRenewalRequest 许可证续期请求
type LicenseRenewalRequest struct {
	NewExpiryDate string  `json:"new_expiry_date" binding:"required"` // 新到期日期
	RenewalCost   string  `json:"renewal_cost"`                       // 续费成本
	Vendor        string  `json:"vendor"`                             // 供应商
	Comment       *string `json:"comment"`                            // 备注
}
