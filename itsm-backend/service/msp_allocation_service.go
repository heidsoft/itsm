package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/mspallocation"
	"itsm-backend/ent/predicate"
	"itsm-backend/ent/tenant"
	"itsm-backend/ent/ticket"
	"itsm-backend/ent/user"
	"itsm-backend/pkg/tenantmode"

	"go.uber.org/zap"
)

// ErrMSPAllocationNotFound 表示找不到匹配的活跃分配：记录不存在、已经解除，
// 或不属于当前 MSP 租户。handler 据此映射 404，不得降级成静默成功。
var ErrMSPAllocationNotFound = errors.New("msp allocation not active")

// MSPAllocationHistoryFilter 分配历史查询条件。
// 可选项使用指针，区分「未传」与「传零值」。
type MSPAllocationHistoryFilter struct {
	// MSPTenantID 是调用者所在 MSP 租户，必填且是唯一的数据边界。
	MSPTenantID      int
	MSPUserID        *int
	CustomerTenantID *int
	// AssignedFrom/AssignedTo 按 assigned_at 过滤分配发生时间（闭区间）。
	AssignedFrom *time.Time
	AssignedTo   *time.Time
	Page         int
	PageSize     int
}

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
		if u.Edges.Tenant == nil || !tenantmode.IsMSPProviderTenantType(string(u.Edges.Tenant.Type)) {
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
	if !isAdmin && !tenantmode.IsCustomerTenantType(string(cust.Type)) {
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

	// 4. 创建新的分配记录。
	// 历史实现在这里还执行过一次「把 DeassignedAtNotNil 的旧记录 SetDeassignedAt(now)」
	// 的更新：它匹配的是已经解除的归档行，于是每次重新分配都会把这些行的历史
	// 结束时间改写为当前时间，分配历史因此不可信。活跃重复已由上面的存在性
	// 检查挡下，这里没有任何需要顺带关闭的记录。
	alloc, err := s.client.MSPAllocation.Create().
		SetMspUserID(mspUserID).
		SetCustomerTenantID(customerTenantID).
		SetRole(role).
		SetAssignedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建分配记录失败: %w", err)
	}

	return s.toDTO(ctx, alloc)
}

// toDTO 转换为 DTO。
// 边的读取必须沿用调用链的 ctx：历史实现固定用 context.Background()，
// 请求取消和超时传不进来，也拿不到链路追踪信息。
func (s *MSPAllocationService) toDTO(ctx context.Context, a *ent.MSPAllocation) (*dto.MSPAllocationDTO, error) {
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

	var deassignedAt *time.Time
	if !a.DeassignedAt.IsZero() {
		deassignedAt = &a.DeassignedAt
	}

	return &dto.MSPAllocationDTO{
		ID:                 a.ID,
		MSPUserID:          a.MspUserID,
		MSPUsername:        mspUsername,
		CustomerTenantID:   customerTenantID,
		CustomerTenantName: customerTenantName,
		Role:               a.Role,
		AssignedAt:         a.AssignedAt,
		DeassignedAt:       deassignedAt,
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
		dto, err := s.toDTO(ctx, a)
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
		dto, err := s.toDTO(ctx, a)
		if err != nil {
			s.logger.Warnw("转换DTO失败", "error", err)
			continue
		}
		dtos = append(dtos, dto)
	}

	return dtos, nil
}

// Deactivate 解除分配。
// 只有确实存在一条活跃分配时才写入结束时间；否则返回 ErrMSPAllocationNotFound，
// handler 映射为 404，而不是把「什么都没解除」报成成功。
func (s *MSPAllocationService) Deactivate(ctx context.Context, mspUserID int, customerTenantID int) error {
	affected, err := s.client.MSPAllocation.Update().
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
	if affected == 0 {
		return ErrMSPAllocationNotFound
	}
	return nil
}

// ListHistory 返回当前 MSP 租户内的分配历史，包含已解除的记录。
//
// 数据边界是 msp_user 所属租户：msp_allocations 表本身没有 tenant_id 列，
// 因此必须通过 HasMspUserWith(user.TenantIDEQ(...)) 收敛，调用者只能读到
// 自己 MSP 租户员工的历史行。
func (s *MSPAllocationService) ListHistory(
	ctx context.Context,
	filter MSPAllocationHistoryFilter,
) ([]*dto.MSPAllocationDTO, int, error) {
	if filter.MSPTenantID == 0 {
		return nil, 0, errors.New("缺少 MSP 租户上下文")
	}

	predicates := []predicate.MSPAllocation{
		mspallocation.HasMspUserWith(user.TenantIDEQ(filter.MSPTenantID)),
	}
	if filter.MSPUserID != nil {
		predicates = append(predicates, mspallocation.MspUserIDEQ(*filter.MSPUserID))
	}
	if filter.CustomerTenantID != nil {
		predicates = append(predicates, mspallocation.HasCustomerTenantWith(tenant.IDEQ(*filter.CustomerTenantID)))
	}
	if filter.AssignedFrom != nil {
		predicates = append(predicates, mspallocation.AssignedAtGTE(*filter.AssignedFrom))
	}
	if filter.AssignedTo != nil {
		// endDate 是日期（无时刻），按「包含当天」处理：上界取次日 00:00 的开区间，
		// 否则当天 00:00 之后发生的分配会被静默丢掉。
		predicates = append(predicates, mspallocation.AssignedAtLT(filter.AssignedTo.AddDate(0, 0, 1)))
	}

	page, pageSize := filter.Page, filter.PageSize
	if page <= 0 {
		page = 1
	}
	// 与 common.GetPaginationFromQuery 的契约保持一致：非法或越界一律回到默认窗口，
	// 避免 offset 为负或 limit 无上限。
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	base := s.client.MSPAllocation.Query().Where(predicates...)

	// Count 与 List 复用同一条件；Ent 的 Where 会就地修改接收者，必须先 Clone。
	total, err := base.Clone().Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("查询分配历史数量失败: %w", err)
	}

	allocations, err := base.
		WithMspUser().
		WithCustomerTenant().
		Order(mspallocation.ByAssignedAt(sql.OrderDesc())).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("查询分配历史失败: %w", err)
	}

	dtos := make([]*dto.MSPAllocationDTO, 0, len(allocations))
	for _, a := range allocations {
		dtos = append(dtos, allocationDTOFromEdges(a))
	}
	return dtos, total, nil
}

// allocationDTOFromEdges 从已预加载的边构造 DTO，避免逐行再查用户与租户。
func allocationDTOFromEdges(a *ent.MSPAllocation) *dto.MSPAllocationDTO {
	var mspUsername, customerTenantName string
	var customerTenantID int
	if a.Edges.MspUser != nil {
		mspUsername = a.Edges.MspUser.Name
	}
	if a.Edges.CustomerTenant != nil {
		customerTenantID = a.Edges.CustomerTenant.ID
		customerTenantName = a.Edges.CustomerTenant.Name
	}

	var deassignedAt *time.Time
	if !a.DeassignedAt.IsZero() {
		deassignedAt = &a.DeassignedAt
	}

	return &dto.MSPAllocationDTO{
		ID:                 a.ID,
		MSPUserID:          a.MspUserID,
		MSPUsername:        mspUsername,
		CustomerTenantID:   customerTenantID,
		CustomerTenantName: customerTenantName,
		Role:               a.Role,
		AssignedAt:         a.AssignedAt,
		DeassignedAt:       deassignedAt,
	}
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
		dtos = append(dtos, allocationDTOFromEdges(a))
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
		if a.Edges.CustomerTenant != nil {
			customers = append(customers, a.Edges.CustomerTenant)
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
