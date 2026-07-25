package cloud

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"go.uber.org/zap"

	"itsm-backend/ent"
	"itsm-backend/ent/cloudaccount"
	"itsm-backend/ent/cloudservice"
	"itsm-backend/ent/configurationitem"
)

// Runner 负责三层架构的调度：Discover → Transform → Reconcile
type Runner struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewRunner 构造 Runner
func NewRunner(client *ent.Client, logger *zap.SugaredLogger) *Runner {
	return &Runner{client: client, logger: logger}
}

// RunAll 执行全量云资源发现
func (r *Runner) RunAll(ctx context.Context, tenantID int, opts ...Option) error {
	cfg := &Config{}
	for _, o := range opts {
		o(cfg)
	}

	// 获取所有启用的云账号
	accounts, err := r.client.CloudAccount.Query().
		Where(cloudaccount.TenantID(tenantID), cloudaccount.IsActive(true)).
		All(ctx)
	if err != nil {
		return fmt.Errorf("查询云账号失败: %w", err)
	}

	var wg sync.WaitGroup
	var once sync.Once
	var topErr error

	for _, account := range accounts {
		wg.Add(1)
		go func(account *ent.CloudAccount) {
			defer wg.Done()
			if err := r.runAccount(ctx, account, cfg); err != nil {
				once.Do(func() { topErr = err })
				r.logger.Warnw("账号发现失败", "account", account.ID, "error", err)
			}
		}(account)
	}
	wg.Wait()

	return topErr
}

func (r *Runner) runAccount(ctx context.Context, account *ent.CloudAccount, cfg *Config) error {
	adapters := GlobalRegistry().GetByAccount(account)
	if len(adapters) == 0 {
		return fmt.Errorf("no adapters for provider=%s", account.Provider)
	}

	for _, adapter := range adapters {
		regions, err := adapter.ListRegions(ctx, account)
		if err != nil {
			return fmt.Errorf("ListRegions: %w", err)
		}

		clients, err := adapter.InitClients(ctx, account, regions)
		if err != nil {
			return fmt.Errorf("InitClients: %w", err)
		}

		for _, region := range regions {
			client := clients[region]
			if client == nil {
				continue
			}
			if err := r.runRegion(ctx, account, adapter, region, client); err != nil {
				r.logger.Warnw("Region 发现失败", "region", region, "error", err)
			}
		}
	}
	return nil
}

func (r *Runner) runRegion(ctx context.Context, account *ent.CloudAccount, adapter CloudDiscoveryAdapter, region string, client Client) error {
	pageResult, err := adapter.DiscoverRegion(ctx, account, region, client, "")
	if err != nil {
		return fmt.Errorf("DiscoverRegion: %w", err)
	}
	return r.reconcile(ctx, account, adapter, region, pageResult)
}

func (r *Runner) reconcile(ctx context.Context, account *ent.CloudAccount, adapter CloudDiscoveryAdapter, region string, pageResult *PageResult) error {
	provider := adapter.Provider()
	serviceCode := adapter.ServiceCode()

	// 查询 CloudService 定义（用于获取 attribute_schema）
	_, err := r.client.CloudService.Query().
		Where(cloudservice.Provider(provider), cloudservice.ServiceCode(serviceCode)).
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return fmt.Errorf("查询 CloudService 失败: %w", err)
	}

	for _, resource := range pageResult.Resources {
		// 按 cloud_resource_id 查找现有 CI
		existing, err := r.client.ConfigurationItem.Query().
			Where(
				configurationitem.TenantID(account.TenantID),
				configurationitem.CiType("cloud"),
				configurationitem.CloudResourceID(resource.ResourceID),
			).
			Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				_, err = r.client.ConfigurationItem.Create().
					SetCiType("cloud").
					SetCloudResourceID(resource.ResourceID).
					SetName(resource.ResourceName).
					SetStatus(mapStatus(resource.Status)).
					SetTenantID(account.TenantID).
					SetCloudProvider(provider).
					SetCloudAccountID(strconv.Itoa(account.ID)).
					SetCloudRegion(region).
					SetAttributes(map[string]interface{}{
						"cloud_service": serviceCode,
						"extra":         resource.Extra,
					}).
					Save(ctx)
				if err != nil {
					r.logger.Warnw("创建 CI 失败", "resourceID", resource.ResourceID, "error", err)
				}
				continue
			}
			r.logger.Warnw("查询 CI 失败", "resourceID", resource.ResourceID, "error", err)
			continue
		}

		// 存在 → 更新
		if existing.Name != resource.ResourceName || existing.Status != mapStatus(resource.Status) {
			_, err = existing.Update().
				SetName(resource.ResourceName).
				SetStatus(mapStatus(resource.Status)).
				SetCloudRegion(region).
				SetAttributes(map[string]interface{}{
					"cloud_service": serviceCode,
					"extra":         resource.Extra,
				}).
				Save(ctx)
			if err != nil {
				r.logger.Warnw("更新 CI 失败", "resourceID", resource.ResourceID, "error", err)
			}
		}
	}
	return nil
}

// mapStatus 统一状态映射
func mapStatus(srcStatus string) string {
	switch srcStatus {
	case "pending", "creating", "Creating":
		return "pending"
	case "active", "running", "Running":
		return "active"
	case "inactive", "stopped", "Stopped", "stopping", "Stopping":
		return "inactive"
	case "retired", "released", "expired", "deleted", "Released", "Expired", "Deleted":
		return "retired"
	default:
		return "inactive"
	}
}

// Config 发现配置
type Config struct {
	ReconcilePolicy ReconcilePolicy
	Concurrency     int
}

// Option 配置选项
type Option func(*Config)

// WithReconcilePolicy 设置对账策略
func WithReconcilePolicy(p ReconcilePolicy) Option {
	return func(c *Config) { c.ReconcilePolicy = p }
}

// WithConcurrency 设置并发数
func WithConcurrency(n int) Option {
	return func(c *Config) { c.Concurrency = n }
}
