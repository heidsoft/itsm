package change

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/change"
	"itsm-backend/ent/cirelationship"
	"itsm-backend/ent/configurationitem"
	"itsm-backend/ent/incident"
	"itsm-backend/handlers/common/datascope"
	"itsm-backend/service"

	"go.uber.org/zap"
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
		// P0-1：变更审批桥接到 BPMN 任务，避免流程实例悬挂
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

func (s *Service) UpdateChange(ctx context.Context, c *Change) (*Change, error) {
	// P1-2: Guard governance fields. Changes must be in draft status to freely edit fields that
	// feed CAB approval / risk snapshot. Reject silent post-submission mutations and force the
	// caller to return to draft for re-approval instead.
	existing, err := s.repo.Get(ctx, c.ID, c.TenantID)
	if err != nil || existing == nil {
		return nil, fmt.Errorf("change not found")
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

func (s *Service) DeleteChange(ctx context.Context, id int, tenantID int) error {
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

	// 3. 解析审批计划（引擎层级 + quorum）。
	//     命中激活 "change" 链时以引擎解析结果（保留每层 approval_type/threshold）覆盖
	//     调用方传入审批人，杜绝「无审批人自审」与越权自选审批人；链阻塞（必需层无审批人
	//     且策略为阻断）→ 失败关闭；无激活链 → 回退旧逻辑（调用方传入 / 默认创建人）。
	var plan []ApprovalLevelPlan
	if s.approvalChain != nil {
		chainPlan, cerr := s.resolveChangeChainPlan(ctx, tenantID, c.ID, c.CreatedBy)
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

	// 4. 提交（写入 change_approval_chains，保留层级与 quorum 元数据）
	if err := s.repo.SubmitForApproval(ctx, changeID, tenantID, plan, req.Comment); err != nil {
		s.logger.Warnw("Failed to atomically submit change", "error", err, "change_id", changeID)
		return nil, fmt.Errorf("提交变更审批失败: %w", err)
	}

	// P0-2：提交审批后同步完成流程中的“变更评估”任务，推进到审批网关。
	// 这一段失败不回滚已提交的业务审批（后续动作会自愈推进流程），仅记录告警。
	if s.approvalBridge != nil {
		if _, bridgeErr := s.approvalBridge.AdvanceBusinessWorkflow(
			ctx, tenantID, submitterID, string(dto.BusinessTypeChange), changeID, nil, 1,
		); bridgeErr != nil {
			s.logger.Warnw("提交变更后推进流程任务失败（非致命，后续动作自愈）", "error", bridgeErr, "change_id", changeID)
		}
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
func (s *Service) resolveChangeChainPlan(ctx context.Context, tenantID, changeID, submitterID int) ([]ApprovalLevelPlan, error) {
	evalCtx := service.ApprovalEvalContext{
		TenantID:    tenantID,
		EntityType:  "change",
		RequesterID: submitterID,
	}
	plan, err := s.approvalChain.ResolveApprovalPlan(ctx, tenantID, "change", evalCtx, nil)
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
func (s *Service) SubmitApproval(ctx context.Context, record *ApprovalRecord, tenantID int) (*ApprovalRecord, error) {
	// Custom business logic: when submitting, we check if change exists
	c, err := s.repo.Get(ctx, record.ChangeID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("change not found")
	}

	// P0-1 修复：审批人来源只能取自本变更的审批链（change_approval_chains），
	// 禁止请求体任意指定。否则持 change:write 的用户可注入以自己为审批人的
	// pending 记录再调用 /transition 直接 approved，绕过 CAB / quorum / 标准
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

	record.Status = "pending"
	record.TenantID = tenantID
	res, err := s.repo.CreateApprovalRecord(ctx, record)
	if err != nil {
		return nil, err
	}

	// Update change status to pending if needed
	if c.Status == "draft" {
		c.Status = "pending"
		if _, err := s.repo.Update(ctx, c); err != nil {
			s.logger.Errorw("SubmitApproval: failed to update change status to pending", "error", err, "change_id", c.ID)
			return nil, fmt.Errorf("failed to update change status: %w", err)
		}
	}

	return res, nil
}

func (s *Service) ProcessApproval(ctx context.Context, recordID int, status string, comment *string, tenantID int) (*ApprovalRecord, error) {
	// 1. Get existing record (we need to know what change it refers to)
	// We might need a repo.GetApprovalRecord method, let's add it or use a workaround if it's missing in repo interface
	// For now, I'll assume I can update by ID directly if the repository implementation allows it

	rec := &ApprovalRecord{
		ID:       recordID,
		TenantID: tenantID,
		Status:   status,
		Comment:  comment,
	}

	res, err := s.repo.UpdateApprovalRecord(ctx, rec)
	if err != nil {
		return nil, err
	}

	// 2. Logic to check if all approvals are done
	if status == "approved" {
		if err := s.checkAndTransitionChange(ctx, res.ChangeID, tenantID); err != nil {
			s.logger.Errorw("ProcessApproval: checkAndTransitionChange failed", "error", err, "change_id", res.ChangeID)
		}
	} else if status == "rejected" {
		// If one rejects, the whole change is rejected?
		c, err := s.repo.Get(ctx, res.ChangeID, tenantID)
		if err != nil {
			s.logger.Errorw("ProcessApproval: failed to get change on rejection", "error", err, "change_id", res.ChangeID)
			return nil, fmt.Errorf("failed to get change: %w", err)
		}
		if c != nil {
			// C-2 修复：通过状态机校验禁止非法转换（例如 cancelled → rejected 终态互跳）
			target := "rejected"
			if !service.IsValidChangeStatusTransition(c.Status, target, c.Type) {
				s.logger.Warnw("ProcessApproval: skip invalid change status transition",
					"change_id", res.ChangeID, "from", c.Status, "to", target)
				return res, nil
			}
			// H-2 修复：事务化更新 change 为终态；提交成功后单独收口 pending chains（CloseChangeApprovalChains 内部使用 rawDB，*ent.Tx 不提供 ExecContext）
			tx, txErr := s.entClient.Tx(ctx)
			if txErr != nil {
				return nil, fmt.Errorf("开启事务失败: %w", txErr)
			}
			defer tx.Rollback()

			if _, updateErr := tx.Change.UpdateOneID(c.ID).
				Where(change.TenantID(tenantID)).
				SetStatus(target).
				Save(ctx); updateErr != nil {
				s.logger.Errorw("ProcessApproval: failed to update change status to rejected", "error", updateErr, "change_id", res.ChangeID)
				return nil, fmt.Errorf("failed to update change status: %w", updateErr)
			}
			if commitErr := tx.Commit(); commitErr != nil {
				return nil, fmt.Errorf("提交事务失败: %w", commitErr)
			}
			if closeErr := service.CloseChangeApprovalChains(ctx, res.ChangeID, tenantID); closeErr != nil {
				s.logger.Errorw("ProcessApproval: 收口审批链失败（非致命，后续状态机兜底）",
					"error", closeErr, "change_id", res.ChangeID, "tenant_id", tenantID)
			}
		}
	}

	return res, nil
}

func (s *Service) checkAndTransitionChange(ctx context.Context, changeID, tenantID int) error {
	chain, err := s.repo.GetApprovalChain(ctx, changeID, tenantID)
	if err != nil {
		s.logger.Errorw("checkAndTransitionChange: failed to get approval chain", "error", err, "change_id", changeID)
		return err
	}
	history, err := s.repo.GetApprovalHistory(ctx, changeID, tenantID)
	if err != nil {
		s.logger.Errorw("checkAndTransitionChange: failed to get approval history", "error", err, "change_id", changeID)
		return err
	}

	// 按 level 聚合每层：审批人集合、已批准/已驳回、该层阈值（quorum）。
	type levelAgg struct {
		required    bool
		threshold   int
		approverIDs map[int]struct{}
		approved    map[int]struct{}
		rejected    bool
	}
	levels := map[int]*levelAgg{}
	var order []int
	for _, item := range chain {
		agg, ok := levels[item.Level]
		if !ok {
			agg = &levelAgg{approverIDs: map[int]struct{}{}, approved: map[int]struct{}{}}
			levels[item.Level] = agg
			order = append(order, item.Level)
		}
		agg.approverIDs[item.ApproverID] = struct{}{}
		if item.IsRequired {
			agg.required = true
		}
		// 该层阈值：引擎写入的 threshold 优先；缺省按类型推导
		// （parallel/all = 候选人数，serial/or = 1）。
		if item.Threshold > 0 {
			agg.threshold = item.Threshold
		} else if item.ApprovalType == "parallel" || item.ApprovalType == "all" {
			agg.threshold = len(agg.approverIDs)
		} else {
			agg.threshold = 1
		}
	}
	// 标记历史决定（租户过滤由仓储保证）。
	// P1 修复：按 (approverID, level) 双重匹配——仅当该审批人确实属于该层级时，
	// 其批准/驳回才计入该层 quorum，避免跨层互相串（一次性审批可满足多层，但绝不串层）。
	for _, h := range history {
		for _, lvl := range h.Levels {
			agg, ok := levels[lvl]
			if !ok {
				continue
			}
			if _, ok := agg.approverIDs[h.ApproverID]; !ok {
				continue
			}
			if h.Status == "approved" {
				agg.approved[h.ApproverID] = struct{}{}
			}
			if h.Status == "rejected" {
				agg.rejected = true
			}
		}
	}

	// 计算每层满意度
	allRequiredSatisfied := true
	anyRequiredRejected := false
	hasRequired := false
	for _, lvl := range order {
		agg := levels[lvl]
		if !agg.required {
			continue
		}
		hasRequired = true
		if agg.rejected {
			anyRequiredRejected = true
			break
		}
		if len(agg.approved) < agg.threshold {
			allRequiredSatisfied = false
		}
	}

	// 任一必需层被驳回 → 整体驳回
	if anyRequiredRejected {
		c, err := s.repo.Get(ctx, changeID, tenantID)
		if err != nil {
			return err
		}
		target := "rejected"
		if !service.IsValidChangeStatusTransition(c.Status, target, c.Type) {
			s.logger.Warnw("checkAndTransitionChange: skip invalid change status transition (reject)",
				"change_id", changeID, "from", c.Status, "to", target)
			return nil
		}
		c.Status = target
		if _, err := s.repo.Update(ctx, c); err != nil {
			s.logger.Errorw("checkAndTransitionChange: failed to update change status to rejected", "error", err, "change_id", changeID)
			return err
		}
		return nil
	}

	// 所有必需层均满足 → 整体批准（至少存在一个必需层）
	if allRequiredSatisfied && hasRequired {
		c, err := s.repo.Get(ctx, changeID, tenantID)
		if err != nil {
			return err
		}
		target := "approved"
		if !service.IsValidChangeStatusTransition(c.Status, target, c.Type) {
			s.logger.Warnw("checkAndTransitionChange: skip invalid change status transition (approve)",
				"change_id", changeID, "from", c.Status, "to", target)
			return nil
		}
		c.Status = target
		if _, err := s.repo.Update(ctx, c); err != nil {
			s.logger.Errorw("checkAndTransitionChange: failed to update change status to approved", "error", err, "change_id", changeID)
			return err
		}
	}
	return nil
}

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
	summary.RequiresCAB = summary.RecommendedRiskLevel == "high" || changeEntity.Type == "emergency" || summary.CriticalCICount > 0
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
		hints = append(hints, "命中关键 CI，建议走 CAB 审批并校验变更窗口。")
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
	if summary.RequiresCAB {
		practices = append(practices, "change_enablement")
	}
	if summary.CriticalCICount > 0 {
		practices = append(practices, "monitoring_and_event_management")
	}
	return practices
}

// TransitionStatus transitions a change to a new status
// For approve/reject actions, verifies user is the designated approver
func (s *Service) TransitionStatus(ctx context.Context, id, tenantID, userID int, targetStatus, comment string) (*Change, error) {
	c, err := s.repo.Get(ctx, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("change not found")
	}

	// Validate state transition (使用 service 包的 canonical 状态机，保证与 legacy service 一致)
	if !service.IsValidChangeStatusTransition(c.Status, targetStatus, c.Type) {
		return nil, fmt.Errorf("无效的状态转换: 从 '%s' 到 '%s'", c.Status, targetStatus)
	}

	// For approval actions, verify user is the approver
	if targetStatus == "approved" || targetStatus == "rejected" {
		history, err := s.repo.GetApprovalHistory(ctx, id, tenantID)
		if err != nil {
			s.logger.Errorw("TransitionStatus: failed to get approval history", "error", err, "change_id", id, "tenant_id", tenantID)
			return nil, fmt.Errorf("failed to get approval history")
		}
		// Find if this user has a pending approval
		isApprover := false
		for _, h := range history {
			if h.ApproverID == userID && h.Status == "pending" {
				isApprover = true
				break
			}
		}
		if !isApprover {
			return nil, fmt.Errorf("用户不是该变更的审批人，无权执行此操作")
		}

		// P0-1：审批先桥接完成对应的 BPMN 待办任务（以流程任务为权威审批来源）。
		// 无关联运行中流程实例时回退为纯业务审批；若存在待办流程任务但完成失败，
		// 则中止业务审批，避免变更状态与流程状态分叉。
		if s.approvalBridge != nil {
			action := "approve"
			if targetStatus == "rejected" {
				action = "reject"
			}
			if _, bridgeErr := s.approvalBridge.CompleteBusinessApprovalTask(
				ctx, tenantID, userID, string(dto.BusinessTypeChange), id, action, comment,
			); bridgeErr != nil {
				return nil, fmt.Errorf("同步流程审批任务失败: %w", bridgeErr)
			}
		}
	}

	// P0-2：业务生命周期动作同步推进 BPMN 流程，避免业务状态与流程状态分叉。
	// 各动作推进的步数按 change_normal_flow 模板节点编排：
	//   rejected  → 驳回收尾（Activity_Reject → 结束）；
	//   scheduled → 完成排期任务（Activity_Schedule → 实施）；
	//   in_progress → 完成实施任务（Activity_Implement → 验证）；
	//   completed → 完成验证（verify_passed=true）+ 关闭变更两步收尾。
	if s.approvalBridge != nil {
		switch targetStatus {
		case "rejected":
			if _, bridgeErr := s.approvalBridge.AdvanceBusinessWorkflow(
				ctx, tenantID, userID, string(dto.BusinessTypeChange), id, nil, 2,
			); bridgeErr != nil {
				return nil, fmt.Errorf("同步流程驳回任务失败: %w", bridgeErr)
			}
		case "scheduled":
			if _, bridgeErr := s.approvalBridge.AdvanceBusinessWorkflow(
				ctx, tenantID, userID, string(dto.BusinessTypeChange), id, nil, 1,
			); bridgeErr != nil {
				return nil, fmt.Errorf("同步流程排期任务失败: %w", bridgeErr)
			}
		case "in_progress":
			if _, bridgeErr := s.approvalBridge.AdvanceBusinessWorkflow(
				ctx, tenantID, userID, string(dto.BusinessTypeChange), id, nil, 1,
			); bridgeErr != nil {
				return nil, fmt.Errorf("同步流程实施任务失败: %w", bridgeErr)
			}
		case "completed":
			if _, bridgeErr := s.approvalBridge.AdvanceBusinessWorkflow(
				ctx, tenantID, userID, string(dto.BusinessTypeChange), id,
				map[string]interface{}{"verify_passed": true}, 4,
			); bridgeErr != nil {
				return nil, fmt.Errorf("同步流程验证任务失败: %w", bridgeErr)
			}
		}
	}

	// For approve action, update the approval record to approved
	if targetStatus == "approved" {
		history, err := s.repo.GetApprovalHistory(ctx, id, tenantID)
		if err != nil {
			s.logger.Warnw("TransitionStatus: failed to get approval history for record update", "error", err)
		} else {
			for _, h := range history {
				if h.ApproverID == userID && h.Status == "pending" {
					approvedStatus := "approved"
					if _, err := s.repo.UpdateApprovalRecord(ctx, &ApprovalRecord{
						ID:       h.ID,
						TenantID: tenantID,
						Status:   approvedStatus,
					}); err != nil {
						s.logger.Warnw("TransitionStatus: failed to update approval record", "error", err, "record_id", h.ID)
					}
					break
				}
			}
		}
	}

	// H-2 / C-2 修复：
	// 1. 终态（rejected/completed/cancelled/rolled_back）需要事务化：写 change + 收口 pending chains
	// 2. 非终态直接更新
	isTerminal := targetStatus == "rejected" || targetStatus == "completed" ||
		targetStatus == "cancelled" || targetStatus == "rolled_back" ||
		targetStatus == "closed"
	if isTerminal {
		tx, txErr := s.entClient.Tx(ctx)
		if txErr != nil {
			return nil, fmt.Errorf("开启事务失败: %w", txErr)
		}
		defer tx.Rollback()

		if _, updateErr := tx.Change.UpdateOneID(c.ID).
			Where(change.TenantID(tenantID)).
			SetStatus(targetStatus).
			Save(ctx); updateErr != nil {
			return nil, fmt.Errorf("failed to update change status: %w", updateErr)
		}
		if commitErr := tx.Commit(); commitErr != nil {
			return nil, fmt.Errorf("提交事务失败: %w", commitErr)
		}
		if closeErr := service.CloseChangeApprovalChains(ctx, id, tenantID); closeErr != nil {
			s.logger.Errorw("TransitionStatus: 收口审批链失败（非致命，后续状态机兜底）",
				"error", closeErr, "change_id", id, "tenant_id", tenantID)
		}
		c.Status = targetStatus
		return c, nil
	}

	c.Status = targetStatus
	return s.repo.Update(ctx, c)
}

// GetApprovalHistory returns approval records for a change
func (s *Service) GetApprovalHistory(ctx context.Context, changeID int, tenantID int) ([]*ApprovalRecord, error) {
	return s.repo.GetApprovalHistory(ctx, changeID, tenantID)
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
