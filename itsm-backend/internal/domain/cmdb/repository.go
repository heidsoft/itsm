package cmdb

import (
	"context"
)

// Repository interface for CMDB domain
type Repository interface {
	// CI CRUD
	CreateCI(ctx context.Context, ci *ConfigurationItem) (*ConfigurationItem, error)
	GetCI(ctx context.Context, id int, tenantID int) (*ConfigurationItem, error)
	ListCIs(ctx context.Context, tenantID int, page, size int, ciTypeID int, status string) ([]*ConfigurationItem, int, error)
	UpdateCI(ctx context.Context, ci *ConfigurationItem) (*ConfigurationItem, error)
	DeleteCI(ctx context.Context, id int, tenantID int) error
	GetStats(ctx context.Context, tenantID int) (*Stats, error)

	// CI Types
	CreateCIType(ctx context.Context, ct *CIType) (*CIType, error)
	GetCIType(ctx context.Context, id int, tenantID int) (*CIType, error)
	ListCITypes(ctx context.Context, tenantID int) ([]*CIType, error)

	// Relationships
	CreateRelationship(ctx context.Context, rel *CIRelationship) (*CIRelationship, error)
	GetRelationships(ctx context.Context, ciID int) ([]*CIRelationship, error)
	DeleteRelationship(ctx context.Context, id int, tenantID int) error
}
