package service

import (
	"context"
	"fmt"
	"strconv"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/servicecatalog"
	"itsm-backend/ent/servicecatalogitem"

	"go.uber.org/zap"
)

type ServiceCatalogService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

func NewServiceCatalogService(client *ent.Client, logger *zap.SugaredLogger) *ServiceCatalogService {
	return &ServiceCatalogService{
		client: client,
		logger: logger,
	}
}

// ListServiceCatalogs 获取服务目录列表
func (s *ServiceCatalogService) ListServiceCatalogs(ctx context.Context, req *dto.GetServiceCatalogsRequest, tenantID int) (*dto.ServiceCatalogListResponse, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Size < 1 {
		req.Size = 20
	}
	if req.Size > 200 {
		req.Size = 200
	}
	query := s.client.ServiceCatalog.Query().
		Where(servicecatalog.TenantID(tenantID))

	// 添加过滤条件
	if req.Category != "" {
		query = query.Where(servicecatalog.Category(req.Category))
	}
	if req.Status != "" {
		query = query.Where(servicecatalog.Status(req.Status))
	}

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取服务目录总数失败: %w", err)
	}

	// 分页查询
	catalogs, err := query.
		Order(ent.Desc(servicecatalog.FieldCreatedAt)).
		Offset((req.Page - 1) * req.Size).
		Limit(req.Size).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取服务目录列表失败: %w", err)
	}

	// 转换为响应格式
	var catalogResponses []dto.ServiceCatalogResponse
	for _, catalog := range catalogs {
		catalogResponses = append(catalogResponses, *dto.ToServiceCatalogResponse(catalog))
	}

	return &dto.ServiceCatalogListResponse{
		Catalogs: catalogResponses,
		Total:    total,
		Page:     req.Page,
		Size:     req.Size,
	}, nil
}

// CreateServiceCatalog 创建服务目录
func (s *ServiceCatalogService) CreateServiceCatalog(ctx context.Context, req *dto.CreateServiceCatalogRequest, tenantID int) (*dto.ServiceCatalogResponse, error) {
	// 将DeliveryTime字符串转换为整数
	deliveryTime := 0
	if req.DeliveryTime != "" {
		parsed, err := strconv.Atoi(req.DeliveryTime)
		if err != nil || parsed < 0 || parsed > 3650 {
			return nil, fmt.Errorf("交付时间必须是 0 到 3650 之间的整数天数")
		}
		deliveryTime = parsed
	}
	if req.Status != "" && req.Status != "active" && req.Status != "inactive" && req.Status != "enabled" && req.Status != "disabled" {
		return nil, fmt.Errorf("无效的服务目录状态")
	}

	requiresApproval := true
	if req.RequiresApproval != nil {
		requiresApproval = *req.RequiresApproval
	}

	create := s.client.ServiceCatalog.Create().
		SetName(req.Name).
		SetCategory(req.Category).
		SetDescription(req.Description).
		SetDeliveryTime(deliveryTime).
		SetStatus(req.Status).
		SetTenantID(tenantID).
		SetIcon(req.Icon).
		SetServiceType(req.ServiceType).
		SetPrice(req.Price).
		SetUnit(req.Unit).
		SetRequiresApproval(requiresApproval).
		SetApprovalLevel(req.ApprovalLevel).
		SetApprovers(req.Approvers).
		SetSLAResponseTime(req.SLAResponseTime).
		SetSLAResolutionTime(req.SLAResolutionTime).
		SetFormSchema(req.FormSchema).
		SetAvailableRegions(req.AvailableRegions).
		SetAvailableSpecs(req.AvailableSpecs).
		SetSortOrder(req.SortOrder)

	catalog, err := create.Save(ctx)
	if err != nil {
		s.logger.Errorf("创建服务目录失败: %v", err)
		return nil, fmt.Errorf("创建服务目录失败: %w", err)
	}

	return dto.ToServiceCatalogResponse(catalog), nil
}

// UpdateServiceCatalog 更新服务目录
func (s *ServiceCatalogService) UpdateServiceCatalog(ctx context.Context, id int, req *dto.UpdateServiceCatalogRequest, tenantID int) (*dto.ServiceCatalogResponse, error) {
	update := s.client.ServiceCatalog.UpdateOneID(id).Where(servicecatalog.TenantID(tenantID))

	if req.Name != nil && *req.Name != "" {
		update = update.SetName(*req.Name)
	}
	if req.Category != nil && *req.Category != "" {
		update = update.SetCategory(*req.Category)
	}
	if req.Description != nil && *req.Description != "" {
		update = update.SetDescription(*req.Description)
	}
	if req.DeliveryTime != nil && *req.DeliveryTime != "" {
		deliveryTime, parseErr := strconv.Atoi(*req.DeliveryTime)
		if parseErr != nil || deliveryTime < 0 || deliveryTime > 3650 {
			return nil, fmt.Errorf("交付时间必须是 0 到 3650 之间的整数天数")
		}
		update = update.SetDeliveryTime(deliveryTime)
	}
	if req.Status != nil && *req.Status != "" {
		if *req.Status != "active" && *req.Status != "inactive" && *req.Status != "enabled" && *req.Status != "disabled" {
			return nil, fmt.Errorf("无效的服务目录状态")
		}
		update = update.SetStatus(*req.Status)
	}
	if req.Icon != nil {
		update = update.SetIcon(*req.Icon)
	}
	if req.ServiceType != nil {
		update = update.SetServiceType(*req.ServiceType)
	}
	if req.Price != nil {
		update = update.SetPrice(*req.Price)
	}
	if req.Unit != nil {
		update = update.SetUnit(*req.Unit)
	}
	if req.RequiresApproval != nil {
		update = update.SetRequiresApproval(*req.RequiresApproval)
	}
	if req.ApprovalLevel != nil {
		update = update.SetApprovalLevel(*req.ApprovalLevel)
	}
	if req.Approvers != nil {
		update = update.SetApprovers(req.Approvers)
	}
	if req.SLAResponseTime != nil {
		update = update.SetSLAResponseTime(*req.SLAResponseTime)
	}
	if req.SLAResolutionTime != nil {
		update = update.SetSLAResolutionTime(*req.SLAResolutionTime)
	}
	if req.FormSchema != nil {
		update = update.SetFormSchema(*req.FormSchema)
	}
	if req.AvailableRegions != nil {
		update = update.SetAvailableRegions(req.AvailableRegions)
	}
	if req.AvailableSpecs != nil {
		update = update.SetAvailableSpecs(req.AvailableSpecs)
	}
	if req.SortOrder != nil {
		update = update.SetSortOrder(*req.SortOrder)
	}

	catalog, err := update.Save(ctx)
	if err != nil {
		s.logger.Errorf("更新服务目录失败: %v", err)
		return nil, fmt.Errorf("更新服务目录失败: %w", err)
	}

	return dto.ToServiceCatalogResponse(catalog), nil
}

// DeleteServiceCatalog 删除服务目录
func (s *ServiceCatalogService) DeleteServiceCatalog(ctx context.Context, id int, tenantID int) error {
	itemCount, err := s.client.ServiceCatalogItem.Query().Where(servicecatalogitem.CatalogID(id), servicecatalogitem.TenantID(tenantID)).Count(ctx)
	if err != nil {
		return fmt.Errorf("检查目录项失败: %w", err)
	}
	if itemCount > 0 {
		return fmt.Errorf("无法删除服务目录：请先下架或迁移其目录项")
	}
	_, err = s.client.ServiceCatalog.Delete().
		Where(
			servicecatalog.IDEQ(id),
			servicecatalog.TenantID(tenantID),
		).
		Exec(ctx)
	return err
}

// GetServiceCatalogByID 根据ID获取服务目录
func (s *ServiceCatalogService) GetServiceCatalogByID(ctx context.Context, id int, tenantID int) (*dto.ServiceCatalogResponse, error) {
	catalog, err := s.client.ServiceCatalog.Query().
		Where(
			servicecatalog.IDEQ(id),
			servicecatalog.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return dto.ToServiceCatalogResponse(catalog), nil
}
