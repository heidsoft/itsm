package service

import (
	"context"
	"fmt"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/servicecatalogitem"

	"go.uber.org/zap"
)

// ServiceCatalogItemService 服务目录项服务
type ServiceCatalogItemService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewServiceCatalogItemService 创建服务目录项服务
func NewServiceCatalogItemService(client *ent.Client, logger *zap.SugaredLogger) *ServiceCatalogItemService {
	return &ServiceCatalogItemService{
		client: client,
		logger: logger,
	}
}

// CreateServiceCatalogItem 创建服务目录项
func (s *ServiceCatalogItemService) CreateServiceCatalogItem(ctx context.Context, req *dto.CreateServiceCatalogItemRequest, tenantID int) (*dto.ServiceCatalogItemResponse, error) {
	// 验证目录是否存在
	_, err := s.client.ServiceCatalog.Get(ctx, req.CatalogID)
	if err != nil {
		return nil, fmt.Errorf("service catalog not found: %w", err)
	}

	// 创建服务项
	item, err := s.client.ServiceCatalogItem.Create().
		SetName(req.Name).
		SetDescription(req.Description).
		SetDetails(req.Details).
		SetCategory(req.Category).
		SetIcon(req.Icon).
		SetFormSchema(req.FormSchema).
		SetNillableSlaID(&req.SlaID).
		SetNillableApprovalChainID(&req.ApprovalChainID).
		SetRequiresApproval(req.RequiresApproval).
		SetEstimatedDays(req.EstimatedDays).
		SetTenantID(tenantID).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create service catalog item", "error", err, "name", req.Name)
		return nil, fmt.Errorf("failed to create service item: %w", err)
	}

	s.logger.Infow("Service catalog item created", "item_id", item.ID, "name", item.Name)
	return s.toItemResponse(item), nil
}

// GetServiceCatalogItem 获取服务目录项
func (s *ServiceCatalogItemService) GetServiceCatalogItem(ctx context.Context, id int, tenantID int) (*dto.ServiceCatalogItemResponse, error) {
	item, err := s.client.ServiceCatalogItem.Query().
		Where(
			servicecatalogitem.ID(id),
			servicecatalogitem.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("service item not found")
		}
		return nil, fmt.Errorf("failed to get service item: %w", err)
	}

	return s.toItemResponse(item), nil
}

// ListServiceCatalogItems 列出服务目录下的所有项
func (s *ServiceCatalogItemService) ListServiceCatalogItems(ctx context.Context, catalogID int, tenantID int) ([]*dto.ServiceCatalogItemResponse, error) {
	// 验证目录是否存在
	_, err := s.client.ServiceCatalog.Get(ctx, catalogID)
	if err != nil {
		return nil, fmt.Errorf("service catalog not found: %w", err)
	}

	items, err := s.client.ServiceCatalogItem.Query().
		Where(
			servicecatalogitem.TenantID(tenantID),
			servicecatalogitem.IsActive(true),
		).
		Order(ent.Asc(servicecatalogitem.FieldName)).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list service catalog items", "error", err, "catalog_id", catalogID)
		return nil, fmt.Errorf("failed to list service items: %w", err)
	}

	var response []*dto.ServiceCatalogItemResponse
	for _, item := range items {
		response = append(response, s.toItemResponse(item))
	}

	return response, nil
}

// UpdateServiceCatalogItem 更新服务目录项
func (s *ServiceCatalogItemService) UpdateServiceCatalogItem(ctx context.Context, id int, req *dto.UpdateServiceCatalogItemRequest, tenantID int) (*dto.ServiceCatalogItemResponse, error) {
	// 验证服务项是否存在
	item, err := s.client.ServiceCatalogItem.Query().
		Where(
			servicecatalogitem.ID(id),
			servicecatalogitem.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("service item not found")
		}
		return nil, fmt.Errorf("failed to get service item: %w", err)
	}

	// 构建更新查询
	update := s.client.ServiceCatalogItem.UpdateOneID(id)
	if req.Name != nil {
		update.SetName(*req.Name)
	}
	if req.Description != nil {
		update.SetDescription(*req.Description)
	}
	if req.Details != nil {
		update.SetDetails(*req.Details)
	}
	if req.Category != nil {
		update.SetCategory(*req.Category)
	}
	if req.Icon != nil {
		update.SetIcon(*req.Icon)
	}
	if req.FormSchema != nil {
		update.SetFormSchema(*req.FormSchema)
	}
	if req.SlaID != nil {
		update.SetSlaID(*req.SlaID)
	}
	if req.ApprovalChainID != nil {
		update.SetApprovalChainID(*req.ApprovalChainID)
	}
	if req.IsActive != nil {
		update.SetIsActive(*req.IsActive)
	}
	if req.RequiresApproval != nil {
		update.SetRequiresApproval(*req.RequiresApproval)
	}
	if req.EstimatedDays != nil {
		update.SetEstimatedDays(*req.EstimatedDays)
	}

	// 执行更新
	updatedItem, err := update.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update service catalog item", "error", err, "item_id", id)
		return nil, fmt.Errorf("failed to update service item: %w", err)
	}

	s.logger.Infow("Service catalog item updated", "item_id", id)
	return s.toItemResponse(updatedItem), nil
}

// DeleteServiceCatalogItem 删除服务目录项
func (s *ServiceCatalogItemService) DeleteServiceCatalogItem(ctx context.Context, id int, tenantID int) error {
	// 验证服务项是否存在
	_, err := s.client.ServiceCatalogItem.Query().
		Where(
			servicecatalogitem.ID(id),
			servicecatalogitem.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("service item not found")
		}
		return fmt.Errorf("failed to get service item: %w", err)
	}

	// 软删除，设置为不活跃
	_, err = s.client.ServiceCatalogItem.UpdateOneID(id).
		SetIsActive(false).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to delete service catalog item", "error", err, "item_id", id)
		return fmt.Errorf("failed to delete service item: %w", err)
	}

	s.logger.Infow("Service catalog item deleted", "item_id", id)
	return nil
}

// toItemResponse 转换为响应对象
func (s *ServiceCatalogItemService) toItemResponse(item *ent.ServiceCatalogItem) *dto.ServiceCatalogItemResponse {
	return &dto.ServiceCatalogItemResponse{
		ID:               item.ID,
		Name:             item.Name,
		Description:      item.Description,
		Details:          item.Details,
		Category:         item.Category,
		Icon:             item.Icon,
		FormSchema:       item.FormSchema,
		SlaID:            item.SlaID,
		ApprovalChainID:  item.ApprovalChainID,
		IsActive:         item.IsActive,
		RequiresApproval: item.RequiresApproval,
		EstimatedDays:    item.EstimatedDays,
		TenantID:         item.TenantID,
		CreatedAt:        item.CreatedAt,
		UpdatedAt:        item.UpdatedAt,
	}
}
