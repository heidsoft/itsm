package service

import (
	"context"
	"fmt"
	"strings"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/configurationitem"

	"go.uber.org/zap"
)

// ConfigurationItemService 配置项服务
type ConfigurationItemService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewConfigurationItemService 创建配置项服务
func NewConfigurationItemService(client *ent.Client, logger *zap.SugaredLogger) *ConfigurationItemService {
	return &ConfigurationItemService{
		client: client,
		logger: logger,
	}
}

// CreateCI 创建配置项
func (s *ConfigurationItemService) CreateCI(ctx context.Context, req *dto.CreateCIRequest, tenantID int) (*dto.CIResponse, error) {
	// 首先获取CI类型
	ciType, err := s.client.CIType.Query().
		Where(ent.Citype.IDEQ(req.CITypeID), ent.Citype.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("CI type not found")
		}
		s.logger.Errorw("Failed to get CI type", "error", err, "ci_type_id", req.CITypeID)
		return nil, fmt.Errorf("failed to get CI type: %w", err)
	}

	// 创建配置项
	create := s.client.ConfigurationItem.Create().
		SetName(req.Name).
		SetCiTypeID(req.CITypeID).
		SetCiType(ciType.Name).
		SetStatus(req.Status).
		SetTenantID(tenantID)

	if req.Environment != "" {
		create.SetEnvironment(req.Environment)
	}
	if req.Criticality != "" {
		create.SetCriticality(req.Criticality)
	}
	if req.AssetTag != "" {
		create.SetAssetTag(req.AssetTag)
	}
	if req.SerialNumber != "" {
		create.SetSerialNumber(req.SerialNumber)
	}
	if req.Model != "" {
		create.SetModel(req.Model)
	}
	if req.Vendor != "" {
		create.SetVendor(req.Vendor)
	}
	if req.Location != "" {
		create.SetLocation(req.Location)
	}
	if req.AssignedTo != "" {
		create.SetAssignedTo(req.AssignedTo)
	}
	if req.OwnedBy != "" {
		create.SetOwnedBy(req.OwnedBy)
	}
	if req.DiscoverySource != "" {
		create.SetDiscoverySource(req.DiscoverySource)
	}
	if req.Source != "" {
		create.SetSource(req.Source)
	}
	if req.Attributes != nil {
		create.SetAttributes(req.Attributes)
	}
	if req.CloudProvider != "" {
		create.SetCloudProvider(req.CloudProvider)
	}
	if req.CloudAccountID != "" {
		create.SetCloudAccountID(req.CloudAccountID)
	}
	if req.CloudRegion != "" {
		create.SetCloudRegion(req.CloudRegion)
	}
	if req.CloudZone != "" {
		create.SetCloudZone(req.CloudZone)
	}
	if req.CloudResourceID != "" {
		create.SetCloudResourceID(req.CloudResourceID)
	}
	if req.CloudResourceType != "" {
		create.SetCloudResourceType(req.CloudResourceType)
	}
	if req.CloudMetadata != nil {
		create.SetCloudMetadata(req.CloudMetadata)
	}
	if req.CloudTags != nil {
		create.SetCloudTags(req.CloudTags)
	}
	if req.CloudMetrics != nil {
		create.SetCloudMetrics(req.CloudMetrics)
	}
	if req.CloudSyncTime != nil {
		create.SetCloudSyncTime(*req.CloudSyncTime)
	}
	if req.CloudSyncStatus != "" {
		create.SetCloudSyncStatus(req.CloudSyncStatus)
	}
	if req.CloudResourceRefID != 0 {
		create.SetCloudResourceRefID(req.CloudResourceRefID)
	}

	ci, err := create.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create configuration item", "error", err, "tenant_id", tenantID, "name", req.Name)
		return nil, fmt.Errorf("failed to create configuration item: %w", err)
	}

	s.logger.Infow("Configuration item created successfully", "ci_id", ci.ID, "tenant_id", tenantID, "name", ci.Name)
	return dto.ToCIResponse(ci), nil
}

// GetCIByID 根据ID获取配置项
func (s *ConfigurationItemService) GetCIByID(ctx context.Context, id, tenantID int, withRelations bool) (*dto.CIResponse, error) {
	query := s.client.ConfigurationItem.Query().
		Where(configurationitem.IDEQ(id), configurationitem.TenantIDEQ(tenantID))

	if withRelations {
		query = query.WithOutgoingRelations(func(q *ent.CIRelationshipQuery) {
			q.WithTargetCI()
		}).WithIncomingRelations(func(q *ent.CIRelationshipQuery) {
			q.WithSourceCI()
		})
	}

	ci, err := query.First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		s.logger.Errorw("Failed to get configuration item", "error", err, "ci_id", id)
		return nil, fmt.Errorf("failed to get configuration item: %w", err)
	}

	return dto.ToCIResponseWithRelations(ci), nil
}

// ListCIs 获取配置项列表
func (s *ConfigurationItemService) ListCIs(ctx context.Context, tenantID int, req *dto.ListCIRequest) (*dto.CIListResponse, error) {
	query := s.client.ConfigurationItem.Query().
		Where(configurationitem.TenantIDEQ(tenantID))

	// 筛选条件
	if req.CITypeID != 0 {
		query = query.Where(configurationitem.CiTypeIDEQ(req.CITypeID))
	}
	if req.Status != "" {
		query = query.Where(configurationitem.StatusEQ(req.Status))
	}
	if req.Environment != "" {
		query = query.Where(configurationitem.EnvironmentEQ(req.Environment))
	}
	if req.Criticality != "" {
		query = query.Where(configurationitem.CriticalityEQ(req.Criticality))
	}
	if req.CloudProvider != "" {
		query = query.Where(configurationitem.CloudProviderEQ(req.CloudProvider))
	}
	if req.CloudAccountID != "" {
		query = query.Where(configurationitem.CloudAccountIDEQ(req.CloudAccountID))
	}
	if req.CloudRegion != "" {
		query = query.Where(configurationitem.CloudRegionEQ(req.CloudRegion))
	}
	if req.AssignedTo != "" {
		query = query.Where(configurationitem.AssignedToEQ(req.AssignedTo))
	}
	if req.OwnedBy != "" {
		query = query.Where(configurationitem.OwnedByEQ(req.OwnedBy))
	}

	// 搜索
	if req.Search != "" {
		search := fmt.Sprintf("%%%s%%", strings.ToLower(req.Search))
		query = query.Where(
			ent.Or(
				configurationitem.NameContainsFold(search),
				configurationitem.AssetTagContainsFold(search),
				configurationitem.SerialNumberContainsFold(search),
				configurationitem.ModelContainsFold(search),
				configurationitem.VendorContainsFold(search),
				configurationitem.CloudResourceIDContainsFold(search),
			),
		)
	}

	// 统计总数
	total, err := query.Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count configuration items", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("failed to count configuration items: %w", err)
	}

	// 分页查询
	ciList, err := query.
		Offset((req.Page - 1) * req.Size).
		Limit(req.Size).
		Order(ent.Desc(configurationitem.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list configuration items", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("failed to list configuration items: %w", err)
	}

	return &dto.CIListResponse{
		Items: dto.ToCIResponseList(ciList),
		Total: total,
		Page:  req.Page,
		Size:  req.Size,
	}, nil
}

// UpdateCI 更新配置项
func (s *ConfigurationItemService) UpdateCI(ctx context.Context, id, tenantID int, req *dto.UpdateCIRequest) (*dto.CIResponse, error) {
	update := s.client.ConfigurationItem.UpdateOneID(id).
		Where(configurationitem.TenantIDEQ(tenantID))

	if req.Name != nil {
		update.SetName(*req.Name)
	}
	if req.Status != nil {
		update.SetStatus(*req.Status)
	}
	if req.Environment != nil {
		update.SetEnvironment(*req.Environment)
	}
	if req.Criticality != nil {
		update.SetCriticality(*req.Criticality)
	}
	if req.AssetTag != nil {
		update.SetAssetTag(*req.AssetTag)
	}
	if req.SerialNumber != nil {
		update.SetSerialNumber(*req.SerialNumber)
	}
	if req.Model != nil {
		update.SetModel(*req.Model)
	}
	if req.Vendor != nil {
		update.SetVendor(*req.Vendor)
	}
	if req.Location != nil {
		update.SetLocation(*req.Location)
	}
	if req.AssignedTo != nil {
		update.SetAssignedTo(*req.AssignedTo)
	}
	if req.OwnedBy != nil {
		update.SetOwnedBy(*req.OwnedBy)
	}
	if req.DiscoverySource != nil {
		update.SetDiscoverySource(*req.DiscoverySource)
	}
	if req.Source != nil {
		update.SetSource(*req.Source)
	}
	if req.Attributes != nil {
		update.SetAttributes(req.Attributes)
	}
	if req.CloudProvider != nil {
		update.SetCloudProvider(*req.CloudProvider)
	}
	if req.CloudAccountID != nil {
		update.SetCloudAccountID(*req.CloudAccountID)
	}
	if req.CloudRegion != nil {
		update.SetCloudRegion(*req.CloudRegion)
	}
	if req.CloudZone != nil {
		update.SetCloudZone(*req.CloudZone)
	}
	if req.CloudResourceID != nil {
		update.SetCloudResourceID(*req.CloudResourceID)
	}
	if req.CloudResourceType != nil {
		update.SetCloudResourceType(*req.CloudResourceType)
	}
	if req.CloudMetadata != nil {
		update.SetCloudMetadata(req.CloudMetadata)
	}
	if req.CloudTags != nil {
		update.SetCloudTags(req.CloudTags)
	}
	if req.CloudMetrics != nil {
		update.SetCloudMetrics(req.CloudMetrics)
	}
	if req.CloudSyncTime != nil {
		update.SetCloudSyncTime(*req.CloudSyncTime)
	}
	if req.CloudSyncStatus != nil {
		update.SetCloudSyncStatus(*req.CloudSyncStatus)
	}
	if req.CloudResourceRefID != nil {
		update.SetCloudResourceRefID(*req.CloudResourceRefID)
	}
	if req.CITypeID != nil {
		// 如果更新了CI类型，需要同步更新CiType字段
		ciType, err := s.client.CIType.Query().
			Where(ent.Citype.IDEQ(*req.CITypeID), ent.Citype.TenantIDEQ(tenantID)).
			First(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return nil, fmt.Errorf("CI type not found")
			}
			s.logger.Errorw("Failed to get CI type", "error", err, "ci_type_id", *req.CITypeID)
			return nil, fmt.Errorf("failed to get CI type: %w", err)
		}
		update.SetCiTypeID(*req.CITypeID)
		update.SetCiType(ciType.Name)
	}

	ci, err := update.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update configuration item", "error", err, "ci_id", id)
		return nil, fmt.Errorf("failed to update configuration item: %w", err)
	}

	s.logger.Infow("Configuration item updated successfully", "ci_id", ci.ID, "tenant_id", tenantID)
	return dto.ToCIResponse(ci), nil
}

// DeleteCI 删除配置项
func (s *ConfigurationItemService) DeleteCI(ctx context.Context, id, tenantID int) error {
	// 检查是否有关联的关系
	outgoingCount, err := s.client.CIRelationship.Query().
		Where(ent.Cirelationship.SourceCIIDEQ(id), ent.Cirelationship.TenantIDEQ(tenantID)).
		Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to check outgoing relations", "error", err, "ci_id", id)
		return fmt.Errorf("failed to check outgoing relations: %w", err)
	}

	incomingCount, err := s.client.CIRelationship.Query().
		Where(ent.Cirelationship.TargetCIIDEQ(id), ent.Cirelationship.TenantIDEQ(tenantID)).
		Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to check incoming relations", "error", err, "ci_id", id)
		return fmt.Errorf("failed to check incoming relations: %w", err)
	}

	if outgoingCount+incomingCount > 0 {
		// 删除所有关联的关系
		_, err := s.client.CIRelationship.Delete().
			Where(
				ent.Or(
					ent.Cirelationship.SourceCIIDEQ(id),
					ent.Cirelationship.TargetCIIDEQ(id),
				),
				ent.Cirelationship.TenantIDEQ(tenantID),
			).
			Exec(ctx)
		if err != nil {
			s.logger.Errorw("Failed to delete related relations", "error", err, "ci_id", id)
			return fmt.Errorf("failed to delete related relations: %w", err)
		}
		s.logger.Infow("Deleted related relations for CI", "ci_id", id, "count", outgoingCount+incomingCount)
	}

	// 删除配置项
	err = s.client.ConfigurationItem.DeleteOneID(id).
		Where(configurationitem.TenantIDEQ(tenantID)).
		Exec(ctx)
	if err != nil {
		s.logger.Errorw("Failed to delete configuration item", "error", err, "ci_id", id)
		return fmt.Errorf("failed to delete configuration item: %w", err)
	}

	s.logger.Infow("Configuration item deleted successfully", "ci_id", id, "tenant_id", tenantID)
	return nil
}

// GetCIStats 获取配置项统计
func (s *ConfigurationItemService) GetCIStats(ctx context.Context, tenantID int) (*dto.CIStatsResponse, error) {
	// 总数量
	total, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.TenantIDEQ(tenantID)).
		Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count total CIs", "error", err)
		return nil, fmt.Errorf("failed to count total CIs: %w", err)
	}

	// 按状态统计
	statusStats, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.TenantIDEQ(tenantID)).
		GroupBy(configurationitem.FieldStatus).
		Aggregate(ent.Count()).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get status stats", "error", err)
		return nil, fmt.Errorf("failed to get status stats: %w", err)
	}

	// 按类型统计
	typeStats, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.TenantIDEQ(tenantID)).
		GroupBy(configurationitem.FieldCiType).
		Aggregate(ent.Count()).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get type stats", "error", err)
		return nil, fmt.Errorf("failed to get type stats: %w", err)
	}

	// 按环境统计
	envStats, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.TenantIDEQ(tenantID)).
		GroupBy(configurationitem.FieldEnvironment).
		Aggregate(ent.Count()).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get environment stats", "error", err)
		return nil, fmt.Errorf("failed to get environment stats: %w", err)
	}

	// 按重要性统计
	criticalityStats, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.TenantIDEQ(tenantID)).
		GroupBy(configurationitem.FieldCriticality).
		Aggregate(ent.Count()).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get criticality stats", "error", err)
		return nil, fmt.Errorf("failed to get criticality stats: %w", err)
	}

	// 构建响应
	response := &dto.CIStatsResponse{
		TotalCount:           total,
		StatusDistribution:   make(map[string]int),
		TypeDistribution:     make(map[string]int),
		EnvironmentDistribution: make(map[string]int),
		CriticalityDistribution: make(map[string]int),
	}

	for _, stat := range statusStats {
		status, ok := stat.Fields[configurationitem.FieldStatus].(string)
		if ok {
			response.StatusDistribution[status] = stat.Count
		}
	}

	for _, stat := range typeStats {
		ciType, ok := stat.Fields[configurationitem.FieldCiType].(string)
		if ok {
			response.TypeDistribution[ciType] = stat.Count
		}
	}

	for _, stat := range envStats {
		env, ok := stat.Fields[configurationitem.FieldEnvironment].(string)
		if ok && env != "" {
			response.EnvironmentDistribution[env] = stat.Count
		}
	}

	for _, stat := range criticalityStats {
		criticality, ok := stat.Fields[configurationitem.FieldCriticality].(string)
		if ok && criticality != "" {
			response.CriticalityDistribution[criticality] = stat.Count
		}
	}

	return response, nil
}
