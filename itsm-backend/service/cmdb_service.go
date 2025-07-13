package service

import (
	"context"
	"fmt"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/configurationitem"
	"strings"
)

type CMDBService struct {
	client *ent.Client
}

func NewCMDBService(client *ent.Client) *CMDBService {
	return &CMDBService{
		client: client,
	}
}

// CreateConfigurationItem 创建配置项
func (s *CMDBService) CreateConfigurationItem(ctx context.Context, req *dto.CreateConfigurationItemRequest, tenantID int) (*dto.ConfigurationItemResponse, error) {
	// 创建配置项
	create := s.client.ConfigurationItem.Create().
		SetName(req.Name).
		SetType(req.Type).
		SetTenantID(tenantID)

	if req.BusinessService != nil {
		create.SetBusinessService(*req.BusinessService)
	}
	if req.Owner != nil {
		create.SetOwner(*req.Owner)
	}
	if req.Environment != nil {
		create.SetEnvironment(*req.Environment)
	}
	if req.Location != nil {
		create.SetLocation(*req.Location)
	}
	if req.Attributes != nil {
		create.SetAttributes(req.Attributes)
	}
	if req.MonitoringData != nil {
		create.SetMonitoringData(req.MonitoringData)
	}

	ci, err := create.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建配置项失败: %w", err)
	}

	// 处理关联关系
	if len(req.RelatedItemIDs) > 0 {
		err = s.client.ConfigurationItem.UpdateOneID(ci.ID).
			AddRelatedItemIDs(req.RelatedItemIDs...).
			Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("设置关联关系失败: %w", err)
		}
	}

	// 重新查询以获取完整信息
	ci, err = s.client.ConfigurationItem.Query().
		Where(configurationitem.ID(ci.ID)).
		WithRelatedItems().
		WithParentItems().
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询配置项失败: %w", err)
	}

	return s.toConfigurationItemResponse(ci), nil
}

// GetConfigurationItem 获取配置项详情
func (s *CMDBService) GetConfigurationItem(ctx context.Context, id int, tenantID int) (*dto.ConfigurationItemResponse, error) {
	ci, err := s.client.ConfigurationItem.Query().
		Where(
			configurationitem.ID(id),
			configurationitem.TenantID(tenantID),
		).
		WithRelatedItems().
		WithParentItems().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("配置项不存在")
		}
		return nil, fmt.Errorf("查询配置项失败: %w", err)
	}

	return s.toConfigurationItemResponse(ci), nil
}

// ListConfigurationItems 获取配置项列表
func (s *CMDBService) ListConfigurationItems(ctx context.Context, req *dto.ListConfigurationItemsRequest, tenantID int) (*dto.ConfigurationItemListResponse, error) {
	query := s.client.ConfigurationItem.Query().
		Where(configurationitem.TenantID(tenantID))

	// 添加过滤条件
	if req.Type != "" {
		query = query.Where(configurationitem.Type(req.Type))
	}
	if req.Status != "" {
		query = query.Where(configurationitem.Status(req.Status))
	}
	if req.BusinessService != "" {
		query = query.Where(configurationitem.BusinessServiceContains(req.BusinessService))
	}
	if req.Owner != "" {
		query = query.Where(configurationitem.OwnerContains(req.Owner))
	}
	if req.Environment != "" {
		query = query.Where(configurationitem.Environment(req.Environment))
	}
	if req.Search != "" {
		query = query.Where(
			configurationitem.Or(
				configurationitem.NameContains(req.Search),
				configurationitem.BusinessServiceContains(req.Search),
				configurationitem.OwnerContains(req.Search),
			),
		)
	}

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询配置项总数失败: %w", err)
	}

	// 分页查询
	offset := (req.Page - 1) * req.Size
	items, err := query.
		WithRelatedItems().
		WithParentItems().
		Offset(offset).
		Limit(req.Size).
		Order(ent.Desc(configurationitem.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询配置项列表失败: %w", err)
	}

	// 转换响应
	responses := make([]dto.ConfigurationItemResponse, len(items))
	for i, item := range items {
		responses[i] = *s.toConfigurationItemResponse(item)
	}

	return &dto.ConfigurationItemListResponse{
		Items: responses,
		Total: total,
		Page:  req.Page,
		Size:  req.Size,
	}, nil
}

// UpdateConfigurationItem 更新配置项
func (s *CMDBService) UpdateConfigurationItem(ctx context.Context, id int, req *dto.UpdateConfigurationItemRequest, tenantID int) (*dto.ConfigurationItemResponse, error) {
	// 检查配置项是否存在
	exists, err := s.client.ConfigurationItem.Query().
		Where(
			configurationitem.ID(id),
			configurationitem.TenantID(tenantID),
		).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("检查配置项失败: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("配置项不存在")
	}

	// 更新配置项
	update := s.client.ConfigurationItem.UpdateOneID(id)

	if req.Name != nil {
		update.SetName(*req.Name)
	}
	if req.Type != nil {
		update.SetType(*req.Type)
	}
	if req.Status != nil {
		update.SetStatus(*req.Status)
	}
	if req.BusinessService != nil {
		update.SetBusinessService(*req.BusinessService)
	}
	if req.Owner != nil {
		update.SetOwner(*req.Owner)
	}
	if req.Environment != nil {
		update.SetEnvironment(*req.Environment)
	}
	if req.Location != nil {
		update.SetLocation(*req.Location)
	}
	if req.Attributes != nil {
		update.SetAttributes(req.Attributes)
	}
	if req.MonitoringData != nil {
		update.SetMonitoringData(req.MonitoringData)
	}

	err = update.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("更新配置项失败: %w", err)
	}

	// 处理关联关系更新
	if req.RelatedItemIDs != nil {
		// 清除现有关联
		err = s.client.ConfigurationItem.UpdateOneID(id).
			ClearRelatedItems().
			Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("清除关联关系失败: %w", err)
		}

		// 添加新关联
		if len(req.RelatedItemIDs) > 0 {
			err = s.client.ConfigurationItem.UpdateOneID(id).
				AddRelatedItemIDs(req.RelatedItemIDs...).
				Exec(ctx)
			if err != nil {
				return nil, fmt.Errorf("设置关联关系失败: %w", err)
			}
		}
	}

	// 重新查询以获取完整信息
	ci, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.ID(id)).
		WithRelatedItems().
		WithParentItems().
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询配置项失败: %w", err)
	}

	return s.toConfigurationItemResponse(ci), nil
}

// DeleteConfigurationItem 删除配置项
func (s *CMDBService) DeleteConfigurationItem(ctx context.Context, id int, tenantID int) error {
	// 检查配置项是否存在
	exists, err := s.client.ConfigurationItem.Query().
		Where(
			configurationitem.ID(id),
			configurationitem.TenantID(tenantID),
		).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("检查配置项失败: %w", err)
	}
	if !exists {
		return fmt.Errorf("配置项不存在")
	}

	// 删除配置项
	err = s.client.ConfigurationItem.DeleteOneID(id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("删除配置项失败: %w", err)
	}

	return nil
}

// GetConfigurationItemStats 获取配置项统计信息
func (s *CMDBService) GetConfigurationItemStats(ctx context.Context, tenantID int) (*dto.ConfigurationItemStatsResponse, error) {
	// 总数统计
	totalCount, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.TenantID(tenantID)).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询总数失败: %w", err)
	}

	// 状态统计
	activeCount, err := s.client.ConfigurationItem.Query().
		Where(
			configurationitem.TenantID(tenantID),
			configurationitem.Status("active"),
		).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询活跃数量失败: %w", err)
	}

	inactiveCount, err := s.client.ConfigurationItem.Query().
		Where(
			configurationitem.TenantID(tenantID),
			configurationitem.Status("inactive"),
		).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询非活跃数量失败: %w", err)
	}

	maintenanceCount, err := s.client.ConfigurationItem.Query().
		Where(
			configurationitem.TenantID(tenantID),
			configurationitem.Status("maintenance"),
		).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询维护数量失败: %w", err)
	}

	// 类型分布统计
	typeDistribution := make(map[string]int)
	types := []string{"server", "database", "application", "network", "storage"}
	for _, t := range types {
		count, err := s.client.ConfigurationItem.Query().
			Where(
				configurationitem.TenantID(tenantID),
				configurationitem.Type(t),
			).
			Count(ctx)
		if err != nil {
			return nil, fmt.Errorf("查询类型分布失败: %w", err)
		}
		typeDistribution[t] = count
	}

	// 环境分布统计
	environmentDistribution := make(map[string]int)
	environments := []string{"production", "staging", "development"}
	for _, env := range environments {
		count, err := s.client.ConfigurationItem.Query().
			Where(
				configurationitem.TenantID(tenantID),
				configurationitem.Environment(env),
			).
			Count(ctx)
		if err != nil {
			return nil, fmt.Errorf("查询环境分布失败: %w", err)
		}
		environmentDistribution[env] = count
	}

	return &dto.ConfigurationItemStatsResponse{
		TotalCount:              totalCount,
		ActiveCount:             activeCount,
		InactiveCount:           inactiveCount,
		MaintenanceCount:        maintenanceCount,
		TypeDistribution:        typeDistribution,
		EnvironmentDistribution: environmentDistribution,
	}, nil
}

// toConfigurationItemResponse 转换为响应格式
func (s *CMDBService) toConfigurationItemResponse(ci *ent.ConfigurationItem) *dto.ConfigurationItemResponse {
	resp := &dto.ConfigurationItemResponse{
		ID:              ci.ID,
		Name:            ci.Name,
		Type:            ci.Type,
		Status:          ci.Status,
		BusinessService: &ci.BusinessService,
		Owner:           &ci.Owner,
		Environment:     &ci.Environment,
		Location:        &ci.Location,
		Attributes:      ci.Attributes,
		MonitoringData:  ci.MonitoringData,
		TenantID:        ci.TenantID,
		CreatedAt:       ci.CreatedAt,
		UpdatedAt:       ci.UpdatedAt,
	}

	// 处理关联项
	if ci.Edges.RelatedItems != nil {
		resp.RelatedItems = make([]dto.ConfigurationItemResponse, len(ci.Edges.RelatedItems))
		for i, item := range ci.Edges.RelatedItems {
			resp.RelatedItems[i] = *s.toConfigurationItemResponse(item)
		}
	}

	// 处理父项
	if ci.Edges.ParentItems != nil {
		resp.ParentItems = make([]dto.ConfigurationItemResponse, len(ci.Edges.ParentItems))
		for i, item := range ci.Edges.ParentItems {
			resp.ParentItems[i] = *s.toConfigurationItemResponse(item)
		}
	}

	return resp
}
