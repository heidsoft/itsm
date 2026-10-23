package change

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/change"
	"itsm-backend/ent/cirelationship"
	"itsm-backend/ent/configurationitem"
	"itsm-backend/ent/incident"
	"itsm-backend/ent/processapprovaldecision"
	"itsm-backend/internal/datascope"
	"itsm-backend/service"

	"go.uber.org/zap"
)

// 领域级哨兵错误：用于让 HTTP 层把「业务规则拒绝」与「系统内部故障」区分开。
// TransitionStatus 的失败若统一包装成 500，状态机冲突、非审批人越权等预期内的业务拒绝
// 会被误报为服务端故障，污染告警且客户端无法据此重试。调用方请用 errors.Is 判定。
var (
	// ErrChangeNotFound 变更不存在或不属于当前租户（→ 404）。
	ErrChangeNotFound = errors.New("change not found")
	// ErrInvalidTransition 状态机不允许该转换（→ 409 Conflict）。
	ErrInvalidTransition = errors.New("invalid state transition")
	// ErrNotApprover 当前用户不是该变更的待办审批人（→ 403）。
	ErrNotApprover = errors.New("user is not an approver of this change")
	// ErrConcurrentModification 状态已被并发请求抢先修改（→ 409 Conflict）。
	ErrConcurrentModification = errors.New("change was modified concurrently")
	// ErrQuorumRequiresWorkflow 审批链要求会签/阈值，但该变更没有绑定运行中的
	// 流程实例，审批决定无处落库、票数无法累积（→ 409 Conflict，需运维配置）。
	ErrQuorumRequiresWorkflow = errors.New("approval quorum requires a running workflow instance")
	// ErrOutcomeNoteRequired 实施失败/回滚未填写原因（Validation 语义，前端按可行动校验呈现）。
	ErrOutcomeNoteRequired = errors.New("change outcome note is required")
)

type Service struct {
	repo           Repository
	logger         *zap.SugaredLogger
	entClient      *ent.Client
	pirService     *service.ChangePIRService
	approvalBridge *service.BPMNApprovalBridge
	// approvalChain 审批链求值引擎：提交变更时按实体类型 "change" 解析租户级激活链，
	// 以解析出的审批人（租户隔离、含 fallback）覆盖调用方传入的审批人，
	// 堵住「无审批人时自审」与越权自选审批人；链阻塞则失败关闭。
	approvalChain *service.ApprovalChainService
}

type workflowCommandCreator interface {
	CreateWithWorkflowCommand(context.Context, *Change) (*Change, error)
}

type atomicApprovalSubmitter interface {
	SubmitForApprovalWithWorkflow(context.Context, int, int, []ApprovalLevelPlan, string, func(*ent.Client) error) error
}

func NewService(repo Repository, entClient *ent.Client, logger *zap.SugaredLogger, approvalChain *service.ApprovalChainService) *Service {
	svc := &Service{
		repo:          repo,
		entClient:     entClient,
		logger:        logger,
		approvalChain: approvalChain,
	}
	// Initialize PIR service
	svc.pirService = service.NewChangePIRService(entClient, logger)
	if entClient != nil {
		// 变更审批桥接到 BPMN 任务，避免流程实例悬挂
		svc.approvalBridge = service.NewBPMNApprovalBridge(entClient, logger)
	}
	return svc
}

// Change methods
func (s *Service) CreateChange(ctx context.Context, c *Change) (*Change, error) {
	s.logger.Infow("Creating change", "title", c.Title, "tenant_id", c.TenantID)
	if creator, ok := s.repo.(workflowCommandCreator); ok {
		return creator.CreateWithWorkflowCommand(ctx, c)
	}
	return s.repo.Create(ctx, c)
}

func (s *Service) GetChange(ctx context.Context, id int, tenantID int) (*Change, error) {
	return s.repo.Get(ctx, id, tenantID)
}

// ListChanges 列出变更单。推广 ticket 的 DataScope 行级权限：
// 管理角色（super_admin/admin/manager/sysadmin）可见全租户，其余角色仅可见
// 本人创建或分配给自己的变更单。currentUserID/currentRole 由 handler 从
// 鉴权中间件注入的 user_id/role 取得。
func (s *Service) ListChanges(ctx context.Context, tenantID int, page, size int, status, search, riskLevel string, currentUserID int, currentRole string) ([]*Change, int, error) {
	dataScope := datascope.DataScopeAll
	if !datascope.IsDataScopeAllRole(currentRole) {
		dataScope = datascope.DataScopeOwnedOrAssigned
	}
	return s.repo.List(ctx, tenantID, page, size, status, search, riskLevel, dataScope, currentUserID)
}

func (s *Service) UpdateChange(ctx context.Context, c *Change, actorID int, actorRole string) (*Change, error) {
	// Guard governance fields. Changes must be in draft status to freely edit fields that
	// feed review approval / risk snapshot. Reject silent post-submission mutations and force the
	// caller to return to draft for re-approval instead.
	existing, err := s.repo.Get(ctx, c.ID, c.TenantID)
	if err != nil || existing == nil {
		return nil, fmt.Errorf("change not found")
	}
	// 写权限 ⊆ 读权限：普通角色仅可修改本人创建或受理的变更单。
	if !datascope.CanWriteResource(actorID, actorRole, existing.CreatedBy, existing.AssigneeID) {
		return nil, common.NewForbiddenError("无权限修改该变更单：仅创建人、受理人或管理员可操作")
	}
	if existing.Status != "draft" {
		if blocked := governanceFieldDiffs(existing, c); len(blocked) > 0 {
			return nil, fmt.Errorf(
				"当前变更状态为 %q，不允许直接修改治理字段 %v。请先将变更退回 draft 后再修改并重新提审",
				existing.Status, blocked,
			)
		}
	}
	return s.repo.Update(ctx, c)
}

// governanceFieldsAlwaysEditable 列示即使在已提审状态下也允许修改的字段（运营类非治理字段）。
// 其余字段若发生变化，将被视为治理字段修改并被拒绝。
var governanceFieldsAlwaysEditable = map[string]struct{}{
	"Title":            {},
	"Description":      {},
	"PlannedStartDate": {},
	"PlannedEndDate":   {},
	"RelatedTickets":   {},
}

// governanceFieldDiffs 返回 existing 与 requested 之间存在差异且不属于
// governanceFieldsAlwaysEditable 的字段名，用于告知调用方被拒绝修改的治理字段。
func governanceFieldDiffs(existing, requested *Change) []string {
	var blocked []string
	check := func(name string, equal bool) {
		if _, ok := governanceFieldsAlwaysEditable[name]; ok {
			return
		}
		if !equal {
			blocked = append(blocked, name)
		}
	}
	check("Type", existing.Type == requested.Type)
	check("Priority", existing.Priority == requested.Priority)
	check("ImpactScope", existing.ImpactScope == requested.ImpactScope)
	check("RiskLevel", existing.RiskLevel == requested.RiskLevel)
	check("Justification", existing.Justification == requested.Justification)
	check("ImplementationPlan", existing.ImplementationPlan == requested.ImplementationPlan)
	check("RollbackPlan", existing.RollbackPlan == requested.RollbackPlan)
	// AffectedCIs 是风险快照输入，修改会改变审批依据
	existingCIs, requestedCIs := existing.AffectedCIs, requested.AffectedCIs
	ciEqual := len(existingCIs) == len(requestedCIs)
	if ciEqual {
		for i := range existingCIs {
			if existingCIs[i] != requestedCIs[i] {
				ciEqual = false
				break
			}
		}
	}
	check("AffectedCIs", ciEqual)
	// 租户/创建人/ID 等不可变字段不属于用户可编辑字段，这里不关心
	return blocked
}

// DeleteChange 删除变更单，受行级写权限约束（写权限 ⊆ 读权限）。
func (s *Service) DeleteChange(ctx context.Context, id int, tenantID int, actorID int, actorRole string) error {
	existing, err := s.repo.Get(ctx, id, tenantID)
	if err != nil || existing == nil {
		if err != nil && !ent.IsNotFound(err) {
			return err
		}
		return fmt.Errorf("change not found")
	}
	if !datascope.CanWriteResource(actorID, actorRole, existing.CreatedBy, existing.AssigneeID) {
		return common.NewForbiddenError("无权限删除该变更单：仅创建人、受理人或管理员可操作")
	}
	return s.repo.Delete(ctx, id, tenantID)
}

func (s *Service) GetStats(ctx context.Context, tenantID int) (*Stats, error) {
	return s.repo.GetStats(ctx, tenantID)
}

// GetCalendarView 获取日历视图数据
func (s *Service) GetCalendarView(ctx context.Context, tenantID int, startDate, endDate, status string) (*dto.ChangeCalendarResponse, error) {
	changes, err := s.repo.ListByDateRange(ctx, tenantID, startDate, endDate, status)
	if err != nil {
		return nil, err
	}

	items := make([]dto.ChangeCalendarItem, 0, len(changes))
	for _, c := range changes {
		var plannedStart, plannedEnd time.Time
		if c.PlannedStartDate != nil {
			plannedStart = *c.PlannedStartDate
		}
		if c.PlannedEndDate != nil {
			plannedEnd = *c.PlannedEndDate
		}

		changeNumber := c.ChangeNumber
		if changeNumber == "" {
			// 存量数据无编号时回退展示 ID
			changeNumber = fmt.Sprintf("C-%d", c.ID)
		}

		items = append(items, dto.ChangeCalendarItem{
			ID:           c.ID,
			Title:        c.Title,
			ChangeNumber: changeNumber,
			Status:       c.Status,
			RiskLevel:    c.RiskLevel,
			Category:     c.Type,
			PlannedStart: plannedStart,
			PlannedEnd:   plannedEnd,
		})
	}

	return &dto.ChangeCalendarResponse{
		Items: items,
		Total: len(items),
	}, nil
}

// ErrSubmitGateBlocked 变更提交被影响分析门禁拦截（→ 422 Validation 语义）。
// 拦截条件：变更绑定了受影响 CI（存在实施风险）但实施计划/回滚计划缺失。
var ErrSubmitGateBlocked = errors.New("change submit blocked by impact analysis gate")

// validateImpactGate 影响摘要消费门禁（2026-10-21 收口「影响摘要尚未成为提交审批的
// 后端门禁」）：关键 CI 命中或存在关联开放事件（RequiresReview）时，必须显式声明
// high 风险等级——影响摘要此前只进 WorkflowHints 提示文案，风险分析与提交流程断链。
// entClient 为 nil（纯 legacy/单测装配）或查询失败时降级放行
// （best-effort，与 checkWindowConflict 同口径）但留 Warn。
func (s *Service) validateImpactGate(ctx context.Context, changeID, tenantID int, c *Change) error {
	if s.entClient == nil {
		return nil
	}
	summary, sumErr := s.GetCMDBImpactSummary(ctx, changeID, tenantID)
	if sumErr != nil {
		s.logger.Warnw("validateImpactGate: 影响摘要查询失败，降级放行",
			"error", sumErr, "change_id", changeID, "tenant_id", tenantID)
		return nil
	}
	if summary.RequiresReview && c.RiskLevel != "high" {
		return common.NewValidationError(
			fmt.Sprintf("该变更命中高风险特征（关键 CI %d 个、关联未关闭事件 %d 个），风险等级必须显式设为 high 才能提交审批；当前为 '%s'。请先在变更详情中完成风险评估",
				summary.CriticalCICount, summary.OpenIncidentCount, c.RiskLevel),
			fmt.Errorf("impact review required: %w", ErrSubmitGateBlocked),
		)
	}
	return nil
}

// validateSubmitGate 提交前影响分析门禁。
//
// GetCMDBImpactSummary 产出的 RequiresReview / RequiresBackoutPlan 推荐若只用于前端展示
// （WorkflowHints）而不被 SubmitChange 消费，绑定受影响 CI 的变更在实施/回滚计划为空时
// 照样提交成功，风险分析与提交流程断链。
//
// 门禁规则（失败关闭，宁严勿松）：
//   - AffectedCIs 非空 ⇒ RequiresBackoutPlan ⇒ 实施计划与回滚计划都必须非空。
//   - 无受影响 CI 不拦截（纯文档型变更不强制绑定计划）。
//
// 返回 *AppError（Validation/422），由 handler 的 RespondError 语义分流。
func (s *Service) validateSubmitGate(c *Change) error {
	if len(c.AffectedCIs) == 0 {
		return nil
	}
	var missing []string
	if strings.TrimSpace(c.ImplementationPlan) == "" {
		missing = append(missing, "实施计划（implementationPlan）")
	}
	if strings.TrimSpace(c.RollbackPlan) == "" {
		missing = append(missing, "回滚计划（rollbackPlan）")
	}
	if len(missing) == 0 {
		return nil
	}
	return common.NewValidationError(
		fmt.Sprintf("该变更关联了 %d 个受影响 CI，提交审批前必须补齐：%s。请先在变更详情中完善计划，或先解除不需要的 CI 关联",
			len(c.AffectedCIs), strings.Join(missing, "、")),
		ErrSubmitGateBlocked,
	)
}

// checkWindowConflict 排期窗口冲突检查。
//
// 场景：同一租户内两个变更的排期窗口（PlannedStart/PlannedEnd）重叠且
// AffectedCIs 有交集时，存在同 CI 双变更并行实施风险（A 的实施动作可能
// 破坏 B 的回滚前提）。只有日历展示（GetCalendarView）时，冲突全靠人工肉眼核对，没有拦截。
//
// 拦截规则（失败关闭）：
//   - 变更自身必须已排期（两个 Planned 字段齐全），否则不检查。
//   - 时间窗重叠 + CI 交集非空 ⇒ 拒绝提交（422），并列出冲突单据。
//   - 仅 pending/approved/scheduled/in_progress 参与冲突（draft 不拦，
//     终态窗口已释放）；审核中（pending）同样拦截——冲突应在审批前暴露。
//
// 查询失败降级放行（best-effort，不阻塞提交主链路）但留 Warn 日志。
func (s *Service) checkWindowConflict(ctx context.Context, c *Change) error {
	if c.PlannedStartDate == nil || c.PlannedEndDate == nil || len(c.AffectedCIs) == 0 {
		return nil
	}
	if !c.PlannedStartDate.Before(*c.PlannedEndDate) {
		// 非法窗口（start >= end）不在此处理：由数据校验负责，这里跳过
		return nil
	}
	others, err := s.repo.FindOverlappingScheduled(ctx, c.TenantID, c.ID, *c.PlannedStartDate, *c.PlannedEndDate)
	if err != nil {
		// 冲突检查属 best-effort 增强门禁：查询故障不应阻断合法提交，
		// 但必须可观测（静默吞错违反静默失败治理纪律）。
		s.logger.Warnw("checkWindowConflict: 查询重叠排期失败，降级放行",
			"error", err, "change_id", c.ID, "tenant_id", c.TenantID)
		return nil
	}

	conflicts := make([]string, 0, len(others))
	for _, o := range others {
		if hasCIIntersection(c.AffectedCIs, o.AffectedCIs) {
			num := o.ChangeNumber
			if num == "" {
				num = fmt.Sprintf("C-%d", o.ID)
			}
			conflicts = append(conflicts, fmt.Sprintf("%s「%s」(%s~%s)",
				num, o.Title,
				o.PlannedStartDate.Format("01-02 15:04"),
				o.PlannedEndDate.Format("01-02 15:04"),
			))
		}
	}
	if len(conflicts) == 0 {
		return nil
	}
	return common.NewValidationError(
		fmt.Sprintf("排期窗口与以下活跃变更冲突（时间重叠且涉及相同 CI），请调整排期或复核受影响 CI：%s",
			strings.Join(conflicts, "；")),
		fmt.Errorf("window conflict: %w", ErrSubmitGateBlocked),
	)
}

// hasCIIntersection 判定两个 CI 集合是否有交集（字符串精确匹配，
// CI 编号/ID 均以字符串形态存储于 AffectedCIs）。
func hasCIIntersection(a, b []string) bool {
	set := make(map[string]struct{}, len(a))
	for _, v := range a {
		set[v] = struct{}{}
	}
	for _, v := range b {
		if _, ok := set[v]; ok {
			return true
		}
	}
	return false
}

// SubmitChange submits a change for approval
// Transitions status from 'draft' to 'pending' and creates approval records for specified approvers
func (s *Service) SubmitChange(ctx context.Context, changeID, tenantID, submitterID int, req *dto.SubmitChangeRequest) (*Change, error) {
	// 1. Get the change
	c, err := s.repo.Get(ctx, changeID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("change not found")
	}

	// 2. Check if change is in draft status
	if c.Status != "draft" {
		return nil, fmt.Errorf("change must be in draft status to submit")
	}

	// 2.5 影响分析门禁：绑定受影响 CI 的变更必须先补齐实施/回滚计划。
	if gateErr := s.validateSubmitGate(c); gateErr != nil {
		return nil, gateErr
	}

	// 2.55 影响摘要消费门禁（2026-10-21 收口「影响摘要尚未成为提交审批的后端门禁」）。
	if gateErr := s.validateImpactGate(ctx, changeID, tenantID, c); gateErr != nil {
		return nil, gateErr
	}

	// 2.6 排期窗口冲突检查：同 CI 时间重叠的活跃变更拦截。
	if gateErr := s.checkWindowConflict(ctx, c); gateErr != nil {
		return nil, gateErr
	}

	// 3. 解析审批计划（引擎层级 + quorum）。
	//     命中激活 "change" 链时以引擎解析结果（保留每层 approval_type/threshold）覆盖
	//     调用方传入审批人，杜绝「无审批人自审」与越权自选审批人；链阻塞（必需层无审批人
	//     且策略为阻断）→ 失败关闭；无激活链 → 回退旧逻辑（调用方传入 / 默认创建人）。
	var plan []ApprovalLevelPlan
	if s.approvalChain != nil {
		chainPlan, cerr := s.resolveChangeChainPlan(ctx, tenantID, c.ID, c.CreatedBy, c.Priority)
		if cerr != nil {
			return nil, fmt.Errorf("变更审批链解析失败，无法提交：%w", cerr)
		}
		plan = chainPlan
	}

	if len(plan) == 0 {
		// 旧逻辑：调用方传入审批人，或缺省创建人；租户校验后构建单级 serial 计划
		// （阈值 1 = 全员顺序必需，等价旧 AND 行为，字节级不变）。
		ids := req.ApproverIDs
		if len(ids) == 0 {
			ids = []int{c.CreatedBy}
			s.logger.Infow("No approvers specified, defaulting to change creator", "change_id", changeID, "creator_id", c.CreatedBy)
		}
		for _, approverID := range ids {
			valid, verr := s.repo.ValidateApproverBelongsToTenant(ctx, approverID, tenantID)
			if verr != nil {
				s.logger.Warnw("Failed to validate approver", "error", verr, "approver_id", approverID)
				return nil, fmt.Errorf("验证审批人失败")
			}
			if !valid {
				s.logger.Warnw("Approver does not belong to tenant", "approver_id", approverID, "tenant_id", tenantID)
				return nil, fmt.Errorf("审批人 %d 不属于当前租户", approverID)
			}
		}
		plan = []ApprovalLevelPlan{{Level: 1, ApprovalType: "serial", Threshold: 1, Required: true, ApproverIDs: ids}}
	}

	// 4. 在同一数据库事务中推进 BPMN 并提交审批状态/审批链/outbox。
	// BPMN 失败时 draft 保持不变；后续任一业务写入失败也会回滚 BPMN 状态。
	var submitErr error
	if submitter, ok := s.repo.(atomicApprovalSubmitter); ok && s.approvalBridge != nil {
		submitErr = submitter.SubmitForApprovalWithWorkflow(ctx, changeID, tenantID, plan, req.Comment, func(txc *ent.Client) error {
			_, err := s.approvalBridge.AdvanceBusinessWorkflowWithClient(
				ctx, txc, tenantID, submitterID, string(dto.BusinessTypeChange), changeID, nil, 1,
			)
			return err
		})
	} else {
		submitErr = s.repo.SubmitForApproval(ctx, changeID, tenantID, plan, req.Comment)
	}
	if submitErr != nil {
		s.logger.Warnw("Failed to atomically submit change", "error", submitErr, "change_id", changeID)
		return nil, fmt.Errorf("提交变更审批失败: %w", submitErr)
	}

	// 审批记录、审批链和 notification.deliver 命令已由仓储在同一事务提交。
	s.logger.Infow("Change submitted for approval", "change_id", changeID, "submitter_id", submitterID, "levels", len(plan))

	c.Status = "pending"
	return c, nil
}

// resolveChangeChainPlan 解析租户内 "change" 类型的激活审批链，按引擎层级结构
// 输出审批计划（含每层的 approval_type / threshold / required / approver_ids）。
// 返回 (nil, nil) 表示无激活链（调用方应回退旧逻辑）；
// 返回 (nil, err) 表示链存在但被阻塞（必需层无审批人且策略为阻断），调用方应失败关闭。
func (s *Service) resolveChangeChainPlan(ctx context.Context, tenantID, changeID, submitterID int, priority string) ([]ApprovalLevelPlan, error) {
	evalCtx := service.ApprovalEvalContext{
		TenantID:    tenantID,
		EntityType:  "change",
		RequesterID: submitterID,
		Priority:    priority,
	}
	plan, err := s.approvalChain.ResolveApprovalPlan(ctx, tenantID, "change", evalCtx, nil)

	// 影子双轨：新路径（BPMN Service Task）影子执行并对比，差异写日志
	s.approvalChain.ShadowCompare(ctx, tenantID, "change", evalCtx, nil, plan, err)

	if err != nil {
		// 未找到激活链或解析异常 → 降级为旧逻辑（不阻断提交可用性，仅记录）
		s.logger.Warnw("变更审批链解析降级为旧逻辑", "error", err, "change_id", changeID, "tenant_id", tenantID)
		return nil, nil
	}
	if plan.Blocked {
		return nil, fmt.Errorf("审批链存在阻塞层级（缺少审批人且策略为阻断）")
	}
	out := make([]ApprovalLevelPlan, 0, len(plan.Levels))
	for _, lv := range plan.Levels {
		if len(lv.ApproverIDs) == 0 {
			// 非必需层且无审批人：引擎已视其 satisfied，提交阶段无审批人可插，跳过。
			continue
		}
		at := lv.ApprovalType
		if at == "" {
			at = "serial"
		}
		thr := lv.Threshold
		if thr <= 0 {
			// serial/or 默认 1；parallel/all 在引擎已置为候选人数。
			if at == "serial" || at == "or" {
				thr = 1
			} else {
				thr = len(lv.ApproverIDs)
			}
		}
		out = append(out, ApprovalLevelPlan{
			Level:        lv.Level,
			ApprovalType: at,
			Threshold:    thr,
			Required:     lv.Required,
			ApproverIDs:  lv.ApproverIDs,
		})
	}
	return out, nil
}

// Approval methods
//
// 审批记录写入与变更状态推进（draft → pending）合并到同一 sql.Tx，
// 由 repository.SubmitApprovalRecordTx 通过条件 UPDATE 统一保证原子性与幂等性。
// 旧逻辑先 CreateApprovalRecord 再用非 CAS 的 repo.Update 改 status，
// 存在"审批记录已写入但状态推进失败"或"状态推进成功但没有审批记录"的部分失败窗口。
func (s *Service) SubmitApproval(ctx context.Context, record *ApprovalRecord, tenantID int) (*ApprovalRecord, error) {
	// Custom business logic: when submitting, we check if change exists
	c, err := s.repo.Get(ctx, record.ChangeID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("change not found")
	}

	// 审批人来源只能取自本变更的审批链（change_approval_chains），
	// 禁止请求体任意指定。否则持 change:write 的用户可注入以自己为审批人的
	// pending 记录再调用 /transition 直接 approved，绕过评审 / quorum / 标准
	// 变更门禁。
	// 1) 审批人必须属于本租户。
	belongs, err := s.repo.ValidateApproverBelongsToTenant(ctx, record.ApproverID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("校验审批人失败: %w", err)
	}
	if !belongs {
		return nil, fmt.Errorf("审批人 %d 不属于租户 %d，无权作为该变更审批人", record.ApproverID, tenantID)
	}
	// 2) 审批人必须是该变更审批链中指定的审批人。
	chain, err := s.repo.GetApprovalChain(ctx, record.ChangeID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("获取审批链失败: %w", err)
	}
	designated := make(map[int]struct{}, len(chain))
	for _, node := range chain {
		designated[node.ApproverID] = struct{}{}
	}
	if _, ok := designated[record.ApproverID]; !ok {
		return nil, fmt.Errorf("用户 %d 不是变更 %d 的指定审批人，无法提交审批", record.ApproverID, record.ChangeID)
	}

	// 3) 在同一 *sql.Tx 中"写审批记录 + CAS 推进 draft→pending"。
	// 仓储负责事务边界与条件 UPDATE；service 不再调用非 CAS 的 Update(c)。
	// 当变更已处于 pending/approved 等状态时仍允许追加审批记录（不同审批人
	// 分轮次提交），返回 promoted=false 表示本轮没有推进状态。
	// rec 已在本事务内 RETURNING 出 ID/CreatedAt，service 直接复用作为响应，
	// 不再二次 CreateApprovalRecord 造成重复写入。
	now := time.Now()
	comment := ""
	if record.Comment != nil {
		comment = *record.Comment
	}
	rec, promoted, err := s.repo.SubmitApprovalRecordTx(ctx, record.ChangeID, tenantID, record.ApproverID, comment, now)
	if err != nil {
		s.logger.Errorw("SubmitApproval: atomic submit failed", "error", err, "change_id", c.ID, "approver_id", record.ApproverID)
		return nil, fmt.Errorf("提交审批失败: %w", err)
	}
	s.logger.Infow("SubmitApproval: atomic submit succeeded",
		"change_id", c.ID, "approver_id", record.ApproverID, "tenant_id", tenantID, "promoted", promoted)
	return rec, nil
}

// 审批决定的写路径唯一化为 TransitionStatus + BPMN 桥接。
// 此前还有一个无路由的 ProcessApproval（legacy）：它先改 change_approvals、
// 再单独改 changes.status、提交后再收口审批链，三段各自成事务且失败只记日志，
// 会留下"审批记录已落定、变更状态未推进"的半写状态；同时绕过桥接，与流程
// 权威状态分叉。2026-10-22 收敛删除，quorum 求值统一走 evaluateApprovalQuorum。
//
// 2026-10-23 进一步删除其配套复核器 checkAndTransitionChange：它虽无生产路由，
// 但自带「读状态 → 校验 → 独立 CAS」的第二条状态写路径（无事务、无 BPMN 桥接、
// 不收口审批链），只差一个调用方就会复活并复现上述半写状态。会签票的累积与
// 求值唯一入口是 TransitionStatus：最后一票经桥接落审计行后，在同一事务内按
// evaluateApprovalQuorum 求值并推进状态（见本文件终态/非终态分支）。

func (s *Service) ConfigureWorkflow(ctx context.Context, changeID, tenantID int, items []*ApprovalChain) error {
	// Clear existing and set new
	if _, err := s.repo.Get(ctx, changeID, tenantID); err != nil {
		return fmt.Errorf("change not found")
	}
	for _, item := range items {
		item.ChangeID = changeID
		item.TenantID = tenantID
	}
	if err := s.repo.ReplaceApprovalChain(ctx, changeID, tenantID, items); err != nil {
		s.logger.Errorw("ConfigureWorkflow: failed to replace approval chain", "error", err, "change_id", changeID)
		return fmt.Errorf("failed to replace approval chain: %w", err)
	}
	return nil
}

func (s *Service) GetApprovalSummary(ctx context.Context, changeID, tenantID int) (interface{}, error) {
	chain, err := s.repo.GetApprovalChain(ctx, changeID, tenantID)
	if err != nil {
		s.logger.Warnw("GetApprovalSummary: failed to get approval chain", "error", err, "change_id", changeID)
		return nil, fmt.Errorf("failed to get approval chain: %w", err)
	}
	history, err := s.repo.GetApprovalHistory(ctx, changeID, tenantID)
	if err != nil {
		s.logger.Warnw("GetApprovalSummary: failed to get approval history", "error", err, "change_id", changeID)
		return nil, fmt.Errorf("failed to get approval history: %w", err)
	}

	return map[string]interface{}{
		"chain":   chain,
		"history": history,
	}, nil
}

// Risk Assessment
func (s *Service) AssessRisk(ctx context.Context, ra *RiskAssessment) (*RiskAssessment, error) {
	return s.repo.CreateRiskAssessment(ctx, ra)
}

func (s *Service) GetRisk(ctx context.Context, changeID int, tenantID int) (*RiskAssessment, error) {
	return s.repo.GetRiskAssessment(ctx, changeID, tenantID)
}

func (s *Service) UpdateRisk(ctx context.Context, ra *RiskAssessment) (*RiskAssessment, error) {
	changeEntity, err := s.repo.Get(ctx, ra.ChangeID, ra.TenantID)
	if err != nil || changeEntity == nil {
		return nil, fmt.Errorf("change not found")
	}
	existing, err := s.repo.GetRiskAssessment(ctx, ra.ChangeID, ra.TenantID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return s.repo.CreateRiskAssessment(ctx, ra)
	}
	return s.repo.UpdateRiskAssessment(ctx, ra)
}

func (s *Service) GetCMDBImpactSummary(ctx context.Context, changeID, tenantID int) (*dto.ChangeCMDBImpactSummary, error) {
	if s.entClient == nil {
		return nil, fmt.Errorf("CMDB impact summary unavailable")
	}

	changeEntity, err := s.repo.Get(ctx, changeID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("change not found")
	}

	ciIDs := make([]int, 0, len(changeEntity.AffectedCIs))
	for _, raw := range changeEntity.AffectedCIs {
		id, err := strconv.Atoi(raw)
		if err != nil || id <= 0 {
			continue
		}
		ciIDs = append(ciIDs, id)
	}

	summary := &dto.ChangeCMDBImpactSummary{
		ChangeID:               changeID,
		AffectedCIs:            ciIDs,
		WorkflowHints:          []string{},
		ITILPractices:          []string{"service_configuration_management", "change_enablement"},
		RecommendedRiskLevel:   "low",
		RecommendedImpactScope: "low",
	}

	if len(ciIDs) == 0 {
		summary.WorkflowHints = append(summary.WorkflowHints, "当前变更未绑定 CI，建议在提交流程前关联受影响配置项。")
		return summary, nil
	}

	cis, err := s.entClient.ConfigurationItem.Query().
		Where(
			configurationitem.TenantID(tenantID),
			configurationitem.IDIn(ciIDs...),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询受影响CI失败: %w", err)
	}

	summary.TotalAffectedCIs = len(cis)
	for _, ci := range cis {
		if ci.Criticality == "high" || ci.Criticality == "critical" {
			summary.CriticalCICount++
		}
	}

	relCount, err := s.entClient.CIRelationship.Query().
		Where(
			cirelationship.TenantID(tenantID),
			cirelationship.IsActive(true),
			cirelationship.Or(
				cirelationship.SourceCiIDIn(ciIDs...),
				cirelationship.TargetCiIDIn(ciIDs...),
			),
			cirelationship.Or(
				cirelationship.StrengthEQ(cirelationship.StrengthHigh),
				cirelationship.StrengthEQ(cirelationship.StrengthCritical),
				cirelationship.ImpactLevelEQ(cirelationship.ImpactLevelHigh),
				cirelationship.ImpactLevelEQ(cirelationship.ImpactLevelCritical),
			),
		).
		Count(ctx)
	if err == nil {
		summary.HighRiskDependencyCount = relCount
	}

	openIncidentCount, err := s.entClient.Incident.Query().
		Where(
			incident.TenantID(tenantID),
			incident.ConfigurationItemIDIn(ciIDs...),
			incident.StatusNotIn("resolved", "closed"),
		).
		Count(ctx)
	if err == nil {
		summary.OpenIncidentCount = openIncidentCount
	}

	summary.RecommendedRiskLevel = recommendRiskLevel(
		summary.TotalAffectedCIs,
		summary.CriticalCICount,
		summary.HighRiskDependencyCount,
		summary.OpenIncidentCount,
		changeEntity.Type,
	)
	summary.RecommendedImpactScope = recommendImpactScope(
		summary.TotalAffectedCIs,
		summary.CriticalCICount,
		summary.HighRiskDependencyCount,
	)
	summary.RequiresReview = summary.RecommendedRiskLevel == "high" || changeEntity.Type == "emergency" || summary.CriticalCICount > 0
	summary.RequiresBackoutPlan = summary.TotalAffectedCIs > 0
	summary.WorkflowHints = buildWorkflowHints(summary, changeEntity.Type)
	summary.ITILPractices = append(summary.ITILPractices, inferITILPractices(summary)...)

	return summary, nil
}

func recommendRiskLevel(totalCIs, criticalCIs, highRiskDependencies, openIncidents int, changeType string) string {
	switch {
	case changeType == "emergency":
		return "high"
	case criticalCIs > 0:
		return "high"
	case highRiskDependencies >= 4:
		return "high"
	case openIncidents >= 2:
		return "high"
	case totalCIs >= 5 || highRiskDependencies > 0 || openIncidents > 0:
		return "medium"
	default:
		return "low"
	}
}

func recommendImpactScope(totalCIs, criticalCIs, highRiskDependencies int) string {
	switch {
	case criticalCIs > 0 || totalCIs >= 5 || highRiskDependencies >= 3:
		return "high"
	case totalCIs >= 2 || highRiskDependencies > 0:
		return "medium"
	default:
		return "low"
	}
}

func buildWorkflowHints(summary *dto.ChangeCMDBImpactSummary, changeType string) []string {
	hints := make([]string, 0, 6)
	if summary.TotalAffectedCIs == 0 {
		hints = append(hints, "补充受影响 CI 后再发起审批，以便自动执行风险分流。")
	}
	if summary.CriticalCICount > 0 {
		hints = append(hints, "命中关键 CI，建议走评审组审批并校验变更窗口。")
	}
	if summary.OpenIncidentCount > 0 {
		hints = append(hints, "受影响 CI 当前存在未关闭事件，建议先做冲突检查和实施前健康确认。")
	}
	if summary.HighRiskDependencyCount > 0 {
		hints = append(hints, "存在高风险依赖，建议在工作流中增加影响确认和回滚演练节点。")
	}
	if changeType == "emergency" {
		hints = append(hints, "紧急变更建议启用快速审批路径，并在实施后自动创建 PIR 任务。")
	}
	if summary.RequiresBackoutPlan {
		hints = append(hints, "建议在提交流程前强制校验回滚计划与实施计划完整性。")
	}
	return hints
}

func inferITILPractices(summary *dto.ChangeCMDBImpactSummary) []string {
	practices := []string{}
	if summary.OpenIncidentCount > 0 {
		practices = append(practices, "incident_management")
	}
	if summary.HighRiskDependencyCount > 0 {
		practices = append(practices, "risk_management")
	}
	if summary.RequiresReview {
		practices = append(practices, "change_enablement")
	}
	if summary.CriticalCICount > 0 {
		practices = append(practices, "monitoring_and_event_management")
	}
	return practices
}

// TransitionStatus 迁移变更状态；approve/reject 需要操作人是对应审批人。
// 非审批分支（schedule/start/complete/close/cancel）只有 change:write RBAC，
// 因此同样补行级 owner 校验（写权限 ⊆ 读权限）；rollback 除外，回滚由独立的
// change:rollback 授权约束。
func (s *Service) TransitionStatus(ctx context.Context, id, tenantID, userID int, targetStatus, comment, actorRole string) (*Change, error) {
	c, err := s.repo.Get(ctx, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("%w: id=%d", ErrChangeNotFound, id)
	}

	if targetStatus != "approved" && targetStatus != "rejected" && targetStatus != "rolled_back" {
		if !datascope.CanWriteResource(userID, actorRole, c.CreatedBy, c.AssigneeID) {
			return nil, common.NewForbiddenError("无权限操作该变更单：仅创建人、受理人或管理员可执行状态流转")
		}
	}

	// Validate state transition (使用 service 包的 canonical 状态机，保证与 legacy service 一致)
	if !service.IsValidChangeStatusTransition(c.Status, targetStatus, c.Type) {
		return nil, fmt.Errorf("%w: 从 '%s' 到 '%s'", ErrInvalidTransition, c.Status, targetStatus)
	}

	// 回滚门禁：清单存在则必须逐项确认（2026-10-22，链路 C「失败可回退」）。
	//
	// 此前 rolled_back 只需一次状态翻转：回滚计划写了几步、走到哪一步、
	// 每步成没成，系统一概不知；审计只能看到「从 in_progress 变成 rolled_back」，
	// PIR 里「影响是否已复原」只能靠人凭记忆填。有了行级清单之后，
	// 「已回滚」才第一次意味着「清单上的每一步都被某个具体的人确认过」。
	// 清单为空时放行（应急回滚不能被"你还没编排清单"挡住）。
	if targetStatus == "rolled_back" {
		if gateErr := s.validateRollbackChecklist(ctx, id, tenantID); gateErr != nil {
			return nil, gateErr
		}
	}

	// 实施结果说明（outcome_note）。
	//
	// failed / rolled_back 必须写原因：这两个状态都表示"没按计划做成"，没有原因的
	// 记录在事后审计里无法区分"技术方案不成立"、"实施步骤走错"还是"窗口不够"，
	// 也就无法反哺风险评估；回滚还要据此判断影响范围是否已复原。
	// completed 的原因是可选的完成说明（有就存，便于复盘）。
	// approve/reject 不走这里：它们的理由由审批桥接写进 ProcessApprovalDecision。
	outcomeNote := strings.TrimSpace(comment)
	if outcomeNote == "" && (targetStatus == "failed" || targetStatus == "rolled_back") {
		return nil, common.NewValidationError(
			fmt.Sprintf("把变更标记为 '%s' 必须填写原因（请求体 reason 或 comment）："+
				"该原因会写入实施结果说明并进入 PIR 与审计，用于判断影响是否已复原、是否需要重新排期", targetStatus),
			ErrOutcomeNoteRequired,
		)
	}

	// scheduled 转换复查排期窗口冲突（2026-10-21 收口）：排期时间可在提交审批后
	// 通过 Update 修改，提交时（SubmitChange）的冲突检查拦不住「批后排期改动」，
	// 所以进入 scheduled 时用实体当前窗口再查一次。查询失败降级放行（同口径）。
	if targetStatus == "scheduled" {
		if gateErr := s.checkWindowConflict(ctx, c); gateErr != nil {
			return nil, gateErr
		}
	}

	// 桥接与业务状态写入收敛到同一事务：流程推进不可逆，若桥接先独立提交而
	// 后续业务 CAS 失败（并发冲突/DB 错误），会留下「流程已批、业务未批」的
	// 永久分叉。事务内任一步失败，defer Rollback 连同桥接写入一起撤销。
	// entClient 为 nil 属纯 legacy/单测装配（与本文件 GetBPMNApprovalDecisions
	// 防御点一致）：无桥接也无 ent 事务，状态 CAS 退回 repo 承担。
	//
	// 事务句柄优先取仓储的 WorkflowTx（原生 *sql.Tx + 绑定 ent client）：
	// 只有它能在同一事务里同时做 ent 写（桥接/状态）与原生 SQL 写
	// （change_approval_chains 无 ent schema），终态收口才具备原子性。
	// 仓储没有原生句柄时（mock/legacy 装配）退化为纯 ent 事务。
	var tx *WorkflowTx
	if s.entClient != nil {
		if starter, ok := s.repo.(workflowTxStarter); ok {
			var txErr error
			tx, txErr = starter.BeginWorkflowTx(ctx)
			if txErr != nil {
				return nil, fmt.Errorf("开启事务失败: %w", txErr)
			}
		} else {
			entTx, txErr := s.entClient.Tx(ctx)
			if txErr != nil {
				return nil, fmt.Errorf("开启事务失败: %w", txErr)
			}
			tx = wrapEntTx(entTx)
		}
		defer tx.Rollback()
	}

	// approvalDecidedByBridge 标记本次审批是否被 BPMN 桥接真正消费（写成
	// ProcessApprovalDecision 审计事实）。会签门禁只在这个前提下求值：
	// 未绑定流程时审批决定不落库，任何"求值"都看不到这一票，强行按链判定
	// 会把单级审批永久卡在 pending（见 api-reference「审批与状态流转」）。
	approvalDecidedByBridge := false

	// For approval actions, verify user is the approver
	if targetStatus == "approved" || targetStatus == "rejected" {
		chain, err := s.repo.GetApprovalChain(ctx, id, tenantID)
		if err != nil {
			s.logger.Errorw("TransitionStatus: failed to get approval chain", "error", err, "change_id", id, "tenant_id", tenantID)
			return nil, fmt.Errorf("failed to get approval chain")
		}
		decisions, err := s.GetBPMNApprovalDecisions(ctx, id, tenantID)
		if err != nil {
			s.logger.Errorw("TransitionStatus: failed to get approval decisions", "error", err, "change_id", id, "tenant_id", tenantID)
			return nil, fmt.Errorf("failed to get approval decisions")
		}
		decidedActors := map[int]struct{}{}
		for _, d := range decisions {
			decidedActors[d.ApproverID] = struct{}{}
		}
		isApprover := false
		for _, c := range chain {
			if c.ApproverID == userID {
				if _, decided := decidedActors[userID]; !decided {
					isApprover = true
					break
				}
			}
		}
		if !isApprover {
			// 找不到该用户的待办审批有两种截然不同的成因，必须区分：
			//  1. 用户确实不是审批人，或已经审批过 → 403（ErrNotApprover）
			//  2. 并发场景下审批已被抢先者消费 → 409（ErrConcurrentModification）
			// 判据：重新读取变更状态。若与本次校验时的快照不同，说明状态已被
			// 其他请求推进，此时返回"不是审批人"会误导调用方（用户其实是审批人，
			// 只是来晚了），也会让运维误判为权限配置问题。
			if fresh, getErr := s.repo.Get(ctx, id, tenantID); getErr == nil && fresh.Status != c.Status {
				return nil, fmt.Errorf("%w: 审批已被并发处理（状态 '%s' → '%s'）",
					ErrConcurrentModification, c.Status, fresh.Status)
			}
			return nil, fmt.Errorf("%w: user_id=%d", ErrNotApprover, userID)
		}

		// 审批先桥接完成对应的 BPMN 待办任务（以流程任务为权威审批来源）。
		// 三态语义：无关联流程实例 → 回退纯业务审批；已绑定但无可操作待办 → bridge 返回
		// conflict 并中止，避免业务状态与流程状态分叉。
		if s.approvalBridge != nil {
			action := "approve"
			if targetStatus == "rejected" {
				action = "reject"
			}
			handled, bridgeErr := s.approvalBridge.CompleteBusinessApprovalTaskWithClient(
				ctx, tx.Client(), tenantID, userID, string(dto.BusinessTypeChange), id, action, comment,
			)
			if bridgeErr != nil {
				return nil, fmt.Errorf("同步流程审批任务失败: %w", bridgeErr)
			}
			approvalDecidedByBridge = handled
			if !handled {
				s.logger.Infow("变更未绑定运行中流程，按审批链直接业务审批",
					"change_id", id, "tenant_id", tenantID, "actor_user_id", userID)
			}
		}
	}

	// 业务生命周期动作同步推进 BPMN 流程，避免业务状态与流程状态分叉。
	// 各动作推进的步数按 change_normal_flow 模板节点编排：
	//   rejected  → 驳回收尾（Activity_Reject → 结束）；
	//   scheduled → 完成排期任务（Activity_Schedule → 实施）；
	//   in_progress → 完成实施任务（Activity_Implement → 验证）；
	//   completed → 完成验证（verify_passed=true）+ 关闭变更两步收尾。
	if s.approvalBridge != nil {
		switch targetStatus {
		case "rejected":
			if _, bridgeErr := s.approvalBridge.AdvanceBusinessWorkflowWithClient(
				ctx, tx.Client(), tenantID, userID, string(dto.BusinessTypeChange), id, nil, 2,
			); bridgeErr != nil {
				return nil, fmt.Errorf("同步流程驳回任务失败: %w", bridgeErr)
			}
		case "scheduled":
			if _, bridgeErr := s.approvalBridge.AdvanceBusinessWorkflowWithClient(
				ctx, tx.Client(), tenantID, userID, string(dto.BusinessTypeChange), id, nil, 1,
			); bridgeErr != nil {
				return nil, fmt.Errorf("同步流程排期任务失败: %w", bridgeErr)
			}
		case "in_progress":
			if _, bridgeErr := s.approvalBridge.AdvanceBusinessWorkflowWithClient(
				ctx, tx.Client(), tenantID, userID, string(dto.BusinessTypeChange), id, nil, 1,
			); bridgeErr != nil {
				return nil, fmt.Errorf("同步流程实施任务失败: %w", bridgeErr)
			}
		case "completed":
			if _, bridgeErr := s.approvalBridge.AdvanceBusinessWorkflowWithClient(
				ctx, tx.Client(), tenantID, userID, string(dto.BusinessTypeChange), id,
				map[string]interface{}{"verify_passed": true}, 4,
			); bridgeErr != nil {
				return nil, fmt.Errorf("同步流程验证任务失败: %w", bridgeErr)
			}
		}
	}

	// 审批决定已由 BPMN bridge 写入 ProcessApprovalDecision（不可变审计表），
	// 无需再回写 legacy change_approvals 表。

	// 会签/阈值门禁：批准必须由审批链求值决定，而不是"当前这一票"。
	//
	// 此前 /transition approved 在桥接完成后直接把状态 CAS 成 approved，会签
	// （parallel/all）、N of M 阈值和多级链的语义只写在无路由的 legacy 入口里，
	// 生产上等于"一个人批掉整单"（两人会签的变更，第一票即 approved）。
	// 现在按链求值：
	//   · 出现必需层驳回票 → 409（已被否决，不能再通过）
	//   · 未达阈值 → 提交事务保留本次决策与流程推进，变更状态保持不变，
	//     返回当前快照（调用方按"已记录、等待其他审批人"处理）
	//   · 达阈值 → 继续走 approved 状态推进
	// 求值读的是 tx.Client() 的决策表：本次审批刚由桥接写入同一事务，若读出
	// 事务外连接会看不到这一票，导致最后一票永远无法达标。
	if targetStatus == "approved" && tx != nil && approvalDecidedByBridge {
		chain, chainErr := s.repo.GetApprovalChain(ctx, id, tenantID)
		if chainErr != nil {
			return nil, fmt.Errorf("failed to get approval chain: %w", chainErr)
		}
		decisions, decErr := s.bpmnApprovalDecisionsWithClient(ctx, tx.Client(), id, tenantID)
		if decErr != nil {
			return nil, fmt.Errorf("failed to get approval decisions: %w", decErr)
		}
		// 决策表不带层级，必须由审批链派生，否则 (approver, level) 计票落空。
		applyChainLevels(chain, decisions)
		outcome := evaluateApprovalQuorum(chain, decisions)
		if outcome.Rejected {
			return nil, fmt.Errorf("%w: 审批链已出现驳回票，不能再通过", ErrInvalidTransition)
		}
		if !outcome.ChainEmpty && !outcome.AllRequiredMet {
			if commitErr := tx.Commit(); commitErr != nil {
				return nil, fmt.Errorf("提交事务失败: %w", commitErr)
			}
			s.logger.Infow("TransitionStatus: 审批决策已记录，会签未集齐，状态保持不变",
				"change_id", id, "tenant_id", tenantID, "actor_user_id", userID,
				"approved_votes", outcome.ApprovedVotes, "required_votes", outcome.RequiredVotes,
				"missing_levels", outcome.MissingLevels)
			// 状态未变：单票不足以放行，调用方按"已记录、等待其他审批人"处理。
			return c, nil
		}
	}

	// 纯业务回退路径的会签门禁：未绑定运行中流程实例（approvalBridge 为 nil 或
	// 桥接返回 handled=false）时，审批决定没有落库位置——ProcessApprovalDecision
	// 的 process_instance_id / process_task_id / process_node_key 都是流程事实，
	// 凭空造一个等于往「不可变审计表」里写假账，所以不能补写决策行。
	// 后果是这一路径的会签票永远无法累积，此时只能认「当前这一票」：
	//   · 单审批人链 / 或签 / 当前票已够阈值 → 继续放行（见下方状态推进）；
	//   · 会签、N of M、多级链未集齐 → 失败关闭（409 + 可行动指引）。
	// 此前该路径在桥接后无条件 CAS approved，等于让会签/阈值链在缺少流程模板时
	// 静默退化成「一个人批掉整单」——与其在文档里写一句"请配置模板"，不如让
	// 缺失配置当场报错（与主路径 2026-10-22 修复的缺陷同源）。
	if targetStatus == "approved" && !approvalDecidedByBridge {
		chain, chainErr := s.repo.GetApprovalChain(ctx, id, tenantID)
		if chainErr != nil {
			return nil, fmt.Errorf("failed to get approval chain: %w", chainErr)
		}
		// 用本票合成一次审批历史：够阈值就放行，不够就拒绝，不依赖任何持久化。
		thisVote := []*ApprovalRecord{{ApproverID: userID, Status: "approved"}}
		applyChainLevels(chain, thisVote)
		fallback := evaluateApprovalQuorum(chain, thisVote)
		if !fallback.ChainEmpty && !fallback.AllRequiredMet {
			s.logger.Warnw("TransitionStatus: 会签链未绑定流程实例，审批无法累积，拒绝放行",
				"change_id", id, "tenant_id", tenantID, "actor_user_id", userID,
				"required_votes", fallback.RequiredVotes, "missing_levels", fallback.MissingLevels)
			return nil, fmt.Errorf(
				"%w: 该变更未绑定运行中的审批流程实例，审批决定无法持久化，会签票无法累积"+
					"（该链需要 %d 票，本次仅 1 票，未达标层级 %v）；"+
					"请为该变更类型配置审批流程模板后重新提交，或将审批链改为单审批人",
				ErrQuorumRequiresWorkflow, fallback.RequiredVotes, fallback.MissingLevels)
		}
	}

	// 终态（rejected/completed/cancelled/rolled_back/failed）需要事务化：写 change +
	// 收口 pending chains；非终态直接更新。
	// failed 是「实施失败」的如实落点：状态机一直允许 in_progress/failed → failed，
	// 但此前没有任何 API 入口能写它，实施失败的变更只能被报成 completed 或直接
	// rolled_back——统计口径与 PIR 结论都随之失真。
	isTerminal := targetStatus == "rejected" || targetStatus == "completed" ||
		targetStatus == "cancelled" || targetStatus == "rolled_back" ||
		targetStatus == "failed" || targetStatus == "closed"
	if isTerminal {
		if tx == nil {
			// 退化路径（entClient==nil 的 mock/legacy 装配）：无事务可依托，但
			// 「状态与结果说明」仍应合成一次条件 UPDATE——分开写会在两次写之间
			// 崩溃时留下「状态已 failed、原因为空」的窗口。仓储实现了组合写
			// （statusNoteUpdater）就用它；未实现（旧 mock）退回两段写，
			// 与历史行为一致。
			if noted, canNote := s.repo.(statusNoteUpdater); canNote {
				ok, casErr := noted.UpdateStatusCASWithNote(ctx, c.ID, tenantID, c.Status, targetStatus, outcomeNote)
				if casErr != nil {
					return nil, fmt.Errorf("failed to update change status: %w", casErr)
				}
				if !ok {
					return nil, fmt.Errorf("%w: 期望 '%s' → '%s'，但状态已被其他请求修改",
						ErrConcurrentModification, c.Status, targetStatus)
				}
			} else {
				ok, casErr := s.repo.UpdateStatusCAS(ctx, c.ID, tenantID, c.Status, targetStatus)
				if casErr != nil {
					return nil, fmt.Errorf("failed to update change status: %w", casErr)
				}
				if !ok {
					return nil, fmt.Errorf("%w: 期望 '%s' → '%s'，但状态已被其他请求修改",
						ErrConcurrentModification, c.Status, targetStatus)
				}
				if outcomeNote != "" {
					// 无事务退化路径里状态已推进，结果说明单独落库。失败仅记日志，
					// 不把已成功的状态推进报成失败（重试会撞 409 而非补写原因）。
					if noteErr := s.repo.SetOutcomeNote(ctx, c.ID, tenantID, outcomeNote); noteErr != nil {
						s.logger.Errorw("TransitionStatus: 记录实施结果说明失败（退化路径）",
							"error", noteErr, "change_id", id, "tenant_id", tenantID)
					}
				}
			}
			// 无事务句柄的装配（mock/legacy）没有原子性可依托，仍需收口，
			// 否则终态变更的审批链会永远停在 pending。失败重试一次后升级日志：
			// 状态已经写入终态，此时报错会让调用方误以为可以重试。
			s.closeChainsDegraded(ctx, id, tenantID)
			c.Status = targetStatus
			c.OutcomeNote = outcomeNote
			return c, nil
		}
		terminalUpdate := tx.Client().Change.UpdateOneID(c.ID).
			Where(change.TenantIDEQ(tenantID), change.StatusEQ(change.Status(c.Status))).
			SetStatus(change.Status(targetStatus))
		// 结果说明与状态推进同一条 UPDATE：分开写会留下"状态已是 failed、原因是空"的
		// 窗口（进程崩溃或后续语句失败），而解释失败原因正是本次改动的目的。
		if outcomeNote != "" {
			terminalUpdate = terminalUpdate.SetOutcomeNote(outcomeNote)
		}
		if _, updateErr := terminalUpdate.Save(ctx); updateErr != nil {
			// StatusEQ 未命中 → 状态已被并发请求抢先推进，属预期内的业务冲突。
			if ent.IsNotFound(updateErr) {
				return nil, fmt.Errorf("%w: 期望 '%s' → '%s'，但状态已被其他请求修改",
					ErrConcurrentModification, c.Status, targetStatus)
			}
			return nil, fmt.Errorf("failed to update change status: %w", updateErr)
		}
		// 审批链收口必须在提交**之前**、且与状态写入同一事务：先提交再收口
		// （旧实现用全局 rawDB，失败只记日志）会留下「变更已终态、审批链仍
		// pending」的半写状态，审批人继续在待办里看到该单。
		rawTxAvailable := true
		if closeErr := tx.ClosePendingApprovalChains(ctx, id, tenantID); closeErr != nil {
			if !errors.Is(closeErr, errRawTxUnavailable) {
				return nil, fmt.Errorf("收口审批链失败: %w", closeErr)
			}
			rawTxAvailable = false
		}
		if commitErr := tx.Commit(); commitErr != nil {
			return nil, fmt.Errorf("提交事务失败: %w", commitErr)
		}
		if !rawTxAvailable {
			// 纯 ent 事务（无 *sql.DB 句柄）：退化为提交后收口并记日志。
			s.closeChainsDegraded(ctx, id, tenantID)
		}
		c.Status = targetStatus
		c.OutcomeNote = outcomeNote
		return c, nil
	}

	// 非终态：CAS 推进状态（乐观锁）。
	// 直接 repo.Update 是 read-then-unconditional-write：并发请求读到同一旧状态、
	// 各自通过状态机校验后都能写入成功，N 个请求全部返回 200，客户端与审计无法
	// 区分"谁真正推进了状态"。CAS 把并发窗口收敛到一次条件更新，抢先者成功、
	// 落后者收到 409；与桥接同事务，CAS 落败时已完成的流程任务一并回滚。
	if tx == nil {
		ok, casErr := s.repo.UpdateStatusCAS(ctx, c.ID, tenantID, c.Status, targetStatus)
		if casErr != nil {
			return nil, fmt.Errorf("failed to update change status: %w", casErr)
		}
		if !ok {
			return nil, fmt.Errorf("%w: 期望 '%s' → '%s'，但状态已被其他请求修改",
				ErrConcurrentModification, c.Status, targetStatus)
		}
	} else {
		affected, err := tx.Client().Change.Update().
			Where(change.IDEQ(c.ID), change.TenantIDEQ(tenantID), change.StatusEQ(change.Status(c.Status))).
			SetStatus(change.Status(targetStatus)).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to update change status: %w", err)
		}
		if affected == 0 {
			return nil, fmt.Errorf("%w: 期望 '%s' → '%s'，但状态已被其他请求修改",
				ErrConcurrentModification, c.Status, targetStatus)
		}
		if commitErr := tx.Commit(); commitErr != nil {
			return nil, fmt.Errorf("提交事务失败: %w", commitErr)
		}
	}

	// 注意：此处绝不能再 `c.Status = targetStatus; return s.repo.Update(ctx, c)`——
	// 那是用陈旧快照把 CAS 刚写好的状态又一次全列无条件覆盖（title/description/
	// affected_cis 全量替换都按读取时旧值回写，并发提交的其他字段编辑会被静默
	// 卡掉，CAS 等于白做）。转换只改变 status 一个字段，与上方终态分支一致：
	// 直接返回携带新状态的内存快照，不做任何二次落库。
	c.Status = targetStatus
	return c, nil
}

// closeChainsDegraded 无事务句柄时的审批链收口兜底（mock/legacy 装配才可达：
// 生产 EntRepository 恒有原生句柄，收口在事务内完成）。此路径上变更状态已
// 写入终态，收口失败不能上抛（调用方会误以为可以重试，实际重试必撞 409），
// 只能重试一次后升级日志，交给运维/补偿任务人工收口。
func (s *Service) closeChainsDegraded(ctx context.Context, id, tenantID int) {
	if closeErr := service.CloseChangeApprovalChains(ctx, id, tenantID); closeErr != nil {
		s.logger.Warnw("TransitionStatus: 收口审批链首次失败（无事务退化路径），重试一次",
			"error", closeErr, "change_id", id, "tenant_id", tenantID)
		if retryErr := service.CloseChangeApprovalChains(ctx, id, tenantID); retryErr != nil {
			s.logger.Errorw("TransitionStatus: 收口审批链失败（无事务退化路径，已重试）——"+
				"变更已终态但审批链可能仍 pending，需要人工收口",
				"error", retryErr, "change_id", id, "tenant_id", tenantID)
		}
	}
}

// GetApprovalHistory returns approval records for a change.
// Migration: Uses ProcessApprovalDecision as the authoritative source (written by
// BPMN bridge) instead of legacy change_approvals table. Levels field is still
// populated from change_approval_chains for frontend approval timeline display.
func (s *Service) GetApprovalHistory(ctx context.Context, changeID int, tenantID int) ([]*ApprovalRecord, error) {
	decisions, err := s.GetBPMNApprovalDecisions(ctx, changeID, tenantID)
	if err != nil {
		return nil, err
	}
	chain, err := s.repo.GetApprovalChain(ctx, changeID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get approval chain for levels: %w", err)
	}
	applyChainLevels(chain, decisions)
	return decisions, nil
}

// GetBPMNApprovalDecisions queries the authoritative ProcessApprovalDecision table
// for all approval decisions related to a change. Results are mapped to
// ApprovalRecord for compatibility with quorum evaluation and the approval
// timeline. ProcessApprovalDecision is the single source of truth for approval
// decisions once the BPMN bridge writes them; change_approvals is legacy.
func (s *Service) GetBPMNApprovalDecisions(ctx context.Context, changeID int, tenantID int) ([]*ApprovalRecord, error) {
	return s.bpmnApprovalDecisionsWithClient(ctx, s.entClient, changeID, tenantID)
}

// bpmnApprovalDecisionsWithClient 用指定 ent 客户端读取审批决定。
// 传入事务内 client 时能读到本事务刚由 BPMN 桥接写入的决策——会签求值必须
// 读得到"当前这一票"，否则最后一票永远无法把整单推过阈值。
func (s *Service) bpmnApprovalDecisionsWithClient(ctx context.Context, client *ent.Client, changeID int, tenantID int) ([]*ApprovalRecord, error) {
	// 未装配 BPMN 桥接（entClient 为 nil，如纯 legacy 部署/单测构造）时视为无决策，
	// 与本文件其他 entClient 防御点一致；调用方按空审批历史处理，走 legacy 审批链语义。
	if client == nil {
		return []*ApprovalRecord{}, nil
	}
	businessID := strconv.Itoa(changeID)
	decisions, err := client.ProcessApprovalDecision.Query().
		Where(
			processapprovaldecision.BusinessType("change"),
			processapprovaldecision.BusinessID(businessID),
			processapprovaldecision.TenantID(tenantID),
		).
		Order(ent.Asc(processapprovaldecision.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query BPMN approval decisions: %w", err)
	}

	records := make([]*ApprovalRecord, 0, len(decisions))
	for _, d := range decisions {
		comment := d.Comment
		var commentPtr *string
		if comment != "" {
			commentPtr = &comment
		}
		approvedAt := d.CreatedAt
		records = append(records, &ApprovalRecord{
			ID:           d.ID,
			ChangeID:     changeID,
			TenantID:     d.TenantID,
			ApproverID:   d.ActorID,
			ApproverName: d.ActorName,
			Status:       d.Decision,
			Comment:      commentPtr,
			ApprovedAt:   &approvedAt,
			CreatedAt:    d.CreatedAt,
		})
	}
	return records, nil
}

// ==================== PIR (Post-Implementation Review) Methods ====================

func (s *Service) CreatePIR(ctx context.Context, req *dto.CreateChangePIRRequest, reviewerID, tenantID int) (*dto.ChangePIRResponse, error) {
	s.logger.Infow("Creating PIR", "change_id", req.ChangeID, "reviewer_id", reviewerID)
	if s.pirService != nil {
		return s.pirService.CreatePIR(ctx, req, reviewerID, tenantID)
	}
	return nil, fmt.Errorf("PIR service not initialized")
}

func (s *Service) GetPIRByChange(ctx context.Context, changeID, tenantID int) (*dto.ChangePIRResponse, error) {
	if s.pirService != nil {
		return s.pirService.GetPIRByChange(ctx, changeID, tenantID)
	}
	return nil, fmt.Errorf("PIR service not initialized")
}

func (s *Service) ListPIRs(ctx context.Context, tenantID int, page, pageSize int, result string) (*dto.ChangePIRListResponse, error) {
	if s.pirService != nil {
		return s.pirService.ListPIRs(ctx, tenantID, page, pageSize, result)
	}
	return nil, fmt.Errorf("PIR service not initialized")
}

func (s *Service) UpdatePIR(ctx context.Context, pirID int, req *dto.UpdateChangePIRRequest, tenantID int) (*dto.ChangePIRResponse, error) {
	if s.pirService != nil {
		return s.pirService.UpdatePIR(ctx, pirID, req, tenantID)
	}
	return nil, fmt.Errorf("PIR service not initialized")
}

func (s *Service) DeletePIR(ctx context.Context, pirID, tenantID int) error {
	if s.pirService != nil {
		return s.pirService.DeletePIR(ctx, pirID, tenantID)
	}
	return fmt.Errorf("PIR service not initialized")
}
