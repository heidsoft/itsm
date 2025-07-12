package service

import (
	"context"
	"fmt"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/servicerequest"
)

// ServiceRequestService 服务请求服务
type ServiceRequestService struct {
	client               *ent.Client
	serviceCatalogService *ServiceCatalogService
}

// NewServiceRequestService 创建服务请求服务实例
func NewServiceRequestService(client *ent.Client, catalogService *ServiceCatalogService) *ServiceRequestService {
	return &ServiceRequestService{
		client:               client,
		serviceCatalogService: catalogService,
	}
}

// CreateServiceRequest 创建服务请求
func (s *ServiceRequestService) CreateServiceRequest(ctx context.Context, req *dto.CreateServiceRequestRequest, requesterID int) (*ent.ServiceRequest, error) {
	// 验证服务目录是否存在且启用
	catalog, err := s.serviceCatalogService.GetServiceCatalogByID(ctx, req.CatalogID)
	if err != nil {
		return nil, err
	}
	
	if catalog.Status != "enabled" {
		return nil, fmt.Errorf("服务目录已禁用")
	}
	
	// 创建服务请求
	serviceRequest, err := s.client.ServiceRequest.Create().
		SetCatalogID(req.CatalogID).
		SetRequesterID(requesterID).
		SetNillableReason(&req.Reason).
		Save(ctx)
	
	if err != nil {
		return nil, fmt.Errorf("创建服务请求失败: %w", err)
	}
	
	return serviceRequest, nil
}

// GetServiceRequestByID 根据ID获取服务请求
func (s *ServiceRequestService) GetServiceRequestByID(ctx context.Context, id int) (*ent.ServiceRequest, error) {
	serviceRequest, err := s.client.ServiceRequest.Query().
		Where(servicerequest.ID(id)).
		WithCatalog().
		WithRequester().
		First(ctx)
	
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("服务请求不存在")
		}
		return nil, fmt.Errorf("获取服务请求失败: %w", err)
	}
	
	return serviceRequest, nil
}

// GetUserServiceRequests 获取用户的服务请求列表
func (s *ServiceRequestService) GetUserServiceRequests(ctx context.Context, req *dto.GetServiceRequestsRequest) (*dto.ServiceRequestListResponse, error) {
	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 10
	}
	
	// 构建查询条件
	query := s.client.ServiceRequest.Query().
		Where(servicerequest.RequesterIDEQ(req.UserID))
	
	// 状态筛选
	if req.Status != "" {
		query = query.Where(servicerequest.StatusEQ(servicerequest.Status(req.Status)))
	}
	
	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取服务请求总数失败: %w", err)
	}
	
	// 分页查询
	requests, err := query.
		WithCatalog().
		WithRequester().
		Offset((req.Page - 1) * req.Size).
		Limit(req.Size).
		Order(ent.Desc(servicerequest.FieldCreatedAt)).
		All(ctx)
	
	if err != nil {
		return nil, fmt.Errorf("获取服务请求列表失败: %w", err)
	}
	
	// 转换响应
	requestResponses := make([]dto.ServiceRequestResponse, len(requests))
	for i, request := range requests {
		requestResponses[i] = *dto.ToServiceRequestResponse(request)
	}
	
	return &dto.ServiceRequestListResponse{
		Requests: requestResponses,
		Total:    total,
		Page:     req.Page,
		Size:     req.Size,
	}, nil
}

// UpdateServiceRequestStatus 更新服务请求状态
func (s *ServiceRequestService) UpdateServiceRequestStatus(ctx context.Context, id int, status string, userID int) (*ent.ServiceRequest, error) {
	// 检查服务请求是否存在
	_, err := s.GetServiceRequestByID(ctx, id)
	if err != nil {
		return nil, err
	}
	
	// 这里可以添加权限检查逻辑
	// 例如：只有管理员或审批人可以更新状态
	
	// 更新状态
	updatedRequest, err := s.client.ServiceRequest.UpdateOneID(id).
		SetStatus(servicerequest.Status(status)).
		Save(ctx)
	
	if err != nil {
		return nil, fmt.Errorf("更新服务请求状态失败: %w", err)
	}
	
	return updatedRequest, nil
}