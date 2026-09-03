package cmdb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"itsm-backend/common"
	"itsm-backend/ent"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Service struct {
	repo             Repository
	productionSvc    *ProductionService
	logger           *zap.SugaredLogger
	discoveryRuntime DiscoveryRuntime
}

type DiscoveryAdapterInspector interface {
	HasAdapter(provider, serviceCode string) bool
}

type DiscoveryRuntime struct {
	Adapters                DiscoveryAdapterInspector
	CredentialResolverReady bool
	WorkerReady             bool
}

func (s *Service) GetDiscoveryHealth() CapabilityStatus {
	adapterReady := s.discoveryRuntime.Adapters != nil && s.discoveryRuntime.Adapters.HasAdapter("aliyun", "ecs")
	missing := make([]string, 0, 3)
	if !adapterReady {
		missing = append(missing, "aliyunEcsAdapter")
	}
	if !s.discoveryRuntime.CredentialResolverReady {
		missing = append(missing, "tenantSecretResolver")
	}
	if !s.discoveryRuntime.WorkerReady {
		missing = append(missing, "discoveryWorker")
	}
	ready := adapterReady && s.discoveryRuntime.CredentialResolverReady && s.discoveryRuntime.WorkerReady
	state := "ready"
	if !ready {
		state = "unready"
	}
	return CapabilityStatus{Key: "cmdbDiscovery", State: state, BuildCapability: true, DeploymentReadiness: ready, ActorPermission: true, MissingRequirements: missing}
}

func NewService(repo Repository, productionSvc *ProductionService, logger *zap.SugaredLogger) *Service {
	return NewServiceWithDiscoveryRuntime(repo, productionSvc, logger, DiscoveryRuntime{})
}

func NewServiceWithDiscoveryRuntime(repo Repository, productionSvc *ProductionService, logger *zap.SugaredLogger, runtime DiscoveryRuntime) *Service {
	return &Service{
		repo:             repo,
		productionSvc:    productionSvc,
		logger:           logger,
		discoveryRuntime: runtime,
	}
}

// Legacy CMDB use cases remain owned by the production CMDB service aggregate.
// These methods are the migration bridge used by Handler while those use cases
// are progressively moved into the domain package.
func (s *Service) SearchCI(c *gin.Context) { s.productionSvc.SearchCI(c) }

func (s *Service) ListCIs(c *gin.Context) { s.productionSvc.ListCIs(c) }

func (s *Service) GetCI(c *gin.Context) { s.productionSvc.GetCI(c) }

func (s *Service) CreateCI(c *gin.Context) { s.productionSvc.CreateCI(c) }

func (s *Service) UpdateCI(c *gin.Context) { s.productionSvc.UpdateCI(c) }

func (s *Service) DeleteCI(c *gin.Context) { s.productionSvc.DeleteCI(c) }

func (s *Service) GetCIStats(c *gin.Context) { s.productionSvc.GetCIStats(c) }

func (s *Service) ListCITypes(c *gin.Context) { s.productionSvc.ListCITypes(c) }

func (s *Service) GetCIType(c *gin.Context) { s.productionSvc.GetCIType(c) }

func (s *Service) CreateCIType(c *gin.Context) { s.productionSvc.CreateCIType(c) }

func (s *Service) UpdateCIType(c *gin.Context) { s.productionSvc.UpdateCIType(c) }

func (s *Service) DeleteCIType(c *gin.Context) { s.productionSvc.DeleteCIType(c) }

func (s *Service) ListCIAttributeDefinitions(c *gin.Context) {
	s.productionSvc.ListCIAttributeDefinitions(c)
}

func (s *Service) GetCIAttributeDefinition(c *gin.Context) {
	s.productionSvc.GetCIAttributeDefinition(c)
}

func (s *Service) CreateCIAttributeDefinition(c *gin.Context) {
	s.productionSvc.CreateCIAttributeDefinition(c)
}

func (s *Service) UpdateCIAttributeDefinition(c *gin.Context) {
	s.productionSvc.UpdateCIAttributeDefinition(c)
}

func (s *Service) DeleteCIAttributeDefinition(c *gin.Context) {
	s.productionSvc.DeleteCIAttributeDefinition(c)
}

func (s *Service) ListCITags(c *gin.Context) { s.productionSvc.ListCITags(c) }

func (s *Service) GetCITag(c *gin.Context) { s.productionSvc.GetCITag(c) }

func (s *Service) CreateCITag(c *gin.Context) { s.productionSvc.CreateCITag(c) }

func (s *Service) UpdateCITag(c *gin.Context) { s.productionSvc.UpdateCITag(c) }

func (s *Service) DeleteCITag(c *gin.Context) { s.productionSvc.DeleteCITag(c) }

func (s *Service) ListSavedViews(c *gin.Context) { s.productionSvc.ListSavedViews(c) }

func (s *Service) GetSavedView(c *gin.Context) { s.productionSvc.GetSavedView(c) }

func (s *Service) CreateSavedView(c *gin.Context) { s.productionSvc.CreateSavedView(c) }

func (s *Service) UpdateSavedView(c *gin.Context) { s.productionSvc.UpdateSavedView(c) }

func (s *Service) DeleteSavedView(c *gin.Context) { s.productionSvc.DeleteSavedView(c) }

func (s *Service) ListImportTasks(c *gin.Context) { s.productionSvc.ListImportTasks(c) }

func (s *Service) GetImportTaskStatus(c *gin.Context) { s.productionSvc.GetImportTaskStatus(c) }

func (s *Service) CreateImportTask(c *gin.Context) { s.productionSvc.CreateImportTask(c) }

func (s *Service) ListExportTasks(c *gin.Context) { s.productionSvc.ListExportTasks(c) }

func (s *Service) GetExportTaskStatus(c *gin.Context) { s.productionSvc.GetExportTaskStatus(c) }

func (s *Service) CreateExportTask(c *gin.Context) { s.productionSvc.CreateExportTask(c) }

func (s *Service) ListCIRelationships(c *gin.Context) { s.productionSvc.ListCIRelationships(c) }

func (s *Service) GetCIRelationship(c *gin.Context) { s.productionSvc.GetCIRelationship(c) }

func (s *Service) CreateCIRelationship(c *gin.Context) { s.productionSvc.CreateCIRelationship(c) }

func (s *Service) UpdateCIRelationship(c *gin.Context) { s.productionSvc.UpdateCIRelationship(c) }

func (s *Service) DeleteCIRelationship(c *gin.Context) { s.productionSvc.DeleteCIRelationship(c) }

func (s *Service) ListCIRelationshipsByCIID(c *gin.Context) {
	s.productionSvc.ListCIRelationshipsByCIID(c)
}

func (s *Service) GetCITopology(c *gin.Context) { s.productionSvc.GetCITopology(c) }

func (s *Service) GetCIImpactAnalysis(c *gin.Context) { s.productionSvc.GetCIImpactAnalysis(c) }

func (s *Service) GetCIHistory(c *gin.Context) { s.productionSvc.GetCIHistory(c) }

func (s *Service) RevertCIVersion(c *gin.Context) { s.productionSvc.RevertCIVersion(c) }

func (s *Service) GetLifecycleHistory(c *gin.Context) { s.productionSvc.GetLifecycleHistory(c) }

func (s *Service) UpdateLifecycleStatus(c *gin.Context) { s.productionSvc.UpdateLifecycleStatus(c) }

func (s *Service) BatchUpdateLifecycleStatus(c *gin.Context) {
	s.productionSvc.BatchUpdateLifecycleStatus(c)
}

func (s *Service) AddTagsToCI(c *gin.Context) { s.productionSvc.AddTagsToCI(c) }

func (s *Service) RemoveTagsFromCI(c *gin.Context) { s.productionSvc.RemoveTagsFromCI(c) }

func (s *Service) BatchCreateCI(c *gin.Context) { s.productionSvc.BatchCreateCI(c) }

func (s *Service) BatchUpdateCI(c *gin.Context) { s.productionSvc.BatchUpdateCI(c) }

func (s *Service) BatchDeleteCI(c *gin.Context) { s.productionSvc.BatchDeleteCI(c) }

func (s *Service) ListRelationshipTypes(c *gin.Context) { s.productionSvc.ListRelationshipTypes(c) }

type CapabilityStatus struct {
	Key                 string   `json:"key"`
	State               string   `json:"state"`
	BuildCapability     bool     `json:"buildCapability"`
	DeploymentReadiness bool     `json:"deploymentReadiness"`
	TenantReadiness     bool     `json:"tenantReadiness"`
	ActorPermission     bool     `json:"actorPermission"`
	MissingRequirements []string `json:"missingRequirements"`
}

func (s *Service) GetDiscoveryCapability(ctx context.Context, tenantID int) (*CapabilityStatus, error) {
	if tenantID <= 0 {
		return nil, fmt.Errorf("tenant ID is required")
	}
	missing := make([]string, 0, 4)
	adapterReady := s.discoveryRuntime.Adapters != nil && s.discoveryRuntime.Adapters.HasAdapter("aliyun", "ecs")
	if !adapterReady {
		missing = append(missing, "aliyunEcsAdapter")
	}
	if !s.discoveryRuntime.CredentialResolverReady {
		missing = append(missing, "tenantSecretResolver")
	}
	if !s.discoveryRuntime.WorkerReady {
		missing = append(missing, "discoveryWorker")
	}
	accounts, err := s.repo.ListCloudAccounts(ctx, tenantID, "aliyun")
	if err != nil {
		return nil, err
	}
	tenantReady := false
	for _, account := range accounts {
		if account.IsActive && account.CredentialRef != "" {
			tenantReady = true
			break
		}
	}
	if !tenantReady {
		missing = append(missing, "tenantCloudAccount")
	}
	deploymentReady := adapterReady && s.discoveryRuntime.CredentialResolverReady && s.discoveryRuntime.WorkerReady
	state := "ready"
	switch {
	case !adapterReady:
		state = "disabled"
	case !deploymentReady:
		state = "unready"
	case !tenantReady:
		state = "unconfigured"
	}
	return &CapabilityStatus{
		Key: "cmdbDiscovery", State: state, BuildCapability: true,
		DeploymentReadiness: deploymentReady, TenantReadiness: tenantReady,
		ActorPermission: true, MissingRequirements: missing,
	}, nil
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
	if err := s.prepareCloudResourceIdentity(ctx, cr); err != nil {
		return nil, err
	}
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
	if err := s.prepareCloudResourceIdentity(ctx, cr); err != nil {
		return nil, err
	}
	result, err := s.repo.UpdateCloudResource(ctx, cr)
	if err != nil {
		s.logger.Errorw("Failed to update cloud resource", "error", err, "id", cr.ID, "resource_id", cr.ResourceID)
		return nil, err
	}
	s.logger.Infow("Cloud resource updated successfully", "id", result.ID, "resource_id", result.ResourceID)
	return result, nil
}

func (s *Service) prepareCloudResourceIdentity(ctx context.Context, resource *CloudResource) error {
	if resource == nil || resource.TenantID <= 0 || resource.CloudAccountID <= 0 || resource.ServiceID <= 0 || strings.TrimSpace(resource.ResourceID) == "" {
		return fmt.Errorf("cloud resource identity fields are required")
	}
	account, err := s.repo.GetCloudAccount(ctx, resource.TenantID, resource.CloudAccountID)
	if err != nil {
		if ent.IsNotFound(err) {
			return common.NewNotFoundError("cloud account")
		}
		return fmt.Errorf("load tenant cloud account: %w", err)
	}
	service, err := s.repo.GetCloudService(ctx, resource.TenantID, resource.ServiceID)
	if err != nil {
		if ent.IsNotFound(err) {
			return common.NewNotFoundError("cloud service")
		}
		return fmt.Errorf("load tenant cloud service: %w", err)
	}
	if account == nil || service == nil || account.TenantID != resource.TenantID || service.TenantID != resource.TenantID {
		return common.NewForbiddenError("cloud resource references must belong to the authenticated tenant")
	}
	provider := strings.ToLower(strings.TrimSpace(account.Provider))
	if provider == "" || provider != strings.ToLower(strings.TrimSpace(service.Provider)) {
		return fmt.Errorf("cloud account and service provider mismatch")
	}
	region := strings.ToLower(strings.TrimSpace(resource.Region))
	zone := strings.ToLower(strings.TrimSpace(resource.Zone))
	scope := "global"
	if zone != "" {
		scope = "zonal"
	} else if region != "" {
		scope = "regional"
	}
	resource.IdentityVersion = 1
	resource.Provider = provider
	resource.Partition = "public"
	resource.CanonicalAccountID = strings.TrimSpace(account.AccountID)
	resource.ResourceScope = scope
	resource.Region = region
	resource.Zone = zone
	resource.ServiceCode = strings.ToLower(strings.TrimSpace(service.ServiceCode))
	resource.ResourceType = strings.ToLower(strings.TrimSpace(service.ResourceTypeCode))
	resource.ResourceID = strings.TrimSpace(resource.ResourceID)
	identity := strings.Join([]string{
		"v1", fmt.Sprint(resource.TenantID), resource.Provider, resource.Partition,
		resource.CanonicalAccountID, resource.ResourceScope, resource.Region,
		resource.ServiceCode, resource.ResourceType, resource.ResourceID,
	}, "|")
	hash := sha256.Sum256([]byte(identity))
	resource.IdentityHash = hex.EncodeToString(hash[:])
	return nil
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
	if ds == nil || ds.TenantID <= 0 {
		return nil, fmt.Errorf("tenant-scoped discovery source is required")
	}
	s.logger.Infow("Creating discovery source", "name", ds.Name, "source_type", ds.SourceType, "tenant_id", ds.TenantID)
	if ds.ReconcilePolicy == "" {
		ds.ReconcilePolicy = "manual"
	}
	if ds.StaleThreshold == 0 {
		ds.StaleThreshold = 3
	}
	if ds.CloudAccountID > 0 {
		account, err := s.repo.GetCloudAccount(ctx, ds.TenantID, ds.CloudAccountID)
		if err != nil {
			if ent.IsNotFound(err) {
				return nil, common.NewNotFoundError("cloud account")
			}
			return nil, fmt.Errorf("load tenant cloud account: %w", err)
		}
		if account == nil || account.TenantID != ds.TenantID || !account.IsActive || account.CredentialRef == "" {
			return nil, common.NewForbiddenError("cloud account must belong to the tenant and be configured and active")
		}
		if ds.Provider != "" && !strings.EqualFold(strings.TrimSpace(ds.Provider), strings.TrimSpace(account.Provider)) {
			return nil, fmt.Errorf("discovery source provider does not match cloud account")
		}
		ds.Provider = strings.ToLower(strings.TrimSpace(account.Provider))
	}
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
