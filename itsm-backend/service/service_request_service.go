package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"itsm-backend/common" // Import common package
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/servicerequest"
	"itsm-backend/ent/servicerequestapproval"
	"itsm-backend/ent/user"

	"net/http" // Import net/http for http.StatusBadRequest

	"go.uber.org/zap"
)

// Constants for approval steps
const (
	ApprovalStepManager  = "manager"
	ApprovalStepIT       = "it"
	ApprovalStepSecurity = "security"
)

type ServiceRequestService struct {
	client          *ent.Client
	logger          *zap.SugaredLogger
	approvalService *ApprovalService
}

func NewServiceRequestService(client *ent.Client, logger *zap.SugaredLogger, approvalService *ApprovalService) *ServiceRequestService {
	return &ServiceRequestService{
		client:          client,
		logger:          logger,
		approvalService: approvalService,
	}
}

const (
	SRStatusSubmitted        = "submitted"
	SRStatusManagerApproved  = "manager_approved"
	SRStatusITApproved       = "it_approved"
	SRStatusSecurityApproved = "security_approved"
	SRStatusRejected         = "rejected"

	ApprovalStatusPending  = "pending"
	ApprovalStatusApproved = "approved"
	ApprovalStatusRejected = "rejected"

	// V1 审批时限配置（小时）
	ApprovalTimeoutManager  = 24 // 部门主管审批：24小时
	ApprovalTimeoutIT       = 48 // IT审批：48小时
	ApprovalTimeoutSecurity = 72 // 安全审批：72小时
)

// CreateServiceRequest 创建服务请求
func (s *ServiceRequestService) CreateServiceRequest(ctx context.Context, req *dto.CreateServiceRequestRequest, requesterID, tenantID int) (*dto.ServiceRequestResponse, error) {
	// 合规确认（V0：必须确认）
	if !req.ComplianceAck {
		return nil, common.NewBadRequestError("必须确认合规条款", nil) // Using common.NewBadRequestError
	}
	// V0 最小策略门禁：申请公网必须提供源 IP 白名单；到期时间建议必须（先作为强校验，便于后续回收作业）
	if req.NeedsPublicIP {
		if len(req.SourceIPWhitelist) == 0 {
			return nil, common.NewBadRequestError("申请公网访问必须提供源IP白名单", nil) // Using common.NewBadRequestError
		}
	}
	if req.ExpireAt == nil {
		return nil, common.NewBadRequestError("必须提供到期时间（用于自动回收）", nil) // Using common.NewBadRequestError
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("开启事务失败: %w", err) // Keep fmt.Errorf for internal errors for now
	}
	defer tx.Rollback()

	create := tx.ServiceRequest.Create().
		SetTenantID(tenantID).
		SetCatalogID(req.CatalogID).
		SetRequesterID(requesterID).
		SetStatus(SRStatusSubmitted).
		SetCurrentLevel(1).
		SetTotalLevels(3).
		SetComplianceAck(req.ComplianceAck).
		SetNeedsPublicIP(req.NeedsPublicIP).
		SetDataClassification(req.DataClassification)

	if req.Title != "" {
		create = create.SetTitle(req.Title)
	}
	if req.Reason != "" {
		create = create.SetReason(req.Reason)
	}
	if req.FormData != nil {
		create = create.SetFormData(req.FormData)
	}
	if req.CostCenter != "" {
		create = create.SetCostCenter(req.CostCenter)
	}
	if req.SourceIPWhitelist != nil {
		create = create.SetSourceIPWhitelist(req.SourceIPWhitelist)
	}
	if req.ExpireAt != nil {
		create = create.SetExpireAt(*req.ExpireAt)
	}

	request, err := create.Save(ctx)

	if err != nil {
		s.logger.Errorf("创建服务请求失败: %v", err)
		return nil, fmt.Errorf("创建服务请求失败: %w", err) // Keep fmt.Errorf for internal errors for now
	}

	// 创建默认三段审批记录（主管→IT→安全），先全部 pending
	steps := []struct {
		level        int
		step         string
		timeoutHours int
	}{
		{1, ApprovalStepManager, ApprovalTimeoutManager},
		{2, ApprovalStepIT, ApprovalTimeoutIT},
		{3, ApprovalStepSecurity, ApprovalTimeoutSecurity},
	}
	now := time.Now()
	for _, st := range steps {
		dueAt := now.Add(time.Duration(st.timeoutHours) * time.Hour)
		_, err := tx.ServiceRequestApproval.Create().
			SetTenantID(tenantID).
			SetServiceRequestID(request.ID).
			SetLevel(st.level).
			SetStep(st.step).
			SetStatus(ApprovalStatusPending).
			SetTimeoutHours(st.timeoutHours).
			SetDueAt(dueAt).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("创建审批记录失败: %w", err) // Keep fmt.Errorf for internal errors for now
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err) // Keep fmt.Errorf for internal errors for now
	}

	// 返回详情（含 approvals）
	return s.GetServiceRequestDetail(ctx, request.ID, tenantID)
}

// ListPendingApprovals 返回当前用户的审批待办（按角色/步骤过滤）。
// 说明：V0 先实现最小可用版本，后续可升级为“审批链配置 + 组织关系”驱动。
func (s *ServiceRequestService) ListPendingApprovals(ctx context.Context, tenantID, actorUserID int, page, size int) (*dto.ServiceRequestListResponse, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 10
	}

	actor, err := s.client.User.Query().
		Where(user.ID(actorUserID), user.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		return nil, common.NewAppError(common.ErrCodeNotFound, "审批人不存在", http.StatusNotFound, err)
	}

	role := strings.ToLower(strings.TrimSpace(actor.Role.String()))
	var targetLevel int
	var requiredStatus string

	switch role {
	case ApprovalStepManager: // Using constant. Assuming role string "manager" matches ApprovalStepManager
		targetLevel = 1
		requiredStatus = SRStatusSubmitted
	case ApprovalStepIT, "agent", "technician": // 'agent' and 'technician' are also IT roles
		targetLevel = 2
		requiredStatus = SRStatusManagerApproved
	case ApprovalStepSecurity:
		targetLevel = 3
		requiredStatus = SRStatusITApproved
	case "admin", "super_admin": // admin and super_admin can view all pending requests
		targetLevel = 0
	default:
		return &dto.ServiceRequestListResponse{Requests: []dto.ServiceRequestResponse{}, Total: 0, Page: page, Size: size}, nil
	}

	base := s.client.ServiceRequest.Query().Where(servicerequest.TenantID(tenantID))
	if targetLevel > 0 {
		base = base.Where(servicerequest.CurrentLevel(targetLevel)).Where(servicerequest.Status(requiredStatus))
	} else {
		// admin/super_admin: any request in submitted/manager_approved/it_approved status is considered pending
		base = base.Where(servicerequest.StatusIn(SRStatusSubmitted, SRStatusManagerApproved, SRStatusITApproved))
	}

	total, err := base.Count(ctx)
	if err != nil {
		return nil, common.NewInternalError("获取待办总数失败", err)
	}

	requests, err := base.
		Order(ent.Desc(servicerequest.FieldCreatedAt)).
		Offset((page - 1) * size).
		Limit(size).
		All(ctx)
	if err != nil {
		return nil, common.NewInternalError("获取待办列表失败", err)
	}

	// manager：只看同部门。当前为 V0 最小实现，通过内存过滤完成。
	// 注意：在大规模数据下，此方法可能导致性能瓶颈。未来优化方向是将部门过滤条件推送到数据库查询层。
	if role == ApprovalStepManager { // Using constant.
		reqIDs := make([]int, 0, len(requests))
		for _, r := range requests {
			reqIDs = append(reqIDs, r.RequesterID)
		}
		users, _ := s.client.User.Query().
			Where(user.TenantIDEQ(tenantID)).
			Where(user.IDIn(reqIDs...)).
			All(ctx)
		deptMap := map[int]string{}
		for _, u := range users {
			deptMap[u.ID] = strings.ToLower(strings.TrimSpace(u.Department))
		}
		actorDept := strings.ToLower(strings.TrimSpace(actor.Department))
		filtered := make([]*ent.ServiceRequest, 0, len(requests))
		for _, r := range requests {
			if actorDept != "" && deptMap[r.RequesterID] == actorDept {
				filtered = append(filtered, r)
			}
		}
		requests = filtered
		total = len(filtered)
	}

	resp := make([]dto.ServiceRequestResponse, 0, len(requests))
	for _, r := range requests {
		resp = append(resp, *dto.ToServiceRequestResponse(r))
	}
	return &dto.ServiceRequestListResponse{Requests: resp, Total: total, Page: page, Size: size}, nil
}

// GetServiceRequest 获取服务请求详情
func (s *ServiceRequestService) GetServiceRequest(ctx context.Context, id, tenantID int) (*dto.ServiceRequestResponse, error) {
	request, err := s.client.ServiceRequest.Query().
		Where(servicerequest.ID(id)).
		Where(servicerequest.TenantID(tenantID)).
		First(ctx)

	if err != nil {
		s.logger.Errorf("获取服务请求失败: %v", err)
		return nil, fmt.Errorf("获取服务请求失败: %w", err)
	}

	return dto.ToServiceRequestResponse(request), nil
}

// GetServiceRequestDetail 获取服务请求详情（包含审批记录）
func (s *ServiceRequestService) GetServiceRequestDetail(ctx context.Context, id, tenantID int) (*dto.ServiceRequestResponse, error) {
	request, err := s.client.ServiceRequest.Query().
		Where(servicerequest.ID(id)).
		Where(servicerequest.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取服务请求失败: %w", err)
	}

	approvals, err := s.client.ServiceRequestApproval.Query().
		Where(servicerequestapproval.TenantID(tenantID)).
		Where(servicerequestapproval.ServiceRequestID(id)).
		Order(ent.Asc(servicerequestapproval.FieldLevel)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取审批记录失败: %w", err)
	}

	resp := dto.ToServiceRequestResponse(request)
	for _, a := range approvals {
		resp.Approvals = append(resp.Approvals, dto.ToServiceRequestApprovalResponse(a))
	}
	return resp, nil
}

// ListServiceRequests 获取服务请求列表
func (s *ServiceRequestService) ListServiceRequests(ctx context.Context, req *dto.GetServiceRequestsRequest, tenantID int) (*dto.ServiceRequestListResponse, error) {
	query := s.client.ServiceRequest.Query().Where(servicerequest.TenantID(tenantID))

	// 添加过滤条件
	if req.Status != "" {
		query = query.Where(servicerequest.Status(req.Status))
	}

	// 仅返回用户自己的请求（req.UserID 由 controller 从认证中间件注入）
	if req.UserID > 0 {
		query = query.Where(servicerequest.RequesterID(req.UserID))
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
	err := s.client.ServiceRequest.Update().
		Where(servicerequest.ID(id)).
		Where(servicerequest.TenantID(tenantID)).
		SetStatus(status).
		Exec(ctx)

	if err != nil {
		s.logger.Errorf("更新服务请求状态失败: %v", err)
		return fmt.Errorf("更新服务请求状态失败: %w", err)
	}

	return nil
}

// ApplyApprovalAction 对服务请求执行审批动作（approve/reject）
func (s *ServiceRequestService) ApplyApprovalAction(ctx context.Context, requestID, tenantID, actorUserID int, action, comment string) (*dto.ServiceRequestResponse, error) {
	if action != "approve" && action != "reject" {
		return nil, common.NewBadRequestError("不支持的动作: "+action, nil)
	}

	// V1 改进：拒绝时必须填写审批意见
	if action == "reject" && strings.TrimSpace(comment) == "" {
		return nil, common.NewBadRequestError("拒绝审批时必须填写审批意见", nil)
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, common.NewInternalError("开启事务失败", err)
	}
	defer tx.Rollback()

	reqEnt, err := tx.ServiceRequest.Query().
		Where(servicerequest.ID(requestID)).
		Where(servicerequest.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return nil, common.NewNotFoundError("服务请求不存在")
	}

	level := reqEnt.CurrentLevel
	if level < 1 {
		level = 1
	}
	if level > reqEnt.TotalLevels {
		level = reqEnt.TotalLevels
	}

	approval, err := tx.ServiceRequestApproval.Query().
		Where(servicerequestapproval.TenantID(tenantID)).
		Where(servicerequestapproval.ServiceRequestID(requestID)).
		Where(servicerequestapproval.Level(level)).
		Where(servicerequestapproval.Status(ApprovalStatusPending)).
		First(ctx)
	if err != nil {
		return nil, common.NewConflictError("当前无可审批步骤", "Service request has no pending approval step at current level")
	}

	// 权限：根据步骤/角色/部门做轻量校验
	// checkApproverEligibility 应该返回 common.AppError 类型，所以可以直接返回
	if err := s.checkApproverEligibility(ctx, tx.Client(), approval.Step, reqEnt.RequesterID, actorUserID, tenantID); err != nil {
		return nil, err // checkApproverEligibility is expected to return an AppError
	}

	actor, err := tx.User.Query().
		Where(user.ID(actorUserID)).
		Where(user.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		return nil, common.NewNotFoundError("审批人不存在")
	}

	now := time.Now()
	status := ApprovalStatusApproved
	nextReqStatus := reqEnt.Status
	nextLevel := reqEnt.CurrentLevel

	if action == "reject" {
		status = ApprovalStatusRejected
		nextReqStatus = SRStatusRejected
	} else {
		// approve
		switch approval.Step {
		case ApprovalStepManager: // Using constant
			nextReqStatus = SRStatusManagerApproved
		case ApprovalStepIT: // Using constant
			nextReqStatus = SRStatusITApproved
		case ApprovalStepSecurity: // Using constant
			nextReqStatus = SRStatusSecurityApproved
		}
		nextLevel = reqEnt.CurrentLevel + 1
	}

	// 更新审批记录
	_, err = tx.ServiceRequestApproval.UpdateOneID(approval.ID).
		SetStatus(status).
		SetAction(action).
		SetComment(comment).
		SetApproverID(actor.ID).
		SetApproverName(actor.Name).
		SetProcessedAt(now).
		Save(ctx)
	if err != nil {
		return nil, common.NewDatabaseError("更新审批记录失败", err)
	}

	// 更新服务请求
	upd := tx.ServiceRequest.UpdateOneID(reqEnt.ID).
		Where(servicerequest.TenantID(tenantID)).
		SetStatus(nextReqStatus)
	if action == "approve" {
		upd = upd.SetCurrentLevel(nextLevel)
	}
	if _, err := upd.Save(ctx); err != nil {
		return nil, common.NewDatabaseError("更新服务请求失败", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, common.NewInternalError("提交事务失败", err)
	}
	return s.GetServiceRequestDetail(ctx, requestID, tenantID)
}

// New constants for user roles
const (
	RoleAdmin      = "admin"
	RoleSuperAdmin = "super_admin"
	RoleManager    = "manager"
	RoleAgent      = "agent"
	RoleTechnician = "technician"
	RoleSecurity   = "security"
)

func (s *ServiceRequestService) checkApproverEligibility(ctx context.Context, client *ent.Client, step string, requesterID, actorUserID, tenantID int) error {
	actor, err := client.User.Query().Where(user.ID(actorUserID), user.TenantIDEQ(tenantID)).First(ctx)
	if err != nil {
		return common.NewNotFoundError("审批人不存在")
	}
	// admin/super_admin 放行
	if actor.Role.String() == RoleAdmin || actor.Role.String() == RoleSuperAdmin {
		return nil
	}

	requester, err := client.User.Query().Where(user.ID(requesterID), user.TenantIDEQ(tenantID)).First(ctx)
	if err != nil {
		return common.NewNotFoundError("申请人不存在")
	}

	role := strings.ToLower(strings.TrimSpace(actor.Role.String()))
	dept := strings.ToLower(strings.TrimSpace(actor.Department))
	reqDept := strings.ToLower(strings.TrimSpace(requester.Department))

	switch step {
	case ApprovalStepManager:
		// 主管：manager 且同部门
		if role == RoleManager && reqDept != "" && dept == reqDept {
			return nil
		}
		return common.NewForbiddenError("无权限执行部门主管审批")
	case ApprovalStepIT:
		// IT：agent/technician（或 admin 已放行）
		if role == RoleAgent || role == RoleTechnician {
			return nil
		}
		return common.NewForbiddenError("无权限执行 IT 审批")
	case ApprovalStepSecurity:
		// 安全：专用角色 security（或 admin/super_admin 已放行）
		if role == RoleSecurity {
			return nil
		}
		return common.NewForbiddenError("无权限执行安全审批（需要 security 角色）")
	default:
		return common.NewBadRequestError(fmt.Sprintf("未知审批步骤: %s", step), nil) // Or InternalError if it implies a system error
	}
}
