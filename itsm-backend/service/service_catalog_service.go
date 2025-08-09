package service

import (
	"context"
	"fmt"
	"strconv"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/servicecatalog"

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
		if parsed, err := strconv.Atoi(req.DeliveryTime); err == nil {
			deliveryTime = parsed
		}
	}

	catalog, err := s.client.ServiceCatalog.Create().
		SetName(req.Name).
		SetCategory(req.Category).
		SetDescription(req.Description).
		SetDeliveryTime(deliveryTime).
		SetStatus(req.Status).
		SetTenantID(tenantID).
		Save(ctx)

	if err != nil {
		s.logger.Errorf("创建服务目录失败: %v", err)
		return nil, fmt.Errorf("创建服务目录失败: %w", err)
	}

	return dto.ToServiceCatalogResponse(catalog), nil
}

// UpdateServiceCatalog 更新服务目录
func (s *ServiceCatalogService) UpdateServiceCatalog(ctx context.Context, id int, req *dto.UpdateServiceCatalogRequest, tenantID int) (*dto.ServiceCatalogResponse, error) {
	update := s.client.ServiceCatalog.UpdateOneID(id)

	if req.Name != "" {
		update = update.SetName(req.Name)
	}
	if req.Category != "" {
		update = update.SetCategory(req.Category)
	}
	if req.Description != "" {
		update = update.SetDescription(req.Description)
	}
	if req.DeliveryTime != "" {
		if deliveryTime, err := strconv.Atoi(req.DeliveryTime); err == nil {
			update = update.SetDeliveryTime(deliveryTime)
		}
	}
	if req.Status != "" {
		update = update.SetStatus(req.Status)
	}

	catalog, err := update.Save(ctx)
	if err != nil {
		s.logger.Errorf("更新服务目录失败: %v", err)
		return nil, fmt.Errorf("更新服务目录失败: %w", err)
	}

	return dto.ToServiceCatalogResponse(catalog), nil
}

// DeleteServiceCatalog 删除服务目录
func (s *ServiceCatalogService) DeleteServiceCatalog(ctx context.Context, id int) error {
	return s.client.ServiceCatalog.DeleteOneID(id).Exec(ctx)
}

// GetServiceCatalogByID 根据ID获取服务目录
func (s *ServiceCatalogService) GetServiceCatalogByID(ctx context.Context, id int) (*dto.ServiceCatalogResponse, error) {
	catalog, err := s.client.ServiceCatalog.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return dto.ToServiceCatalogResponse(catalog), nil
}
