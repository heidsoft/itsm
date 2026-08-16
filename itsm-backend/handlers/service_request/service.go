package service_request

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/handlers/cmdb"
	"itsm-backend/handlers/service_catalog"
	"itsm-backend/service"

	"go.uber.org/zap"
)

// Constants for approval steps
const (
	ApprovalStepManager  = "manager"
	ApprovalStepIT       = "it"
	ApprovalStepSecurity = "security"

	SRStatusSubmitted        = "submitted"
	SRStatusManagerApproved  = "manager_approved"
	SRStatusITApproved       = "it_approved"
	SRStatusSecurityApproved = "security_approved"
	SRStatusRejected         = "rejected"
	SRStatusProvisioning     = "provisioning"
	SRStatusDelivered        = "delivered"
	SRStatusFailed           = "failed"
	SRStatusCancelled        = "cancelled"

	ApprovalStatusPending  = "pending"
	ApprovalStatusApproved = "approved"
	ApprovalStatusRejected = "rejected"

	// V1 审批时限配置（小时）
	ApprovalTimeoutManager  = 24
	ApprovalTimeoutIT       = 48
	ApprovalTimeoutSecurity = 72

	// Roles
	RoleAdmin      = "admin"
	RoleSuperAdmin = "super_admin"
	RoleManager    = "manager"
	RoleAgent      = "agent"
	RoleTechnician = "technician"
	RoleSecurity   = "security"
)

type Service struct {
	repo           Repository
	scRepo         service_catalog.Repository
	cmdbRepo       cmdb.Repository
	logger         *zap.SugaredLogger
	approvalBridge *service.BPMNApprovalBridge
	// approvalChain 审批链求值引擎：驱动服务请求的分级/会签/或签/fallback 审批。
	// 仅当 BPMN 桥接未处理（无运行中流程实例）时消费，避免双轨推进。
	approvalChain *service.ApprovalChainService
}

func NewService(repo Repository, scRepo service_catalog.Repository, cmdbRepo cmdb.Repository, entClient *ent.Client, logger *zap.SugaredLogger, approvalChain *service.ApprovalChainService) *Service {
	svc := &Service{
		repo:          repo,
		scRepo:        scRepo,
		cmdbRepo:      cmdbRepo,
		logger:        logger,
		approvalChain: approvalChain,
	}
	if entClient != nil {
		// P0-1：服务请求审批桥接到 BPMN 任务，避免流程实例悬挂
		svc.approvalBridge = service.NewBPMNApprovalBridge(entClient, logger)
	}
	return svc
}

// Create submits a new service request
func (s *Service) Create(ctx context.Context, tenantID, requesterID int, catalogID int, reqData *ServiceRequest) (*ServiceRequest, error) {
	if _, _, err := s.repo.GetUserContext(ctx, requesterID, tenantID); err != nil {
		return nil, common.NewBadRequestError("Requester not found or inactive", err)
	}
	// 1. Validate Service Catalog
	cat, err := s.scRepo.Get(ctx, tenantID, catalogID)
	if err != nil {
		return nil, common.NewNotFoundError("Service Catalog not found")
	}
	if cat.CloudServiceID > 0 && cat.CITypeID == 0 {
		return nil, common.NewBadRequestError("关联云服务时必须配置CI类型", nil)
	}
	if cat.Status != "enabled" && cat.Status != "active" {
		return nil, common.NewBadRequestError("Service Catalog is not enabled", nil)
	}

	// 2. Validate Request Data
	if !reqData.ComplianceAck {
		return nil, common.NewBadRequestError("Compliance acknowledgement required", nil)
	}
	if reqData.NeedsPublicIP && len(reqData.SourceIPWhitelist) == 0 {
		return nil, common.NewBadRequestError("Source IP whitelist required for public IP", nil)
	}
	if reqData.ExpireAt == nil {
		return nil, common.NewBadRequestError("Expiration date required", nil)
	}
	if !reqData.ExpireAt.After(time.Now()) {
		return nil, common.NewBadRequestError("Expiration date must be in the future", nil)
	}
	if strings.TrimSpace(reqData.Title) == "" {
		return nil, common.NewBadRequestError("Request title is required", nil)
	}
	switch reqData.DataClassification {
	case "public", "internal", "confidential", "restricted":
	default:
		return nil, common.NewBadRequestError("Invalid data classification", nil)
	}

	// 3. Prepare Service Request
	newReq := &ServiceRequest{
		TenantID:           tenantID,
		CatalogID:          catalogID,
		RequesterID:        requesterID,
		Status:             SRStatusSubmitted,
		CurrentLevel:       1,
		TotalLevels:        3,
		ComplianceAck:      reqData.ComplianceAck,
		NeedsPublicIP:      reqData.NeedsPublicIP,
		DataClassification: reqData.DataClassification,
		Title:              reqData.Title,
		Reason:             reqData.Reason,
		FormData:           reqData.FormData,
		CostCenter:         reqData.CostCenter,
		SourceIPWhitelist:  reqData.SourceIPWhitelist,
		ExpireAt:           reqData.ExpireAt,
	}

	if cat.CITypeID > 0 {
		ciID, err := s.ensureLinkedCI(ctx, tenantID, cat, reqData)
		if err != nil {
			return nil, err
		}
		newReq.CiID = ciID
	}

	// 4. 构建审批步骤：若服务目录关联了审批链，则用审批链求值引擎解析
	//    审批人/会签阈值/fallback（全程租户隔离，堵住跨租户注入与无审批人自审）；
	//    否则回退默认三级（manager/it/security）。状态机/前端/履约门禁仅支持三级，
	//    故审批链超过 3 级时仅取前 3 级（多于 3 级的链式审批为后续独立重构项）。
	now := time.Now()
	totalLevels := 3
	approvals := make([]*ServiceRequestApproval, 0)

	if s.approvalChain != nil {
		if plan, perr := s.resolveServiceRequestChain(ctx, tenantID, requesterID); perr == nil && plan != nil && len(plan.Levels) > 0 {
			levels := plan.Levels
			if len(levels) > 3 {
				levels = levels[:3]
				s.logger.Warnw("服务请求审批链层级超过3，仅取前3级", "catalogID", catalogID, "levels", len(plan.Levels))
			}
			stepLabels := []string{ApprovalStepManager, ApprovalStepIT, ApprovalStepSecurity}
			for i, lv := range levels {
				label := stepLabels[i]
				timeout := []int{ApprovalTimeoutManager, ApprovalTimeoutIT, ApprovalTimeoutSecurity}[i]
				dueAt := now.Add(time.Duration(timeout) * time.Hour)
				approvals = append(approvals, &ServiceRequestApproval{
					TenantID:     tenantID,
					Level:        i + 1,
					Step:         label,
					Status:       ApprovalStatusPending,
					TimeoutHours: timeout,
					DueAt:        &dueAt,
					Node: map[string]interface{}{
						"approver_ids":     intsToIfaces(lv.ApproverIDs),
						"approver_names":   stringsToIfaces(lv.ApproverNames),
						"quorum_type":      lv.ApprovalType,
						"quorum_threshold": lv.Threshold,
						"required":         lv.Required,
						"approved_ids":     []interface{}{},
						"step_label":       label,
					},
				})
			}
			totalLevels = len(levels)
		} else if perr != nil {
			s.logger.Warnw("审批链解析失败，回退默认三级审批", "catalogID", catalogID, "err", perr)
		}
	}

	if len(approvals) == 0 {
		// 默认三级审批（manager/it/security）
		defaultSteps := []struct {
			level        int
			step         string
			timeoutHours int
		}{
			{1, ApprovalStepManager, ApprovalTimeoutManager},
			{2, ApprovalStepIT, ApprovalTimeoutIT},
			{3, ApprovalStepSecurity, ApprovalTimeoutSecurity},
		}
		for _, st := range defaultSteps {
			dueAt := now.Add(time.Duration(st.timeoutHours) * time.Hour)
			approvals = append(approvals, &ServiceRequestApproval{
				TenantID:     tenantID,
				Level:        st.level,
				Step:         st.step,
				Status:       ApprovalStatusPending,
				TimeoutHours: st.timeoutHours,
				DueAt:        &dueAt,
			})
		}
		totalLevels = 3
	}

	newReq.TotalLevels = totalLevels

	// 5. Save
	created, err := s.repo.Create(ctx, newReq, approvals)
	if err != nil {
		s.logger.Errorw("Failed to create service request", "error", err)
		return nil, common.NewInternalError("Failed to create service request", err)
	}

	return created, nil
}

func (s *Service) ensureLinkedCI(ctx context.Context, tenantID int, cat *service_catalog.ServiceCatalog, reqData *ServiceRequest) (int, error) {
	_ = cat
	cloudResourceRefID := parseIntField(reqData.FormData, "cloud_resource_ref_id")
	if cloudResourceRefID > 0 {
		existing, err := s.cmdbRepo.GetCIByCloudResourceRefID(ctx, tenantID, cloudResourceRefID)
		if err == nil && existing != nil {
			return existing.ID, nil
		}
		if err != nil && !ent.IsNotFound(err) {
			return 0, common.NewInternalError("查询关联CI失败", err)
		}
	}
	// 新 CI 必须在审批完成后的 provisioning 阶段由履约器/连接器创建，
	// 提交申请时不能提前向 CMDB 写入 active 资产。
	return 0, nil
}

func parseIntField(formData map[string]interface{}, key string) int {
	if formData == nil {
		return 0
	}
	switch v := formData[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return 0
		}
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
	}
	return 0
}

// Get retrieves a service request with approvals
func (s *Service) Get(ctx context.Context, id, tenantID int) (*ServiceRequest, []*ServiceRequestApproval, error) {
	req, approvals, err := s.repo.GetWithApprovals(ctx, id, tenantID)
	if err != nil {
		return nil, nil, err
	}
	return req, approvals, nil
}

// ApplyApproval processes an approval action
func (s *Service) ApplyApproval(ctx context.Context, id, tenantID, actorID int, action, comment string, userRole, userDept string) (*ServiceRequest, []*ServiceRequestApproval, error) {
	// 1. Validate Inputs
	if action != "approve" && action != "reject" {
		return nil, nil, common.NewBadRequestError("Invalid action: "+action, nil)
	}
	if action == "reject" && strings.TrimSpace(comment) == "" {
		return nil, nil, common.NewBadRequestError("Comment required for rejection", nil)
	}

	// 2. Get Request
	req, approvals, err := s.repo.GetWithApprovals(ctx, id, tenantID)
	if err != nil {
		return nil, nil, err
	}
	actorDept, actorName, err := s.repo.GetUserContext(ctx, actorID, tenantID)
	if err != nil {
		return nil, nil, common.NewForbiddenError("Approver not found or inactive")
	}
	if actorID == req.RequesterID && !isServiceRequestAdmin(userRole) {
		return nil, nil, common.NewForbiddenError("Requesters cannot approve their own service requests")
	}

	// 3. Find Pending Approval for Current Level
	var currentApproval *ServiceRequestApproval
	for _, app := range approvals {
		if app.Level == req.CurrentLevel && app.Status == ApprovalStatusPending {
			currentApproval = app
			break
		}
	}
	if currentApproval == nil {
		return nil, nil, common.NewConflictError("No pending approval found for current level", "")
	}

	// 4. Check Permissions
	requesterDept, _, err := s.repo.GetUserContext(ctx, req.RequesterID, tenantID)
	if err != nil {
		return nil, nil, common.NewConflictError("Requester is no longer available", "")
	}
	if actorDept == "" {
		actorDept = userDept
	}
	if err := s.checkEligibility(actorID, userRole, actorDept, requesterDept, currentApproval); err != nil {
		return nil, nil, err
	}

	// P0-1：审批先桥接完成对应的 BPMN 待办任务（以流程任务为权威审批来源，
	// 仅流程任务指派人才可完成；无关联运行中流程实例时该调用为 no-op）。
	// 桥接仅完成流程任务、做操作人鉴权，不改变服务请求状态；请求状态的层级/
	// quorum 推进仍由下方逻辑统一处理（审批链驱动或遗留三级），避免双轨分叉。
	if s.approvalBridge != nil {
		if _, bridgeErr := s.approvalBridge.CompleteBusinessApprovalTask(
			ctx, tenantID, actorID, string(dto.BusinessTypeServiceRequest), id, action, comment,
		); bridgeErr != nil {
			return nil, nil, common.NewInternalError("同步流程审批任务失败", bridgeErr)
		}
	}

	// 5. Process
	now := time.Now()
	status := ApprovalStatusApproved
	nextReqStatus := req.Status
	nextLevel := req.CurrentLevel

	if action == "reject" {
		status = ApprovalStatusRejected
		nextReqStatus = SRStatusRejected
	} else {
		// 审批链驱动的层级：按 node 中解析出的审批人 + 会签阈值(quorum)判定本级是否通过。
		// 串行(阈值1)行为与遗留一致；会签(阈值=审批人数)需全员批准才推进。
		if nodeIDs, ok := currentApproval.Node["approver_ids"].([]interface{}); ok && len(nodeIDs) > 0 {
			threshold := 1
			if t, ok2 := currentApproval.Node["quorum_threshold"].(float64); ok2 && int(t) > 0 {
				threshold = int(t)
			}
			// 记录本次审批人（去重）
			approved := []interface{}{}
			if existing, ok3 := currentApproval.Node["approved_ids"].([]interface{}); ok3 {
				approved = append(approved, existing...)
			}
			already := false
			for _, a := range approved {
				if aid, ok4 := a.(float64); ok4 && int(aid) == actorID {
					already = true
					break
				}
			}
			if !already {
				approved = append(approved, float64(actorID))
			}
			currentApproval.Node["approved_ids"] = approved

			if len(approved) >= threshold {
				// 本级 quorum 满足
				nextReqStatus = srLevelApprovedStatus(req.CurrentLevel, req.TotalLevels)
				if req.CurrentLevel < req.TotalLevels {
					nextLevel = req.CurrentLevel + 1
				}
			} else {
				// quorum 未满足：本级仍 pending，仅记录审批进度，不推进请求/状态机
				currentApproval.Action = action
				currentApproval.Comment = comment
				currentApproval.ApproverID = &actorID
				currentApproval.ApproverName = actorName
				currentApproval.ProcessedAt = &now
				if err := s.repo.UpdateApproval(ctx, currentApproval); err != nil {
					return nil, nil, common.NewDatabaseError("Failed to update approval", err)
				}
				return s.Get(ctx, id, tenantID)
			}
		} else {
			// 遗留三级：按步骤映射状态
			switch currentApproval.Step {
			case ApprovalStepManager:
				nextReqStatus = SRStatusManagerApproved
			case ApprovalStepIT:
				nextReqStatus = SRStatusITApproved
			case ApprovalStepSecurity:
				nextReqStatus = SRStatusSecurityApproved
			}
			if req.CurrentLevel < req.TotalLevels {
				nextLevel = req.CurrentLevel + 1
			}
		}
	}

	// 6. Update Entities
	currentApproval.Status = status
	currentApproval.Action = action
	currentApproval.Comment = comment
	currentApproval.ApproverID = &actorID
	currentApproval.ApproverName = actorName
	currentApproval.ProcessedAt = &now

	req.Status = nextReqStatus
	if action == "approve" {
		req.CurrentLevel = nextLevel
	}

	if err := s.repo.UpdateRequestAndApproval(ctx, req, currentApproval); err != nil {
		return nil, nil, common.NewDatabaseError("Failed to update request", err)
	}

	return s.Get(ctx, id, tenantID)
}

func (s *Service) checkEligibility(actorID int, actorRole, actorDept, requesterDept string, approval *ServiceRequestApproval) error {
	actorRole = strings.ToLower(actorRole)
	if actorRole == RoleAdmin || actorRole == RoleSuperAdmin {
		return nil
	}

	// 审批链驱动的层级：仅链上解析出的租户内审批人可审（堵住跨租户注入 / 无审批人自审）。
	if approval != nil {
		if ids, ok := approval.Node["approver_ids"].([]interface{}); ok && len(ids) > 0 {
			for _, a := range ids {
				if aid, ok2 := a.(float64); ok2 && int(aid) == actorID {
					return nil
				}
			}
			return common.NewForbiddenError("无权限执行该审批链层级的审批")
		}
	}

	step := ""
	if approval != nil {
		step = approval.Step
	}
	switch step {
	case ApprovalStepManager:
		if actorRole == RoleManager && actorDept != "" && strings.EqualFold(actorDept, requesterDept) {
			return nil
		}
	case ApprovalStepIT:
		if actorRole == RoleAgent || actorRole == RoleTechnician {
			return nil
		}
	case ApprovalStepSecurity:
		if actorRole == RoleSecurity {
			return nil
		}
	}
	return common.NewForbiddenError("Permission denied for this approval step")
}

// srLevelApprovedStatus 返回第 level 级审批通过后的请求状态。
// 末级统一为 security_approved（履约门禁），保证与状态机/前端/履约一致；
// 中间级为 manager_approved / it_approved，与遗留三级语义对齐。
func srLevelApprovedStatus(level, total int) string {
	if level >= total {
		return SRStatusSecurityApproved
	}
	if level == 1 {
		return SRStatusManagerApproved
	}
	return SRStatusITApproved
}

// resolveServiceRequestChain 解析租户内 service_request 类型的激活审批链并求值。
// 返回 nil 表示应回退默认三级（无激活链 / 解析失败 / 链阻塞 / 无层级）。
// 注：当前按实体类型解析（租户级激活链），未绑定到具体服务目录项；
// 若需按目录项维度绑定审批链，需为 ServiceCatalog 增加 approval_chain_id 字段
// （schema 变更 + 迁移），列为后续独立重构项。
func (s *Service) resolveServiceRequestChain(ctx context.Context, tenantID, requesterID int) (*service.ApprovalChainEvaluation, error) {
	evalCtx := service.ApprovalEvalContext{
		TenantID:    tenantID,
		EntityType:  "service_request",
		RequesterID: requesterID,
	}
	plan, err := s.approvalChain.ResolveApprovalPlan(ctx, tenantID, "service_request", evalCtx, nil)
	if err != nil {
		return nil, err
	}
	if plan.Blocked {
		return nil, fmt.Errorf("审批链存在阻塞层级（缺少审批人且策略为阻断）")
	}
	return plan, nil
}

func intsToIfaces(in []int) []interface{} {
	out := make([]interface{}, 0, len(in))
	for _, v := range in {
		out = append(out, float64(v))
	}
	return out
}

func stringsToIfaces(in []string) []interface{} {
	out := make([]interface{}, 0, len(in))
	for _, v := range in {
		out = append(out, v)
	}
	return out
}

// ListPendingApprovals lists pending approvals for current user
func (s *Service) ListPendingApprovals(ctx context.Context, tenantID, userID int, role string, page, size int) ([]*ServiceRequest, int, error) {
	role = strings.ToLower(role)
	var targetLevel int
	var requiredStatus string
	var requesterDept string
	actorDept, _, err := s.repo.GetUserContext(ctx, userID, tenantID)
	if err != nil {
		return nil, 0, common.NewForbiddenError("Approver not found or inactive")
	}

	switch role {
	case ApprovalStepManager:
		targetLevel = 1
		requiredStatus = SRStatusSubmitted
		requesterDept = actorDept
	case ApprovalStepIT, RoleAgent, RoleTechnician:
		targetLevel = 2
		requiredStatus = SRStatusManagerApproved
	case ApprovalStepSecurity:
		targetLevel = 3
		requiredStatus = SRStatusITApproved
	case RoleAdmin, RoleSuperAdmin:
		targetLevel = 0
	default:
		return nil, 0, common.NewForbiddenError("Role is not eligible to approve service requests")
	}

	return s.repo.ListPendingApprovals(ctx, tenantID, targetLevel, requiredStatus, requesterDept, page, size)
}

func (s *Service) List(ctx context.Context, tenantID int, filters ListFilters) ([]*ServiceRequest, int, error) {
	return s.repo.List(ctx, tenantID, filters)
}

// Update updates a service request
func (s *Service) Update(ctx context.Context, id, tenantID, actorID int, actorRole string, reqData *ServiceRequest) (*ServiceRequest, error) {
	// 1. Get existing request
	req, _, err := s.repo.GetWithApprovals(ctx, id, tenantID)
	if err != nil {
		return nil, common.NewNotFoundError("Service Request not found")
	}
	if req.Status != SRStatusSubmitted {
		return nil, common.NewConflictError("Only submitted requests can be edited", req.Status)
	}
	if actorID != req.RequesterID && !isServiceRequestAdmin(actorRole) {
		return nil, common.NewForbiddenError("Only the requester or an administrator can edit this request")
	}

	// 2. Update fields
	if reqData.Title != "" {
		req.Title = reqData.Title
	}
	if reqData.Reason != "" {
		req.Reason = reqData.Reason
	}
	if reqData.FormData != nil {
		req.FormData = reqData.FormData
	}
	if reqData.CostCenter != "" {
		req.CostCenter = reqData.CostCenter
	}
	if reqData.DataClassification != "" {
		req.DataClassification = reqData.DataClassification
	}
	if reqData.NeedsPublicIPSet {
		req.NeedsPublicIP = reqData.NeedsPublicIP
	}
	if reqData.SourceIPWhitelist != nil {
		req.SourceIPWhitelist = reqData.SourceIPWhitelist
	}
	if reqData.ExpireAt != nil {
		req.ExpireAt = reqData.ExpireAt
	}
	if reqData.ComplianceAckSet {
		req.ComplianceAck = reqData.ComplianceAck
	}

	// 3. Save
	if err := s.repo.Update(ctx, req); err != nil {
		s.logger.Errorw("Failed to update service request", "error", err)
		return nil, common.NewInternalError("Failed to update service request", err)
	}

	return req, nil
}

// UpdateStatus updates a service request status with tenant isolation.
func (s *Service) UpdateStatus(ctx context.Context, id, tenantID, actorID int, actorRole, status string) error {
	req, _, err := s.repo.GetWithApprovals(ctx, id, tenantID)
	if err != nil {
		return common.NewNotFoundError("Service Request not found")
	}
	if strings.TrimSpace(status) == "" {
		return common.NewBadRequestError("Status is required", nil)
	}
	if !isValidServiceRequestOperationalTransition(req.Status, status) {
		return common.NewConflictError("Invalid service request status transition", req.Status+" -> "+status)
	}
	if status == SRStatusCancelled {
		if actorID != req.RequesterID && !isServiceRequestAdmin(actorRole) {
			return common.NewForbiddenError("Only the requester or an administrator can cancel this request")
		}
	} else if !isServiceRequestOperator(actorRole) {
		return common.NewForbiddenError("Only service operators can update fulfillment status")
	}
	if err := s.repo.UpdateStatus(ctx, req, status, actorID); err != nil {
		s.logger.Errorw("Failed to update service request status", "error", err, "id", id, "status", status)
		return common.NewInternalError("Failed to update service request status", err)
	}
	return nil
}

// Delete deletes a service request
func (s *Service) Delete(ctx context.Context, id, tenantID, actorID int, actorRole string) error {
	// 1. Get existing request
	req, _, err := s.repo.GetWithApprovals(ctx, id, tenantID)
	if err != nil {
		return common.NewNotFoundError("Service Request not found")
	}
	if actorID != req.RequesterID && !isServiceRequestAdmin(actorRole) {
		return common.NewForbiddenError("Only the requester or an administrator can delete this request")
	}
	if req.Status != SRStatusSubmitted && req.Status != SRStatusRejected && req.Status != SRStatusCancelled {
		return common.NewConflictError("Only submitted, rejected, or cancelled requests can be deleted", req.Status)
	}

	// 2. Delete
	if err := s.repo.Delete(ctx, req); err != nil {
		s.logger.Errorw("Failed to delete service request", "error", err)
		return common.NewInternalError("Failed to delete service request", err)
	}

	return nil
}

func isServiceRequestAdmin(role string) bool {
	role = strings.ToLower(strings.TrimSpace(role))
	return role == RoleAdmin || role == RoleSuperAdmin
}

func isServiceRequestOperator(role string) bool {
	role = strings.ToLower(strings.TrimSpace(role))
	return isServiceRequestAdmin(role) || role == RoleAgent || role == RoleTechnician
}

func isValidServiceRequestOperationalTransition(current, next string) bool {
	if current == next {
		return true
	}
	transitions := map[string]map[string]struct{}{
		SRStatusSubmitted:        {SRStatusCancelled: {}},
		SRStatusManagerApproved:  {SRStatusCancelled: {}},
		SRStatusITApproved:       {SRStatusCancelled: {}},
		SRStatusSecurityApproved: {SRStatusProvisioning: {}, SRStatusCancelled: {}},
		SRStatusProvisioning:     {SRStatusDelivered: {}, SRStatusFailed: {}},
		SRStatusFailed:           {SRStatusProvisioning: {}, SRStatusCancelled: {}},
		SRStatusRejected:         {},
		SRStatusDelivered:        {},
		SRStatusCancelled:        {},
	}
	allowed, ok := transitions[current]
	if !ok {
		return false
	}
	_, ok = allowed[next]
	return ok
}
