package cmdb

import (
	"context"
)

// Repository interface for CMDB domain
// 仅保留云账号/云服务/云资源/发现/对账职责；CI/CIType/关系的死代码接口已删除。
type Repository interface {
	// Cloud services
	CreateCloudService(ctx context.Context, cs *CloudService) (*CloudService, error)
	ListCloudServices(ctx context.Context, tenantID int, provider string) ([]*CloudService, error)
	GetCloudService(ctx context.Context, tenantID int, id int) (*CloudService, error)
	UpdateCloudService(ctx context.Context, cs *CloudService) (*CloudService, error)
	DeleteCloudService(ctx context.Context, id int, tenantID int) error

	// Cloud accounts
	CreateCloudAccount(ctx context.Context, ca *CloudAccount) (*CloudAccount, error)
	ListCloudAccounts(ctx context.Context, tenantID int, provider string) ([]*CloudAccount, error)
	GetCloudAccount(ctx context.Context, tenantID int, id int) (*CloudAccount, error)
	UpdateCloudAccount(ctx context.Context, ca *CloudAccount) (*CloudAccount, error)
	DeleteCloudAccount(ctx context.Context, id int, tenantID int) error

	// Cloud resources
	ListCloudResources(ctx context.Context, tenantID int, provider string, serviceID int, region string) ([]*CloudResource, error)
	GetCloudResource(ctx context.Context, tenantID int, id int) (*CloudResource, error)
	CreateCloudResource(ctx context.Context, cr *CloudResource) (*CloudResource, error)
	UpdateCloudResource(ctx context.Context, cr *CloudResource) (*CloudResource, error)
	DeleteCloudResource(ctx context.Context, id int, tenantID int) error
	ListCIsForReconciliation(ctx context.Context, tenantID int) ([]*ConfigurationItem, error)
	GetCIByCloudResourceRefID(ctx context.Context, tenantID int, cloudResourceRefID int) (*ConfigurationItem, error)

	// Discovery
	CreateDiscoverySource(ctx context.Context, ds *DiscoverySource) (*DiscoverySource, error)
	ListDiscoverySources(ctx context.Context, tenantID int) ([]*DiscoverySource, error)
	CreateDiscoveryJob(ctx context.Context, job *DiscoveryJob) (*DiscoveryJob, error)
	ListDiscoveryResults(ctx context.Context, tenantID int, jobID int) ([]*DiscoveryResult, error)
}
