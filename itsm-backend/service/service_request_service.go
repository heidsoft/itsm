package service

import (
	"context"
	"fmt"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/servicerequest"

	"go.uber.org/zap"
)

type ServiceRequestService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

func NewServiceRequestService(client *ent.Client, logger *zap.SugaredLogger) *ServiceRequestService {
	return &ServiceRequestService{
		client: client,
		logger: logger,
	}
}

// CreateServiceRequest 创建服务请求
func (s *ServiceRequestService) CreateServiceRequest(ctx context.Context, req *dto.CreateServiceRequestRequest, requesterID, tenantID int) (*dto.ServiceRequestResponse, error) {
	request, err := s.client.ServiceRequest.Create().
		SetCatalogID(req.CatalogID).
		SetRequesterID(requesterID).
		SetReason(req.Reason).
		Save(ctx)

	if err != nil {
		s.logger.Errorf("创建服务请求失败: %v", err)
		return nil, fmt.Errorf("创建服务请求失败: %w", err)
	}

	return dto.ToServiceRequestResponse(request), nil
}

// GetServiceRequest 获取服务请求详情
func (s *ServiceRequestService) GetServiceRequest(ctx context.Context, id, tenantID int) (*dto.ServiceRequestResponse, error) {
	request, err := s.client.ServiceRequest.Query().
		Where(servicerequest.ID(id)).
		First(ctx)

	if err != nil {
		s.logger.Errorf("获取服务请求失败: %v", err)
		return nil, fmt.Errorf("获取服务请求失败: %w", err)
	}

	return dto.ToServiceRequestResponse(request), nil
}

// ListServiceRequests 获取服务请求列表
func (s *ServiceRequestService) ListServiceRequests(ctx context.Context, req *dto.GetServiceRequestsRequest, tenantID int) (*dto.ServiceRequestListResponse, error) {
	query := s.client.ServiceRequest.Query()

	// 添加过滤条件
	if req.Status != "" {
		query = query.Where(servicerequest.Status(req.Status))
	}

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取服务请求总数失败: %w", err)
	}

	// 分页查询
	requests, err := query.
		Order(ent.Desc(servicerequest.FieldCreatedAt)).
		Offset((req.Page - 1) * req.Size).
		Limit(req.Size).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("获取服务请求列表失败: %w", err)
	}

	// 转换为响应格式
	var requestResponses []dto.ServiceRequestResponse
	for _, request := range requests {
		requestResponses = append(requestResponses, *dto.ToServiceRequestResponse(request))
	}

	return &dto.ServiceRequestListResponse{
		Requests: requestResponses,
		Total:    total,
		Page:     req.Page,
		Size:     req.Size,
	}, nil
}

// UpdateServiceRequestStatus 更新服务请求状态
func (s *ServiceRequestService) UpdateServiceRequestStatus(ctx context.Context, id int, status string, tenantID int) error {
	err := s.client.ServiceRequest.UpdateOneID(id).
		SetStatus(status).
		Exec(ctx)

	if err != nil {
		s.logger.Errorf("更新服务请求状态失败: %v", err)
		return fmt.Errorf("更新服务请求状态失败: %w", err)
	}

	return nil
}
