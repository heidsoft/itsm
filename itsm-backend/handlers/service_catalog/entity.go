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
	Page     int
	Size     int
}

// ServiceStats holds statistics for service catalog
type ServiceStats struct {
	TotalServices     int            `json:"totalServices"`
	PublishedServices int            `json:"publishedServices"`
	Categories        map[string]int `json:"categories"`
}
