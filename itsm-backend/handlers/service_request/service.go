package service_request

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"itsm-backend/common"
	domainrole "itsm-backend/domain/role"
	domainServiceRequest "itsm-backend/domain/servicerequest"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/handlers/cmdb"
	"itsm-backend/handlers/service_catalog"
	"itsm-backend/internal/datascope"
	"itsm-backend/service"
	"itsm-backend/service/sla"

	"go.uber.org/zap"
)

// Constants for approval steps
const (
	ApprovalStepManager  = "manager"
	ApprovalStepIT       = "it"
	ApprovalStepSecurity = "security"

	SRStatusSubmitted        = domainServiceRequest.StatusSubmitted
	SRStatusManagerApproved  = domainServiceRequest.StatusManagerApproved
	SRStatusITApproved       = domainServiceRequest.StatusITApproved
	SRStatusSecurityApproved = domainServiceRequest.StatusSecurityApproved
	SRStatusRejected         = domainServiceRequest.StatusRejected
	SRStatusProvisioning     = domainServiceRequest.StatusProvisioning
	SRStatusDelivered        = domainServiceRequest.StatusDelivered
	SRStatusFailed           = domainServiceRequest.StatusFailed
	SRStatusCancelled        = domainServiceRequest.StatusCancelled

	ApprovalStatusPending   = "pending"
	ApprovalStatusApproved  = "approved"
	ApprovalStatusRejected  = "rejected"
	ApprovalStatusCancelled = "cancelled"

	// V1 审批时限配置（小时）
	ApprovalTimeoutManager  = 24
	ApprovalTimeoutIT       = 48
	ApprovalTimeoutSecurity = 72

	// Roles：单一源在 domain/role。
	// 保留域内别名以最小化调用点改动；新代码一律用 domainrole 常量。
	RoleAdmin      = domainrole.Admin
	RoleSuperAdmin = domainrole.SuperAdmin
	RoleManager    = domainrole.Manager
	RoleAgent      = domainrole.Agent
	RoleTechnician = domainrole.Technician
	RoleITAdmin    = domainrole.ITAdmin
	RoleSecAdmin   = domainrole.SecurityAdmin
)

type workflowCommandCreator interface {
	CreateWithWorkflowCommand(context.Context, *ServiceRequest, []*ServiceRequestApproval) (*ServiceRequest, error)
}

type Service struct {
	repo           Repository
	scRepo         service_catalog.Repository
	cmdbRepo       cmdb.Repository
	client         *ent.Client
	logger         *zap.SugaredLogger
	approvalBridge *service.BPMNApprovalBridge
	// approvalChain 解析审批展示/投影层级；审批写入必须先成功完成 BPMN 待办,
	// 不再允许无运行中实例或无待办时回退到业务侧裁决。
	approvalChain *service.ApprovalChainService
	// slaSvc 为服务请求提供 SLA 绑定与读面（2026-10-21 统一绑定落地）。
	slaSvc                *service.TicketSLAService
	workflowOutboxEnabled bool
	// notifier 供运维批量终止旧版流程实例后提示发起人重建；未注入时静默跳过。
	notifier SystemNotifier
}

func NewService(repo Repository, scRepo service_catalog.Repository, cmdbRepo cmdb.Repository, entClient *ent.Client, logger *zap.SugaredLogger, approvalChain *service.ApprovalChainService) *Service {
	svc := &Service{
		repo:          repo,
		scRepo:        scRepo,
		cmdbRepo:      cmdbRepo,
		client:        entClient,
		logger:        logger,
		approvalChain: approvalChain,
	}
	if entClient != nil {
		// 服务请求审批桥接到 BPMN 任务，避免流程实例悬挂
		svc.approvalBridge = service.NewBPMNApprovalBridge(entClient, logger)
	}
	return svc
}

// EnableWorkflowOutbox opts the service into the atomic outbox path:
// Create will call CreateWithWorkflowCommand so the business row and the
// CommandStartBPMN outbox row share one transaction.
func (s *Service) EnableWorkflowOutbox() { s.workflowOutboxEnabled = true }

var _ service.ProvisioningDeliveryListener = (*Service)(nil)

// SetSLAService 注入 SLA 服务（装配缺失时服务请求降级为无 SLA 绑定，不阻断创建）。
func (s *Service) SetSLAService(slaSvc *service.TicketSLAService) { s.slaSvc = slaSvc }

// bindSLAOnCreate 服务请求创建后绑定 SLA：按 service_type="service_request" 匹配
// SLA 定义并写入 sla_states（与工单/事件/问题同一 Store、同一多态表）。
// 服务请求无 priority 字段——匹配走「不带优先级」的定义（租户为服务请求
// 单独配的定义即可命中）；无匹配定义即不绑定（与问题域同一决策：
// 不套用隐藏默认值，避免存量请求被静默绑上非预期时限）。
func (s *Service) bindSLAOnCreate(ctx context.Context, tenantID, requestID int) {
	if s.slaSvc == nil {
		return
	}
	slaResult, err := s.slaSvc.CalculateSLADeadlineFromRequest(ctx, tenantID, "service_request", "")
	if err != nil {
		s.logger.Warnw("Failed to calculate service request SLA deadline, continuing without SLA binding",
			"tenant_id", tenantID, "request_id", requestID, "error", err)
		return
	}
	if slaResult == nil || slaResult.SLADefinitionID == 0 {
		return
	}
	store := s.slaSvc.SLAStore()
	if store == nil {
		return
	}
	if err := store.SaveDeadlines(ctx, tenantID, "service_request", requestID, slaResult.SLADefinitionID, slaResult.ResponseDeadline, slaResult.ResolutionDeadline); err != nil {
		s.logger.Warnw("Failed to save service request SLA state, continuing", "request_id", requestID, "error", err)
	}
}

// OnProvisioningStarted 交付任务开始执行：SLA 首次响应从这一刻起算。
func (s *Service) OnProvisioningStarted(ctx context.Context, tenantID, serviceRequestID, actorUserID int) {
	s.markSLAFirstResponse(ctx, tenantID, serviceRequestID)
}

// OnProvisioningFinished 交付任务结束：成功时记录解决并推进流程作业节点；
// 失败不推进也不写解决时间，保留待办供修复后重试。
func (s *Service) OnProvisioningFinished(ctx context.Context, tenantID, serviceRequestID, actorUserID int, delivered bool, detail string) {
	if !delivered {
		return
	}
	s.markSLAFirstResponse(ctx, tenantID, serviceRequestID)
	s.markSLAResolved(ctx, tenantID, serviceRequestID)
	s.advanceDeliveryWorkflow(ctx, tenantID, actorUserID, serviceRequestID)
}

func (s *Service) serviceRequestSLAStore() *sla.Store {
	if s.slaSvc == nil {
		return nil
	}
	return s.slaSvc.SLAStore()
}

// markSLAFirstResponse 只补记首次响应：重复审批或重复回调不得把响应时间越改越晚。
func (s *Service) markSLAFirstResponse(ctx context.Context, tenantID, requestID int) {
	store := s.serviceRequestSLAStore()
	if store == nil {
		return
	}
	state, err := store.GetState(ctx, tenantID, "service_request", requestID)
	if err != nil {
		if !ent.IsNotFound(err) {
			s.logger.Warnw("Failed to load service request SLA state", "request_id", requestID, "error", err)
		}
		return
	}
	if !state.FirstResponseAt.IsZero() {
		return
	}
	if err := store.RecordFirstResponse(ctx, tenantID, "service_request", requestID, time.Now()); err != nil {
		s.logger.Warnw("Failed to record service request SLA first response", "request_id", requestID, "error", err)
	}
}

// markSLAResolved 记录解决时间；已 breached 的状态由 Store 保持 breached。
func (s *Service) markSLAResolved(ctx context.Context, tenantID, requestID int) {
	store := s.serviceRequestSLAStore()
	if store == nil {
		return
	}
	state, err := store.GetState(ctx, tenantID, "service_request", requestID)
	if err != nil {
		if !ent.IsNotFound(err) {
			s.logger.Warnw("Failed to load service request SLA state", "request_id", requestID, "error", err)
		}
		return
	}
	if !state.ResolvedAt.IsZero() {
		return
	}
	if err := store.RecordResolution(ctx, tenantID, "service_request", requestID, time.Now()); err != nil {
		s.logger.Warnw("Failed to record service request SLA resolution", "request_id", requestID, "error", err)
	}
}

// advanceDeliveryWorkflow 在交付成功后关闭模板里的作业待办（执行服务/用户确认）。
// 失败只告警：业务状态已 delivered，流程收敛问题不应把交付结果改回失败。
func (s *Service) advanceDeliveryWorkflow(ctx context.Context, tenantID, actorUserID, requestID int) {
	if s.approvalBridge == nil {
		return
	}
	completed, err := s.approvalBridge.CompleteBusinessWorkTasks(
		ctx, tenantID, actorUserID, string(dto.BusinessTypeServiceRequest), requestID,
		map[string]interface{}{"satisfied": true, "deliveryResult": "delivered"}, 5,
	)
	if err != nil {
		s.logger.Warnw("交付完成后推进流程作业任务失败",
			"request_id", requestID, "tenant_id", tenantID, "error", err)
		return
	}
	if completed > 0 {
		s.logger.Infow("交付完成后已推进流程作业任务",
			"request_id", requestID, "tenant_id", tenantID, "completed", completed)
	}
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

	// 服务目录声明免审批时直接进入履约门禁（security_approved），不生成审批行：
	// 默认三级只是「目录要求审批却没配审批链」的兜底，不能反过来给免审批项制造待办。
	if !cat.RequiresApproval {
		approvedAt := now
		newReq.Status = SRStatusSecurityApproved
		newReq.CurrentLevel = 0
		newReq.TotalLevels = 0
		newReq.ApprovedAt = &approvedAt
		totalLevels = 0
	} else if s.approvalChain != nil {
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

	if cat.RequiresApproval && len(approvals) == 0 {
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

	// 4.5 自动分配审批人：按角色解析（manager/it_admin/security_admin），
	// 若同部门找不到则回退到租户内同角色用户，最后回退 super_admin。
	// 防止审批链解析失败时出现「审批人=空」的全员可审/无人可审状态。
	requesterDept, _, _ := s.repo.GetUserContext(ctx, requesterID, tenantID)
	s.enrichApprovalsWithApprovers(ctx, tenantID, approvals, requesterDept, requesterID)

	// 4.6 提交那一刻收敛审批计划：名单剔除发起人、会签阈值钳制到实际可批人数。
	// 会启动真实 BPMN 流程时额外 fail-closed：任一级无人可批就在创建阶段拒绝，
	// 而不是等请求落库、流程启动后用户点审批按钮才发现是死路（只能删掉重提）。
	s.normalizeApprovalPlan(approvals, requesterID)
	if s.workflowOutboxEnabled {
		if err := s.ensureActionableApprovalPlan(approvals); err != nil {
			return nil, err
		}
	}

	// 5. Save（outbox 启用时在同一事务内 enqueue CommandStartBPMN，
	// 保证业务行与流程启动命令的原子性；与 incident/ticket/change 域对齐。）
	var created *ServiceRequest
	if s.workflowOutboxEnabled {
		if creator, ok := s.repo.(workflowCommandCreator); ok {
			created, err = creator.CreateWithWorkflowCommand(ctx, newReq, approvals)
			if err != nil {
				s.logger.Errorw("Failed to create service request with workflow command", "error", err)
				return nil, common.NewInternalError("Failed to create service request", err)
			}
		}
	}
	if created == nil {
		created, err = s.repo.Create(ctx, newReq, approvals)
		if err != nil {
			s.logger.Errorw("Failed to create service request", "error", err)
			return nil, common.NewInternalError("Failed to create service request", err)
		}
	}

	// SLA 绑定在请求落库后进行：查不到定义只是「无 SLA」，不阻断创建
	// （与 ticket/incident/problem 同一口径——能力缺失降级，主流程不失败）。
	s.bindSLAOnCreate(ctx, tenantID, created.ID)
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

// GetForActor 读取服务请求并执行行级可见性检查。
// 不可见时返回 NotFound 而非 Forbidden：同租户内的存在性本身也不该泄漏。
func (s *Service) GetForActor(ctx context.Context, id, tenantID, actorID int, actorRole string) (*ServiceRequest, []*ServiceRequestApproval, error) {
	req, approvals, err := s.Get(ctx, id, tenantID)
	if err != nil {
		return nil, nil, err
	}
	if !canReadServiceRequest(req, approvals, actorID, actorRole) {
		return nil, nil, common.NewNotFoundError("Service Request")
	}
	return req, approvals, nil
}

// canReadServiceRequest 判定行级可见性：管理角色、发起人、经办人、
// 以及审批链上出现过的审批人可见。
func canReadServiceRequest(req *ServiceRequest, approvals []*ServiceRequestApproval, actorID int, actorRole string) bool {
	if datascope.IsDataScopeAllRole(actorRole) {
		return true
	}
	if actorID <= 0 {
		return false
	}
	if actorID == req.RequesterID {
		return true
	}
	if req.ProcessorID != nil && *req.ProcessorID == actorID {
		return true
	}
	for _, app := range approvals {
		if app.ApproverID != nil && *app.ApproverID == actorID {
			return true
		}
		if nodeApproverContains(app.Node, actorID) {
			return true
		}
	}
	return false
}

func nodeApproverContains(node map[string]interface{}, userID int) bool {
	return nodeUserListContains(node, "approver_ids", userID)
}

func nodeUserListContains(node map[string]interface{}, key string, userID int) bool {
	if node == nil {
		return false
	}
	ids, ok := node[key].([]interface{})
	if !ok {
		return false
	}
	for _, raw := range ids {
		switch value := raw.(type) {
		case float64:
			if int(value) == userID {
				return true
			}
		case int:
			if value == userID {
				return true
			}
		case int64:
			if int(value) == userID {
				return true
			}
		case string:
			if parsed, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && parsed == userID {
				return true
			}
		}
	}
	return false
}

// ApplyApproval processes an approval action
func (s *Service) ApplyApproval(ctx context.Context, id, tenantID, actorID int, action, comment string, userRole, userDept string) (*ServiceRequest, []*ServiceRequestApproval, error) {
	// 1. Validate Inputs before any tenant-scoped reads.
	if tenantID <= 0 || actorID <= 0 {
		return nil, nil, common.NewUnauthorizedError("缺少审批身份或租户上下文")
	}
	if id <= 0 {
		return nil, nil, common.NewBadRequestError("Invalid ID", nil)
	}
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
	// 审批只存在于审批阶段：终态或已进入履约的请求即使残留 pending 审批行也不得再投票。
	if req.CurrentLevel <= 0 || req.Status != srApprovalStageStatus(req.CurrentLevel) {
		return nil, nil, common.NewConflictError("当前请求不在审批阶段", req.Status)
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

	// 审批人集合只含发起人时，BPMN 会按「审批任务不得落到发起人」的规则把待办
	// 留空指派；业务层若继续走完成路径，只会拿到引擎的原始错误并被包成 500
	// 「操作失败」——用户不知道是自己不能批还是流程坏了。这里提前给出可行动
	// 结论：既不放宽自审批禁令，也不让单管理员租户在审批按钮上撞 500。
	//
	// 只拦「非空且全是发起人」这一种确定形态：approver_ids 缺失是历史数据，
	// 可能仍有静态指派的待办可完成，不属于这里能判定的场景。
	if onlyRequester := approvalOnlyHasRequester(currentApproval.Node, req.RequesterID); onlyRequester {
		return nil, nil, common.NewBusinessError(common.ConflictCode,
			"该审批节点的审批人集合只包含发起人本人，流程禁止自审批：请为服务目录配置其他审批人，或改由他人提交申请", "")
	}

	// normalize 之后名单为空会打显式标记（= 引擎不会指派任何待办）。
	// 创建阶段已 fail-closed，这里兜住修复前落库的历史请求：点审批时给可行动
	// 结论，而不是把引擎的「无待办」原始错误包成 500「操作失败」。
	if s.workflowOutboxEnabled {
		if unstaffed, _ := currentApproval.Node["approval_unstaffed"].(bool); unstaffed {
			role := approvalStepRole(currentApproval.Step)
			return nil, nil, common.NewBusinessError(common.ConflictCode,
				fmt.Sprintf("第 %d 级审批（%s）没有可审批的人：请为租户内除申请人外的用户分配 role=%s 角色，再由该用户完成审批；也可删除本请求重新提交",
					currentApproval.Level, currentApproval.Step, role), "")
		}
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

	// BPMN 待办是审批写入的前置条件，缺少 bridge/实例/待办均不得回退直批。
	// BPMN 任务完成与业务写入必须在同一事务内，防止"流程已批、业务未批"的不一致。
	if s.approvalBridge == nil || s.client == nil {
		return nil, nil, common.NewBusinessError(common.ConflictCode, "没有可处理的 BPMN 审批任务，请先确认流程绑定与待办状态", "")
	}
	// 待办不存在时不得收集投票：会签要等凑齐 quorum 才完成待办，但前提是流程确实
	// 停在该审批节点上，否则业务侧会记下流程从未裁决的票。
	bound, err := s.approvalBridge.CheckActionableApprovalTask(ctx, tenantID, string(dto.BusinessTypeServiceRequest), id)
	if err != nil {
		return nil, nil, err
	}
	if !bound {
		return nil, nil, common.NewBusinessError(common.ConflictCode, "没有可处理的 BPMN 审批任务，请先确认流程绑定与待办状态", "")
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("开启审批事务失败: %w", err)
	}
	defer tx.Rollback()

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
			// 阈值取不到时兜底取「该层级候选审批人数」而非 1，
			// 避免会签（需全员批准）退化为任意一人即可通过导致 threshold 退化。
			threshold := len(nodeIDs)
			if t, ok2 := currentApproval.Node["quorum_threshold"].(float64); ok2 && int(t) > 0 {
				threshold = int(t)
			}

			// Node 是整块读改写的 JSON，并发投票靠 version CAS 串行化：
			// 写冲突即重读最新 approved_ids 重新合并，两票不再互相覆盖导致 quorum 永不达标。
			quorumMet := false
			for attempt := 0; attempt < 3; attempt++ {
				if fresh, ferr := s.repo.GetApprovalWithClient(ctx, tx.Client(), id, currentApproval.Level, tenantID); ferr == nil && fresh != nil && fresh.Node != nil {
					currentApproval = fresh
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
					quorumMet = true
					break
				}

				// quorum 未满足：本级仍 pending，仅记录审批进度，不推进请求/状态机
				currentApproval.Action = action
				currentApproval.Comment = comment
				currentApproval.ApproverID = &actorID
				currentApproval.ApproverName = actorName
				currentApproval.ProcessedAt = &now
				if err := s.repo.UpdateApprovalWithClient(ctx, tx.Client(), currentApproval); err != nil {
					if errors.Is(err, ErrApprovalWriteConflict) {
						continue
					}
					return nil, nil, common.NewDatabaseError("Failed to update approval", err)
				}
				if err := tx.Commit(); err != nil {
					return nil, nil, fmt.Errorf("提交审批事务失败: %w", err)
				}
				// 会签未凑齐时已记下一票，同样属于首次人工响应。
				s.markSLAFirstResponse(ctx, tenantID, id)
				return s.Get(ctx, id, tenantID)
			}
			if !quorumMet {
				return nil, nil, common.NewBusinessError(common.ConflictCode, "审批并发冲突，请刷新后重试", "")
			}
			// 本级 quorum 满足
			nextReqStatus = srLevelApprovedStatus(req.CurrentLevel, req.TotalLevels)
			if req.CurrentLevel < req.TotalLevels {
				nextLevel = req.CurrentLevel + 1
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
	// 会签只有凑齐 quorum 的那一票才能完成 BPMN 任务：过早完成会让流程推进到下一级，
	// 而本级业务审批仍是 pending，后续投票会落到错误的节点上。
	completed, bridgeErr := s.approvalBridge.CompleteBusinessApprovalTaskWithClient(
		ctx, tx.Client(), tenantID, actorID, string(dto.BusinessTypeServiceRequest), id, action, comment,
	)
	if bridgeErr != nil {
		return nil, nil, fmt.Errorf("同步流程审批任务失败: %w", bridgeErr)
	}
	if !completed {
		return nil, nil, common.NewBusinessError(common.ConflictCode, "没有可处理的 BPMN 审批任务，请先确认流程绑定与待办状态", "")
	}

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

	if err := s.repo.UpdateRequestAndApprovalWithClient(ctx, tx.Client(), req, currentApproval); err != nil {
		// 审批/请求行被并发终审或推进：涉及已完成的 BPMN 桥接动作，不可自动重放，明确报冲突。
		if errors.Is(err, ErrApprovalWriteConflict) || errors.Is(err, ErrRequestWriteConflict) {
			return nil, nil, common.NewBusinessError(common.ConflictCode, "审批已被并发处理，请刷新后重试", "")
		}
		return nil, nil, common.NewDatabaseError("Failed to update request", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("提交审批事务失败: %w", err)
	}

	// 审批动作本身就是对该请求的首次人工响应。
	s.markSLAFirstResponse(ctx, tenantID, id)

	return s.Get(ctx, id, tenantID)
}

// approvalOnlyHasRequester 读出 node.approver_ids，判断这一级是否「只有发起人
// 一个审批人」。这种形态下 BPMN 永远不指派待办（自审批禁令），业务层必须提前
// 给出可行动结论，而不是把引擎错误包成 500。
// 返回 false 的情形包括：名单为空（历史数据，可能有静态指派待办）与名单里
// 还有别人（正常多人审批）。
func approvalOnlyHasRequester(node map[string]interface{}, requesterID int) bool {
	if node == nil {
		return false
	}
	raw, ok := node["approver_ids"].([]interface{})
	if !ok || len(raw) == 0 {
		return false
	}
	for _, item := range raw {
		switch typed := item.(type) {
		case float64:
			if int(typed) != requesterID {
				return false
			}
		case int:
			if typed != requesterID {
				return false
			}
		case int64:
			if int(typed) != requesterID {
				return false
			}
		default:
			// 形态不认识的条目不算「只含发起人」——宁可放过也不误拦。
			return false
		}
	}
	return true
}

func (s *Service) checkEligibility(actorID int, actorRole, actorDept, requesterDept string, approval *ServiceRequestApproval) error {
	actorRole = strings.ToLower(actorRole)
	if actorRole == RoleAdmin || actorRole == RoleSuperAdmin {
		return nil
	}

	// 审批链驱动的层级：仅链上解析出的租户内审批人可审（堵住跨租户注入 / 无审批人自审）。
	// 名单可能是内存态 int（创建时 intsToIfaces 写入）或 JSONB 往返后的 float64，
	// 统一走 nodeApproverIDs 解析——只认其中一种形态会误判合法审批人为 403。
	if approval != nil {
		if ids := nodeApproverIDs(approval.Node); len(ids) > 0 {
			for _, aid := range ids {
				if aid == actorID {
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
	// 词表与 DB 真实角色对齐（domain/role 单一源：users.role 实际值
	// manager/it_admin/security_admin）。approver_ids 包含分支已覆盖正常路径，
	// 此处是审批链未持久化 approver_ids 时的兜底。
	switch step {
	case ApprovalStepManager:
		if actorRole == RoleManager && actorDept != "" && strings.EqualFold(actorDept, requesterDept) {
			return nil
		}
	case ApprovalStepIT:
		// IT 层由 it_admin 主签，agent/technician 作为一线/二线处理人可代签
		if actorRole == RoleITAdmin || actorRole == RoleAgent || actorRole == RoleTechnician {
			return nil
		}
	case ApprovalStepSecurity:
		if actorRole == RoleSecAdmin {
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

// srApprovalStageStatus 返回「第 level 级待审时」请求应处的状态。
// 与 srLevelApprovedStatus 互为逆映射，用于拒绝终态/履约态残留审批行的投票。
func srApprovalStageStatus(level int) string {
	switch level {
	case 1:
		return SRStatusSubmitted
	case 2:
		return SRStatusManagerApproved
	default:
		return SRStatusITApproved
	}
}

// resolveApproversForStep 根据审批步骤角色解析租户内的候选审批人。
// 步骤到角色的映射：manager→manager, it→it_admin, security→security_admin。
// 若指定部门则优先返回同部门用户，其次回退到租户内同角色用户。
// 找不到候选人不返回错误（会记录 warn），由 admin/super_admin 兜底。
//
// requesterID 会被排除在候选之外：流程引擎对审批任务有自审批禁令，会把发起人
// 从候选池里剔除（reason=requester_filtered_from_pool）。业务层若把发起人写进
// 名单，就会出现「名单非空、可批人数为 0」的分裂状态，待办留空指派。这里在
// 源头就与引擎同口径，让 approver_ids 恒等于「真正能批的人」。
func (s *Service) resolveApproversForStep(ctx context.Context, tenantID int, step string, requesterDept string, requesterID int) []int {
	roleByStep := map[string]string{
		ApprovalStepManager:  RoleManager,
		ApprovalStepIT:       RoleITAdmin,
		ApprovalStepSecurity: RoleSecAdmin,
	}
	role, ok := roleByStep[step]
	if !ok {
		return nil
	}
	// 三级回退：① 同部门同角色 → ② 租户内同角色（不限部门）→ ③ super_admin 兜底。
	// 注意第②级必须存在：种子/真实数据中 manager/security 往往不在 requester 同部门，
	// 缺了它审批链会退化为「审批人=requester 自己（admin）」的独审状态。
	users, err := s.repo.FindActiveUsersByRole(ctx, tenantID, role, requesterDept)
	if err != nil || len(users) == 0 {
		if requesterDept != "" {
			users, err = s.repo.FindActiveUsersByRole(ctx, tenantID, role, "")
		}
	}
	if err != nil || len(users) == 0 {
		s.logger.Warnw("未找到匹配审批人，使用 super_admin 兜底",
			"step", step, "role", role, "department", requesterDept, "err", err)
		// Fallback: 任何 super_admin
		admins, adminErr := s.repo.FindActiveUsersByRole(ctx, tenantID, RoleSuperAdmin, "")
		if adminErr == nil {
			users = admins
		}
	}
	return excludeRequesterFromApprovers(users, requesterID)
}

// excludeRequesterFromApprovers 剔除名单里的发起人。
// 剔除后为空是有意义的结果（= 这一级无人可批），不在这里兜底放宽。
func excludeRequesterFromApprovers(ids []int, requesterID int) []int {
	if requesterID <= 0 {
		return ids
	}
	kept := make([]int, 0, len(ids))
	for _, id := range ids {
		if id != requesterID {
			kept = append(kept, id)
		}
	}
	return kept
}

// nodeApproverIDs 从审批 node 读出 approver_ids，兼容内存态 int 与 JSON 往返后的
// float64 两种形态。读不出来返回 nil。
func nodeApproverIDs(node map[string]interface{}) []int {
	if node == nil {
		return nil
	}
	raw, ok := node["approver_ids"].([]interface{})
	if !ok {
		return nil
	}
	ids := make([]int, 0, len(raw))
	for _, item := range raw {
		switch typed := item.(type) {
		case float64:
			if typed > 0 {
				ids = append(ids, int(typed))
			}
		case int:
			if typed > 0 {
				ids = append(ids, typed)
			}
		case int64:
			if typed > 0 {
				ids = append(ids, int(typed))
			}
		}
	}
	return ids
}

// nodeInt 从审批 node 读整数，兼容 int 与 float64。
func nodeInt(node map[string]interface{}, key string) (int, bool) {
	if node == nil {
		return 0, false
	}
	switch typed := node[key].(type) {
	case float64:
		return int(typed), true
	case int:
		return typed, true
	case int64:
		return int(typed), true
	}
	return 0, false
}

// approvalStepRole 返回审批步骤要求的角色码，用于生成可行动的错误文案。
func approvalStepRole(step string) string {
	switch step {
	case ApprovalStepManager:
		return RoleManager
	case ApprovalStepIT:
		return RoleITAdmin
	case ApprovalStepSecurity:
		return RoleSecAdmin
	}
	return RoleSuperAdmin
}

// normalizeApprovalPlan 在创建阶段把审批计划收敛成与流程引擎同口径的形态：
// 剔除发起人、钳制会签阈值，并给「无人可批」的级别打上显式标记。
//
// 不做这一步的后果在真实租户里很硬：单管理员租户提交后，三级审批的名单里只有
// 发起人自己，引擎按自审批禁令把待办留空指派，请求从此卡在审批环节无法推进
// ——用户只能删掉重提。名单要先收敛，后续的 fail-closed 判定才不会误判。
func (s *Service) normalizeApprovalPlan(approvals []*ServiceRequestApproval, requesterID int) {
	for _, app := range approvals {
		if app.Status != ApprovalStatusPending {
			continue
		}
		if app.Node == nil {
			app.Node = make(map[string]interface{})
		}

		// 审批链路径也会把发起人算进候选（链按角色求值，不看谁提交），
		// 这里统一剔除，两条路径共用同一个口径。
		candidates := excludeRequesterFromApprovers(nodeApproverIDs(app.Node), requesterID)
		app.Node["approver_ids"] = intsToIfaces(candidates)

		// 阈值钳制：会签阈值不能超过实际能批的人数。「三人会签」配在只有一个
		// 可批人的租户上，这一级同样永远凑不齐票。宁可降级为「一人通过即可」，
		// 也不能让请求卡死。
		threshold, ok := nodeInt(app.Node, "quorum_threshold")
		if !ok || threshold < 1 {
			threshold = 1
		}
		if len(candidates) == 0 {
			// 无人可批时阈值没有执行意义，归一到 1，避免历史数据里残留
			// 「三人会签」把后续人工补审批人后的重试再次卡死。
			threshold = 1
		} else if threshold > len(candidates) {
			threshold = len(candidates)
		}
		app.Node["quorum_threshold"] = threshold

		// 显式标记，而不是靠「名单为空」推断：历史数据里空名单可能仍有静态
		// 指派的待办可完成，只有 normalize 之后仍为空才是真的无人可批。
		delete(app.Node, "approval_unstaffed")
		if len(candidates) == 0 {
			app.Node["approval_unstaffed"] = true
		}
	}
}

// ensureActionableApprovalPlan 在请求会进入真实 BPMN 流程时做 fail-closed：
// 任一级无人可批就在创建阶段拒绝，并给出可行动文案。
//
// 只挂在 outbox 起流路径上：死路的根因是引擎对审批任务的空指派，而没有
// CommandStartBPMN 的部署/测试形态走的是同步业务审批，不存在这个失败面。
// 把校验放在创建时，用户当场就知道缺什么人，而不是提交后走到审批按钮才发现
// 请求已经注定卡死、只能删掉重提。
func (s *Service) ensureActionableApprovalPlan(approvals []*ServiceRequestApproval) error {
	for _, app := range approvals {
		if app.Status != ApprovalStatusPending {
			continue
		}
		unstaffed, _ := app.Node["approval_unstaffed"].(bool)
		if !unstaffed {
			continue
		}
		role := approvalStepRole(app.Step)
		return common.NewBadRequestError(
			fmt.Sprintf("第 %d 级审批（%s）没有可审批的人：租户内除申请人外没有 role=%s 的在职用户。请为该角色至少分配一名其他用户，或改由他人提交本申请",
				app.Level, app.Step, role), nil)
	}
	return nil
}

// enrichApprovalsWithApprovers 补充审批步骤的 approver_ids，确保 checkEligibility
// 能正确路由到具体审批人。
func (s *Service) enrichApprovalsWithApprovers(ctx context.Context, tenantID int, approvals []*ServiceRequestApproval, requesterDept string, requesterID int) {
	for _, app := range approvals {
		if app.Status != ApprovalStatusPending {
			continue
		}
		// 已有 approver_ids 的不覆盖
		if existing, ok := app.Node["approver_ids"].([]interface{}); ok && len(existing) > 0 {
			continue
		}
		if app.Node == nil {
			app.Node = make(map[string]interface{})
		}
		ids := s.resolveApproversForStep(ctx, tenantID, app.Step, requesterDept, requesterID)
		app.Node["approver_ids"] = intsToIfaces(ids)
		app.Node["step_label"] = app.Step
		// 默认三级审批每级是「任一审批人批准即通过」，必须显式落阈值。
		// 不写的话审批执行侧会按「候选审批人数」兜底成本级全员会签：
		// 发起人也在候选名单里时（单管理员租户常见），流程层禁止发起人审批
		// 待办，这一级就永远凑不齐票，请求卡在「已批 1/2 人」。
		if _, ok := app.Node["quorum_threshold"]; !ok {
			app.Node["quorum_threshold"] = 1
		}
	}
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

	// 影子双轨：新路径（BPMN Service Task）影子执行并对比，差异写日志
	s.approvalChain.ShadowCompare(ctx, tenantID, "service_request", evalCtx, nil, plan, err)

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
	role = strings.ToLower(strings.TrimSpace(role))
	if tenantID <= 0 || userID <= 0 {
		return nil, 0, common.NewUnauthorizedError("缺少审批身份或租户上下文")
	}
	actorDept, _, err := s.repo.GetUserContext(ctx, userID, tenantID)
	if err != nil {
		return nil, 0, common.NewForbiddenError("Approver not found or inactive")
	}

	// 管理员看全租户待办；其他人只看到「自己确实能批」的那些，
	// 口径与 ApplyApproval/checkEligibility 一致（approver_ids 会员制优先）。
	if isServiceRequestAdmin(role) {
		return s.repo.ListPendingApprovals(ctx, tenantID, 0, "", "", page, size)
	}
	switch role {
	case RoleManager, RoleITAdmin, RoleAgent, RoleTechnician, RoleSecAdmin:
	default:
		return nil, 0, common.NewForbiddenError("Role is not eligible to approve service requests")
	}
	return s.repo.ListApproverPendingRequests(ctx, tenantID, userID, role, actorDept, page, size)
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

	// 2. 复用 Create 的业务校验（写路径一致性）：编辑态也须满足同样的合规约束，
	// 否则攻击者可在 submitted 阶段绕过创建时的校验写入不合规数据。
	if reqData.ComplianceAckSet && !reqData.ComplianceAck {
		return nil, common.NewBadRequestError("Compliance acknowledgement required", nil)
	}
	if reqData.DataClassification != "" {
		switch reqData.DataClassification {
		case "public", "internal", "confidential", "restricted":
		default:
			return nil, common.NewBadRequestError("Invalid data classification", nil)
		}
	}
	if reqData.NeedsPublicIPSet && reqData.NeedsPublicIP && len(reqData.SourceIPWhitelist) == 0 {
		return nil, common.NewBadRequestError("Source IP whitelist required for public IP", nil)
	}
	if reqData.ExpireAt != nil && !reqData.ExpireAt.After(time.Now()) {
		return nil, common.NewBadRequestError("Expiration date must be in the future", nil)
	}

	// 3. Update fields
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
		return nil, s.classifyWriteFailure(ctx, req, err, "Failed to update service request")
	}

	return req, nil
}

// classifyWriteFailure 区分条件更新 0 行命中的两种成因。
// Ent 把「版本被并发推进」和「记录已不存在」统一报 NotFoundError；
// 按 id+tenant 复查存在性：仍在 → 版本冲突（409 带双版本号），不在 → 404。
func (s *Service) classifyWriteFailure(ctx context.Context, req *ServiceRequest, opErr error, failMessage string) error {
	if !ent.IsNotFound(opErr) {
		s.logger.Errorw(failMessage, "error", opErr, "id", req.ID)
		return common.NewInternalError(failMessage, opErr)
	}
	current, checkErr := s.repo.Get(ctx, req.ID, req.TenantID)
	if checkErr != nil {
		return common.NewNotFoundError("Service Request")
	}
	return common.NewVersionConflictError("服务请求", req.ID, req.Version, current.Version)
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
		return s.classifyWriteFailure(ctx, req, err, "Failed to update service request status")
	}
	switch status {
	case SRStatusCancelled:
		s.cancelApprovalFollowUps(ctx, tenantID, actorID, id)
	case SRStatusProvisioning:
		s.markSLAFirstResponse(ctx, tenantID, id)
	case SRStatusDelivered:
		// 运营手工置为已交付时走与自动交付相同的 SLA/流程收口。
		s.markSLAFirstResponse(ctx, tenantID, id)
		s.markSLAResolved(ctx, tenantID, id)
		s.advanceDeliveryWorkflow(ctx, tenantID, actorID, id)
	}
	return nil
}

// cancelApprovalFollowUps 在请求已取消后收口审批与流程：
// 待审批步骤置为 cancelled，运行中的流程实例终止，避免留下永久悬挂的待办与计时器。
// 业务取消已经提交，后续失败只告警不回滚——回滚会让用户看到「取消失败」但状态已变。
func (s *Service) cancelApprovalFollowUps(ctx context.Context, tenantID, actorID, requestID int) {
	if affected, err := s.repo.CancelPendingApprovals(ctx, tenantID, requestID); err != nil {
		s.logger.Errorw("取消服务请求后失效待审批记录失败",
			"request_id", requestID, "tenant_id", tenantID, "error", err)
	} else if affected > 0 {
		s.logger.Infow("服务请求已取消，待审批记录已失效",
			"request_id", requestID, "tenant_id", tenantID, "affected", affected)
	}

	if s.approvalBridge == nil {
		return
	}
	if err := s.approvalBridge.TerminateBusinessWorkflow(
		ctx, tenantID, actorID, string(dto.BusinessTypeServiceRequest), requestID, "服务请求已取消",
	); err != nil {
		s.logger.Errorw("取消服务请求后终止流程实例失败",
			"request_id", requestID, "tenant_id", tenantID, "error", err)
	}
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
		return s.classifyWriteFailure(ctx, req, err, "Failed to delete service request")
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
	allowed, ok := serviceRequestStatusTransitions[current]
	if !ok {
		return false
	}
	_, ok = allowed[next]
	return ok
}

// serviceRequestStatusTransitions 是服务请求运营状态机的单一事实源。
// 提为包级变量以便守卫测试枚举「状态机实际承认的值集合」，与
// common.StatusVocabulary 的数据库 CHECK 词表做包含关系断言。
var serviceRequestStatusTransitions = map[string]map[string]struct{}{
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

// GetServiceRequestSLA 读取服务请求的 SLA 状态（2026-10-21 统一绑定落地）。
//
// 契约红线（与 problem 域同一口径）：bound=false 是「没配 SLA」的如实回答，
// 绝不返回伪数据让调用方把「能力没接/没配置」读成「SLA 正常且未超时」。
// 存储态为 active 时按 SLAStatus 实时推导 warning/breached（与 problem 一致）。
func (s *Service) GetServiceRequestSLA(ctx context.Context, id, tenantID, actorID int, actorRole string) (*dto.ServiceRequestSLAResponse, error) {
	req, _, err := s.GetForActor(ctx, id, tenantID, actorID, actorRole)
	if err != nil {
		return nil, err
	}
	info := &dto.ServiceRequestSLAResponse{
		RequestID: req.ID,
		Title:     req.Title,
		Status:    req.Status,
		Bound:     false,
		SLAStatus: "unbound",
	}

	if s.slaSvc == nil {
		return info, nil
	}
	store := s.slaSvc.SLAStore()
	if store == nil {
		return info, nil
	}
	state, err := store.GetState(ctx, tenantID, "service_request", id)
	if err != nil || state == nil {
		// 无 sla_state 行 = 未绑定（租户没配 service_request 类型定义）。
		return info, nil
	}

	info.Bound = true
	info.SLADefinitionID = state.SLADefinitionID
	info.SLAStatus = state.Status
	info.PausedDurationSec = state.PausedDurationSeconds
	if !state.ResponseDeadline.IsZero() {
		t := state.ResponseDeadline
		info.ResponseDeadline = &t
	}
	if !state.ResolutionDeadline.IsZero() {
		t := state.ResolutionDeadline
		info.ResolutionDeadline = &t
	}
	if !state.FirstResponseAt.IsZero() {
		t := state.FirstResponseAt
		info.FirstResponseAt = &t
	}
	if !state.ResolvedAt.IsZero() {
		t := state.ResolvedAt
		info.ResolvedAt = &t
	}

	// 暂停中的计时冻结与已解决都不做实时推导，直接信任存储态。
	if state.Status != "paused" && state.ResolvedAt.IsZero() {
		derived := sla.SLAStatus(time.Now(),
			info.ResponseDeadline, info.ResolutionDeadline,
			state.FirstResponseAt, state.ResolvedAt)
		if derived == "breached" {
			info.SLAStatus = "breached"
		} else if derived == "warning" && state.Status == "active" {
			info.SLAStatus = "warning"
		}
	}
	return info, nil
}
