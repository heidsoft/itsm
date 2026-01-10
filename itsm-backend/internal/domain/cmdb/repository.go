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

	// Relationship types
	ListRelationshipTypes(ctx context.Context, tenantID int) ([]*RelationshipType, error)

	// Cloud services
	CreateCloudService(ctx context.Context, cs *CloudService) (*CloudService, error)
	ListCloudServices(ctx context.Context, tenantID int, provider string) ([]*CloudService, error)
	GetCloudService(ctx context.Context, tenantID int, id int) (*CloudService, error)

	// Cloud accounts
	CreateCloudAccount(ctx context.Context, ca *CloudAccount) (*CloudAccount, error)
	ListCloudAccounts(ctx context.Context, tenantID int, provider string) ([]*CloudAccount, error)

	// Cloud resources
	ListCloudResources(ctx context.Context, tenantID int, provider string, serviceID int, region string) ([]*CloudResource, error)
	ListCIsForReconciliation(ctx context.Context, tenantID int) ([]*ConfigurationItem, error)
	GetCIByCloudResourceRefID(ctx context.Context, tenantID int, cloudResourceRefID int) (*ConfigurationItem, error)

	// Discovery
	CreateDiscoverySource(ctx context.Context, ds *DiscoverySource) (*DiscoverySource, error)
	ListDiscoverySources(ctx context.Context, tenantID int) ([]*DiscoverySource, error)
	CreateDiscoveryJob(ctx context.Context, job *DiscoveryJob) (*DiscoveryJob, error)
	ListDiscoveryResults(ctx context.Context, tenantID int, jobID int) ([]*DiscoveryResult, error)
}
