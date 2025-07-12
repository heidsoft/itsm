package service

import (
	"context"
	"fmt"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/tenant"
	"time"

	"go.uber.org/zap"
)

type TenantService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

func NewTenantService(client *ent.Client, logger *zap.SugaredLogger) *TenantService {
	return &TenantService{
		client: client,
		logger: logger,
	}
}

// CreateTenant 创建租户
func (s *TenantService) CreateTenant(ctx context.Context, req *dto.CreateTenantRequest) (*ent.Tenant, error) {
	// 检查租户代码是否已存在
	exists, err := s.client.Tenant.
		Query().
		Where(tenant.CodeEQ(req.Code)).
		Exist(ctx)
	if err != nil {
		s.logger.Errorf("检查租户代码失败: %v", err)
		return nil, fmt.Errorf("检查租户代码失败: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("租户代码已存在: %s", req.Code)
	}

	// 创建租户
	tenantEntity, err := s.client.Tenant.
		Create().
		SetName(req.Name).
		SetCode(req.Code).
		SetNillableDomain(req.Domain).
		SetType(req.Type).
		SetStatus(string(TenantStatusActive)).
		SetSettings(req.Settings).
		SetQuota(req.Quota).
		SetNillableExpiresAt(req.ExpiresAt).
		Save(ctx)
	if err != nil {
		s.logger.Errorf("创建租户失败: %v", err)
		return nil, fmt.Errorf("创建租户失败: %w", err)
	}

	s.logger.Infof("成功创建租户: %s (%s)", tenantEntity.Name, tenantEntity.Code)
	return tenantEntity, nil
}

// GetTenantByCode 根据代码获取租户
func (s *TenantService) GetTenantByCode(ctx context.Context, code string) (*ent.Tenant, error) {
	tenantEntity, err := s.client.Tenant.
		Query().
		Where(tenant.CodeEQ(code)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("租户不存在: %s", code)
		}
		s.logger.Errorf("查询租户失败: %v", err)
		return nil, fmt.Errorf("查询租户失败: %w", err)
	}
	return tenantEntity, nil
}

// UpdateTenantStatus 更新租户状态
func (s *TenantService) UpdateTenantStatus(ctx context.Context, tenantID int, status string) error {
	err := s.client.Tenant.
		UpdateOneID(tenantID).
		SetStatus(status).
		SetUpdatedAt(time.Now()).
		Exec(ctx)
	if err != nil {
		s.logger.Errorf("更新租户状态失败: %v", err)
		return fmt.Errorf("更新租户状态失败: %w", err)
	}

	s.logger.Infof("成功更新租户状态: %d -> %s", tenantID, status)
	return nil
}

// ListTenants 获取租户列表
func (s *TenantService) ListTenants(ctx context.Context, req *dto.ListTenantsRequest) ([]*ent.Tenant, int, error) {
	query := s.client.Tenant.Query()

	// 状态过滤
	if req.Status != "" {
		query = query.Where(tenant.StatusEQ(req.Status))
	}

	// 类型过滤
	if req.Type != "" {
		query = query.Where(tenant.TypeEQ(req.Type))
	}

	// 搜索过滤
	if req.Search != "" {
		query = query.Where(
			tenant.Or(
				tenant.NameContains(req.Search),
				tenant.CodeContains(req.Search),
			),
		)
	}

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		s.logger.Errorf("获取租户总数失败: %v", err)
		return nil, 0, fmt.Errorf("获取租户总数失败: %w", err)
	}

	// 分页查询
	tenants, err := query.
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		Order(ent.Desc(tenant.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		s.logger.Errorf("获取租户列表失败: %v", err)
		return nil, 0, fmt.Errorf("获取租户列表失败: %w", err)
	}

	return tenants, total, nil
}
