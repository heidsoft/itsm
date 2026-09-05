// Package cloud provides cloud account, cloud service and cloud resource
// management (multi-cloud inventory) HTTP handlers.
//
// The handler layer is a thin HTTP facade: it parses requests, extracts the
// tenant from the auth context via middleware.TenantIDOrUnauthorized, and
// delegates all business logic to the domain Service interface below.
// Persistence and business rules live in the underlying service implementation.
package cloud

import (
	"context"

	"itsm-backend/dto"
	"itsm-backend/ent"
)

// Service defines the business capability surface consumed by the cloud
// handler. It covers three entity families (accounts, services, resources),
// each with full CRUD plus list filtering.
type Service interface {
	// Cloud accounts
	CreateCloudAccount(ctx context.Context, tenantID int, req *dto.CreateCloudAccountRequest) (*ent.CloudAccount, error)
	GetCloudAccount(ctx context.Context, tenantID, id int) (*ent.CloudAccount, error)
	UpdateCloudAccount(ctx context.Context, tenantID, id int, req *dto.UpdateCloudAccountRequest) (*ent.CloudAccount, error)
	DeleteCloudAccount(ctx context.Context, tenantID, id int) error
	ListCloudAccounts(ctx context.Context, tenantID int, req *dto.ListCloudAccountsRequest) ([]*ent.CloudAccount, int, error)

	// Cloud services (catalog of cloud offerings)
	CreateCloudService(ctx context.Context, tenantID int, req *dto.CreateCloudServiceRequest) (*ent.CloudService, error)
	GetCloudService(ctx context.Context, tenantID, id int) (*ent.CloudService, error)
	UpdateCloudService(ctx context.Context, tenantID, id int, req *dto.UpdateCloudServiceRequest) (*ent.CloudService, error)
	DeleteCloudService(ctx context.Context, tenantID, id int) error
	ListCloudServices(ctx context.Context, tenantID int, req *dto.ListCloudServicesRequest) ([]*ent.CloudService, int, error)

	// Cloud resources (discovered inventory items)
	CreateCloudResource(ctx context.Context, tenantID int, req *dto.CreateCloudResourceRequest) (*ent.CloudResource, error)
	GetCloudResource(ctx context.Context, tenantID, id int) (*ent.CloudResource, error)
	UpdateCloudResource(ctx context.Context, tenantID, id int, req *dto.UpdateCloudResourceRequest) (*ent.CloudResource, error)
	DeleteCloudResource(ctx context.Context, tenantID, id int) error
	ListCloudResources(ctx context.Context, tenantID int, req *dto.ListCloudResourcesRequest) ([]*ent.CloudResource, int, error)
}
