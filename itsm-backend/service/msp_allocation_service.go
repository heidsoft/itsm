package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/mspallocation"
	"itsm-backend/ent/ticket"
	"itsm-backend/ent/tenant"
	"itsm-backend/ent/user"

	"go.uber.org/zap"
)

// TenantTypeMSP MSP服务提供商类型
const TenantTypeMSP = "msp"

// TenantTypeCustomer 客户类型
const TenantTypeCustomer = "customer"

// MSPAllocationService MSP 分配业务服务
type MSPAllocationService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewMSPAllocationService 创建 MSP 分配服务实例
func NewMSPAllocationService(client *ent.Client, logger *zap.SugaredLogger) *MSPAllocationService {
	return &MSPAllocationService{
		client: client,
		logger: logger,
	}
}

// Create 创建新的 MSP 分配
// operatorRole: 操作者角色，如果为 "super_admin" 或 "sysadmin" 则跳过租户类型验证
func (s *MSPAllocationService) Create(
	ctx context.Context,
	mspUserID int,
	customerTenantID int,
	role string,
	operatorRole ...string,
) (*dto.MSPAllocationDTO, error) {
	// 检查是否是管理员操作
	isAdmin := len(operatorRole) > 0 && (operatorRole[0] == "super_admin" || operatorRole[0] == "sysadmin")

	// 1. 验证 MSP 用户必须是 MSP 租户（管理员除外）
	if !isAdmin {
		u, err := s.client.User.Query().
			Where(user.IDEQ(mspUserID)).
			WithTenant().
			Only(ctx)
		if err != nil {
			return nil, fmt.Errorf("MSP用户不存在: %w", err)
		}
		if u.Edges.Tenant == nil || u.Edges.Tenant.Type != TenantTypeMSP {
			return nil, fmt.Errorf("用户不属于MSP租户")
		}
	}

	// 2. 验证客户租户（管理员除外，需要是 customer 类型）
	cust, err := s.client.Tenant.Query().
		Where(tenant.IDEQ(customerTenantID)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("客户租户不存在: %w", err)
	}
	if !isAdmin && cust.Type != TenantTypeCustomer {
		return nil, fmt.Errorf("目标租户不是客户类型")
	}

	// 3. 检查是否已存在有效的未解除分配
	exists, err := s.client.MSPAllocation.Query().
		Where(
			mspallocation.MspUserIDEQ(mspUserID),
			mspallocation.HasCustomerTenantWith(tenant.IDEQ(customerTenantID)),
			mspallocation.DeassignedAtIsNil(),
		).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询现有分配失败: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("已存在有效分配记录")
	}

	// 4. 如果有已解除的旧记录，标记为已解除（避免重复）
	// 查询可能存在的未解除记录并更新
	_, err = s.client.MSPAllocation.Update().
		Where(
			mspallocation.MspUserIDEQ(mspUserID),
			mspallocation.HasCustomerTenantWith(tenant.IDEQ(customerTenantID)),
			mspallocation.DeassignedAtNotNil(),
		).
		SetDeassignedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Warnw("更新旧分配记录失败", "error", err)
	}

	// 5. 创建新的分配记录
	alloc, err := s.client.MSPAllocation.Create().
		SetMspUserID(mspUserID).
		AddCustomerTenantIDs(customerTenantID).
		SetRole(role).
		SetAssignedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建分配记录失败: %w", err)
	}

	return s.toDTO(alloc)
}

// toDTO 转换为 DTO
func (s *MSPAllocationService) toDTO(a *ent.MSPAllocation) (*dto.MSPAllocationDTO, error) {
	// 查询关联的用户和租户
	ctx := context.Background()

	var mspUsername string
	var customerTenantID int
	var customerTenantName string

	mspUser, err := a.QueryMspUser().Only(ctx)
	if err == nil && mspUser != nil {
		mspUsername = mspUser.Name
	}

	customers, err := a.QueryCustomerTenant().All(ctx)
	if err == nil && len(customers) > 0 {
		customerTenantID = customers[0].ID
		customerTenantName = customers[0].Name
	}

	return &dto.MSPAllocationDTO{
		ID:                 a.ID,
		MSPUserID:          a.MspUserID,
		MSPUsername:        mspUsername,
		CustomerTenantID:   customerTenantID,
		CustomerTenantName: customerTenantName,
		Role:              a.Role,
		AssignedAt:        a.AssignedAt,
		DeassignedAt:       &a.DeassignedAt,
	}, nil
}

// ListByMSPUser 根据 MSP 用户 ID 获取其所有分配
func (s *MSPAllocationService) ListByMSPUser(ctx context.Context, mspUserID int) ([]*dto.MSPAllocationDTO, error) {
	allocations, err := s.client.MSPAllocation.Query().
		Where(mspallocation.MspUserIDEQ(mspUserID)).
		WithCustomerTenant().
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询分配失败: %w", err)
	}

	dtos := make([]*dto.MSPAllocationDTO, 0, len(allocations))
	for _, a := range allocations {
		dto, err := s.toDTO(a)
		if err != nil {
			s.logger.Warnw("转换DTO失败", "error", err)
			continue
		}
		dtos = append(dtos, dto)
	}

	return dtos, nil
}

// ListByCustomer 根据客户租户 ID 获取所有分配
func (s *MSPAllocationService) ListByCustomer(ctx context.Context, customerTenantID int) ([]*dto.MSPAllocationDTO, error) {
	allocations, err := s.client.MSPAllocation.Query().
		Where(mspallocation.HasCustomerTenantWith(tenant.IDEQ(customerTenantID))).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询分配失败: %w", err)
	}

	dtos := make([]*dto.MSPAllocationDTO, 0, len(allocations))
	for _, a := range allocations {
		dto, err := s.toDTO(a)
		if err != nil {
			s.logger.Warnw("转换DTO失败", "error", err)
			continue
		}
		dtos = append(dtos, dto)
	}

	return dtos, nil
}

// Deactivate 解除分配
func (s *MSPAllocationService) Deactivate(ctx context.Context, mspUserID int, customerTenantID int) error {
	_, err := s.client.MSPAllocation.Update().
		Where(
			mspallocation.MspUserIDEQ(mspUserID),
			mspallocation.HasCustomerTenantWith(tenant.IDEQ(customerTenantID)),
			mspallocation.DeassignedAtIsNil(),
		).
		SetDeassignedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("解除分配失败: %w", err)
	}
	return nil
}

// GetActiveAllocations 获取所有有效分配
func (s *MSPAllocationService) GetActiveAllocations(ctx context.Context) ([]*dto.MSPAllocationDTO, error) {
	allocations, err := s.client.MSPAllocation.Query().
		Where(mspallocation.DeassignedAtIsNil()).
		WithCustomerTenant().
		WithMspUser().
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询有效分配失败: %w", err)
	}

	dtos := make([]*dto.MSPAllocationDTO, 0, len(allocations))
	for _, a := range allocations {
		var mspUsername, customerTenantName string
		var customerTenantID int
		if a.Edges.MspUser != nil {
			mspUsername = a.Edges.MspUser.Name
		}
		if len(a.Edges.CustomerTenant) > 0 {
			customerTenantID = a.Edges.CustomerTenant[0].ID
			customerTenantName = a.Edges.CustomerTenant[0].Name
		}

		dtos = append(dtos, &dto.MSPAllocationDTO{
			ID:                 a.ID,
			MSPUserID:          a.MspUserID,
			MSPUsername:        mspUsername,
			CustomerTenantID:   customerTenantID,
			CustomerTenantName: customerTenantName,
			Role:              a.Role,
			AssignedAt:        a.AssignedAt,
			DeassignedAt:       &a.DeassignedAt,
		})
	}

	return dtos, nil
}

// GetMSPCustomers 获取指定 MSP 用户可访问的客户列表
func (s *MSPAllocationService) GetMSPCustomers(ctx context.Context, mspUserID int) ([]*ent.Tenant, error) {
	allocations, err := s.client.MSPAllocation.Query().
		Where(
			mspallocation.MspUserIDEQ(mspUserID),
			mspallocation.DeassignedAtIsNil(),
		).
		WithCustomerTenant().
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询客户列表失败: %w", err)
	}

	customers := make([]*ent.Tenant, 0, len(allocations))
	for _, a := range allocations {
		if len(a.Edges.CustomerTenant) > 0 {
			customers = append(customers, a.Edges.CustomerTenant[0])
		}
	}

	return customers, nil
}

// GetTicketsForCustomer 获取指定客户租户的工单列表
func (s *MSPAllocationService) GetTicketsForCustomer(ctx context.Context, customerTenantID int, page, pageSize int) ([]*ent.Ticket, int, error) {
	query := s.client.Ticket.Query().
		Where(ticket.TenantIDEQ(customerTenantID))

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("查询工单数量失败: %w", err)
	}

	tickets, err := query.
		Order(ent.Desc(ticket.FieldCreatedAt)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("查询工单列表失败: %w", err)
	}

	return tickets, total, nil
}
