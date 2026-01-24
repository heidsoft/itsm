package service

import (
	"context"
	"database/sql"
	"fmt"
	"itsm-backend/dto"
	"time"

	"go.uber.org/zap"
)

// ProblemInvestigationService 问题调查服务
type ProblemInvestigationService struct {
	db     *sql.DB
	logger *zap.SugaredLogger
}

// NewProblemInvestigationService 创建问题调查服务
func NewProblemInvestigationService(db *sql.DB, logger *zap.SugaredLogger) *ProblemInvestigationService {
	return &ProblemInvestigationService{
		db:     db,
		logger: logger,
	}
}

// GetRootCauseAnalysis 获取根本原因分析
func (s *ProblemInvestigationService) GetRootCauseAnalysis(ctx context.Context, id int, tenantID int) (*dto.RootCauseAnalysisResponse, error) {
	var analysis dto.RootCauseAnalysisResponse
	err := s.db.QueryRowContext(ctx, `
		SELECT prca.id, prca.problem_id, prca.analyst_id, u1.name, prca.analysis_method,
		       prca.root_cause_description, prca.contributing_factors, prca.evidence, prca.confidence_level,
		       prca.analysis_date, prca.reviewed_by, u2.name, prca.review_date,
		       prca.created_at, prca.updated_at
		FROM problem_root_cause_analyses prca
		JOIN users u1 ON prca.analyst_id = u1.id
		LEFT JOIN users u2 ON prca.reviewed_by = u2.id
		WHERE prca.id = $1
	`, id).Scan(
		&analysis.ID, &analysis.ProblemID, &analysis.AnalystID, &analysis.AnalystName,
		&analysis.AnalysisMethod, &analysis.RootCauseDescription, &analysis.ContributingFactors,
		&analysis.Evidence, &analysis.ConfidenceLevel, &analysis.AnalysisDate,
		&analysis.ReviewedBy, &analysis.ReviewedByName, &analysis.ReviewDate,
		&analysis.CreatedAt, &analysis.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("根因分析不存在")
		}
		return nil, fmt.Errorf("查询根因分析失败: %v", err)
	}
	return &analysis, nil
}

// GetProblemSolution 获取解决方案
func (s *ProblemInvestigationService) GetProblemSolution(ctx context.Context, id int, tenantID int) (*dto.ProblemSolutionResponse, error) {
	var solution dto.ProblemSolutionResponse
	err := s.db.QueryRowContext(ctx, `
		SELECT ps.id, ps.problem_id, ps.solution_type, ps.solution_description, ps.proposed_by, u1.name,
		       ps.proposed_date, ps.status, ps.priority, ps.estimated_effort_hours, ps.estimated_cost,
		       ps.risk_assessment, ps.approval_status, ps.approved_by, u2.name, ps.approval_date,
		       ps.created_at, ps.updated_at
		FROM problem_solutions ps
		JOIN users u1 ON ps.proposed_by = u1.id
		LEFT JOIN users u2 ON ps.approved_by = u2.id
		WHERE ps.id = $1
	`, id).Scan(
		&solution.ID, &solution.ProblemID, &solution.SolutionType, &solution.SolutionDescription,
		&solution.ProposedBy, &solution.ProposedByName, &solution.ProposedDate, &solution.Status,
		&solution.Priority, &solution.EstimatedEffortHours, &solution.EstimatedCost,
		&solution.RiskAssessment, &solution.ApprovalStatus, &solution.ApprovedBy, &solution.ApprovedByName,
		&solution.ApprovalDate, &solution.CreatedAt, &solution.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("解决方案不存在")
		}
		return nil, fmt.Errorf("查询解决方案失败: %v", err)
	}
	return &solution, nil
}

// CreateProblemInvestigation 创建问题调查
func (s *ProblemInvestigationService) CreateProblemInvestigation(ctx context.Context, req *dto.CreateProblemInvestigationRequest, tenantID int) (*dto.ProblemInvestigationResponse, error) {
	// 检查问题是否存在
	var problemTitle string
	err := s.db.QueryRowContext(ctx, "SELECT title FROM problems WHERE id = $1 AND tenant_id = $2", req.ProblemID, tenantID).Scan(&problemTitle)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("问题不存在")
		}
		return nil, fmt.Errorf("查询问题失败: %v", err)
	}

	// 检查是否已存在调查记录
	var existingID int
	err = s.db.QueryRowContext(ctx, "SELECT id FROM problem_investigations WHERE problem_id = $1", req.ProblemID).Scan(&existingID)
	if err == nil {
		return nil, fmt.Errorf("该问题已存在调查记录")
	}

	// 创建调查记录
	var investigationID int
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO problem_investigations (problem_id, investigator_id, estimated_completion_date, investigation_summary, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		RETURNING id
	`, req.ProblemID, req.InvestigatorID, req.EstimatedCompletionDate, req.InvestigationSummary, time.Now()).Scan(&investigationID)
	if err != nil {
		return nil, fmt.Errorf("创建问题调查失败: %v", err)
	}

	// 获取调查者姓名
	var investigatorName string
	err = s.db.QueryRowContext(ctx, "SELECT name FROM users WHERE id = $1", req.InvestigatorID).Scan(&investigatorName)
	if err != nil {
		investigatorName = "未知用户"
	}

	// 更新问题状态为"调查中"
	_, err = s.db.ExecContext(ctx, "UPDATE problems SET status = 'in_progress' WHERE id = $1", req.ProblemID)
	if err != nil {
		s.logger.Warnw("Failed to update problem status", "problem_id", req.ProblemID, "error", err)
	}

	return &dto.ProblemInvestigationResponse{
		ID:                      investigationID,
		ProblemID:               req.ProblemID,
		InvestigatorID:          req.InvestigatorID,
		InvestigatorName:        investigatorName,
		Status:                  dto.InvestigationStatusInProgress,
		StartDate:               time.Now(),
		EstimatedCompletionDate: req.EstimatedCompletionDate,
		InvestigationSummary:    &req.InvestigationSummary,
		CreatedAt:               time.Now(),
		UpdatedAt:               time.Now(),
	}, nil
}

// GetProblemInvestigation 获取问题调查详情
func (s *ProblemInvestigationService) GetProblemInvestigation(ctx context.Context, investigationID, tenantID int) (*dto.ProblemInvestigationResponse, error) {
	var investigation dto.ProblemInvestigationResponse
	err := s.db.QueryRowContext(ctx, `
		SELECT pi.id, pi.problem_id, pi.investigator_id, u.name, pi.status, pi.start_date, 
		       pi.estimated_completion_date, pi.actual_completion_date, pi.investigation_summary,
		       pi.created_at, pi.updated_at
		FROM problem_investigations pi
		JOIN users u ON pi.investigator_id = u.id
		JOIN problems p ON pi.problem_id = p.id
		WHERE pi.id = $1 AND p.tenant_id = $2
	`, investigationID, tenantID).Scan(
		&investigation.ID, &investigation.ProblemID, &investigation.InvestigatorID, &investigation.InvestigatorName,
		&investigation.Status, &investigation.StartDate, &investigation.EstimatedCompletionDate,
		&investigation.ActualCompletionDate, &investigation.InvestigationSummary,
		&investigation.CreatedAt, &investigation.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("问题调查不存在")
		}
		return nil, fmt.Errorf("获取问题调查失败: %v", err)
	}

	return &investigation, nil
}

// UpdateProblemInvestigation 更新问题调查
func (s *ProblemInvestigationService) UpdateProblemInvestigation(ctx context.Context, investigationID int, req *dto.UpdateProblemInvestigationRequest, tenantID int) (*dto.ProblemInvestigationResponse, error) {
	// 检查调查记录是否存在
	investigation, err := s.GetProblemInvestigation(ctx, investigationID, tenantID)
	if err != nil {
		return nil, err
	}

	// 构建更新查询
	query := "UPDATE problem_investigations SET updated_at = $1"
	args := []interface{}{time.Now()}
	argIndex := 2

	if req.Status != nil {
		query += fmt.Sprintf(", status = $%d", argIndex)
		args = append(args, *req.Status)
		argIndex++
	}
	if req.EstimatedCompletionDate != nil {
		query += fmt.Sprintf(", estimated_completion_date = $%d", argIndex)
		args = append(args, *req.EstimatedCompletionDate)
		argIndex++
	}
	if req.ActualCompletionDate != nil {
		query += fmt.Sprintf(", actual_completion_date = $%d", argIndex)
		args = append(args, *req.ActualCompletionDate)
		argIndex++
	}
	if req.InvestigationSummary != nil {
		query += fmt.Sprintf(", investigation_summary = $%d", argIndex)
		args = append(args, *req.InvestigationSummary)
		argIndex++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argIndex)
	args = append(args, investigationID)

	_, err = s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("更新问题调查失败: %v", err)
	}

	// 如果状态更新为完成，同时更新问题状态
	if req.Status != nil && *req.Status == dto.InvestigationStatusCompleted {
		_, err = s.db.ExecContext(ctx, "UPDATE problems SET status = 'resolved' WHERE id = $1", investigation.ProblemID)
		if err != nil {
			s.logger.Warnw("Failed to update problem status", "problem_id", investigation.ProblemID, "error", err)
		}
	}

	// 返回更新后的调查记录
	return s.GetProblemInvestigation(ctx, investigationID, tenantID)
}

// CreateInvestigationStep 创建调查步骤
func (s *ProblemInvestigationService) CreateInvestigationStep(ctx context.Context, req *dto.CreateInvestigationStepRequest, tenantID int) (*dto.InvestigationStepResponse, error) {
	// 检查调查记录是否存在
	var problemID int
	err := s.db.QueryRowContext(ctx, `
		SELECT pi.problem_id FROM problem_investigations pi
		JOIN problems p ON pi.problem_id = p.id
		WHERE pi.id = $1 AND p.tenant_id = $2
	`, req.InvestigationID, tenantID).Scan(&problemID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("调查记录不存在")
		}
		return nil, fmt.Errorf("查询调查记录失败: %v", err)
	}

	// 创建调查步骤
	var stepID int
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO problem_investigation_steps (investigation_id, step_number, step_title, step_description, assigned_to, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		RETURNING id
	`, req.InvestigationID, req.StepNumber, req.StepTitle, req.StepDescription, req.AssignedTo, req.Notes, time.Now()).Scan(&stepID)
	if err != nil {
		return nil, fmt.Errorf("创建调查步骤失败: %v", err)
	}

	// 获取分配人员姓名
	var assignedToName *string
	if req.AssignedTo != nil {
		var name string
		err = s.db.QueryRowContext(ctx, "SELECT name FROM users WHERE id = $1", *req.AssignedTo).Scan(&name)
		if err == nil {
			assignedToName = &name
		}
	}

	return &dto.InvestigationStepResponse{
		ID:              stepID,
		InvestigationID: req.InvestigationID,
		StepNumber:      req.StepNumber,
		StepTitle:       req.StepTitle,
		StepDescription: req.StepDescription,
		Status:          dto.StepStatusPending,
		AssignedTo:      req.AssignedTo,
		AssignedToName:  assignedToName,
		Notes:           &req.Notes,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}, nil
}

// UpdateInvestigationStep 更新调查步骤
func (s *ProblemInvestigationService) UpdateInvestigationStep(ctx context.Context, stepID int, req *dto.UpdateInvestigationStepRequest, tenantID int) (*dto.InvestigationStepResponse, error) {
	// 检查步骤是否存在
	var investigationID int
	err := s.db.QueryRowContext(ctx, `
		SELECT pis.investigation_id FROM problem_investigation_steps pis
		JOIN problem_investigations pi ON pis.investigation_id = pi.id
		JOIN problems p ON pi.problem_id = p.id
		WHERE pis.id = $1 AND p.tenant_id = $2
	`, stepID, tenantID).Scan(&investigationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("调查步骤不存在")
		}
		return nil, fmt.Errorf("查询调查步骤失败: %v", err)
	}

	// 构建更新查询
	query := "UPDATE problem_investigation_steps SET updated_at = $1"
	args := []interface{}{time.Now()}
	argIndex := 2

	if req.StepTitle != nil {
		query += fmt.Sprintf(", step_title = $%d", argIndex)
		args = append(args, *req.StepTitle)
		argIndex++
	}
	if req.StepDescription != nil {
		query += fmt.Sprintf(", step_description = $%d", argIndex)
		args = append(args, *req.StepDescription)
		argIndex++
	}
	if req.Status != nil {
		query += fmt.Sprintf(", status = $%d", argIndex)
		args = append(args, *req.Status)
		argIndex++
	}
	if req.AssignedTo != nil {
		query += fmt.Sprintf(", assigned_to = $%d", argIndex)
		args = append(args, *req.AssignedTo)
		argIndex++
	}
	if req.StartDate != nil {
		query += fmt.Sprintf(", start_date = $%d", argIndex)
		args = append(args, *req.StartDate)
		argIndex++
	}
	if req.CompletionDate != nil {
		query += fmt.Sprintf(", completion_date = $%d", argIndex)
		args = append(args, *req.CompletionDate)
		argIndex++
	}
	if req.Notes != nil {
		query += fmt.Sprintf(", notes = $%d", argIndex)
		args = append(args, *req.Notes)
		argIndex++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argIndex)
	args = append(args, stepID)

	_, err = s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("更新调查步骤失败: %v", err)
	}

	// 返回更新后的步骤
	return s.getInvestigationStep(ctx, stepID)
}

// getInvestigationStep 获取调查步骤详情
func (s *ProblemInvestigationService) getInvestigationStep(ctx context.Context, stepID int) (*dto.InvestigationStepResponse, error) {
	var step dto.InvestigationStepResponse
	err := s.db.QueryRowContext(ctx, `
		SELECT pis.id, pis.investigation_id, pis.step_number, pis.step_title, pis.step_description,
		       pis.status, pis.assigned_to, u.name, pis.start_date, pis.completion_date, pis.notes,
		       pis.created_at, pis.updated_at
		FROM problem_investigation_steps pis
		LEFT JOIN users u ON pis.assigned_to = u.id
		WHERE pis.id = $1
	`, stepID).Scan(
		&step.ID, &step.InvestigationID, &step.StepNumber, &step.StepTitle, &step.StepDescription,
		&step.Status, &step.AssignedTo, &step.AssignedToName, &step.StartDate, &step.CompletionDate, &step.Notes,
		&step.CreatedAt, &step.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("获取调查步骤失败: %v", err)
	}

	return &step, nil
}

// CreateRootCauseAnalysis 创建根本原因分析
func (s *ProblemInvestigationService) CreateRootCauseAnalysis(ctx context.Context, req *dto.CreateRootCauseAnalysisRequest, tenantID int) (*dto.RootCauseAnalysisResponse, error) {
	// 检查问题是否存在
	var problemTitle string
	err := s.db.QueryRowContext(ctx, "SELECT title FROM problems WHERE id = $1 AND tenant_id = $2", req.ProblemID, tenantID).Scan(&problemTitle)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("问题不存在")
		}
		return nil, fmt.Errorf("查询问题失败: %v", err)
	}

	// 检查是否已存在根本原因分析
	var existingID int
	err = s.db.QueryRowContext(ctx, "SELECT id FROM problem_root_cause_analyses WHERE problem_id = $1", req.ProblemID).Scan(&existingID)
	if err == nil {
		return nil, fmt.Errorf("该问题已存在根本原因分析")
	}

	// 创建根本原因分析
	var analysisID int
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO problem_root_cause_analyses (problem_id, analyst_id, analysis_method, root_cause_description, 
		                                       contributing_factors, evidence, confidence_level, analysis_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8, $8)
		RETURNING id
	`, req.ProblemID, req.AnalystID, req.AnalysisMethod, req.RootCauseDescription,
		req.ContributingFactors, req.Evidence, req.ConfidenceLevel, time.Now()).Scan(&analysisID)
	if err != nil {
		return nil, fmt.Errorf("创建根本原因分析失败: %v", err)
	}

	// 获取分析师姓名
	var analystName string
	err = s.db.QueryRowContext(ctx, "SELECT name FROM users WHERE id = $1", req.AnalystID).Scan(&analystName)
	if err != nil {
		analystName = "未知用户"
	}

	return &dto.RootCauseAnalysisResponse{
		ID:                   analysisID,
		ProblemID:            req.ProblemID,
		AnalystID:            req.AnalystID,
		AnalystName:          analystName,
		AnalysisMethod:       req.AnalysisMethod,
		RootCauseDescription: req.RootCauseDescription,
		ContributingFactors:  &req.ContributingFactors,
		Evidence:             &req.Evidence,
		ConfidenceLevel:      req.ConfidenceLevel,
		AnalysisDate:         time.Now(),
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}, nil
}

// CreateProblemSolution 创建问题解决方案
func (s *ProblemInvestigationService) CreateProblemSolution(ctx context.Context, req *dto.CreateProblemSolutionRequest, tenantID int) (*dto.ProblemSolutionResponse, error) {
	// 检查问题是否存在
	var problemTitle string
	err := s.db.QueryRowContext(ctx, "SELECT title FROM problems WHERE id = $1 AND tenant_id = $2", req.ProblemID, tenantID).Scan(&problemTitle)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("问题不存在")
		}
		return nil, fmt.Errorf("查询问题失败: %v", err)
	}

	// 创建解决方案
	var solutionID int
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO problem_solutions (problem_id, solution_type, solution_description, proposed_by, proposed_date,
		                             status, priority, estimated_effort_hours, estimated_cost, risk_assessment,
		                             approval_status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12)
		RETURNING id
	`, req.ProblemID, req.SolutionType, req.SolutionDescription, req.ProposedBy, time.Now(),
		dto.SolutionStatusProposed, req.Priority, req.EstimatedEffortHours, req.EstimatedCost, req.RiskAssessment,
		"pending", time.Now()).Scan(&solutionID)
	if err != nil {
		return nil, fmt.Errorf("创建问题解决方案失败: %v", err)
	}

	// 获取提议者姓名
	var proposedByName string
	err = s.db.QueryRowContext(ctx, "SELECT name FROM users WHERE id = $1", req.ProposedBy).Scan(&proposedByName)
	if err != nil {
		proposedByName = "未知用户"
	}

	return &dto.ProblemSolutionResponse{
		ID:                   solutionID,
		ProblemID:            req.ProblemID,
		SolutionType:         req.SolutionType,
		SolutionDescription:  req.SolutionDescription,
		ProposedBy:           req.ProposedBy,
		ProposedByName:       proposedByName,
		ProposedDate:         time.Now(),
		Status:               dto.SolutionStatusProposed,
		Priority:             req.Priority,
		EstimatedEffortHours: req.EstimatedEffortHours,
		EstimatedCost:        req.EstimatedCost,
		RiskAssessment:       &req.RiskAssessment,
		ApprovalStatus:       "pending",
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}, nil
}

// GetProblemInvestigationSummary 获取问题调查摘要
func (s *ProblemInvestigationService) GetProblemInvestigationSummary(ctx context.Context, problemID, tenantID int) (*dto.ProblemInvestigationSummaryResponse, error) {
	// 检查问题是否存在
	var problemTitle string
	err := s.db.QueryRowContext(ctx, "SELECT title FROM problems WHERE id = $1 AND tenant_id = $2", problemID, tenantID).Scan(&problemTitle)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("问题不存在")
		}
		return nil, fmt.Errorf("查询问题失败: %v", err)
	}

	summary := &dto.ProblemInvestigationSummaryResponse{}

	// 获取调查记录
	var investigationID int
	err = s.db.QueryRowContext(ctx, "SELECT id FROM problem_investigations WHERE problem_id = $1", problemID).Scan(&investigationID)
	if err == nil {
		investigation, err := s.GetProblemInvestigation(ctx, investigationID, tenantID)
		if err == nil {
			summary.Investigation = investigation
		}
	}

	// 获取调查步骤
	if summary.Investigation != nil {
		rows, err := s.db.QueryContext(ctx, `
			SELECT pis.id, pis.investigation_id, pis.step_number, pis.step_title, pis.step_description,
			       pis.status, pis.assigned_to, u.name, pis.start_date, pis.completion_date, pis.notes,
			       pis.created_at, pis.updated_at
			FROM problem_investigation_steps pis
			LEFT JOIN users u ON pis.assigned_to = u.id
			WHERE pis.investigation_id = $1
			ORDER BY pis.step_number
		`, investigationID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var step dto.InvestigationStepResponse
				err := rows.Scan(
					&step.ID, &step.InvestigationID, &step.StepNumber, &step.StepTitle, &step.StepDescription,
					&step.Status, &step.AssignedTo, &step.AssignedToName, &step.StartDate, &step.CompletionDate, &step.Notes,
					&step.CreatedAt, &step.UpdatedAt,
				)
				if err == nil {
					summary.Steps = append(summary.Steps, &step)
				}
			}
		}
	}

	// 获取根本原因分析
	var analysisID int
	err = s.db.QueryRowContext(ctx, "SELECT id FROM problem_root_cause_analyses WHERE problem_id = $1", problemID).Scan(&analysisID)
	if err == nil {
		rows, err := s.db.QueryContext(ctx, `
			SELECT prca.id, prca.problem_id, prca.analyst_id, u1.name, prca.analysis_method,
			       prca.root_cause_description, prca.contributing_factors, prca.evidence, prca.confidence_level,
			       prca.analysis_date, prca.reviewed_by, u2.name, prca.review_date,
			       prca.created_at, prca.updated_at
			FROM problem_root_cause_analyses prca
			JOIN users u1 ON prca.analyst_id = u1.id
			LEFT JOIN users u2 ON prca.reviewed_by = u2.id
			WHERE prca.problem_id = $1
		`, problemID)
		if err == nil {
			defer rows.Close()
			if rows.Next() {
				var analysis dto.RootCauseAnalysisResponse
				err := rows.Scan(
					&analysis.ID, &analysis.ProblemID, &analysis.AnalystID, &analysis.AnalystName,
					&analysis.AnalysisMethod, &analysis.RootCauseDescription, &analysis.ContributingFactors,
					&analysis.Evidence, &analysis.ConfidenceLevel, &analysis.AnalysisDate,
					&analysis.ReviewedBy, &analysis.ReviewedByName, &analysis.ReviewDate,
					&analysis.CreatedAt, &analysis.UpdatedAt,
				)
				if err == nil {
					summary.RootCauseAnalysis = &analysis
				}
			}
		}
	}

	// 获取解决方案
	rows, err := s.db.QueryContext(ctx, `
		SELECT ps.id, ps.problem_id, ps.solution_type, ps.solution_description, ps.proposed_by, u1.name,
		       ps.proposed_date, ps.status, ps.priority, ps.estimated_effort_hours, ps.estimated_cost,
		       ps.risk_assessment, ps.approval_status, ps.approved_by, u2.name, ps.approval_date,
		       ps.created_at, ps.updated_at
		FROM problem_solutions ps
		JOIN users u1 ON ps.proposed_by = u1.id
		LEFT JOIN users u2 ON ps.approved_by = u2.id
		WHERE ps.problem_id = $1
		ORDER BY ps.created_at DESC
	`, problemID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var solution dto.ProblemSolutionResponse
			err := rows.Scan(
				&solution.ID, &solution.ProblemID, &solution.SolutionType, &solution.SolutionDescription,
				&solution.ProposedBy, &solution.ProposedByName, &solution.ProposedDate, &solution.Status,
				&solution.Priority, &solution.EstimatedEffortHours, &solution.EstimatedCost,
				&solution.RiskAssessment, &solution.ApprovalStatus, &solution.ApprovedBy, &solution.ApprovedByName,
				&solution.ApprovalDate, &solution.CreatedAt, &solution.UpdatedAt,
			)
			if err == nil {
				summary.Solutions = append(summary.Solutions, &solution)
			}
		}
	}

	return summary, nil
}

// UpdateRootCauseAnalysis 更新根本原因分析
func (s *ProblemInvestigationService) UpdateRootCauseAnalysis(ctx context.Context, id int, req *dto.UpdateRootCauseAnalysisRequest, tenantID int) (*dto.RootCauseAnalysisResponse, error) {
	s.logger.Infow("Updating root cause analysis", "id", id, "tenant_id", tenantID)

	// 检查根因分析是否存在
	var existingAnalysis dto.RootCauseAnalysisResponse
	err := s.db.QueryRowContext(ctx, `
		SELECT prca.id, prca.problem_id, prca.analyst_id, u1.name, prca.analysis_method,
		       prca.root_cause_description, prca.contributing_factors, prca.evidence, prca.confidence_level,
		       prca.analysis_date, prca.reviewed_by, u2.name, prca.review_date,
		       prca.created_at, prca.updated_at
		FROM problem_root_cause_analyses prca
		JOIN users u1 ON prca.analyst_id = u1.id
		LEFT JOIN users u2 ON prca.reviewed_by = u2.id
		WHERE prca.id = $1
	`, id).Scan(
		&existingAnalysis.ID, &existingAnalysis.ProblemID, &existingAnalysis.AnalystID, &existingAnalysis.AnalystName,
		&existingAnalysis.AnalysisMethod, &existingAnalysis.RootCauseDescription, &existingAnalysis.ContributingFactors,
		&existingAnalysis.Evidence, &existingAnalysis.ConfidenceLevel, &existingAnalysis.AnalysisDate,
		&existingAnalysis.ReviewedBy, &existingAnalysis.ReviewedByName, &existingAnalysis.ReviewDate,
		&existingAnalysis.CreatedAt, &existingAnalysis.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("根因分析不存在")
		}
		return nil, fmt.Errorf("查询根因分析失败: %v", err)
	}

	// 构建更新语句
	updateSQL := "UPDATE problem_root_cause_analyses SET updated_at = $1"
	var params []interface{}
	paramIndex := 2

	if req.AnalysisMethod != nil {
		updateSQL += fmt.Sprintf(", analysis_method = $%d", paramIndex)
		params = append(params, *req.AnalysisMethod)
		paramIndex++
	}
	if req.RootCauseDescription != nil {
		updateSQL += fmt.Sprintf(", root_cause_description = $%d", paramIndex)
		params = append(params, *req.RootCauseDescription)
		paramIndex++
	}
	if req.ContributingFactors != nil {
		updateSQL += fmt.Sprintf(", contributing_factors = $%d", paramIndex)
		params = append(params, *req.ContributingFactors)
		paramIndex++
	}
	if req.Evidence != nil {
		updateSQL += fmt.Sprintf(", evidence = $%d", paramIndex)
		params = append(params, *req.Evidence)
		paramIndex++
	}
	if req.ConfidenceLevel != nil {
		updateSQL += fmt.Sprintf(", confidence_level = $%d", paramIndex)
		params = append(params, *req.ConfidenceLevel)
		paramIndex++
	}
	if req.ReviewedBy != nil {
		updateSQL += fmt.Sprintf(", reviewed_by = $%d", paramIndex)
		params = append(params, *req.ReviewedBy)
		paramIndex++
	}

	updateSQL += fmt.Sprintf(" WHERE id = $%d", paramIndex)
	params = append(params, id)

	now := time.Now()
	params[0] = now

	_, err = s.db.ExecContext(ctx, updateSQL, params...)
	if err != nil {
		return nil, fmt.Errorf("更新根因分析失败: %v", err)
	}

	// 获取更新后的数据
	return s.GetRootCauseAnalysis(ctx, id, tenantID)
}

// DeleteRootCauseAnalysis 删除根本原因分析
func (s *ProblemInvestigationService) DeleteRootCauseAnalysis(ctx context.Context, id int, tenantID int) error {
	s.logger.Infow("Deleting root cause analysis", "id", id, "tenant_id", tenantID)

	// 检查是否存在
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM problem_root_cause_analyses WHERE id = $1", id).Scan(&count)
	if err != nil {
		return fmt.Errorf("查询根因分析失败: %v", err)
	}
	if count == 0 {
		return fmt.Errorf("根因分析不存在")
	}

	// 删除
	_, err = s.db.ExecContext(ctx, "DELETE FROM problem_root_cause_analyses WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("删除根因分析失败: %v", err)
	}

	return nil
}

// UpdateProblemSolution 更新解决方案
func (s *ProblemInvestigationService) UpdateProblemSolution(ctx context.Context, id int, req *dto.UpdateProblemSolutionRequest, tenantID int) (*dto.ProblemSolutionResponse, error) {
	s.logger.Infow("Updating problem solution", "id", id, "tenant_id", tenantID)

	// 检查解决方案是否存在
	var existingSolution dto.ProblemSolutionResponse
	err := s.db.QueryRowContext(ctx, `
		SELECT ps.id, ps.problem_id, ps.solution_type, ps.solution_description, ps.proposed_by, u1.name,
		       ps.proposed_date, ps.status, ps.priority, ps.estimated_effort_hours, ps.estimated_cost,
		       ps.risk_assessment, ps.approval_status, ps.approved_by, u2.name, ps.approval_date,
		       ps.created_at, ps.updated_at
		FROM problem_solutions ps
		JOIN users u1 ON ps.proposed_by = u1.id
		LEFT JOIN users u2 ON ps.approved_by = u2.id
		WHERE ps.id = $1
	`, id).Scan(
		&existingSolution.ID, &existingSolution.ProblemID, &existingSolution.SolutionType, &existingSolution.SolutionDescription,
		&existingSolution.ProposedBy, &existingSolution.ProposedByName, &existingSolution.ProposedDate, &existingSolution.Status,
		&existingSolution.Priority, &existingSolution.EstimatedEffortHours, &existingSolution.EstimatedCost,
		&existingSolution.RiskAssessment, &existingSolution.ApprovalStatus, &existingSolution.ApprovedBy, &existingSolution.ApprovedByName,
		&existingSolution.ApprovalDate, &existingSolution.CreatedAt, &existingSolution.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("解决方案不存在")
		}
		return nil, fmt.Errorf("查询解决方案失败: %v", err)
	}

	// 构建更新语句
	updateSQL := "UPDATE problem_solutions SET updated_at = $1"
	var params []interface{}
	paramIndex := 2

	if req.SolutionType != nil {
		updateSQL += fmt.Sprintf(", solution_type = $%d", paramIndex)
		params = append(params, *req.SolutionType)
		paramIndex++
	}
	if req.SolutionDescription != nil {
		updateSQL += fmt.Sprintf(", solution_description = $%d", paramIndex)
		params = append(params, *req.SolutionDescription)
		paramIndex++
	}
	if req.Status != nil {
		updateSQL += fmt.Sprintf(", status = $%d", paramIndex)
		params = append(params, *req.Status)
		paramIndex++
	}
	if req.Priority != nil {
		updateSQL += fmt.Sprintf(", priority = $%d", paramIndex)
		params = append(params, *req.Priority)
		paramIndex++
	}
	if req.EstimatedEffortHours != nil {
		updateSQL += fmt.Sprintf(", estimated_effort_hours = $%d", paramIndex)
		params = append(params, *req.EstimatedEffortHours)
		paramIndex++
	}
	if req.EstimatedCost != nil {
		updateSQL += fmt.Sprintf(", estimated_cost = $%d", paramIndex)
		params = append(params, *req.EstimatedCost)
		paramIndex++
	}
	if req.RiskAssessment != nil {
		updateSQL += fmt.Sprintf(", risk_assessment = $%d", paramIndex)
		params = append(params, *req.RiskAssessment)
		paramIndex++
	}

	updateSQL += fmt.Sprintf(" WHERE id = $%d", paramIndex)
	params = append(params, id)

	now := time.Now()
	params[0] = now

	_, err = s.db.ExecContext(ctx, updateSQL, params...)
	if err != nil {
		return nil, fmt.Errorf("更新解决方案失败: %v", err)
	}

	// 获取更新后的数据
	return s.GetProblemSolution(ctx, id, tenantID)
}

// DeleteProblemSolution 删除解决方案
func (s *ProblemInvestigationService) DeleteProblemSolution(ctx context.Context, id int, tenantID int) error {
	s.logger.Infow("Deleting problem solution", "id", id, "tenant_id", tenantID)

	// 检查是否存在
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM problem_solutions WHERE id = $1", id).Scan(&count)
	if err != nil {
		return fmt.Errorf("查询解决方案失败: %v", err)
	}
	if count == 0 {
		return fmt.Errorf("解决方案不存在")
	}

	// 删除
	_, err = s.db.ExecContext(ctx, "DELETE FROM problem_solutions WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("删除解决方案失败: %v", err)
	}

	return nil
}

// ApproveSolution 审批解决方案
func (s *ProblemInvestigationService) ApproveSolution(ctx context.Context, id int, approverID int, approved bool, comment string, tenantID int) (*dto.ProblemSolutionResponse, error) {
	s.logger.Infow("Approving solution", "id", id, "approver_id", approverID, "approved", approved)

	// 检查解决方案是否存在
	var solution dto.ProblemSolutionResponse
	err := s.db.QueryRowContext(ctx, `
		SELECT ps.id, ps.problem_id, ps.solution_type, ps.solution_description, ps.proposed_by, u1.name,
		       ps.proposed_date, ps.status, ps.priority, ps.estimated_effort_hours, ps.estimated_cost,
		       ps.risk_assessment, ps.approval_status, ps.approved_by, u2.name, ps.approval_date,
		       ps.created_at, ps.updated_at
		FROM problem_solutions ps
		JOIN users u1 ON ps.proposed_by = u1.id
		LEFT JOIN users u2 ON ps.approved_by = u2.id
		WHERE ps.id = $1
	`, id).Scan(
		&solution.ID, &solution.ProblemID, &solution.SolutionType, &solution.SolutionDescription,
		&solution.ProposedBy, &solution.ProposedByName, &solution.ProposedDate, &solution.Status,
		&solution.Priority, &solution.EstimatedEffortHours, &solution.EstimatedCost,
		&solution.RiskAssessment, &solution.ApprovalStatus, &solution.ApprovedBy, &solution.ApprovedByName,
		&solution.ApprovalDate, &solution.CreatedAt, &solution.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("解决方案不存在")
		}
		return nil, fmt.Errorf("查询解决方案失败: %v", err)
	}

	// 更新审批状态
	approvalStatus := "rejected"
	if approved {
		approvalStatus = "approved"
	}

	now := time.Now()
	_, err = s.db.ExecContext(ctx, `
		UPDATE problem_solutions
		SET approval_status = $1, approved_by = $2, approval_date = $3, updated_at = $3
		WHERE id = $4
	`, approvalStatus, approverID, now, id)
	if err != nil {
		return nil, fmt.Errorf("更新审批状态失败: %v", err)
	}

	// 如果批准，更新解决方案状态为待实施
	if approved {
		_, err = s.db.ExecContext(ctx, `
			UPDATE problem_solutions SET status = $1, updated_at = $2 WHERE id = $3
		`, dto.SolutionStatusPendingImplementation, now, id)
		if err != nil {
			s.logger.Warnw("Failed to update solution status after approval", "error", err)
		}
	}

	return s.GetProblemSolution(ctx, id, tenantID)
}
