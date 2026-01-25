package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/change"

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

// CreateChangeApproval 创建变更审批记录
func (s *ChangeApprovalService) CreateChangeApproval(ctx context.Context, req *dto.CreateChangeApprovalRequest, tenantID int) (*dto.ChangeApprovalResponse, error) {
	// 验证变更是否存在
	_, err := s.client.Change.Get(ctx, req.ChangeID)
	if err != nil {
		return nil, fmt.Errorf("change not found: %w", err)
	}

	// 验证审批人是否存在
	approver, err := s.client.User.Get(ctx, req.ApproverID)
	if err != nil {
		return nil, fmt.Errorf("approver not found: %w", err)
	}

	// 插入审批记录
	query := `
		INSERT INTO change_approvals (change_id, approver_id, status, comment, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`
	
	var id int
	var createdAt time.Time
	err = s.rawDB.QueryRowContext(ctx, query,
		req.ChangeID,
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
func (s *ChangeApprovalService) UpdateChangeApproval(ctx context.Context, approvalID int, req *dto.UpdateChangeApprovalRequest, tenantID int) (*dto.ChangeApprovalResponse, error) {
	// 更新审批状态
	query := `
		UPDATE change_approvals 
		SET status = $1, comment = $2, approved_at = $3, updated_at = $4
		WHERE id = $5
		RETURNING id, change_id, approver_id, status, comment, approved_at, created_at
	`
	
	var id, changeID, approverID int
	var status, comment string
	var approvedAt *time.Time
	var createdAt time.Time
	
	err := s.rawDB.QueryRowContext(ctx, query,
		string(req.Status),
		req.Comment,
		time.Now(),
		time.Now(),
		approvalID,
	).Scan(&id, &changeID, &approverID, &status, &comment, &approvedAt, &createdAt)

	if err != nil {
		s.logger.Errorw("Failed to update change approval", "error", err, "approval_id", approvalID)
		return nil, fmt.Errorf("failed to update change approval: %w", err)
	}

	// 获取审批人信息
	approver, err := s.client.User.Get(ctx, approverID)
	if err != nil {
		s.logger.Warnw("Failed to get approver info", "error", err, "user_id", approverID)
	}

	// 如果审批通过，更新变更状态
	if req.Status == dto.ChangeStatusApproved {
		// 检查是否所有必需审批都已完成
		if err := s.checkAndUpdateChangeStatus(ctx, changeID, tenantID); err != nil {
			s.logger.Warnw("Failed to update change status", "error", err, "change_id", changeID)
		}
	}

	// 构建响应
	response := &dto.ChangeApprovalResponse{
		ID:           id,
		ChangeID:     changeID,
		ApproverID:   approverID,
		ApproverName: approver.Name,
		Status:       req.Status,
		Comment:      req.Comment,
		ApprovedAt:   approvedAt,
		CreatedAt:    createdAt,
	}

	s.logger.Infow("Change approval updated", "approval_id", id, "status", req.Status)
	return response, nil
}

// CreateChangeApprovalWorkflow 创建变更审批工作流
func (s *ChangeApprovalService) CreateChangeApprovalWorkflow(ctx context.Context, req *dto.ChangeApprovalWorkflowRequest, tenantID int) error {
	// 验证变更是否存在
	_, err := s.client.Change.Get(ctx, req.ChangeID)
	if err != nil {
		return fmt.Errorf("change not found: %w", err)
	}

	// 开始事务
	tx, err := s.rawDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 删除现有的审批链
	_, err = tx.ExecContext(ctx, "DELETE FROM change_approval_chains WHERE change_id = $1", req.ChangeID)
	if err != nil {
		return fmt.Errorf("failed to delete existing approval chains: %w", err)
	}

	// 创建新的审批链
	for _, item := range req.ApprovalChain {
		// 验证审批人是否存在
		_, err := s.client.User.Get(ctx, item.ApproverID)
		if err != nil {
			return fmt.Errorf("approver not found: %w", err)
		}

		query := `
			INSERT INTO change_approval_chains (change_id, level, approver_id, role, status, is_required, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`
		
		_, err = tx.ExecContext(ctx, query,
			req.ChangeID,
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
	// 验证变更是否存在
	_, err := s.client.Change.Get(ctx, changeID)
	if err != nil {
		return nil, fmt.Errorf("change not found: %w", err)
	}

	// 获取审批链
	chainQuery := `
		SELECT id, level, approver_id, role, status, is_required, created_at
		FROM change_approval_chains 
		WHERE change_id = $1 
		ORDER BY level
	`
	
	chainRows, err := s.rawDB.QueryContext(ctx, chainQuery, changeID)
	if err != nil {
		return nil, fmt.Errorf("failed to query approval chains: %w", err)
	}
	defer chainRows.Close()

	var approvalChains []dto.ChangeApprovalChainResponse
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

		// 获取审批人姓名
		approver, err := s.client.User.Get(ctx, approverID)
		approverName := ""
		if err == nil {
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

		approvalChains = append(approvalChains, chainItem)
		totalLevels = level

		if status == "pending" {
			pendingApprovals = append(pendingApprovals, chainItem)
			if currentLevel == 0 {
				currentLevel = level
			}
		}
	}

	// 获取审批历史
	historyQuery := `
		SELECT id, change_id, approver_id, status, comment, approved_at, created_at
		FROM change_approvals 
		WHERE change_id = $1 
		ORDER BY created_at
	`
	
	historyRows, err := s.rawDB.QueryContext(ctx, historyQuery, changeID)
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

		// 获取审批人姓名
		approver, err := s.client.User.Get(ctx, approverID)
		approverName := ""
		if err == nil {
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
		ChangeID:          changeID,
		CurrentLevel:      currentLevel,
		TotalLevels:       totalLevels,
		ApprovalStatus:    approvalStatus,
		NextApprover:      nextApprover,
		ApprovalHistory:   approvalHistory,
		PendingApprovals:  pendingApprovals,
	}

	return summary, nil
}

// CreateChangeRiskAssessment 创建变更风险评估
func (s *ChangeApprovalService) CreateChangeRiskAssessment(ctx context.Context, req *dto.CreateChangeRiskAssessmentRequest, tenantID int) (*dto.ChangeRiskAssessmentResponse, error) {
	// 验证变更是否存在
	_, err := s.client.Change.Get(ctx, req.ChangeID)
	if err != nil {
		return nil, fmt.Errorf("change not found: %w", err)
	}

	// 插入风险评估记录
	query := `
		INSERT INTO change_risk_assessments (
			change_id, risk_level, risk_description, impact_analysis, 
			mitigation_measures, contingency_plan, risk_owner, risk_review_date,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at
	`
	
	var id int
	var createdAt time.Time
	err = s.rawDB.QueryRowContext(ctx, query,
		req.ChangeID,
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
	// 验证变更是否存在
	_, err := s.client.Change.Get(ctx, changeID)
	if err != nil {
		return nil, fmt.Errorf("change not found: %w", err)
	}

	// 查询风险评估
	query := `
		SELECT id, change_id, risk_level, risk_description, impact_analysis,
		       mitigation_measures, contingency_plan, risk_owner, risk_review_date,
		       created_at, updated_at
		FROM change_risk_assessments 
		WHERE change_id = $1
	`
	
	var id int
	var riskLevel, riskDescription, impactAnalysis, mitigationMeasures, contingencyPlan, riskOwner string
	var riskReviewDate *time.Time
	var createdAt, updatedAt time.Time

	err = s.rawDB.QueryRowContext(ctx, query, changeID).Scan(
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
	// 验证变更是否存在
	_, err := s.client.Change.Get(ctx, req.ChangeID)
	if err != nil {
		return nil, fmt.Errorf("change not found: %w", err)
	}

	// 序列化任务、前置条件和依赖关系
	tasksJSON := SafeMarshal(req.Tasks)
	prerequisitesJSON := SafeMarshal(req.Prerequisites)
	dependenciesJSON := SafeMarshal(req.Dependencies)

	// 插入实施计划
	query := `
		INSERT INTO change_implementation_plans (
			change_id, phase, description, tasks, responsible, start_date, end_date,
			prerequisites, dependencies, success_criteria, status, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, created_at
	`
	
	var id int
	var createdAt time.Time
	err = s.rawDB.QueryRowContext(ctx, query,
		req.ChangeID,
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
	// 验证变更是否存在
	_, err := s.client.Change.Get(ctx, req.ChangeID)
	if err != nil {
		return nil, fmt.Errorf("change not found: %w", err)
	}

	// 序列化触发条件和回滚步骤
	triggerConditionsJSON := SafeMarshal(req.TriggerConditions)
	rollbackStepsJSON := SafeMarshal(req.RollbackSteps)

	// 插入回滚计划
	query := `
		INSERT INTO change_rollback_plans (
			change_id, trigger_conditions, rollback_steps, responsible, estimated_time,
			communication_plan, test_plan, approval_required, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at
	`
	
	var id int
	var createdAt time.Time
	err = s.rawDB.QueryRowContext(ctx, query,
		req.ChangeID,
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
	// 验证变更是否存在
	_, err := s.client.Change.Get(ctx, changeID)
	if err != nil {
		return nil, fmt.Errorf("change not found: %w", err)
	}

	// 验证回滚计划是否存在
	var planExists bool
	err = s.rawDB.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM change_rollback_plans WHERE id = $1)", rollbackPlanID).Scan(&planExists)
	if err != nil || !planExists {
		return nil, fmt.Errorf("rollback plan not found")
	}

	// 获取发起人姓名
	initiator, err := s.client.User.Get(ctx, initiatedBy)
	initiatorName := ""
	if err == nil {
		initiatorName = initiator.Name
	}

	// 插入回滚执行记录
	query := `
		INSERT INTO change_rollback_executions (
			change_id, rollback_plan_id, trigger_reason, initiated_by, status,
			start_time, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`
	
	var id int
	var createdAt time.Time
	err = s.rawDB.QueryRowContext(ctx, query,
		changeID,
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

// checkAndUpdateChangeStatus 检查并更新变更状态
func (s *ChangeApprovalService) checkAndUpdateChangeStatus(ctx context.Context, changeID, tenantID int) error {
	// 检查是否所有必需审批都已完成
	query := `
		SELECT COUNT(*) as total_required, 
		       COUNT(CASE WHEN status = 'approved' THEN 1 END) as approved_count
		FROM change_approval_chains 
		WHERE change_id = $1 AND is_required = true
	`
	
	var totalRequired, approvedCount int
	err := s.rawDB.QueryRowContext(ctx, query, changeID).Scan(&totalRequired, &approvedCount)
	if err != nil {
		return fmt.Errorf("failed to check approval status: %w", err)
	}

	// 如果所有必需审批都已完成，更新变更状态为已批准
	if totalRequired > 0 && totalRequired == approvedCount {
		_, err = s.client.Change.Update().
			Where(change.ID(changeID)).
			SetStatus(string(dto.ChangeStatusApproved)).
			Save(ctx)
		
		if err != nil {
			return fmt.Errorf("failed to update change status: %w", err)
		}

		s.logger.Infow("Change status updated to approved", "change_id", changeID)
	}

	return nil
}
