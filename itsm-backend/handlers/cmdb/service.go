package cmdb

import (
	"context"

	"go.uber.org/zap"
)

type Service struct {
	repo   Repository
	logger *zap.SugaredLogger
}

func NewService(repo Repository, logger *zap.SugaredLogger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

// 未注册路由的 CI / CIType / 关系相关方法属死代码，已删除；
// 线上 CI 能力由 service/configuration_item_service.go 提供。

// Cloud services
func (s *Service) CreateCloudService(ctx context.Context, cs *CloudService) (*CloudService, error) {
	s.logger.Infow("Creating cloud service", "provider", cs.Provider, "service_code", cs.ServiceCode, "tenant_id", cs.TenantID)
	result, err := s.repo.CreateCloudService(ctx, cs)
	if err != nil {
		s.logger.Errorw("Failed to create cloud service", "error", err, "provider", cs.Provider, "service_code", cs.ServiceCode)
		return nil, err
	}
	s.logger.Infow("Cloud service created successfully", "id", result.ID, "provider", result.Provider, "service_code", result.ServiceCode)
	return result, nil
}

func (s *Service) ListCloudServices(ctx context.Context, tenantID int, provider string) ([]*CloudService, error) {
	s.logger.Infow("Listing cloud services", "tenant_id", tenantID, "provider", provider)
	result, err := s.repo.ListCloudServices(ctx, tenantID, provider)
	if err != nil {
		s.logger.Errorw("Failed to list cloud services", "error", err, "tenant_id", tenantID, "provider", provider)
		return nil, err
	}
	s.logger.Infow("Listed cloud services successfully", "count", len(result), "tenant_id", tenantID)
	return result, nil
}

func (s *Service) GetCloudService(ctx context.Context, tenantID int, id int) (*CloudService, error) {
	s.logger.Infow("Getting cloud service", "id", id, "tenant_id", tenantID)
	result, err := s.repo.GetCloudService(ctx, tenantID, id)
	if err != nil {
		s.logger.Errorw("Failed to get cloud service", "error", err, "id", id, "tenant_id", tenantID)
		return nil, err
	}
	s.logger.Infow("Got cloud service successfully", "id", id, "provider", result.Provider)
	return result, nil
}

func (s *Service) UpdateCloudService(ctx context.Context, cs *CloudService) (*CloudService, error) {
	s.logger.Infow("Updating cloud service", "id", cs.ID, "provider", cs.Provider, "tenant_id", cs.TenantID)
	result, err := s.repo.UpdateCloudService(ctx, cs)
	if err != nil {
		s.logger.Errorw("Failed to update cloud service", "error", err, "id", cs.ID, "provider", cs.Provider)
		return nil, err
	}
	s.logger.Infow("Cloud service updated successfully", "id", result.ID, "provider", result.Provider)
	return result, nil
}

func (s *Service) DeleteCloudService(ctx context.Context, id int, tenantID int) error {
	s.logger.Infow("Deleting cloud service", "id", id, "tenant_id", tenantID)
	err := s.repo.DeleteCloudService(ctx, id, tenantID)
	if err != nil {
		s.logger.Errorw("Failed to delete cloud service", "error", err, "id", id, "tenant_id", tenantID)
		return err
	}
	s.logger.Infow("Cloud service deleted successfully", "id", id, "tenant_id", tenantID)
	return nil
}

// Cloud accounts
func (s *Service) CreateCloudAccount(ctx context.Context, ca *CloudAccount) (*CloudAccount, error) {
	s.logger.Infow("Creating cloud account", "provider", ca.Provider, "account_id", ca.AccountID, "tenant_id", ca.TenantID)
	result, err := s.repo.CreateCloudAccount(ctx, ca)
	if err != nil {
		s.logger.Errorw("Failed to create cloud account", "error", err, "provider", ca.Provider, "account_id", ca.AccountID)
		return nil, err
	}
	s.logger.Infow("Cloud account created successfully", "id", result.ID, "provider", result.Provider, "account_id", result.AccountID)
	return result, nil
}

func (s *Service) ListCloudAccounts(ctx context.Context, tenantID int, provider string) ([]*CloudAccount, error) {
	s.logger.Infow("Listing cloud accounts", "tenant_id", tenantID, "provider", provider)
	result, err := s.repo.ListCloudAccounts(ctx, tenantID, provider)
	if err != nil {
		s.logger.Errorw("Failed to list cloud accounts", "error", err, "tenant_id", tenantID, "provider", provider)
		return nil, err
	}
	s.logger.Infow("Listed cloud accounts successfully", "count", len(result), "tenant_id", tenantID)
	return result, nil
}

func (s *Service) GetCloudAccount(ctx context.Context, tenantID int, id int) (*CloudAccount, error) {
	s.logger.Infow("Getting cloud account", "id", id, "tenant_id", tenantID)
	result, err := s.repo.GetCloudAccount(ctx, tenantID, id)
	if err != nil {
		s.logger.Errorw("Failed to get cloud account", "error", err, "id", id, "tenant_id", tenantID)
		return nil, err
	}
	s.logger.Infow("Got cloud account successfully", "id", id, "provider", result.Provider, "account_id", result.AccountID)
	return result, nil
}

func (s *Service) UpdateCloudAccount(ctx context.Context, ca *CloudAccount) (*CloudAccount, error) {
	s.logger.Infow("Updating cloud account", "id", ca.ID, "provider", ca.Provider, "tenant_id", ca.TenantID)
	result, err := s.repo.UpdateCloudAccount(ctx, ca)
	if err != nil {
		s.logger.Errorw("Failed to update cloud account", "error", err, "id", ca.ID, "provider", ca.Provider)
		return nil, err
	}
	s.logger.Infow("Cloud account updated successfully", "id", result.ID, "provider", result.Provider, "account_id", result.AccountID)
	return result, nil
}

func (s *Service) DeleteCloudAccount(ctx context.Context, id int, tenantID int) error {
	s.logger.Infow("Deleting cloud account", "id", id, "tenant_id", tenantID)
	err := s.repo.DeleteCloudAccount(ctx, id, tenantID)
	if err != nil {
		s.logger.Errorw("Failed to delete cloud account", "error", err, "id", id, "tenant_id", tenantID)
		return err
	}
	s.logger.Infow("Cloud account deleted successfully", "id", id, "tenant_id", tenantID)
	return nil
}

// Cloud resources
func (s *Service) ListCloudResources(ctx context.Context, tenantID int, provider string, serviceID int, region string) ([]*CloudResource, error) {
	s.logger.Infow("Listing cloud resources", "tenant_id", tenantID, "provider", provider, "service_id", serviceID, "region", region)
	result, err := s.repo.ListCloudResources(ctx, tenantID, provider, serviceID, region)
	if err != nil {
		s.logger.Errorw("Failed to list cloud resources", "error", err, "tenant_id", tenantID, "provider", provider, "service_id", serviceID, "region", region)
		return nil, err
	}
	s.logger.Infow("Listed cloud resources successfully", "count", len(result), "tenant_id", tenantID)
	return result, nil
}

func (s *Service) GetCloudResource(ctx context.Context, tenantID int, id int) (*CloudResource, error) {
	s.logger.Infow("Getting cloud resource", "id", id, "tenant_id", tenantID)
	result, err := s.repo.GetCloudResource(ctx, tenantID, id)
	if err != nil {
		s.logger.Errorw("Failed to get cloud resource", "error", err, "id", id, "tenant_id", tenantID)
		return nil, err
	}
	s.logger.Infow("Got cloud resource successfully", "id", id, "resource_id", result.ResourceID)
	return result, nil
}

func (s *Service) CreateCloudResource(ctx context.Context, cr *CloudResource) (*CloudResource, error) {
	s.logger.Infow("Creating cloud resource", "resource_id", cr.ResourceID, "service_id", cr.ServiceID, "tenant_id", cr.TenantID)
	result, err := s.repo.CreateCloudResource(ctx, cr)
	if err != nil {
		s.logger.Errorw("Failed to create cloud resource", "error", err, "resource_id", cr.ResourceID, "service_id", cr.ServiceID)
		return nil, err
	}
	s.logger.Infow("Cloud resource created successfully", "id", result.ID, "resource_id", result.ResourceID)
	return result, nil
}

func (s *Service) UpdateCloudResource(ctx context.Context, cr *CloudResource) (*CloudResource, error) {
	s.logger.Infow("Updating cloud resource", "id", cr.ID, "resource_id", cr.ResourceID, "tenant_id", cr.TenantID)
	result, err := s.repo.UpdateCloudResource(ctx, cr)
	if err != nil {
		s.logger.Errorw("Failed to update cloud resource", "error", err, "id", cr.ID, "resource_id", cr.ResourceID)
		return nil, err
	}
	s.logger.Infow("Cloud resource updated successfully", "id", result.ID, "resource_id", result.ResourceID)
	return result, nil
}

func (s *Service) DeleteCloudResource(ctx context.Context, id int, tenantID int) error {
	s.logger.Infow("Deleting cloud resource", "id", id, "tenant_id", tenantID)
	err := s.repo.DeleteCloudResource(ctx, id, tenantID)
	if err != nil {
		s.logger.Errorw("Failed to delete cloud resource", "error", err, "id", id, "tenant_id", tenantID)
		return err
	}
	s.logger.Infow("Cloud resource deleted successfully", "id", id, "tenant_id", tenantID)
	return nil
}

func (s *Service) GetReconciliation(ctx context.Context, tenantID int) (*ReconciliationResult, error) {
	resources, err := s.repo.ListCloudResources(ctx, tenantID, "", 0, "")
	if err != nil {
		return nil, err
	}
	cis, err := s.repo.ListCIsForReconciliation(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	resourceIndex := make(map[int]*CloudResource, len(resources))
	for _, res := range resources {
		resourceIndex[res.ID] = res
	}

	usedResources := make(map[int]struct{})
	var orphanCIs []*ConfigurationItem
	var unlinkedCIs []*ConfigurationItem

	for _, ci := range cis {
		if ci.CloudResourceRefID > 0 {
			if _, ok := resourceIndex[ci.CloudResourceRefID]; ok {
				usedResources[ci.CloudResourceRefID] = struct{}{}
			} else {
				orphanCIs = append(orphanCIs, ci)
			}
		} else if ci.CloudResourceID != "" {
			unlinkedCIs = append(unlinkedCIs, ci)
		}
	}

	var unboundResources []*CloudResource
	for _, res := range resources {
		if _, ok := usedResources[res.ID]; !ok {
			unboundResources = append(unboundResources, res)
		}
	}

	result := &ReconciliationResult{
		Summary: ReconciliationSummary{
			ResourceTotal:        len(resources),
			BoundResourceCount:   len(usedResources),
			UnboundResourceCount: len(unboundResources),
			OrphanCICount:        len(orphanCIs),
			UnlinkedCICount:      len(unlinkedCIs),
		},
		UnboundResources: unboundResources,
		OrphanCIs:        orphanCIs,
		UnlinkedCIs:      unlinkedCIs,
	}
	return result, nil
}

// Discovery
func (s *Service) CreateDiscoverySource(ctx context.Context, ds *DiscoverySource) (*DiscoverySource, error) {
	s.logger.Infow("Creating discovery source", "name", ds.Name, "source_type", ds.SourceType, "tenant_id", ds.TenantID)
	result, err := s.repo.CreateDiscoverySource(ctx, ds)
	if err != nil {
		s.logger.Errorw("Failed to create discovery source", "error", err, "name", ds.Name, "source_type", ds.SourceType)
		return nil, err
	}
	s.logger.Infow("Discovery source created successfully", "id", result.ID, "name", result.Name)
	return result, nil
}

func (s *Service) ListDiscoverySources(ctx context.Context, tenantID int) ([]*DiscoverySource, error) {
	s.logger.Infow("Listing discovery sources", "tenant_id", tenantID)
	result, err := s.repo.ListDiscoverySources(ctx, tenantID)
	if err != nil {
		s.logger.Errorw("Failed to list discovery sources", "error", err, "tenant_id", tenantID)
		return nil, err
	}
	s.logger.Infow("Listed discovery sources successfully", "count", len(result), "tenant_id", tenantID)
	return result, nil
}

func (s *Service) CreateDiscoveryJob(ctx context.Context, job *DiscoveryJob) (*DiscoveryJob, error) {
	s.logger.Infow("Creating discovery job", "source_id", job.SourceID, "tenant_id", job.TenantID)
	result, err := s.repo.CreateDiscoveryJob(ctx, job)
	if err != nil {
		s.logger.Errorw("Failed to create discovery job", "error", err, "source_id", job.SourceID)
		return nil, err
	}
	s.logger.Infow("Discovery job created successfully", "id", result.ID, "source_id", result.SourceID)
	return result, nil
}

func (s *Service) ListDiscoveryResults(ctx context.Context, tenantID int, jobID int) ([]*DiscoveryResult, error) {
	s.logger.Infow("Listing discovery results", "tenant_id", tenantID, "job_id", jobID)
	result, err := s.repo.ListDiscoveryResults(ctx, tenantID, jobID)
	if err != nil {
		s.logger.Errorw("Failed to list discovery results", "error", err, "tenant_id", tenantID, "job_id", jobID)
		return nil, err
	}
	s.logger.Infow("Listed discovery results successfully", "count", len(result), "tenant_id", tenantID)
	return result, nil
}
