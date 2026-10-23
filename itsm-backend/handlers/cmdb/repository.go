package cmdb

import (
	"context"
	"time"
)

// CloudResourceFilter 云资源列表的过滤条件；零值字段一律表示「不过滤」。
//
// Provider 走云账号边（资源本身不存 provider），因此它依赖账号归属而不是自报表字段；
// CloudAccountID/ServiceID 用 >0 判定，0 是「未传」而不是「按 0 过滤」。
type CloudResourceFilter struct {
	Provider       string
	CloudAccountID int
	ServiceID      int
	Region         string
	Status         string
	Search         string
}

// Repository interface for CMDB domain
// 仅保留云账号/云服务/云资源/发现/对账职责；CI/CIType/关系的死代码接口已删除。
//
// 三个 Delete 返回受影响行数：tenant+id 命中 0 行不是「删除成功」，
// 由 service 翻成 not found，否则跨租户删除会对外报成功。
type Repository interface {
	// Cloud services
	CreateCloudService(ctx context.Context, cs *CloudService) (*CloudService, error)
	ListCloudServices(ctx context.Context, tenantID int, provider string) ([]*CloudService, error)
	GetCloudService(ctx context.Context, tenantID int, id int) (*CloudService, error)
	UpdateCloudService(ctx context.Context, cs *CloudService) (*CloudService, error)
	DeleteCloudService(ctx context.Context, id int, tenantID int) (int, error)

	// Cloud accounts
	CreateCloudAccount(ctx context.Context, ca *CloudAccount) (*CloudAccount, error)
	ListCloudAccounts(ctx context.Context, tenantID int, provider string) ([]*CloudAccount, error)
	GetCloudAccount(ctx context.Context, tenantID int, id int) (*CloudAccount, error)
	UpdateCloudAccount(ctx context.Context, ca *CloudAccount) (*CloudAccount, error)
	DeleteCloudAccount(ctx context.Context, id int, tenantID int) (int, error)

	// Cloud resources
	ListCloudResources(ctx context.Context, tenantID int, filter CloudResourceFilter) ([]*CloudResource, error)
	ListCloudResourcesPage(ctx context.Context, tenantID int, filter CloudResourceFilter, page int, pageSize int) ([]*CloudResource, int, error)
	GetCloudResource(ctx context.Context, tenantID int, id int) (*CloudResource, error)
	CreateCloudResource(ctx context.Context, cr *CloudResource) (*CloudResource, error)
	UpdateCloudResource(ctx context.Context, cr *CloudResource) (*CloudResource, error)
	DeleteCloudResource(ctx context.Context, id int, tenantID int) (int, error)
	ListCIsForReconciliation(ctx context.Context, tenantID int) ([]*ConfigurationItem, error)
	GetCIByCloudResourceRefID(ctx context.Context, tenantID int, cloudResourceRefID int) (*ConfigurationItem, error)

	// Discovery
	CreateDiscoverySource(ctx context.Context, ds *DiscoverySource) (*DiscoverySource, error)
	ListDiscoverySources(ctx context.Context, tenantID int) ([]*DiscoverySource, error)
	GetDiscoverySource(ctx context.Context, tenantID int, id string) (*DiscoverySource, error)
	TouchDiscoverySourceSuccess(ctx context.Context, tenantID int, id string, at time.Time) error
	CreateDiscoveryJob(ctx context.Context, job *DiscoveryJob) (*DiscoveryJob, error)
	// CreateDiscoveryJobWithCommand 在同一事务内落 discovery_jobs 行并入队
	// commandType 出箱命令（payload 携带 discoveryJobId），保证「有 job 必有命令」。
	CreateDiscoveryJobWithCommand(ctx context.Context, job *DiscoveryJob, commandType string, payload map[string]interface{}) (*DiscoveryJob, error)
	GetDiscoveryJob(ctx context.Context, tenantID, id int) (*DiscoveryJob, error)
	FindDiscoveryJobByIdempotencyKey(ctx context.Context, tenantID int, sourceID, key string) (*DiscoveryJob, error)
	ListDiscoveryJobs(ctx context.Context, tenantID int, status string, page, pageSize int) ([]*DiscoveryJob, int, error)
	// UpdateDiscoveryJob 全字段写回（执行器 claim/进度/终态推进的持久化通道）。
	UpdateDiscoveryJob(ctx context.Context, job *DiscoveryJob) error
	CreateDiscoveryResult(ctx context.Context, res *DiscoveryResult) (*DiscoveryResult, error)
	ListDiscoveryResults(ctx context.Context, tenantID int, jobID int) ([]*DiscoveryResult, error)
	// GetCloudResourceByIdentity 按规范身份哈希取资源（发现 diff 的现状读取面）。
	GetCloudResourceByIdentity(ctx context.Context, tenantID int, identityHash string) (*CloudResource, error)
	// ListCloudResourcesByAccountService 按账号+服务维度列资源（缺失扫描：
	// 本轮快照里没出现的资源要推进 missing_count）。
	ListCloudResourcesByAccountService(ctx context.Context, tenantID, cloudAccountID int, serviceCode string) ([]*CloudResource, error)
	// UpdateDiscoveredCloudResource 发现执行器专用的资源写回：全字段 +
	// first_seen_at/last_seen_at（通用 UpdateCloudResource 不覆盖 seen 时间戳）。
	UpdateDiscoveredCloudResource(ctx context.Context, cr *CloudResource) error
}
