package service_catalog

import (
	"context"
	"time"
)

// ServiceCatalog represents the core domain entity
type ServiceCatalog struct {
	ID             int
	Name           string
	Category       string
	Description    string
	DeliveryTime   int
	CITypeID       int
	CloudServiceID int
	Status         string
	TenantID       int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Repository defines the interface for data persistence
type Repository interface {
	Create(ctx context.Context, catalog *ServiceCatalog) (*ServiceCatalog, error)
	Get(ctx context.Context, tenantID int, id int) (*ServiceCatalog, error)
	List(ctx context.Context, tenantID int, filters ListFilters) ([]*ServiceCatalog, int, error)
	Search(ctx context.Context, tenantID int, keyword string, filters ListFilters) ([]*ServiceCatalog, int, error)
	Update(ctx context.Context, tenantID int, catalog *ServiceCatalog) (*ServiceCatalog, error)
	Delete(ctx context.Context, tenantID int, id int) error
	Count(ctx context.Context, tenantID int, filters ListFilters) (int, error)
	CountByCategory(ctx context.Context, tenantID int) (map[string]int, error)
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
