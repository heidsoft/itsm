package service_catalog

import "context"

// Repository defines the interface for data persistence.
// 实现见 repository_impl.go（EntRepository）。
type Repository interface {
	Create(ctx context.Context, catalog *ServiceCatalog) (*ServiceCatalog, error)
	Get(ctx context.Context, tenantID int, id int) (*ServiceCatalog, error)
	List(ctx context.Context, tenantID int, filters ListFilters) ([]*ServiceCatalog, int, error)
	Search(ctx context.Context, tenantID int, keyword string, filters ListFilters) ([]*ServiceCatalog, int, error)
	Update(ctx context.Context, tenantID int, catalog *ServiceCatalog) (*ServiceCatalog, error)
	Delete(ctx context.Context, tenantID int, id int) error
	Count(ctx context.Context, tenantID int, filters ListFilters) (int, error)
	CountByCategory(ctx context.Context, tenantID int) (map[string]int, error)
	NameExists(ctx context.Context, tenantID int, name string, excludeID int) (bool, error)
	ValidateReferences(ctx context.Context, tenantID, ciTypeID, cloudServiceID int) error
}
