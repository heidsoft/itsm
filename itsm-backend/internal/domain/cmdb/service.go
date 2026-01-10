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

// CI operations
func (s *Service) CreateCI(ctx context.Context, ci *ConfigurationItem) (*ConfigurationItem, error) {
	s.logger.Infow("Creating CI", "name", ci.Name, "type_id", ci.CITypeID)
	return s.repo.CreateCI(ctx, ci)
}

func (s *Service) GetCI(ctx context.Context, id int, tenantID int) (*ConfigurationItem, error) {
	return s.repo.GetCI(ctx, id, tenantID)
}

func (s *Service) ListCIs(ctx context.Context, tenantID int, page, size int, ciTypeID int, status string) ([]*ConfigurationItem, int, error) {
	return s.repo.ListCIs(ctx, tenantID, page, size, ciTypeID, status)
}

func (s *Service) UpdateCI(ctx context.Context, ci *ConfigurationItem) (*ConfigurationItem, error) {
	return s.repo.UpdateCI(ctx, ci)
}

func (s *Service) DeleteCI(ctx context.Context, id int, tenantID int) error {
	return s.repo.DeleteCI(ctx, id, tenantID)
}

func (s *Service) GetStats(ctx context.Context, tenantID int) (*Stats, error) {
	return s.repo.GetStats(ctx, tenantID)
}

// CIType operations
func (s *Service) ListTypes(ctx context.Context, tenantID int) ([]*CIType, error) {
	return s.repo.ListCITypes(ctx, tenantID)
}

func (s *Service) CreateType(ctx context.Context, ct *CIType) (*CIType, error) {
	return s.repo.CreateCIType(ctx, ct)
}

// Relationship operations
func (s *Service) CreateRelationship(ctx context.Context, rel *CIRelationship) (*CIRelationship, error) {
	return s.repo.CreateRelationship(ctx, rel)
}

func (s *Service) GetCIRelationships(ctx context.Context, ciID int) ([]*CIRelationship, error) {
	return s.repo.GetRelationships(ctx, ciID)
}

// Relationship types
func (s *Service) ListRelationshipTypes(ctx context.Context, tenantID int) ([]*RelationshipType, error) {
	return s.repo.ListRelationshipTypes(ctx, tenantID)
}

// Cloud services
func (s *Service) CreateCloudService(ctx context.Context, cs *CloudService) (*CloudService, error) {
	return s.repo.CreateCloudService(ctx, cs)
}

func (s *Service) ListCloudServices(ctx context.Context, tenantID int, provider string) ([]*CloudService, error) {
	return s.repo.ListCloudServices(ctx, tenantID, provider)
}

// Cloud accounts
func (s *Service) CreateCloudAccount(ctx context.Context, ca *CloudAccount) (*CloudAccount, error) {
	return s.repo.CreateCloudAccount(ctx, ca)
}

func (s *Service) ListCloudAccounts(ctx context.Context, tenantID int, provider string) ([]*CloudAccount, error) {
	return s.repo.ListCloudAccounts(ctx, tenantID, provider)
}

// Cloud resources
func (s *Service) ListCloudResources(ctx context.Context, tenantID int, provider string, serviceID int, region string) ([]*CloudResource, error) {
	return s.repo.ListCloudResources(ctx, tenantID, provider, serviceID, region)
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
	return s.repo.CreateDiscoverySource(ctx, ds)
}

func (s *Service) ListDiscoverySources(ctx context.Context, tenantID int) ([]*DiscoverySource, error) {
	return s.repo.ListDiscoverySources(ctx, tenantID)
}

func (s *Service) CreateDiscoveryJob(ctx context.Context, job *DiscoveryJob) (*DiscoveryJob, error) {
	return s.repo.CreateDiscoveryJob(ctx, job)
}

func (s *Service) ListDiscoveryResults(ctx context.Context, tenantID int, jobID int) ([]*DiscoveryResult, error) {
	return s.repo.ListDiscoveryResults(ctx, tenantID, jobID)
}
