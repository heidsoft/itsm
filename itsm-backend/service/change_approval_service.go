package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/change"
	entuser "itsm-backend/ent/user"

	"go.uber.org/zap"
)

// ChangeApprovalService 变更审批服务
type ChangeApprovalService struct {
	client *ent.Client
	rawDB  *sql.DB
	logger *zap.SugaredLogger
}

// NewChangeApprovalService 创建变更审批服务
func NewChangeApprovalService(client *ent.Client, rawDB *sql.DB, logger *zap.SugaredLogger) *ChangeApprovalService {
	return &ChangeApprovalService{
		client: client,
		rawDB:  rawDB,
		logger: logger,
	}
}

// getTenantUser 查询用户并强制 tenant_id 校验。所有本服务内 User 查询都应走这里，
// 避免通过 User.Get(ctx, id) 枚举跨租户用户 ID。
func (s *ChangeApprovalService) getTenantUser(ctx context.Context, userID, tenantID int) (*ent.User, error) {
	return s.client.User.Query().
		Where(entuser.ID(userID), entuser.TenantID(tenantID)).
		Only(ctx)
}

// CreateChangeApproval 创建变更审批记录
func (s *ChangeApprovalService) CreateChangeApproval(ctx context.Context, req *dto.CreateChangeApprovalRequest, tenantID int) (*dto.ChangeApprovalResponse, error) {
	// 验证变更是否存在且属于当前租户
	changeEntity, err := s.client.Change.Get(ctx, req.ChangeID)
	if err != nil {
		return nil, fmt.Errorf("change not found: %w", err)
	}
	if changeEntity.TenantID != tenantID {
		return nil, fmt.Errorf("change does not belong to current tenant")
	}

	// 验证审批人是否存在且属于当前租户
	approver, err := s.getTenantUser(ctx, req.ApproverID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("approver not found: %w", err)
	}

	// 插入审批记录
	query := `
		INSERT INTO change_approvals (change_id, tenant_id, approver_id, status, comment, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`

	var id int
	var createdAt time.Time
	err = s.rawDB.QueryRowContext(
		ctx, query,
		req.ChangeID,
		tenantID,
		req.ApproverID,
		"pending",
		req.Comment,
		time.Now(),
		time.Now(),
	).Scan(&id, &createdAt)
	if err != nil {
		s.logger.Errorw("Failed to create change approval", "error", err, "change_id", req.ChangeID)
		return nil, fmt.Errorf("failed to create change approval: %w", err)
	}

	// 构建响应
	response := &dto.ChangeApprovalResponse{
		ID:           id,
		ChangeID:     req.ChangeID,
		ApproverID:   req.ApproverID,
		ApproverName: approver.Name,
		Status:       dto.ChangeStatusPending,
		Comment:      req.Comment,
		CreatedAt:    createdAt,
	}

	s.logger.Infow("Change approval created", "approval_id", id, "change_id", req.ChangeID)
	return response, nil
}

// UpdateChangeApproval 更新变更审批状态
// currentUserID: 当前登录用户 ID，用于身份校验（防越权代人审批）；<=0 表示跳过校验（受信任的内部调用如 CAB 已单独校验成员身份）
func (s *ChangeApprovalService) UpdateChangeApproval(ctx context.Context, approvalID int, req *dto.UpdateChangeApprovalRequest, tenantID int, currentUserID int) (*dto.ChangeApprovalResponse, error) {
	// 先读原审批记录，校验 approver_id == currentUserID（防越权代人审批）
	if currentUserID > 0 {
		var recordApprover int
		var recordStatus string
		if scanErr := s.rawDB.QueryRowContext(ctx,
			`SELECT approver_id, status FROM change_approvals WHERE id = $1 AND tenant_id = $2`,
			approvalID, tenantID,
		).Scan(&recordApprover, &recordStatus); scanErr != nil {
			s.logger.Warnw("approval not found or cross-tenant", "approval_id", approvalID, "tenant_id", tenantID, "error", scanErr)
			return nil, fmt.Errorf("审批记录不存在")
		}
		if recordApprover != currentUserID {
			s.logger.Warnw("Approver identity mismatch, deny", "approval_id", approvalID,
				"record_approver", recordApprover, "current_user", currentUserID, "tenant_id", tenantID)
			return nil, fmt.Errorf("无权代替他人审批（approval=%d）", approvalID)
		}
		// 已处理过的审批不允许再次覆盖
		if recordStatus != "" && recordStatus != string(dto.ChangeApprovalStatusPending) {
			return nil, fmt.Errorf("该审批已处于终态（%s），不允许重复处理", recordStatus)
		}
	}

	// 更新审批状态（tenant_id 过滤，防止跨租户修改）
	query := `
		UPDATE change_approvals 
		SET status = $1, comment = $2, approved_at = $3, updated_at = $4
		WHERE id = $5 AND tenant_id = $6
		RETURNING id, change_id, approver_id, status, comment, approved_at, created_at
	`

	var id, changeID, approverID int
	var status, comment string
	var approvedAt *time.Time
	var createdAt time.Time

	err := s.rawDB.QueryRowContext(
		ctx, query,
		string(req.Status),
		req.Comment,
		time.Now(),
		time.Now(),
		approvalID,
		tenantID,
	).Scan(&id, &changeID, &approverID, &status, &comment, &approvedAt, &createdAt)
	if err != nil {
		s.logger.Errorw("Failed to update change approval", "error", err, "approval_id", approvalID)
		return nil, fmt.Errorf("failed to update change approval: %w", err)
	}

	// 获取审批人信息（租户过滤）
	approver, err := s.getTenantUser(ctx, approverID, tenantID)
	if err != nil {
		s.logger.Warnw("Failed to get approver info", "error", err, "user_id", approverID)
	}

	// 将审批决定同步回审批链。
	// 修复：此前审批只写入 change_approvals 历史表，change_approval_chains 的状态从未被更新，
	// 导致 checkAndUpdateChangeStatus 永远统计不到已批准节点，变更状态无法流转到 approved。
	if req.Status == dto.ChangeApprovalStatusApproved || req.Status == dto.ChangeApprovalStatusRejected {
		chainStatus := "approved"
		if req.Status == dto.ChangeApprovalStatusRejected {
			chainStatus = "rejected"
		}
		if _, serr := s.rawDB.ExecContext(ctx,
			`UPDATE change_approval_chains SET status = $1 WHERE change_id = $2 AND approver_id = $3 AND tenant_id = $4 AND status = 'pending'`,
			chainStatus, changeID, approverID, tenantID,
		); serr != nil {
			s.logger.Warnw("Failed to sync approval chain status", "error", serr, "change_id", changeID, "approver_id", approverID)
		}
	}

	// 根据审批决定重新计算并更新变更状态（批准或驳回都需重算）
	if req.Status == dto.ChangeApprovalStatusApproved || req.Status == dto.ChangeApprovalStatusRejected {
		if err := s.checkAndUpdateChangeStatus(ctx, changeID, tenantID); err != nil {
			s.logger.Warnw("Failed to update change status", "error", err, "change_id", changeID)
		}
	}

	// 构建响应
	approverName := ""
	if approver != nil {
		approverName = approver.Name
	}
	response := &dto.ChangeApprovalResponse{
		ID:           id,
		ChangeID:     changeID,
		ApproverID:   approverID,
		ApproverName: approverName,
		Status:       dto.ChangeStatus(req.Status),
		Comment:      req.Comment,
		ApprovedAt:   approvedAt,
		CreatedAt:    createdAt,
	}

	s.logger.Infow("Change approval updated", "approval_id", id, "status", req.Status)
	return response, nil
}

// CreateChangeApprovalWorkflow 创建变更审批工作流
func (s *ChangeApprovalService) CreateChangeApprovalWorkflow(ctx context.Context, req *dto.ChangeApprovalWorkflowRequest, tenantID int) error {
	// 验证变更是否存在且属于当前租户（防止跨租户销毁他人审批链，C1 修复）
	changeEntity, err := s.client.Change.Get(ctx, req.ChangeID)
	if err != nil {
		return fmt.Errorf("change not found: %w", err)
	}
	if changeEntity.TenantID != tenantID {
		return fmt.Errorf("change does not belong to current tenant")
	}

	// 开始事务
	tx, err := s.rawDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 删除现有的审批链（tenant_id 兜底过滤，双重保险）
	_, err = tx.ExecContext(ctx, "DELETE FROM change_approval_chains WHERE change_id = $1 AND tenant_id = $2", req.ChangeID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete existing approval chains: %w", err)
	}

	// 创建新的审批链
	for _, item := range req.ApprovalChain {
		// 验证审批人是否存在且属于当前租户
		if _, err := s.getTenantUser(ctx, item.ApproverID, tenantID); err != nil {
			return fmt.Errorf("approver %d not found in tenant: %w", item.ApproverID, err)
		}

		query := `
			INSERT INTO change_approval_chains (change_id, tenant_id, level, approver_id, role, status, is_required, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`

		_, err = tx.ExecContext(
			ctx, query,
			req.ChangeID,
			tenantID,
			item.Level,
			item.ApproverID,
			item.Role,
			"pending",
			item.IsRequired,
			time.Now(),
		)
		if err != nil {
			return fmt.Errorf("failed to create approval chain item: %w", err)
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Infow("Change approval workflow created", "change_id", req.ChangeID, "levels", len(req.ApprovalChain))
	return nil
}

// GetChangeApprovalSummary 获取变更审批摘要
func (s *ChangeApprovalService) GetChangeApprovalSummary(ctx context.Context, changeID, tenantID int) (*dto.ChangeApprovalSummary, error) {
	// 验证变更是否存在且属于当前租户
	changeEntity, err := s.client.Change.Get(ctx, changeID)
	if err != nil {
		return nil, fmt.Errorf("change not found: %w", err)
	}
	if changeEntity.TenantID != tenantID {
		return nil, fmt.Errorf("change does not belong to current tenant")
	}

	// 获取审批链（tenant_id 过滤）
	chainQuery := `
		SELECT id, level, approver_id, role, status, is_required, created_at
		FROM change_approval_chains 
		WHERE change_id = $1 AND tenant_id = $2
		ORDER BY level
	`

	chainRows, err := s.rawDB.QueryContext(ctx, chainQuery, changeID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query approval chains: %w", err)
	}
	defer chainRows.Close()

	var currentLevel int
	var totalLevels int
	var pendingApprovals []dto.ChangeApprovalChainResponse

	for chainRows.Next() {
		var id, level, approverID int
		var role, status string
		var isRequired bool
		var createdAt time.Time

		err := chainRows.Scan(&id, &level, &approverID, &role, &status, &isRequired, &createdAt)
		if err != nil {
			continue
		}

		// 获取审批人姓名（租户过滤）
		approver, err := s.getTenantUser(ctx, approverID, tenantID)
		approverName := ""
		if err == nil && approver != nil {
			approverName = approver.Name
		}

		chainItem := dto.ChangeApprovalChainResponse{
			ID:           id,
			ChangeID:     changeID,
			Level:        level,
			ApproverID:   approverID,
			ApproverName: approverName,
			Role:         role,
			Status:       status,
			IsRequired:   isRequired,
			CreatedAt:    createdAt,
		}

		totalLevels = level

		if status == "pending" {
			pendingApprovals = append(pendingApprovals, chainItem)
			if currentLevel == 0 {
				currentLevel = level
			}
		}
	}

	// 获取审批历史（tenant_id 过滤）
	historyQuery := `
		SELECT id, change_id, approver_id, status, comment, approved_at, created_at
		FROM change_approvals 
		WHERE change_id = $1 AND tenant_id = $2
		ORDER BY created_at
	`

	historyRows, err := s.rawDB.QueryContext(ctx, historyQuery, changeID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query approval history: %w", err)
	}
	defer historyRows.Close()

	var approvalHistory []dto.ChangeApprovalResponse
	for historyRows.Next() {
		var id, changeID, approverID int
		var status, comment string
		var approvedAt *time.Time
		var createdAt time.Time

		err := historyRows.Scan(&id, &changeID, &approverID, &status, &comment, &approvedAt, &createdAt)
		if err != nil {
			continue
		}

		// 获取审批人姓名（租户过滤）
		approver, err := s.getTenantUser(ctx, approverID, tenantID)
		approverName := ""
		if err == nil && approver != nil {
			approverName = approver.Name
		}

		historyItem := dto.ChangeApprovalResponse{
			ID:           id,
			ChangeID:     changeID,
			ApproverID:   approverID,
			ApproverName: approverName,
			Status:       dto.ChangeStatus(status),
			Comment:      &comment,
			ApprovedAt:   approvedAt,
			CreatedAt:    createdAt,
		}

		approvalHistory = append(approvalHistory, historyItem)
	}

	// 确定审批状态
	approvalStatus := "pending"
	if len(pendingApprovals) == 0 {
		approvalStatus = "completed"
	} else if len(approvalHistory) > 0 {
		approvalStatus = "in_progress"
	}

	// 确定下一个审批人
	var nextApprover *string
	if len(pendingApprovals) > 0 {
		nextApprover = &pendingApprovals[0].ApproverName
	}

	summary := &dto.ChangeApprovalSummary{
		ChangeID:         changeID,
		CurrentLevel:     currentLevel,
		TotalLevels:      totalLevels,
		ApprovalStatus:   approvalStatus,
		NextApprover:     nextApprover,
		ApprovalHistory:  approvalHistory,
		PendingApprovals: pendingApprovals,
	}

	return summary, nil
}

// CreateChangeRiskAssessment 创建变更风险评估
func (s *ChangeApprovalService) CreateChangeRiskAssessment(ctx context.Context, req *dto.CreateChangeRiskAssessmentRequest, tenantID int) (*dto.ChangeRiskAssessmentResponse, error) {
	// 验证变更是否存在且属于当前租户
	changeEntity, err := s.client.Change.Get(ctx, req.ChangeID)
	if err != nil {
		return nil, fmt.Errorf("change not found: %w", err)
	}
	if changeEntity.TenantID != tenantID {
		return nil, fmt.Errorf("change does not belong to current tenant")
	}

	// 插入风险评估记录（含 tenant_id）
	query := `
		INSERT INTO change_risk_assessments (
			change_id, tenant_id, risk_level, risk_description, impact_analysis, 
			mitigation_measures, contingency_plan, risk_owner, risk_review_date,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at
	`

	var id int
	var createdAt time.Time
	err = s.rawDB.QueryRowContext(
		ctx, query,
		req.ChangeID,
		tenantID,
		string(req.RiskLevel),
		req.RiskDescription,
		req.ImpactAnalysis,
		req.MitigationMeasures,
		req.ContingencyPlan,
		req.RiskOwner,
		req.RiskReviewDate,
		time.Now(),
		time.Now(),
	).Scan(&id, &createdAt)
	if err != nil {
		s.logger.Errorw("Failed to create risk assessment", "error", err, "change_id", req.ChangeID)
		return nil, fmt.Errorf("failed to create risk assessment: %w", err)
	}

	// 构建响应
	response := &dto.ChangeRiskAssessmentResponse{
		ID:                 id,
		ChangeID:           req.ChangeID,
		RiskLevel:          req.RiskLevel,
		RiskDescription:    req.RiskDescription,
		ImpactAnalysis:     req.ImpactAnalysis,
		MitigationMeasures: req.MitigationMeasures,
		ContingencyPlan:    req.ContingencyPlan,
		RiskOwner:          req.RiskOwner,
		RiskReviewDate:     req.RiskReviewDate,
		CreatedAt:          createdAt,
		UpdatedAt:          createdAt,
	}

	s.logger.Infow("Risk assessment created", "assessment_id", id, "change_id", req.ChangeID)
	return response, nil
}

// GetChangeRiskAssessment 获取变更风险评估
func (s *ChangeApprovalService) GetChangeRiskAssessment(ctx context.Context, changeID, tenantID int) (*dto.ChangeRiskAssessmentResponse, error) {
	// 验证变更是否存在且属于当前租户
	changeEntity, err := s.client.Change.Get(ctx, changeID)
	if err != nil {
		return nil, fmt.Errorf("change not found: %w", err)
	}
	if changeEntity.TenantID != tenantID {
		return nil, fmt.Errorf("change does not belong to current tenant")
	}

	// 查询风险评估（tenant_id 过滤）
	query := `
		SELECT id, change_id, risk_level, risk_description, impact_analysis,
		       mitigation_measures, contingency_plan, risk_owner, risk_review_date,
		       created_at, updated_at
		FROM change_risk_assessments 
		WHERE change_id = $1 AND tenant_id = $2
	`

	var id int
	var riskLevel, riskDescription, impactAnalysis, mitigationMeasures, contingencyPlan, riskOwner string
	var riskReviewDate *time.Time
	var createdAt, updatedAt time.Time

	err = s.rawDB.QueryRowContext(ctx, query, changeID, tenantID).Scan(
		&id, &changeID, &riskLevel, &riskDescription, &impactAnalysis,
		&mitigationMeasures, &contingencyPlan, &riskOwner, &riskReviewDate,
		&createdAt, &updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("risk assessment not found")
		}
		return nil, fmt.Errorf("failed to get risk assessment: %w", err)
	}

	// 构建响应
	response := &dto.ChangeRiskAssessmentResponse{
		ID:                 id,
		ChangeID:           changeID,
		RiskLevel:          dto.ChangeRisk(riskLevel),
		RiskDescription:    riskDescription,
		ImpactAnalysis:     impactAnalysis,
		MitigationMeasures: mitigationMeasures,
		ContingencyPlan:    contingencyPlan,
		RiskOwner:          riskOwner,
		RiskReviewDate:     riskReviewDate,
		CreatedAt:          createdAt,
		UpdatedAt:          updatedAt,
	}

	return response, nil
}

// CreateChangeImplementationPlan 创建变更实施计划
func (s *ChangeApprovalService) CreateChangeImplementationPlan(ctx context.Context, req *dto.CreateChangeImplementationPlanRequest, tenantID int) (*dto.ChangeImplementationPlanResponse, error) {
	// 验证变更是否存在且属于当前租户
	changeEntity, err := s.client.Change.Get(ctx, req.ChangeID)
	if err != nil {
		return nil, fmt.Errorf("change not found: %w", err)
	}
	if changeEntity.TenantID != tenantID {
		return nil, fmt.Errorf("change does not belong to current tenant")
	}

	// 序列化任务、前置条件和依赖关系
	tasksJSON := SafeMarshal(req.Tasks)
	prerequisitesJSON := SafeMarshal(req.Prerequisites)
	dependenciesJSON := SafeMarshal(req.Dependencies)

	// 插入实施计划（写入 tenant_id 强化多租户隔离）
	query := `
		INSERT INTO change_implementation_plans (
			change_id, tenant_id, phase, description, tasks, responsible, start_date, end_date,
			prerequisites, dependencies, success_criteria, status, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, created_at
	`

	var id int
	var createdAt time.Time
	err = s.rawDB.QueryRowContext(
		ctx, query,
		req.ChangeID,
		tenantID,
		req.Phase,
		req.Description,
		tasksJSON,
		req.Responsible,
		req.StartDate,
		req.EndDate,
		prerequisitesJSON,
		dependenciesJSON,
		req.SuccessCriteria,
		"pending",
		time.Now(),
		time.Now(),
	).Scan(&id, &createdAt)
	if err != nil {
		s.logger.Errorw("Failed to create implementation plan", "error", err, "change_id", req.ChangeID)
		return nil, fmt.Errorf("failed to create implementation plan: %w", err)
	}

	// 构建响应
	response := &dto.ChangeImplementationPlanResponse{
		ID:              id,
		ChangeID:        req.ChangeID,
		Phase:           req.Phase,
		Description:     req.Description,
		Tasks:           req.Tasks,
		Responsible:     req.Responsible,
		StartDate:       req.StartDate,
		EndDate:         req.EndDate,
		Prerequisites:   req.Prerequisites,
		Dependencies:    req.Dependencies,
		SuccessCriteria: req.SuccessCriteria,
		Status:          "pending",
		CreatedAt:       createdAt,
		UpdatedAt:       createdAt,
	}

	s.logger.Infow("Implementation plan created", "plan_id", id, "change_id", req.ChangeID)
	return response, nil
}

// CreateChangeRollbackPlan 创建变更回滚计划
func (s *ChangeApprovalService) CreateChangeRollbackPlan(ctx context.Context, req *dto.CreateChangeRollbackPlanRequest, tenantID int) (*dto.ChangeRollbackPlanResponse, error) {
	// 验证变更是否存在且属于当前租户
	changeEntity, err := s.client.Change.Get(ctx, req.ChangeID)
	if err != nil {
		return nil, fmt.Errorf("change not found: %w", err)
	}
	if changeEntity.TenantID != tenantID {
		return nil, fmt.Errorf("change does not belong to current tenant")
	}

	// 序列化触发条件和回滚步骤
	triggerConditionsJSON := SafeMarshal(req.TriggerConditions)
	rollbackStepsJSON := SafeMarshal(req.RollbackSteps)

	// 插入回滚计划（含 tenant_id 保证多租户隔离）
	query := `
		INSERT INTO change_rollback_plans (
			change_id, tenant_id, trigger_conditions, rollback_steps, responsible, estimated_time,
			communication_plan, test_plan, approval_required, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at
	`

	var id int
	var createdAt time.Time
	err = s.rawDB.QueryRowContext(
		ctx, query,
		req.ChangeID,
		tenantID,
		triggerConditionsJSON,
		rollbackStepsJSON,
		req.Responsible,
		req.EstimatedTime,
		req.CommunicationPlan,
		req.TestPlan,
		req.ApprovalRequired,
		time.Now(),
		time.Now(),
	).Scan(&id, &createdAt)
	if err != nil {
		s.logger.Errorw("Failed to create rollback plan", "error", err, "change_id", req.ChangeID)
		return nil, fmt.Errorf("failed to create rollback plan: %w", err)
	}

	// 构建响应
	response := &dto.ChangeRollbackPlanResponse{
		ID:                id,
		ChangeID:          req.ChangeID,
		TriggerConditions: req.TriggerConditions,
		RollbackSteps:     req.RollbackSteps,
		Responsible:       req.Responsible,
		EstimatedTime:     req.EstimatedTime,
		CommunicationPlan: req.CommunicationPlan,
		TestPlan:          req.TestPlan,
		ApprovalRequired:  req.ApprovalRequired,
		CreatedAt:         createdAt,
		UpdatedAt:         createdAt,
	}

	s.logger.Infow("Rollback plan created", "plan_id", id, "change_id", req.ChangeID)
	return response, nil
}

// ExecuteChangeRollback 执行变更回滚
func (s *ChangeApprovalService) ExecuteChangeRollback(ctx context.Context, changeID, rollbackPlanID, initiatedBy int, triggerReason string, tenantID int) (*dto.ChangeRollbackExecutionResponse, error) {
	// 验证变更是否存在且属于当前租户
	changeEntity, err := s.client.Change.Get(ctx, changeID)
	if err != nil {
		return nil, fmt.Errorf("change not found: %w", err)
	}
	if changeEntity.TenantID != tenantID {
		return nil, fmt.Errorf("change does not belong to current tenant")
	}

	// 验证回滚计划是否存在（严格校验 tenant_id，防跨租户）
	var planExists bool
	err = s.rawDB.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM change_rollback_plans WHERE id = $1 AND tenant_id = $2)",
		rollbackPlanID, tenantID,
	).Scan(&planExists)
	if err != nil || !planExists {
		return nil, fmt.Errorf("rollback plan not found")
	}

	// 获取发起人姓名（租户过滤）
	initiator, err := s.getTenantUser(ctx, initiatedBy, tenantID)
	initiatorName := ""
	if err == nil && initiator != nil {
		initiatorName = initiator.Name
	}

	// 插入回滚执行记录（写入 tenant_id 保持一致性）
	query := `
		INSERT INTO change_rollback_executions (
			change_id, tenant_id, rollback_plan_id, trigger_reason, initiated_by, status,
			start_time, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at
	`

	var id int
	var createdAt time.Time
	err = s.rawDB.QueryRowContext(
		ctx, query,
		changeID,
		tenantID,
		rollbackPlanID,
		triggerReason,
		initiatedBy,
		"initiated",
		time.Now(),
		time.Now(),
		time.Now(),
	).Scan(&id, &createdAt)
	if err != nil {
		s.logger.Errorw("Failed to create rollback execution", "error", err, "change_id", changeID)
		return nil, fmt.Errorf("failed to create rollback execution: %w", err)
	}

	// 构建响应
	response := &dto.ChangeRollbackExecutionResponse{
		ID:              id,
		ChangeID:        changeID,
		RollbackPlanID:  rollbackPlanID,
		TriggerReason:   triggerReason,
		InitiatedBy:     initiatedBy,
		InitiatedByName: initiatorName,
		Status:          "initiated",
		StartTime:       &createdAt,
		CreatedAt:       createdAt,
		UpdatedAt:       createdAt,
	}

	s.logger.Infow("Rollback execution initiated", "execution_id", id, "change_id", changeID)
	return response, nil
}

// checkAndUpdateChangeStatus 根据审批链重新计算并更新变更状态。
// 规则：
//   - 任一「必需」审批人驳回 -> 变更整体驳回(rejected)
//   - 全部「必需」审批人都已批准 -> 变更批准(approved)
//   - 否则保持现状（仍有待审批节点）
func (s *ChangeApprovalService) checkAndUpdateChangeStatus(ctx context.Context, changeID, tenantID int) error {
	query := `
		SELECT COUNT(*) as total_required,
		       COUNT(CASE WHEN status = 'approved' THEN 1 END) as approved_count,
		       COUNT(CASE WHEN status = 'rejected' THEN 1 END) as rejected_count
		FROM change_approval_chains
		WHERE change_id = $1 AND tenant_id = $2 AND is_required = true
	`

	var totalRequired, approvedCount, rejectedCount int
	err := s.rawDB.QueryRowContext(ctx, query, changeID, tenantID).Scan(&totalRequired, &approvedCount, &rejectedCount)
	if err != nil {
		return fmt.Errorf("failed to check approval status: %w", err)
	}

	// 任一必需审批人驳回，整体驳回（tenant 过滤）
	if rejectedCount > 0 {
		_, err = s.client.Change.Update().
			Where(change.ID(changeID), change.TenantID(tenantID)).
			SetStatus(string(dto.ChangeStatusRejected)).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed to update change status: %w", err)
		}
		s.logger.Infow("Change status updated to rejected", "change_id", changeID)
		return nil
	}

	// 所有必需审批人都已批准，更新变更状态为已批准（tenant 过滤）
	if totalRequired > 0 && totalRequired == approvedCount {
		_, err = s.client.Change.Update().
			Where(change.ID(changeID), change.TenantID(tenantID)).
			SetStatus(string(dto.ChangeStatusApproved)).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed to update change status: %w", err)
		}

		s.logger.Infow("Change status updated to approved", "change_id", changeID)
		return nil
	}

	// 兜底：审批链全部为「非必需」时（totalRequired=0），只要有任一节点批准且无节点驳回，
	// 视为整体批准，避免 change 永远停留在 pending_approval。
	if totalRequired == 0 {
		var totalAll, approvedAll int
		err = s.rawDB.QueryRowContext(ctx, `
			SELECT COUNT(*) AS total_all,
			       COUNT(CASE WHEN status = 'approved' THEN 1 END) AS approved_all
			FROM change_approval_chains
			WHERE change_id = $1 AND tenant_id = $2
		`, changeID, tenantID).Scan(&totalAll, &approvedAll)
		if err != nil {
			return fmt.Errorf("failed to fallback check approval status: %w", err)
		}
		if totalAll > 0 && approvedAll > 0 {
			if _, err = s.client.Change.Update().
				Where(change.ID(changeID), change.TenantID(tenantID)).
				SetStatus(string(dto.ChangeStatusApproved)).
				Save(ctx); err != nil {
				return fmt.Errorf("failed to update change status (fallback): %w", err)
			}
			s.logger.Infow("Change status updated to approved (all-optional fallback)", "change_id", changeID)
		}
	}

	return nil
}
