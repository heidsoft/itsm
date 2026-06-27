package change

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/cirelationship"
	"itsm-backend/ent/configurationitem"
	"itsm-backend/ent/incident"
	"itsm-backend/service"

	"go.uber.org/zap"
)

type Service struct {
	repo       Repository
	logger     *zap.SugaredLogger
	entClient  *ent.Client
	pirService *service.ChangePIRService
}

func NewService(repo Repository, entClient *ent.Client, logger *zap.SugaredLogger) *Service {
	svc := &Service{
		repo:      repo,
		entClient: entClient,
		logger:    logger,
	}
	// Initialize PIR service
	svc.pirService = service.NewChangePIRService(entClient, logger)
	return svc
}

// Change methods
func (s *Service) CreateChange(ctx context.Context, c *Change) (*Change, error) {
	s.logger.Infow("Creating change", "title", c.Title, "tenant_id", c.TenantID)
	return s.repo.Create(ctx, c)
}

func (s *Service) GetChange(ctx context.Context, id int, tenantID int) (*Change, error) {
	return s.repo.Get(ctx, id, tenantID)
}

func (s *Service) ListChanges(ctx context.Context, tenantID int, page, size int, status, search string) ([]*Change, int, error) {
	return s.repo.List(ctx, tenantID, page, size, status, search)
}

func (s *Service) UpdateChange(ctx context.Context, c *Change) (*Change, error) {
	return s.repo.Update(ctx, c)
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

		items = append(items, dto.ChangeCalendarItem{
			ID:           c.ID,
			Title:        c.Title,
			ChangeNumber: fmt.Sprintf("C-%d", c.ID),
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

	// 3. Update change status to pending
	c.Status = "pending"
	updatedChange, err := s.repo.Update(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("failed to update change status: %w", err)
	}

	// 3.5. If no approvers specified, default to the change creator
	if len(req.ApproverIDs) == 0 {
		req.ApproverIDs = []int{c.CreatedBy}
		s.logger.Infow("No approvers specified, defaulting to change creator", "change_id", changeID, "creator_id", c.CreatedBy)
	}

	// 4. Validate approvers belong to the same tenant before creating approval records
	for _, approverID := range req.ApproverIDs {
		valid, err := s.repo.ValidateApproverBelongsToTenant(ctx, approverID, tenantID)
		if err != nil {
			s.logger.Warnw("Failed to validate approver", "error", err, "approver_id", approverID)
			return nil, fmt.Errorf("验证审批人失败")
		}
		if !valid {
			s.logger.Warnw("Approver does not belong to tenant", "approver_id", approverID, "tenant_id", tenantID)
			return nil, fmt.Errorf("审批人 %d 不属于当前租户", approverID)
		}
	}

	// 5. Create approval request records for each approver
	for _, approverID := range req.ApproverIDs {
		record := &ApprovalRecord{
			ChangeID:   changeID,
			ApproverID: approverID,
			Status:     "pending",
			Comment:    &req.Comment,
		}
		_, err := s.repo.CreateApprovalRecord(ctx, record)
		if err != nil {
			s.logger.Warnw("Failed to create approval record", "error", err, "change_id", changeID, "approver_id", approverID)
			// Continue creating other records even if one fails
		}
	}

	// 5.5. Create approval chain entries for each approver
	chainItems := make([]*ApprovalChain, 0, len(req.ApproverIDs))
	for i, approverID := range req.ApproverIDs {
		chainItems = append(chainItems, &ApprovalChain{
			ChangeID:   changeID,
			Level:      i + 1,
			ApproverID: approverID,
			Role:       "approver",
			Status:     "pending",
			IsRequired: true,
		})
	}
	if err := s.repo.CreateApprovalChain(ctx, chainItems); err != nil {
		s.logger.Warnw("Failed to create approval chain", "error", err, "change_id", changeID)
		// Non-fatal: approval records are already created
	}

	// 6. Notify approvers (optional - to be implemented later or via async)
	// For now, just log the submission
	s.logger.Infow("Change submitted for approval", "change_id", changeID, "submitter_id", submitterID, "approvers", req.ApproverIDs)

	return updatedChange, nil
}

// Approval methods
func (s *Service) SubmitApproval(ctx context.Context, record *ApprovalRecord, tenantID int) (*ApprovalRecord, error) {
	// Custom business logic: when submitting, we check if change exists
	c, err := s.repo.Get(ctx, record.ChangeID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("change not found")
	}

	record.Status = "pending"
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
		ID:      recordID,
		Status:  status,
		Comment: comment,
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
			c.Status = "rejected"
			if _, err := s.repo.Update(ctx, c); err != nil {
				s.logger.Errorw("ProcessApproval: failed to update change status to rejected", "error", err, "change_id", res.ChangeID)
				return nil, fmt.Errorf("failed to update change status: %w", err)
			}
		}
	}

	return res, nil
}

func (s *Service) checkAndTransitionChange(ctx context.Context, changeID, tenantID int) error {
	chain, err := s.repo.GetApprovalChain(ctx, changeID)
	if err != nil {
		s.logger.Errorw("checkAndTransitionChange: failed to get approval chain", "error", err, "change_id", changeID)
		return err
	}
	history, err := s.repo.GetApprovalHistory(ctx, changeID, tenantID)
	if err != nil {
		s.logger.Errorw("checkAndTransitionChange: failed to get approval history", "error", err, "change_id", changeID)
		return err
	}

	// Simple logic: if all required members approved, transition to 'approved'
	allApproved := true
	requiredCount := 0
	approvedMap := make(map[int]bool)
	for _, h := range history {
		if h.Status == "approved" {
			approvedMap[h.ApproverID] = true
		}
	}

	for _, item := range chain {
		if item.IsRequired {
			requiredCount++
			if !approvedMap[item.ApproverID] {
				allApproved = false
				break
			}
		}
	}

	if allApproved && requiredCount > 0 {
		c, err := s.repo.Get(ctx, changeID, tenantID)
		if err != nil {
			s.logger.Errorw("checkAndTransitionChange: failed to get change", "error", err, "change_id", changeID)
			return err
		}
		if c != nil {
			c.Status = "approved"
			if _, err := s.repo.Update(ctx, c); err != nil {
				s.logger.Errorw("checkAndTransitionChange: failed to update change status to approved", "error", err, "change_id", changeID)
				return err
			}
		}
	}
	return nil
}

func (s *Service) ConfigureWorkflow(ctx context.Context, changeID int, items []*ApprovalChain) error {
	// Clear existing and set new
	if err := s.repo.DeleteApprovalChain(ctx, changeID); err != nil {
		s.logger.Errorw("ConfigureWorkflow: failed to delete approval chain", "error", err, "change_id", changeID)
		return fmt.Errorf("failed to delete approval chain: %w", err)
	}
	if err := s.repo.CreateApprovalChain(ctx, items); err != nil {
		s.logger.Errorw("ConfigureWorkflow: failed to create approval chain", "error", err, "change_id", changeID)
		return fmt.Errorf("failed to create approval chain: %w", err)
	}
	return nil
}

func (s *Service) GetApprovalSummary(ctx context.Context, changeID, tenantID int) (interface{}, error) {
	chain, err := s.repo.GetApprovalChain(ctx, changeID)
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

func (s *Service) GetRisk(ctx context.Context, changeID int) (*RiskAssessment, error) {
	return s.repo.GetRiskAssessment(ctx, changeID)
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
func (s *Service) TransitionStatus(ctx context.Context, id, tenantID, userID int, targetStatus string) (*Change, error) {
	c, err := s.repo.Get(ctx, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("change not found")
	}

	// Validate state transition
	if !isValidChangeStatusTransition(c.Status, targetStatus) {
		return nil, fmt.Errorf("无效的状态转换: 从 '%s' 到 '%s'", c.Status, targetStatus)
	}

	// For approval actions, verify user is the approver
	if targetStatus == "approved" || targetStatus == "rejected" {
		history, err := s.repo.GetApprovalHistory(ctx, id, tenantID)
		if err != nil {
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
						ID:     h.ID,
						Status: approvedStatus,
					}); err != nil {
						s.logger.Warnw("TransitionStatus: failed to update approval record", "error", err, "record_id", h.ID)
					}
					break
				}
			}
		}
	}

	c.Status = targetStatus
	return s.repo.Update(ctx, c)
}

// isValidChangeStatusTransition validates state transitions for Change entities
func isValidChangeStatusTransition(currentStatus, newStatus string) bool {
	validTransitions := map[string][]string{
		"draft":       {"pending", "cancelled"},
		"pending":     {"approved", "rejected", "cancelled"},
		"approved":    {"scheduled", "in_progress", "cancelled"},
		"rejected":    {},
		"scheduled":   {"in_progress", "cancelled"},
		"in_progress": {"completed", "failed", "cancelled"},
		"completed":   {},
		"failed":      {"in_progress", "cancelled"},
		"cancelled":   {},
	}

	allowed, ok := validTransitions[currentStatus]
	if !ok {
		return false
	}

	for _, status := range allowed {
		if status == newStatus {
			return true
		}
	}
	return false
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
