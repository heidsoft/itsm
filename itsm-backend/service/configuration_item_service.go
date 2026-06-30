package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/cirelationship"
	"itsm-backend/ent/citag"
	"itsm-backend/ent/citype"
	"itsm-backend/ent/configurationitem"

	"go.uber.org/zap"
)

// ConfigurationItemService 配置项服务
type ConfigurationItemService struct {
	client          *ent.Client
	logger          *zap.SugaredLogger
	historyService  *CIHistoryService
	tagService      *CITagService
}

// NewConfigurationItemService 创建配置项服务
func NewConfigurationItemService(client *ent.Client, logger *zap.SugaredLogger, historyService *CIHistoryService, tagService *CITagService) *ConfigurationItemService {
	return &ConfigurationItemService{
		client:         client,
		logger:         logger,
		historyService: historyService,
		tagService:     tagService,
	}
}

// CreateCI 创建配置项
func (s *ConfigurationItemService) CreateCI(ctx context.Context, req *dto.CreateCIRequest, tenantID int) (*dto.CIResponse, error) {
	ciTypeID := req.CITypeID
	if ciTypeID == 0 {
		ciTypeID = req.CITypeIDSnake
	}
	if ciTypeID == 0 {
		return nil, fmt.Errorf("CI type id is required")
	}

	// 首先获取CI类型
	ciType, err := s.client.CIType.Query().
		Where(citype.IDEQ(ciTypeID), citype.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("CI type not found")
		}
		s.logger.Errorw("Failed to get CI type", "error", err, "ci_type_id", ciTypeID)
		return nil, fmt.Errorf("failed to get CI type: %w", err)
	}

	// 创建配置项
	create := s.client.ConfigurationItem.Create().
		SetName(req.Name).
		SetCiTypeID(ciTypeID).
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
	if err == nil {
		// 记录创建历史
		operatorID, _ := ctx.Value("user_id").(int)
		operatorName, _ := ctx.Value("user_name").(string)
		_ = s.historyService.RecordCIHistory(ctx, ci.ID, tenantID, operatorID, operatorName, "create", "", nil, ci)
	}
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
			q.WithTargetCi()
		}).WithIncomingRelations(func(q *ent.CIRelationshipQuery) {
			q.WithSourceCi()
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
		search := strings.TrimSpace(req.Search)
		query = query.Where(
			configurationitem.Or(
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
	// 查询更新前的CI数据
	oldCI, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.IDEQ(id), configurationitem.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("CI not found")
		}
		s.logger.Errorw("Failed to get old CI for update", "error", err, "ci_id", id)
		return nil, fmt.Errorf("failed to get CI: %w", err)
	}

	update := s.client.ConfigurationItem.UpdateOneID(id).
		Where(configurationitem.TenantIDEQ(tenantID))

	if req.Name != "" {
		update.SetName(req.Name)
	}
	if req.Status != "" {
		update.SetStatus(req.Status)
	}
	if req.Environment != "" {
		update.SetEnvironment(req.Environment)
	}
	if req.Criticality != "" {
		update.SetCriticality(req.Criticality)
	}
	if req.AssetTag != "" {
		update.SetAssetTag(req.AssetTag)
	}
	if req.SerialNumber != "" {
		update.SetSerialNumber(req.SerialNumber)
	}
	if req.Model != "" {
		update.SetModel(req.Model)
	}
	if req.Vendor != "" {
		update.SetVendor(req.Vendor)
	}
	if req.Location != "" {
		update.SetLocation(req.Location)
	}
	if req.AssignedTo != "" {
		update.SetAssignedTo(req.AssignedTo)
	}
	if req.OwnedBy != "" {
		update.SetOwnedBy(req.OwnedBy)
	}
	if req.DiscoverySource != "" {
		update.SetDiscoverySource(req.DiscoverySource)
	}
	if req.Source != "" {
		update.SetSource(req.Source)
	}
	if req.Attributes != nil {
		update.SetAttributes(req.Attributes)
	}
	if req.CloudProvider != "" {
		update.SetCloudProvider(req.CloudProvider)
	}
	if req.CloudAccountID != "" {
		update.SetCloudAccountID(req.CloudAccountID)
	}
	if req.CloudRegion != "" {
		update.SetCloudRegion(req.CloudRegion)
	}
	if req.CloudZone != "" {
		update.SetCloudZone(req.CloudZone)
	}
	if req.CloudResourceID != "" {
		update.SetCloudResourceID(req.CloudResourceID)
	}
	if req.CloudResourceType != "" {
		update.SetCloudResourceType(req.CloudResourceType)
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
	if req.CloudSyncStatus != "" {
		update.SetCloudSyncStatus(req.CloudSyncStatus)
	}
	if req.CloudResourceRefID != 0 {
		update.SetCloudResourceRefID(req.CloudResourceRefID)
	}
	ciTypeID := req.CITypeID
	if ciTypeID == 0 {
		ciTypeID = req.CITypeIDSnake
	}
	if ciTypeID != 0 {
		// 如果更新了CI类型，需要同步更新CiType字段
		ciType, err := s.client.CIType.Query().
			Where(citype.IDEQ(ciTypeID), citype.TenantIDEQ(tenantID)).
			First(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return nil, fmt.Errorf("CI type not found")
			}
			s.logger.Errorw("Failed to get CI type", "error", err, "ci_type_id", ciTypeID)
			return nil, fmt.Errorf("failed to get CI type: %w", err)
		}
		update.SetCiTypeID(ciTypeID)
		update.SetCiType(ciType.Name)
	}

	ci, err := update.Save(ctx)
	if err == nil {
		// 记录更新历史
		operatorID, _ := ctx.Value("user_id").(int)
		operatorName, _ := ctx.Value("user_name").(string)
		_ = s.historyService.RecordCIHistory(ctx, ci.ID, tenantID, operatorID, operatorName, "update", "", oldCI, ci)
	}
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
	// 查询要删除的CI数据
	ci, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.IDEQ(id), configurationitem.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("CI not found")
		}
		s.logger.Errorw("Failed to get CI for delete", "error", err, "ci_id", id)
		return fmt.Errorf("failed to get CI: %w", err)
	}

	outgoingCount, err := s.client.CIRelationship.Query().
		Where(cirelationship.SourceCiIDEQ(id), cirelationship.TenantIDEQ(tenantID)).
		Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to check outgoing relations", "error", err, "ci_id", id)
		return fmt.Errorf("failed to check outgoing relations: %w", err)
	}

	incomingCount, err := s.client.CIRelationship.Query().
		Where(cirelationship.TargetCiIDEQ(id), cirelationship.TenantIDEQ(tenantID)).
		Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to check incoming relations", "error", err, "ci_id", id)
		return fmt.Errorf("failed to check incoming relations: %w", err)
	}

	if outgoingCount+incomingCount > 0 {
		// 删除所有关联的关系
		_, err := s.client.CIRelationship.Delete().
			Where(
				cirelationship.Or(
					cirelationship.SourceCiIDEQ(id),
					cirelationship.TargetCiIDEQ(id),
				),
				cirelationship.TenantIDEQ(tenantID),
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

	if err == nil {
		// 记录删除历史
		operatorID, _ := ctx.Value("user_id").(int)
		operatorName, _ := ctx.Value("user_name").(string)
		_ = s.historyService.RecordCIHistory(ctx, id, tenantID, operatorID, operatorName, "delete", "", ci, nil)
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

	type statusCount struct {
		Status string `json:"status"`
		Count  int    `json:"count"`
	}
	type ciTypeCount struct {
		CiType string `json:"ci_type"`
		Count  int    `json:"count"`
	}
	type environmentCount struct {
		Environment string `json:"environment"`
		Count       int    `json:"count"`
	}
	type criticalityCount struct {
		Criticality string `json:"criticality"`
		Count       int    `json:"count"`
	}

	// 按状态统计
	var statusStats []statusCount
	err = s.client.ConfigurationItem.Query().
		Where(configurationitem.TenantIDEQ(tenantID)).
		GroupBy(configurationitem.FieldStatus).
		Aggregate(ent.Count()).
		Scan(ctx, &statusStats)
	if err != nil {
		s.logger.Errorw("Failed to get status stats", "error", err)
		return nil, fmt.Errorf("failed to get status stats: %w", err)
	}

	// 按类型统计
	var typeStats []ciTypeCount
	err = s.client.ConfigurationItem.Query().
		Where(configurationitem.TenantIDEQ(tenantID)).
		GroupBy(configurationitem.FieldCiType).
		Aggregate(ent.Count()).
		Scan(ctx, &typeStats)
	if err != nil {
		s.logger.Errorw("Failed to get type stats", "error", err)
		return nil, fmt.Errorf("failed to get type stats: %w", err)
	}

	// 按环境统计
	var envStats []environmentCount
	err = s.client.ConfigurationItem.Query().
		Where(configurationitem.TenantIDEQ(tenantID)).
		GroupBy(configurationitem.FieldEnvironment).
		Aggregate(ent.Count()).
		Scan(ctx, &envStats)
	if err != nil {
		s.logger.Errorw("Failed to get environment stats", "error", err)
		return nil, fmt.Errorf("failed to get environment stats: %w", err)
	}

	// 按重要性统计
	var criticalityStats []criticalityCount
	err = s.client.ConfigurationItem.Query().
		Where(configurationitem.TenantIDEQ(tenantID)).
		GroupBy(configurationitem.FieldCriticality).
		Aggregate(ent.Count()).
		Scan(ctx, &criticalityStats)
	if err != nil {
		s.logger.Errorw("Failed to get criticality stats", "error", err)
		return nil, fmt.Errorf("failed to get criticality stats: %w", err)
	}

	// 构建响应
	response := &dto.CIStatsResponse{
		TotalCount:              total,
		StatusDistribution:      make(map[string]int),
		TypeDistribution:        make(map[string]int),
		EnvironmentDistribution: make(map[string]int),
		CriticalityDistribution: make(map[string]int),
	}

	for _, stat := range statusStats {
		if stat.Status != "" {
			response.StatusDistribution[stat.Status] = stat.Count
		}
	}

	for _, stat := range typeStats {
		if stat.CiType != "" {
			response.TypeDistribution[stat.CiType] = stat.Count
		}
	}

	for _, stat := range envStats {
		if stat.Environment != "" {
			response.EnvironmentDistribution[stat.Environment] = stat.Count
		}
	}

	for _, stat := range criticalityStats {
		if stat.Criticality != "" {
			response.CriticalityDistribution[stat.Criticality] = stat.Count
		}
	}

	return response, nil
}

// AddTagsToCI 给CI添加标签
func (s *ConfigurationItemService) AddTagsToCI(ctx context.Context, ciID, tenantID int, tagIDs []int) (*dto.CIResponse, error) {
	// 检查CI是否存在
	ci, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.IDEQ(ciID), configurationitem.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("CI not found")
		}
		s.logger.Errorw("Failed to get CI for adding tags", "error", err, "ci_id", ciID)
		return nil, fmt.Errorf("failed to get CI: %w", err)
	}

	// 检查标签是否都存在且属于当前租户
	tags, err := s.client.CITag.Query().
		Where(
			citag.IDIn(tagIDs...),
			citag.TenantIDEQ(tenantID),
		).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get tags", "error", err, "tag_ids", tagIDs)
		return nil, fmt.Errorf("failed to get tags: %w", err)
	}
	if len(tags) != len(tagIDs) {
		return nil, fmt.Errorf("some tags not found or not belong to current tenant")
	}

	// 添加标签关联
	err = s.client.ConfigurationItem.UpdateOneID(ciID).
		AddTagIDs(tagIDs...).
		Exec(ctx)
	if err != nil {
		s.logger.Errorw("Failed to add tags to CI", "error", err, "ci_id", ciID, "tag_ids", tagIDs)
		return nil, fmt.Errorf("failed to add tags: %w", err)
	}

	// 重新加载CI数据
	updatedCI, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.IDEQ(ciID)).
		WithOutgoingRelations().
		WithIncomingRelations().
		WithTags().
		First(ctx)
	if err != nil {
		s.logger.Errorw("Failed to reload CI after adding tags", "error", err, "ci_id", ciID)
		return nil, fmt.Errorf("failed to reload CI: %w", err)
	}

	// 记录历史
	operatorID, _ := ctx.Value("user_id").(int)
	operatorName, _ := ctx.Value("user_name").(string)
	_ = s.historyService.RecordCIHistory(ctx, ciID, tenantID, operatorID, operatorName, "update", "Added tags", ci, updatedCI)

	s.logger.Infow("Tags added to CI successfully", "ci_id", ciID, "tag_ids", tagIDs)
	return dto.ToCIResponseWithRelations(updatedCI), nil
}

// RemoveTagsFromCI 从CI移除标签
func (s *ConfigurationItemService) RemoveTagsFromCI(ctx context.Context, ciID, tenantID int, tagIDs []int) (*dto.CIResponse, error) {
	// 检查CI是否存在
	ci, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.IDEQ(ciID), configurationitem.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("CI not found")
		}
		s.logger.Errorw("Failed to get CI for removing tags", "error", err, "ci_id", ciID)
		return nil, fmt.Errorf("failed to get CI: %w", err)
	}

	// 移除标签关联
	err = s.client.ConfigurationItem.UpdateOneID(ciID).
		RemoveTagIDs(tagIDs...).
		Exec(ctx)
	if err != nil {
		s.logger.Errorw("Failed to remove tags from CI", "error", err, "ci_id", ciID, "tag_ids", tagIDs)
		return nil, fmt.Errorf("failed to remove tags: %w", err)
	}

	// 重新加载CI数据
	updatedCI, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.IDEQ(ciID)).
		WithOutgoingRelations().
		WithIncomingRelations().
		WithTags().
		First(ctx)
	if err != nil {
		s.logger.Errorw("Failed to reload CI after removing tags", "error", err, "ci_id", ciID)
		return nil, fmt.Errorf("failed to reload CI: %w", err)
	}

	// 记录历史
	operatorID, _ := ctx.Value("user_id").(int)
	operatorName, _ := ctx.Value("user_name").(string)
	_ = s.historyService.RecordCIHistory(ctx, ciID, tenantID, operatorID, operatorName, "update", "Removed tags", ci, updatedCI)

	s.logger.Infow("Tags removed from CI successfully", "ci_id", ciID, "tag_ids", tagIDs)
	return dto.ToCIResponseWithRelations(updatedCI), nil
}

// BatchCreateCI 批量创建CI
func (s *ConfigurationItemService) BatchCreateCI(ctx context.Context, req *dto.BatchCreateCIRequest, tenantID int) (*dto.BatchOperationResponse, error) {
	if len(req.Items) == 0 {
		return nil, fmt.Errorf("no CI items provided")
	}
	if len(req.Items) > 100 {
		return nil, fmt.Errorf("batch size cannot exceed 100")
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		s.logger.Errorw("Failed to start transaction for batch create", "error", err)
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}

	successCount := 0
	failedIDs := make([]int, 0)
	errors := make([]string, 0)

	for i, item := range req.Items {
		// 这里用索引作为临时ID，因为还没创建
		tempID := i + 1

		// 检查CI类型
		ciTypeID := item.CITypeID
		if ciTypeID == 0 {
			ciTypeID = item.CITypeIDSnake
		}
		if ciTypeID == 0 {
			failedIDs = append(failedIDs, tempID)
			errors = append(errors, fmt.Sprintf("CI %d: CI type id is required", tempID))
			continue
		}

		ciType, err := tx.CIType.Query().
			Where(citype.IDEQ(ciTypeID), citype.TenantIDEQ(tenantID)).
			First(ctx)
		if err != nil {
			failedIDs = append(failedIDs, tempID)
			if ent.IsNotFound(err) {
				errors = append(errors, fmt.Sprintf("CI %d: CI type %d not found", tempID, ciTypeID))
			} else {
				errors = append(errors, fmt.Sprintf("CI %d: failed to get CI type: %v", tempID, err))
			}
			continue
		}

		// 创建CI
		create := tx.ConfigurationItem.Create().
			SetName(item.Name).
			SetCiTypeID(ciTypeID).
			SetCiType(ciType.Name).
			SetStatus(item.Status).
			SetTenantID(tenantID)

		if item.Environment != "" {
			create.SetEnvironment(item.Environment)
		}
		if item.Criticality != "" {
			create.SetCriticality(item.Criticality)
		}
		if item.AssetTag != "" {
			create.SetAssetTag(item.AssetTag)
		}
		if item.SerialNumber != "" {
			create.SetSerialNumber(item.SerialNumber)
		}
		if item.Model != "" {
			create.SetModel(item.Model)
		}
		if item.Vendor != "" {
			create.SetVendor(item.Vendor)
		}
		if item.Location != "" {
			create.SetLocation(item.Location)
		}
		if item.AssignedTo != "" {
			create.SetAssignedTo(item.AssignedTo)
		}
		if item.OwnedBy != "" {
			create.SetOwnedBy(item.OwnedBy)
		}
		if item.DiscoverySource != "" {
			create.SetDiscoverySource(item.DiscoverySource)
		}
		if item.Source != "" {
			create.SetSource(item.Source)
		}
		if item.Attributes != nil {
			create.SetAttributes(item.Attributes)
		}
		if item.CloudProvider != "" {
			create.SetCloudProvider(item.CloudProvider)
		}
		if item.CloudAccountID != "" {
			create.SetCloudAccountID(item.CloudAccountID)
		}
		if item.CloudRegion != "" {
			create.SetCloudRegion(item.CloudRegion)
		}
		if item.CloudZone != "" {
			create.SetCloudZone(item.CloudZone)
		}
		if item.CloudResourceID != "" {
			create.SetCloudResourceID(item.CloudResourceID)
		}
		if item.CloudResourceType != "" {
			create.SetCloudResourceType(item.CloudResourceType)
		}
		if item.CloudMetadata != nil {
			create.SetCloudMetadata(item.CloudMetadata)
		}
		if item.CloudTags != nil {
			create.SetCloudTags(item.CloudTags)
		}
		if item.CloudMetrics != nil {
			create.SetCloudMetrics(item.CloudMetrics)
		}
		if item.CloudSyncTime != nil {
			create.SetCloudSyncTime(*item.CloudSyncTime)
		}
		if item.CloudSyncStatus != "" {
			create.SetCloudSyncStatus(item.CloudSyncStatus)
		}
		if item.CloudResourceRefID != 0 {
			create.SetCloudResourceRefID(item.CloudResourceRefID)
		}

		ci, err := create.Save(ctx)
		if err != nil {
			failedIDs = append(failedIDs, tempID)
			errors = append(errors, fmt.Sprintf("CI %d: failed to create: %v", tempID, err))
			continue
		}

		// 记录历史
		operatorID, _ := ctx.Value("user_id").(int)
		operatorName, _ := ctx.Value("user_name").(string)
		_ = s.historyService.RecordCIHistory(ctx, ci.ID, tenantID, operatorID, operatorName, "create", "Batch created", nil, ci)

		successCount++
	}

	if len(failedIDs) > 0 {
		// 有失败的，回滚事务
		if err := tx.Rollback(); err != nil {
			s.logger.Errorw("Failed to rollback transaction for batch create", "error", err)
			return nil, fmt.Errorf("failed to rollback: %w", err)
		}
	} else {
		// 全部成功，提交事务
		if err := tx.Commit(); err != nil {
			s.logger.Errorw("Failed to commit transaction for batch create", "error", err)
			return nil, fmt.Errorf("failed to commit: %w", err)
		}
	}

	s.logger.Infow("Batch create CI completed", "success", successCount, "failed", len(failedIDs), "tenant_id", tenantID)
	return &dto.BatchOperationResponse{
		SuccessCount: successCount,
		FailedCount:  len(failedIDs),
		FailedIDs:    failedIDs,
		Errors:       errors,
	}, nil
}

// BatchUpdateCI 批量更新CI
func (s *ConfigurationItemService) BatchUpdateCI(ctx context.Context, req *dto.BatchUpdateCIRequest, tenantID int) (*dto.BatchOperationResponse, error) {
	if len(req.IDs) == 0 {
		return nil, fmt.Errorf("no CI IDs provided")
	}
	if len(req.IDs) > 100 {
		return nil, fmt.Errorf("batch size cannot exceed 100")
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		s.logger.Errorw("Failed to start transaction for batch update", "error", err)
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}

	successCount := 0
	failedIDs := make([]int, 0)
	errors := make([]string, 0)

	// 先检查所有CI是否存在
	cis, err := tx.ConfigurationItem.Query().
		Where(
			configurationitem.IDIn(req.IDs...),
			configurationitem.TenantIDEQ(tenantID),
		).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to query CIs for batch update", "error", err)
		return nil, fmt.Errorf("failed to query CIs: %w", err)
	}

	// 检查是否有CI不存在
	existingIDs := make(map[int]bool)
	for _, ci := range cis {
		existingIDs[ci.ID] = true
	}
	for _, id := range req.IDs {
		if !existingIDs[id] {
			failedIDs = append(failedIDs, id)
			errors = append(errors, fmt.Sprintf("CI %d: not found", id))
		}
	}

	if len(failedIDs) > 0 {
		// 有CI不存在，直接返回错误，不执行更新
		if err := tx.Rollback(); err != nil {
			s.logger.Errorw("Failed to rollback transaction for batch update", "error", err)
			return nil, fmt.Errorf("failed to rollback: %w", err)
		}
		return &dto.BatchOperationResponse{
			SuccessCount: 0,
			FailedCount:  len(failedIDs),
			FailedIDs:    failedIDs,
			Errors:       errors,
		}, nil
	}

	// 处理CI类型更新
	var ciType *ent.CIType
	if req.Updates.CITypeID != 0 || req.Updates.CITypeIDSnake != 0 {
		ciTypeID := req.Updates.CITypeID
		if ciTypeID == 0 {
			ciTypeID = req.Updates.CITypeIDSnake
		}
		ciType, err = tx.CIType.Query().
			Where(citype.IDEQ(ciTypeID), citype.TenantIDEQ(tenantID)).
			First(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return nil, fmt.Errorf("CI type %d not found", ciTypeID)
			}
			s.logger.Errorw("Failed to get CI type for batch update", "error", err, "ci_type_id", ciTypeID)
			return nil, fmt.Errorf("failed to get CI type: %w", err)
		}
	}

	// 执行批量更新
	for _, id := range req.IDs {
		// 获取更新前的CI
		oldCI, err := tx.ConfigurationItem.Query().
			Where(configurationitem.IDEQ(id), configurationitem.TenantIDEQ(tenantID)).
			First(ctx)
		if err != nil {
			failedIDs = append(failedIDs, id)
			errors = append(errors, fmt.Sprintf("CI %d: failed to get old CI: %v", id, err))
			continue
		}

		update := tx.ConfigurationItem.UpdateOneID(id)

		if req.Updates.Name != "" {
			update.SetName(req.Updates.Name)
		}
		if req.Updates.Status != "" {
			update.SetStatus(req.Updates.Status)
		}
		if req.Updates.Environment != "" {
			update.SetEnvironment(req.Updates.Environment)
		}
		if req.Updates.Criticality != "" {
			update.SetCriticality(req.Updates.Criticality)
		}
		if req.Updates.AssetTag != "" {
			update.SetAssetTag(req.Updates.AssetTag)
		}
		if req.Updates.SerialNumber != "" {
			update.SetSerialNumber(req.Updates.SerialNumber)
		}
		if req.Updates.Model != "" {
			update.SetModel(req.Updates.Model)
		}
		if req.Updates.Vendor != "" {
			update.SetVendor(req.Updates.Vendor)
		}
		if req.Updates.Location != "" {
			update.SetLocation(req.Updates.Location)
		}
		if req.Updates.AssignedTo != "" {
			update.SetAssignedTo(req.Updates.AssignedTo)
		}
		if req.Updates.OwnedBy != "" {
			update.SetOwnedBy(req.Updates.OwnedBy)
		}
		if req.Updates.DiscoverySource != "" {
			update.SetDiscoverySource(req.Updates.DiscoverySource)
		}
		if req.Updates.Source != "" {
			update.SetSource(req.Updates.Source)
		}
		if req.Updates.Attributes != nil {
			update.SetAttributes(req.Updates.Attributes)
		}
		if req.Updates.CloudProvider != "" {
			update.SetCloudProvider(req.Updates.CloudProvider)
		}
		if req.Updates.CloudAccountID != "" {
			update.SetCloudAccountID(req.Updates.CloudAccountID)
		}
		if req.Updates.CloudRegion != "" {
			update.SetCloudRegion(req.Updates.CloudRegion)
		}
		if req.Updates.CloudZone != "" {
			update.SetCloudZone(req.Updates.CloudZone)
		}
		if req.Updates.CloudResourceID != "" {
			update.SetCloudResourceID(req.Updates.CloudResourceID)
		}
		if req.Updates.CloudResourceType != "" {
			update.SetCloudResourceType(req.Updates.CloudResourceType)
		}
		if req.Updates.CloudMetadata != nil {
			update.SetCloudMetadata(req.Updates.CloudMetadata)
		}
		if req.Updates.CloudTags != nil {
			update.SetCloudTags(req.Updates.CloudTags)
		}
		if req.Updates.CloudMetrics != nil {
			update.SetCloudMetrics(req.Updates.CloudMetrics)
		}
		if req.Updates.CloudSyncTime != nil {
			update.SetCloudSyncTime(*req.Updates.CloudSyncTime)
		}
		if req.Updates.CloudSyncStatus != "" {
			update.SetCloudSyncStatus(req.Updates.CloudSyncStatus)
		}
		if req.Updates.CloudResourceRefID != 0 {
			update.SetCloudResourceRefID(req.Updates.CloudResourceRefID)
		}
		if ciType != nil {
			update.SetCiTypeID(ciType.ID)
			update.SetCiType(ciType.Name)
		}

		// 执行更新
		updatedCI, err := update.Save(ctx)
		if err != nil {
			failedIDs = append(failedIDs, id)
			errors = append(errors, fmt.Sprintf("CI %d: failed to update: %v", id, err))
			continue
		}

		// 记录历史
		operatorID, _ := ctx.Value("user_id").(int)
		operatorName, _ := ctx.Value("user_name").(string)
		_ = s.historyService.RecordCIHistory(ctx, id, tenantID, operatorID, operatorName, "update", "Batch updated", oldCI, updatedCI)

		successCount++
	}

	if len(failedIDs) > 0 {
		// 有失败的，回滚事务
		if err := tx.Rollback(); err != nil {
			s.logger.Errorw("Failed to rollback transaction for batch update", "error", err)
			return nil, fmt.Errorf("failed to rollback: %w", err)
		}
	} else {
		// 全部成功，提交事务
		if err := tx.Commit(); err != nil {
			s.logger.Errorw("Failed to commit transaction for batch update", "error", err)
			return nil, fmt.Errorf("failed to commit: %w", err)
		}
	}

	s.logger.Infow("Batch update CI completed", "success", successCount, "failed", len(failedIDs), "tenant_id", tenantID)
	return &dto.BatchOperationResponse{
		SuccessCount: successCount,
		FailedCount:  len(failedIDs),
		FailedIDs:    failedIDs,
		Errors:       errors,
	}, nil
}

// BatchDeleteCI 批量删除CI
func (s *ConfigurationItemService) BatchDeleteCI(ctx context.Context, req *dto.BatchDeleteCIRequest, tenantID int) (*dto.BatchOperationResponse, error) {
	if len(req.IDs) == 0 {
		return nil, fmt.Errorf("no CI IDs provided")
	}
	if len(req.IDs) > 100 {
		return nil, fmt.Errorf("batch size cannot exceed 100")
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		s.logger.Errorw("Failed to start transaction for batch delete", "error", err)
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}

	successCount := 0
	failedIDs := make([]int, 0)
	errors := make([]string, 0)

	// 先检查所有CI是否存在
	cis, err := tx.ConfigurationItem.Query().
		Where(
			configurationitem.IDIn(req.IDs...),
			configurationitem.TenantIDEQ(tenantID),
		).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to query CIs for batch delete", "error", err)
		return nil, fmt.Errorf("failed to query CIs: %w", err)
	}

	// 检查是否有CI不存在
	existingIDs := make(map[int]bool)
	ciMap := make(map[int]*ent.ConfigurationItem)
	for _, ci := range cis {
		existingIDs[ci.ID] = true
		ciMap[ci.ID] = ci
	}
	for _, id := range req.IDs {
		if !existingIDs[id] {
			failedIDs = append(failedIDs, id)
			errors = append(errors, fmt.Sprintf("CI %d: not found", id))
		}
	}

	if len(failedIDs) > 0 {
		// 有CI不存在，直接返回错误
		if err := tx.Rollback(); err != nil {
			s.logger.Errorw("Failed to rollback transaction for batch delete", "error", err)
			return nil, fmt.Errorf("failed to rollback: %w", err)
		}
		return &dto.BatchOperationResponse{
			SuccessCount: 0,
			FailedCount:  len(failedIDs),
			FailedIDs:    failedIDs,
			Errors:       errors,
		}, nil
	}

	// 执行批量删除
	for _, id := range req.IDs {
		ci := ciMap[id]

		// 删除关联的关系
		_, err := tx.CIRelationship.Delete().
			Where(
				cirelationship.Or(
					cirelationship.SourceCiIDEQ(id),
					cirelationship.TargetCiIDEQ(id),
				),
				cirelationship.TenantIDEQ(tenantID),
			).
			Exec(ctx)
		if err != nil {
			failedIDs = append(failedIDs, id)
			errors = append(errors, fmt.Sprintf("CI %d: failed to delete relations: %v", id, err))
			continue
		}

		// 删除CI
		err = tx.ConfigurationItem.DeleteOneID(id).
			Where(configurationitem.TenantIDEQ(tenantID)).
			Exec(ctx)
		if err != nil {
			failedIDs = append(failedIDs, id)
			errors = append(errors, fmt.Sprintf("CI %d: failed to delete: %v", id, err))
			continue
		}

		// 记录历史
		operatorID, _ := ctx.Value("user_id").(int)
		operatorName, _ := ctx.Value("user_name").(string)
		_ = s.historyService.RecordCIHistory(ctx, id, tenantID, operatorID, operatorName, "delete", "Batch deleted", ci, nil)

		successCount++
	}

	if len(failedIDs) > 0 {
		// 有失败的，回滚事务
		if err := tx.Rollback(); err != nil {
			s.logger.Errorw("Failed to rollback transaction for batch delete", "error", err)
			return nil, fmt.Errorf("failed to rollback: %w", err)
		}
	} else {
		// 全部成功，提交事务
		if err := tx.Commit(); err != nil {
			s.logger.Errorw("Failed to commit transaction for batch delete", "error", err)
			return nil, fmt.Errorf("failed to commit: %w", err)
		}
	}

	s.logger.Infow("Batch delete CI completed", "success", successCount, "failed", len(failedIDs), "tenant_id", tenantID)
	return &dto.BatchOperationResponse{
		SuccessCount: successCount,
		FailedCount:  len(failedIDs),
		FailedIDs:    failedIDs,
		Errors:       errors,
	}, nil
}

// SearchCI 高级搜索CI
func (s *ConfigurationItemService) SearchCI(ctx context.Context, tenantID int, req *dto.CISearchRequest) (*dto.ListResponse[dto.CIResponse], error) {
	query := s.client.ConfigurationItem.Query().
		Where(configurationitem.TenantID(tenantID))

	// 应用过滤条件
	query = s.applySearchFilters(query, &req.Filters)

	// 统计总数
	total, err := query.Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count CI for search", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("统计CI数量失败: %w", err)
	}

	// 排序
	if req.SortBy != "" {
		sortField := s.convertSortField(req.SortBy)
		if req.SortOrder == "asc" {
			query = query.Order(ent.Asc(sortField))
		} else {
			query = query.Order(ent.Desc(sortField))
		}
	} else {
		// 默认按ID降序
		query = query.Order(ent.Desc(configurationitem.FieldID))
	}

	// 分页查询
	cis, err := query.
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		WithCiTypeRef().
		WithTags().
		WithOutgoingRelations(func(q *ent.CIRelationshipQuery) {
			q.WithTargetCi()
		}).
		WithIncomingRelations(func(q *ent.CIRelationshipQuery) {
			q.WithSourceCi()
		}).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to search CI", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("查询CI失败: %w", err)
	}

	// 转换为DTO
	items := make([]dto.CIResponse, len(cis))
	for i, ci := range cis {
		items[i] = *dto.ToCIResponseWithRelations(ci)
	}

	return &dto.ListResponse[dto.CIResponse]{
		Items: items,
		Total: total,
		Page:  req.Page,
		Size:  req.PageSize,
	}, nil
}

// applySearchFilters 应用搜索过滤条件
func (s *ConfigurationItemService) applySearchFilters(query *ent.ConfigurationItemQuery, filters *dto.CISearchFilter) *ent.ConfigurationItemQuery {
	// 全文搜索
	if filters.Keyword != "" {
		query = query.Where(
			configurationitem.Or(
				configurationitem.NameContains(filters.Keyword),
				configurationitem.AssetTagContains(filters.Keyword),
				configurationitem.SerialNumberContains(filters.Keyword),
				configurationitem.ModelContains(filters.Keyword),
				configurationitem.VendorContains(filters.Keyword),
				configurationitem.LocationContains(filters.Keyword),
				configurationitem.AssignedToContains(filters.Keyword),
				configurationitem.OwnedByContains(filters.Keyword),
				configurationitem.CloudResourceIDContains(filters.Keyword),
			),
		)
	}

	// CI类型过滤
	if filters.CITypeID > 0 {
		query = query.Where(configurationitem.CiTypeID(filters.CITypeID))
	}

	// 状态过滤
	if filters.Status != "" {
		query = query.Where(configurationitem.Status(filters.Status))
	}

	// 环境过滤
	if filters.Environment != "" {
		query = query.Where(configurationitem.Environment(filters.Environment))
	}

	// 重要级别过滤
	if filters.Criticality != "" {
		query = query.Where(configurationitem.Criticality(filters.Criticality))
	}

	// 资产标签过滤
	if filters.AssetTag != "" {
		query = query.Where(configurationitem.AssetTagContains(filters.AssetTag))
	}

	// 序列号过滤
	if filters.SerialNumber != "" {
		query = query.Where(configurationitem.SerialNumberContains(filters.SerialNumber))
	}

	// 厂商过滤
	if filters.Vendor != "" {
		query = query.Where(configurationitem.VendorContains(filters.Vendor))
	}

	// 位置过滤
	if filters.Location != "" {
		query = query.Where(configurationitem.LocationContains(filters.Location))
	}

	// 负责人过滤
	if filters.AssignedTo != "" {
		query = query.Where(configurationitem.AssignedToContains(filters.AssignedTo))
	}

	// 归属人过滤
	if filters.OwnedBy != "" {
		query = query.Where(configurationitem.OwnedByContains(filters.OwnedBy))
	}

	// 云厂商过滤
	if filters.CloudProvider != "" {
		query = query.Where(configurationitem.CloudProvider(filters.CloudProvider))
	}

	// 云区域过滤
	if filters.CloudRegion != "" {
		query = query.Where(configurationitem.CloudRegion(filters.CloudRegion))
	}

	// 云资源ID过滤
	if filters.CloudResourceID != "" {
		query = query.Where(configurationitem.CloudResourceIDContains(filters.CloudResourceID))
	}

	// 时间范围过滤
	if filters.DateFrom != nil && !filters.DateFrom.IsZero() {
		query = query.Where(configurationitem.CreatedAtGTE(*filters.DateFrom))
	}
	if filters.DateTo != nil && !filters.DateTo.IsZero() {
		query = query.Where(configurationitem.CreatedAtLTE(*filters.DateTo))
	}

	// 标签过滤
	if len(filters.TagIDs) > 0 {
		query = query.Where(configurationitem.HasTagsWith(citag.IDIn(filters.TagIDs...)))
	}

	// 自定义属性过滤（ent 不支持 JSON key/value 谓词，此处跳过，由调用方在内存中过滤）
	// TODO: 如需数据库层 JSON 过滤，可使用 entgo WithMutation + raw SQL predicate

	return query
}

// convertSortField 转换排序字段为数据库字段名
func (s *ConfigurationItemService) convertSortField(sortBy string) string {
	fieldMap := map[string]string{
		"id":                  configurationitem.FieldID,
		"name":                configurationitem.FieldName,
		"ci_type":             configurationitem.FieldCiType,
		"status":              configurationitem.FieldStatus,
		"environment":         configurationitem.FieldEnvironment,
		"criticality":         configurationitem.FieldCriticality,
		"asset_tag":           configurationitem.FieldAssetTag,
		"serial_number":       configurationitem.FieldSerialNumber,
		"vendor":              configurationitem.FieldVendor,
		"location":            configurationitem.FieldLocation,
		"assigned_to":         configurationitem.FieldAssignedTo,
		"owned_by":            configurationitem.FieldOwnedBy,
		"cloud_provider":      configurationitem.FieldCloudProvider,
		"cloud_region":        configurationitem.FieldCloudRegion,
		"created_at":          configurationitem.FieldCreatedAt,
		"updated_at":          configurationitem.FieldUpdatedAt,
		"lifecycle_status":    configurationitem.FieldLifecycleStatus,
		"effective_at":        configurationitem.FieldEffectiveAt,
		"expire_at":           configurationitem.FieldExpireAt,
	}

	if field, ok := fieldMap[sortBy]; ok {
		return field
	}

	// 默认按ID排序
	return configurationitem.FieldID
}

// UpdateLifecycleStatus 更新CI生命周期状态
func (s *ConfigurationItemService) UpdateLifecycleStatus(ctx context.Context, id int, tenantID int, status string, remark string, operatorID int, operatorName string) (*dto.CIResponse, error) {
	// 校验状态是否合法
	validStatuses := map[string]bool{
		"draft":        true,
		"online":       true,
		"maintenance":  true,
		"offline":      true,
		"scrapped":     true,
	}
	if !validStatuses[status] {
		return nil, fmt.Errorf("无效的生命周期状态: %s", status)
	}

	// 获取当前CI
	ci, err := s.client.ConfigurationItem.Query().
		Where(
			configurationitem.ID(id),
			configurationitem.TenantID(tenantID),
		).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("CI不存在")
		}
		s.logger.Errorw("Failed to get CI for lifecycle update", "error", err, "ci_id", id)
		return nil, fmt.Errorf("获取CI失败: %w", err)
	}

	// 如果状态没有变化，直接返回
	if ci.LifecycleStatus == status {
		return dto.ToCIResponse(ci), nil
	}

	// 更新状态
	updatedCI, err := s.client.ConfigurationItem.UpdateOne(ci).
		SetLifecycleStatus(status).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update CI lifecycle status", "error", err, "ci_id", id, "status", status)
		return nil, fmt.Errorf("更新生命周期状态失败: %w", err)
	}

	// 记录变更历史
	// LifecycleHistory 字段尚未在 ent schema 中定义，暂时跳过写入历史记录
	// TODO: 在 ent/schema/configurationitem.go 中添加 lifecycle_history 字段后恢复
	_ = time.Now() // keep time import

	// 记录到CI变更历史
	_ = s.historyService.RecordCIHistory(ctx, id, tenantID, operatorID, operatorName, "update", remark, ci, updatedCI)

	s.logger.Infow("CI lifecycle status updated", "ci_id", id, "old_status", ci.LifecycleStatus, "new_status", status, "tenant_id", tenantID)

	// 重新加载CI完整信息
	fullCI, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.ID(id)).
		WithOutgoingRelations().
		WithIncomingRelations().
		WithTags().
		First(ctx)
	if err != nil {
		return dto.ToCIResponse(updatedCI), nil
	}

	return dto.ToCIResponseWithRelations(fullCI), nil
}

// BatchUpdateLifecycleStatus 批量更新CI生命周期状态
func (s *ConfigurationItemService) BatchUpdateLifecycleStatus(ctx context.Context, ids []int, tenantID int, status string, remark string, operatorID int, operatorName string) (*dto.BatchOperationResponse, error) {
	successCount := 0
	failedCount := 0
	failedIDs := make([]int, 0)
	errors := make([]string, 0)

	for _, id := range ids {
		_, err := s.UpdateLifecycleStatus(ctx, id, tenantID, status, remark, operatorID, operatorName)
		if err != nil {
			failedCount++
			failedIDs = append(failedIDs, id)
			errors = append(errors, fmt.Sprintf("CI %d: %v", id, err))
		} else {
			successCount++
		}
	}

	return &dto.BatchOperationResponse{
		SuccessCount: successCount,
		FailedCount:  failedCount,
		FailedIDs:    failedIDs,
		Errors:       errors,
	}, nil
}

// GetLifecycleHistory 获取CI生命周期变更历史
// TODO: ent schema 添加 lifecycle_history 字段后完整实现
func (s *ConfigurationItemService) GetLifecycleHistory(ctx context.Context, id int, tenantID int) ([]map[string]interface{}, error) {
	_, err := s.client.ConfigurationItem.Query().
		Where(
			configurationitem.ID(id),
			configurationitem.TenantID(tenantID),
		).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("CI不存在")
		}
		s.logger.Errorw("Failed to get CI lifecycle history", "error", err, "ci_id", id)
		return nil, fmt.Errorf("获取生命周期历史失败: %w", err)
	}

	// LifecycleHistory 字段尚未在 ent schema 中定义，返回空列表
	return []map[string]interface{}{}, nil
}
