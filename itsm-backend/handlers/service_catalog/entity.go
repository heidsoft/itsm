package service_catalog

import (
	"time"
)

// ServiceCatalog represents the core domain entity
type ServiceCatalog struct {
	ID                int
	Name              string
	Category          string
	Description       string
	Icon              string
	ServiceType       string
	Price             float64
	DeliveryTime      int
	Unit              string
	RequiresApproval  bool
	ApprovalLevel     int
	Approvers         []int
	SLAResponseTime   int
	SLAResolutionTime int
	CITypeID          int
	CloudServiceID    int
	FormSchema        map[string]interface{}
	AvailableRegions  []string
	AvailableSpecs    []string
	Status            string
	TenantID          int
	IsActive          bool
	SortOrder         int
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// ListFilters defines available filters for listing catalogs
type ListFilters struct {
	Category string
	Status   string
	// Keyword 匹配 name 或 description，与 Search 同一口径。
	Keyword string
	// CiTypeID / CloudServiceID 为 0 表示不过滤。管理页的关联筛选必须服务端生效，
	// 否则只能对「已加载的第一页」二次过滤再冒充全量结果。
	CiTypeID       int
	CloudServiceID int
	Page           int
	PageSize       int
}

// CatalogPatch 是服务目录的显式部分更新意图：nil 表示调用方未提交该字段，保持原值；
// 非 nil 表示按携带的值写入，包括 0、false、空串这类合法业务值。
//
// Approvers/AvailableRegions/AvailableSpecs 不在这里：请求 DTO 上是裸切片，
// 无法区分「未传」与「传空数组」，清空列表需要请求侧先指针化。
type CatalogPatch struct {
	Name              *string
	Category          *string
	Description       *string
	Icon              *string
	ServiceType       *string
	Price             *float64
	DeliveryTime      *int
	Unit              *string
	RequiresApproval  *bool
	ApprovalLevel     *int
	SLAResponseTime   *int
	SLAResolutionTime *int
	CITypeID          *int
	CloudServiceID    *int
	FormSchema        *map[string]interface{}
	Status            *string
	SortOrder         *int

	// 请求 DTO 上是裸切片，nil 与空数组无法区分，因此这里只能表达「非空才改」。
	Approvers        []int
	AvailableRegions []string
	AvailableSpecs   []string
}

// ServiceStats holds statistics for service catalog
type ServiceStats struct {
	TotalServices     int            `json:"totalServices"`
	PublishedServices int            `json:"publishedServices"`
	Categories        map[string]int `json:"categories"`
}
