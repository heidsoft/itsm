package service

import (
	"context"
	"fmt"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/servicecatalog"
)

// ServiceCatalogService 服务目录服务
type ServiceCatalogService struct {
	client *ent.Client
}

// NewServiceCatalogService 创建服务目录服务实例
func NewServiceCatalogService(client *ent.Client) *ServiceCatalogService {
	return &ServiceCatalogService{
		client: client,
	}
}

// GetServiceCatalogs 获取服务目录列表
func (s *ServiceCatalogService) GetServiceCatalogs(ctx context.Context, req *dto.GetServiceCatalogsRequest) (*dto.ServiceCatalogListResponse, error) {
	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 10
	}
	
	// 构建查询条件
	query := s.client.ServiceCatalog.Query()
	
	// 分类筛选
	if req.Category != "" {
		query = query.Where(servicecatalog.CategoryEQ(req.Category))
	}
	
	// 状态筛选
	if req.Status != "" {
		query = query.Where(servicecatalog.StatusEQ(servicecatalog.Status(req.Status)))
	}
	
	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取服务目录总数失败: %w", err)
	}
	
	// 分页查询
	catalogs, err := query.
		Offset((req.Page - 1) * req.Size).
		Limit(req.Size).
		Order(ent.Desc(servicecatalog.FieldCreatedAt)).
		All(ctx)
	
	if err != nil {
		return nil, fmt.Errorf("获取服务目录列表失败: %w", err)
	}
	
	// 转换响应
	catalogResponses := make([]dto.ServiceCatalogResponse, len(catalogs))
	for i, catalog := range catalogs {
		catalogResponses[i] = *dto.ToServiceCatalogResponse(catalog)
	}
	
	return &dto.ServiceCatalogListResponse{
		Catalogs: catalogResponses,
		Total:    total,
		Page:     req.Page,
		Size:     req.Size,
	}, nil
}

// GetServiceCatalogByID 根据ID获取服务目录
func (s *ServiceCatalogService) GetServiceCatalogByID(ctx context.Context, id int) (*ent.ServiceCatalog, error) {
	catalog, err := s.client.ServiceCatalog.Query().
		Where(servicecatalog.ID(id)).
		First(ctx)
	
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("服务目录不存在")
		}
		return nil, fmt.Errorf("获取服务目录失败: %w", err)
	}
	
	return catalog, nil
}

// CreateServiceCatalog 创建服务目录
func (s *ServiceCatalogService) CreateServiceCatalog(ctx context.Context, req *dto.CreateServiceCatalogRequest) (*ent.ServiceCatalog, error) {
	status := servicecatalog.StatusEnabled
	if req.Status != "" {
		status = servicecatalog.Status(req.Status)
	}

	catalog, err := s.client.ServiceCatalog.Create().
		SetName(req.Name).
		SetCategory(req.Category).
		SetDescription(req.Description).
		SetDeliveryTime(req.DeliveryTime).
		SetStatus(status).
		Save(ctx)

	if err != nil {
		return nil, fmt.Errorf("创建服务目录失败: %w", err)
	}

	return catalog, nil
}

// UpdateServiceCatalog 更新服务目录
func (s *ServiceCatalogService) UpdateServiceCatalog(ctx context.Context, id int, req *dto.UpdateServiceCatalogRequest) (*ent.ServiceCatalog, error) {
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
		update = update.SetDeliveryTime(req.DeliveryTime)
	}
	if req.Status != "" {
		update = update.SetStatus(servicecatalog.Status(req.Status))
	}

	catalog, err := update.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("服务目录不存在")
		}
		return nil, fmt.Errorf("更新服务目录失败: %w", err)
	}

	return catalog, nil
}

// DeleteServiceCatalog 删除服务目录
func (s *ServiceCatalogService) DeleteServiceCatalog(ctx context.Context, id int) error {
	err := s.client.ServiceCatalog.DeleteOneID(id).Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("服务目录不存在")
		}
		return fmt.Errorf("删除服务目录失败: %w", err)
	}
	return nil
}