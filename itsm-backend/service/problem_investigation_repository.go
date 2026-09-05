package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"itsm-backend/dto"
)

// problemInvestigationRepository 负责封装 problem investigation 关联表（无 Ent schema）的 SQL。
//
// 设计原则：
// 1. 所有方法都强制 tenant 隔离，要么用 IN (SELECT ... FROM problems WHERE tenant_id)，
//    要么用 JOIN problems p ON ... WHERE p.tenant_id = $。
// 2. 状态机校验 / 业务编排留在 service 层；repository 只暴露原子数据操作。
// 3. 多表 JOIN 一次性读出 DTO，避免 service 反复 round-trip。
//
// 被封装表：
// problem_investigations、problem_investigation_steps、problem_root_cause_analyses、problem_solutions。
type problemInvestigationRepository struct {
	db *sql.DB
}

func newProblemInvestigationRepository(db *sql.DB) *problemInvestigationRepository {
	if db == nil {
		return nil
	}
	return &problemInvestigationRepository{db: db}
}

// ============================================================
// problems 表（部分查询通过 Ent，但跨表 JOIN + tenant 校验仍走 repository）
// ============================================================

// FetchProblemTitle 取出 problem 标题，用于"问题不存在"提示。
func (r *problemInvestigationRepository) FetchProblemTitle(ctx context.Context, problemID, tenantID int) (string, bool, error) {
	if r == nil || r.db == nil {
		return "", false, errors.New("problem investigation repository not initialised")
	}
	const query = `SELECT title FROM problems WHERE id = $1 AND tenant_id = $2`
	var title string
	if err := r.db.QueryRowContext(ctx, query, problemID, tenantID).Scan(&title); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("fetch problem title: %w", err)
	}
	return title, true, nil
}

// CurrentProblemStatus 取出 problem 当前状态。
func (r *problemInvestigationRepository) CurrentProblemStatus(ctx context.Context, problemID, tenantID int) (string, error) {
	if r == nil || r.db == nil {
		return "", errors.New("problem investigation repository not initialised")
	}
	const query = `SELECT status FROM problems WHERE id = $1 AND tenant_id = $2`
	var status string
	if err := r.db.QueryRowContext(ctx, query, problemID, tenantID).Scan(&status); err != nil {
		return "", fmt.Errorf("fetch problem status: %w", err)
	}
	return status, nil
}

// UpdateProblemStatusWithGuard 乐观并发更新 problem 状态：WHERE id + status = current。
//
// 返回 affected==true 表示更新成功，affected==false 表示状态被其他请求改动。
func (r *problemInvestigationRepository) UpdateProblemStatusWithGuard(ctx context.Context, problemID, tenantID int, current, target string, now time.Time) (bool, error) {
	if r == nil || r.db == nil {
		return false, errors.New("problem investigation repository not initialised")
	}
	const query = `
		UPDATE problems
		SET status = $1, updated_at = $2
		WHERE id = $3 AND tenant_id = $4 AND status = $5
	`
	res, err := r.db.ExecContext(ctx, query, target, now, problemID, tenantID, current)
	if err != nil {
		return false, fmt.Errorf("update problem status guarded: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected: %w", err)
	}
	return affected == 1, nil
}

// ============================================================
// problem_investigations 表
// ============================================================

// InvestigationExistsByProblem 检查某 problem 是否已存在调查记录。
func (r *problemInvestigationRepository) InvestigationExistsByProblem(ctx context.Context, problemID, tenantID int) (bool, error) {
	if r == nil || r.db == nil {
		return false, errors.New("problem investigation repository not initialised")
	}
	const query = `
		SELECT pi.id
		FROM problem_investigations pi
		JOIN problems p ON pi.problem_id = p.id
		WHERE pi.problem_id = $1 AND p.tenant_id = $2
	`
	var id int
	if err := r.db.QueryRowContext(ctx, query, problemID, tenantID).Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("check investigation exists: %w", err)
	}
	return true, nil
}

// CreateInvestigation 在 problem_investigations 插入新调查。
func (r *problemInvestigationRepository) CreateInvestigation(ctx context.Context, problemID, investigatorID int, estimatedCompletion *time.Time, summary *string, now time.Time) (int, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("problem investigation repository not initialised")
	}
	const query = `
		INSERT INTO problem_investigations
		    (problem_id, investigator_id, estimated_completion_date, investigation_summary, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		RETURNING id
	`
	var id int
	if err := r.db.QueryRowContext(ctx, query, problemID, investigatorID, estimatedCompletion, summary, now).Scan(&id); err != nil {
		return 0, fmt.Errorf("create investigation: %w", err)
	}
	return id, nil
}

// FetchInvestigationIDByProblem 取出某 problem 关联的调查 ID（用于 GetSummary）。
func (r *problemInvestigationRepository) FetchInvestigationIDByProblem(ctx context.Context, problemID, tenantID int) (int, bool, error) {
	if r == nil || r.db == nil {
		return 0, false, errors.New("problem investigation repository not initialised")
	}
	const query = `
		SELECT pi.id FROM problem_investigations pi
		JOIN problems p ON pi.problem_id = p.id
		WHERE pi.problem_id = $1 AND p.tenant_id = $2
	`
	var id int
	if err := r.db.QueryRowContext(ctx, query, problemID, tenantID).Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("fetch investigation id: %w", err)
	}
	return id, true, nil
}

// FetchInvestigationProblemID 取出某调查记录的 problemID，并强制校验租户。
func (r *problemInvestigationRepository) FetchInvestigationProblemID(ctx context.Context, investigationID, tenantID int) (int, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("problem investigation repository not initialised")
	}
	const query = `
		SELECT pi.problem_id FROM problem_investigations pi
		JOIN problems p ON pi.problem_id = p.id
		WHERE pi.id = $1 AND p.tenant_id = $2
	`
	var pid int
	if err := r.db.QueryRowContext(ctx, query, investigationID, tenantID).Scan(&pid); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("fetch investigation problem id: %w", err)
	}
	return pid, nil
}

// investigationUpdate 动态更新字段集合（nil 表示不更新）。
type investigationUpdate struct {
	Status                  *string
	EstimatedCompletionDate *time.Time
	ActualCompletionDate    *time.Time
	InvestigationSummary    *string
}

// UpdateInvestigation 动态更新 problem_investigations 行，tenant 由子查询保证。
func (r *problemInvestigationRepository) UpdateInvestigation(ctx context.Context, investigationID int, upd investigationUpdate, tenantID, nowUnix int64) error {
	if r == nil || r.db == nil {
		return errors.New("problem investigation repository not initialised")
	}
	now := time.Unix(nowUnix, 0)
	columns := []string{"updated_at = $1"}
	args := []interface{}{now}
	idx := 2
	if upd.Status != nil {
		columns = append(columns, fmt.Sprintf("status = $%d", idx))
		args = append(args, *upd.Status)
		idx++
	}
	if upd.EstimatedCompletionDate != nil {
		columns = append(columns, fmt.Sprintf("estimated_completion_date = $%d", idx))
		args = append(args, *upd.EstimatedCompletionDate)
		idx++
	}
	if upd.ActualCompletionDate != nil {
		columns = append(columns, fmt.Sprintf("actual_completion_date = $%d", idx))
		args = append(args, *upd.ActualCompletionDate)
		idx++
	}
	if upd.InvestigationSummary != nil {
		columns = append(columns, fmt.Sprintf("investigation_summary = $%d", idx))
		args = append(args, *upd.InvestigationSummary)
		idx++
	}
	query := "UPDATE problem_investigations SET " + strings.Join(columns, ", ") +
		fmt.Sprintf(" WHERE id = $%d AND problem_id IN (SELECT id FROM problems WHERE tenant_id = $%d)", idx, idx+1)
	args = append(args, investigationID, tenantID)
	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("update investigation: %w", err)
	}
	return nil
}

// ============================================================
// problem_investigation_steps 表
// ============================================================

// InvestigationStepCreate 调查步骤写入行。
type InvestigationStepCreate struct {
	InvestigationID  int
	StepNumber       int
	StepTitle        string
	StepDescription  string
	AssignedTo       *int
	Notes            *string
}

// CreateInvestigationStep 插入调查步骤。
func (r *problemInvestigationRepository) CreateInvestigationStep(ctx context.Context, step *InvestigationStepCreate, now time.Time) (int, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("problem investigation repository not initialised")
	}
	const query = `
		INSERT INTO problem_investigation_steps
		    (investigation_id, step_number, step_title, step_description, assigned_to, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		RETURNING id
	`
	var id int
	if err := r.db.QueryRowContext(ctx, query,
		step.InvestigationID, step.StepNumber, step.StepTitle, step.StepDescription,
		step.AssignedTo, step.Notes, now,
	).Scan(&id); err != nil {
		return 0, fmt.Errorf("create investigation step: %w", err)
	}
	return id, nil
}

// FetchStepInvestigationID 取出 step 所属 investigation，并通过 problems JOIN 校验租户。
func (r *problemInvestigationRepository) FetchStepInvestigationID(ctx context.Context, stepID, tenantID int) (int, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("problem investigation repository not initialised")
	}
	const query = `
		SELECT pis.investigation_id FROM problem_investigation_steps pis
		JOIN problem_investigations pi ON pis.investigation_id = pi.id
		JOIN problems p ON pi.problem_id = p.id
		WHERE pis.id = $1 AND p.tenant_id = $2
	`
	var iid int
	if err := r.db.QueryRowContext(ctx, query, stepID, tenantID).Scan(&iid); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("fetch step investigation id: %w", err)
	}
	return iid, nil
}

// stepUpdate 动态更新字段集合。
type stepUpdate struct {
	StepTitle       *string
	StepDescription *string
	Status          *string
	AssignedTo      *int
	StartDate       *time.Time
	CompletionDate  *time.Time
	Notes           *string
}

// UpdateStep 动态更新 problem_investigation_steps 行。
func (r *problemInvestigationRepository) UpdateStep(ctx context.Context, stepID int, upd stepUpdate, now time.Time) error {
	if r == nil || r.db == nil {
		return errors.New("problem investigation repository not initialised")
	}
	columns := []string{"updated_at = $1"}
	args := []interface{}{now}
	idx := 2
	if upd.StepTitle != nil {
		columns = append(columns, fmt.Sprintf("step_title = $%d", idx))
		args = append(args, *upd.StepTitle)
		idx++
	}
	if upd.StepDescription != nil {
		columns = append(columns, fmt.Sprintf("step_description = $%d", idx))
		args = append(args, *upd.StepDescription)
		idx++
	}
	if upd.Status != nil {
		columns = append(columns, fmt.Sprintf("status = $%d", idx))
		args = append(args, *upd.Status)
		idx++
	}
	if upd.AssignedTo != nil {
		columns = append(columns, fmt.Sprintf("assigned_to = $%d", idx))
		args = append(args, *upd.AssignedTo)
		idx++
	}
	if upd.StartDate != nil {
		columns = append(columns, fmt.Sprintf("start_date = $%d", idx))
		args = append(args, *upd.StartDate)
		idx++
	}
	if upd.CompletionDate != nil {
		columns = append(columns, fmt.Sprintf("completion_date = $%d", idx))
		args = append(args, *upd.CompletionDate)
		idx++
	}
	if upd.Notes != nil {
		columns = append(columns, fmt.Sprintf("notes = $%d", idx))
		args = append(args, *upd.Notes)
		idx++
	}
	query := "UPDATE problem_investigation_steps SET " + strings.Join(columns, ", ") +
		fmt.Sprintf(" WHERE id = $%d", idx)
	args = append(args, stepID)
	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("update step: %w", err)
	}
	return nil
}

// ============================================================
// problem_root_cause_analyses 表
// ============================================================

// RootCauseExistsByProblem 检查某 problem 是否已存在根因分析。
func (r *problemInvestigationRepository) RootCauseExistsByProblem(ctx context.Context, problemID, tenantID int) (bool, error) {
	if r == nil || r.db == nil {
		return false, errors.New("problem investigation repository not initialised")
	}
	const query = `
		SELECT rca.id FROM problem_root_cause_analyses rca
		JOIN problems p ON rca.problem_id = p.id
		WHERE rca.problem_id = $1 AND p.tenant_id = $2
	`
	var id int
	if err := r.db.QueryRowContext(ctx, query, problemID, tenantID).Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("check root cause exists: %w", err)
	}
	return true, nil
}

// FetchRootCauseAnalysisIDByProblem 同上。
func (r *problemInvestigationRepository) FetchRootCauseAnalysisIDByProblem(ctx context.Context, problemID, tenantID int) (int, bool, error) {
	if r == nil || r.db == nil {
		return 0, false, errors.New("problem investigation repository not initialised")
	}
	const query = `
		SELECT rca.id FROM problem_root_cause_analyses rca
		JOIN problems p ON rca.problem_id = p.id
		WHERE rca.problem_id = $1 AND p.tenant_id = $2
	`
	var id int
	if err := r.db.QueryRowContext(ctx, query, problemID, tenantID).Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("fetch root cause id: %w", err)
	}
	return id, true, nil
}

// RootCauseAnalysisCreate 根因分析写入行。
type RootCauseAnalysisCreate struct {
	ProblemID            int
	AnalystID            int
	AnalysisMethod       string
	RootCauseDescription string
	ContributingFactors  *string
	Evidence             *string
	ConfidenceLevel      *float64
}

// CreateRootCauseAnalysis 插入根因分析。
func (r *problemInvestigationRepository) CreateRootCauseAnalysis(ctx context.Context, row *RootCauseAnalysisCreate, now time.Time) (int, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("problem investigation repository not initialised")
	}
	const query = `
		INSERT INTO problem_root_cause_analyses
		    (problem_id, analyst_id, analysis_method, root_cause_description,
		     contributing_factors, evidence, confidence_level, analysis_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8, $8)
		RETURNING id
	`
	var id int
	if err := r.db.QueryRowContext(ctx, query,
		row.ProblemID, row.AnalystID, row.AnalysisMethod, row.RootCauseDescription,
		row.ContributingFactors, row.Evidence, row.ConfidenceLevel, now,
	).Scan(&id); err != nil {
		return 0, fmt.Errorf("create root cause analysis: %w", err)
	}
	return id, nil
}

// rootCauseUpdate 动态更新字段集合。
type rootCauseUpdate struct {
	AnalysisMethod       *string
	RootCauseDescription *string
	ContributingFactors  *string
	Evidence             *string
	ConfidenceLevel      *float64
	ReviewedBy           *int
}

// UpdateRootCauseAnalysis 动态更新根因分析行，tenant 由子查询保证。
func (r *problemInvestigationRepository) UpdateRootCauseAnalysis(ctx context.Context, id, tenantID int, upd rootCauseUpdate, now time.Time) error {
	if r == nil || r.db == nil {
		return errors.New("problem investigation repository not initialised")
	}
	columns := []string{"updated_at = $1"}
	args := []interface{}{now}
	idx := 2
	if upd.AnalysisMethod != nil {
		columns = append(columns, fmt.Sprintf("analysis_method = $%d", idx))
		args = append(args, *upd.AnalysisMethod)
		idx++
	}
	if upd.RootCauseDescription != nil {
		columns = append(columns, fmt.Sprintf("root_cause_description = $%d", idx))
		args = append(args, *upd.RootCauseDescription)
		idx++
	}
	if upd.ContributingFactors != nil {
		columns = append(columns, fmt.Sprintf("contributing_factors = $%d", idx))
		args = append(args, *upd.ContributingFactors)
		idx++
	}
	if upd.Evidence != nil {
		columns = append(columns, fmt.Sprintf("evidence = $%d", idx))
		args = append(args, *upd.Evidence)
		idx++
	}
	if upd.ConfidenceLevel != nil {
		columns = append(columns, fmt.Sprintf("confidence_level = $%d", idx))
		args = append(args, *upd.ConfidenceLevel)
		idx++
	}
	if upd.ReviewedBy != nil {
		columns = append(columns, fmt.Sprintf("reviewed_by = $%d", idx))
		args = append(args, *upd.ReviewedBy)
		idx++
	}
	query := "UPDATE problem_root_cause_analyses SET " + strings.Join(columns, ", ") +
		fmt.Sprintf(" WHERE id = $%d AND problem_id IN (SELECT id FROM problems WHERE tenant_id = $%d)", idx, idx+1)
	args = append(args, id, tenantID)
	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("update root cause analysis: %w", err)
	}
	return nil
}

// DeleteRootCauseAnalysis 删除根因分析。
func (r *problemInvestigationRepository) DeleteRootCauseAnalysis(ctx context.Context, id, tenantID int) (bool, error) {
	if r == nil || r.db == nil {
		return false, errors.New("problem investigation repository not initialised")
	}
	const query = `
		DELETE FROM problem_root_cause_analyses
		WHERE id = $1 AND problem_id IN (SELECT id FROM problems WHERE tenant_id = $2)
	`
	res, err := r.db.ExecContext(ctx, query, id, tenantID)
	if err != nil {
		return false, fmt.Errorf("delete root cause analysis: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected: %w", err)
	}
	return affected == 1, nil
}

// CountRootCauseAnalysis 检查根因分析是否存在。
func (r *problemInvestigationRepository) CountRootCauseAnalysis(ctx context.Context, id, tenantID int) (int, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("problem investigation repository not initialised")
	}
	const query = `
		SELECT COUNT(*) FROM problem_root_cause_analyses
		WHERE id = $1 AND problem_id IN (SELECT id FROM problems WHERE tenant_id = $2)
	`
	var count int
	if err := r.db.QueryRowContext(ctx, query, id, tenantID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count root cause: %w", err)
	}
	return count, nil
}

// ============================================================
// problem_solutions 表
// ============================================================

// ProblemSolutionCreate 解决方案写入行。
type ProblemSolutionCreate struct {
	ProblemID            int
	SolutionType         string
	SolutionDescription  string
	ProposedBy           int
	Status               string
	Priority             string
	EstimatedEffortHours *int
	EstimatedCost        *float64
	RiskAssessment       *string
	ApprovalStatus       string
}

// CreateProblemSolution 插入解决方案。
func (r *problemInvestigationRepository) CreateProblemSolution(ctx context.Context, row *ProblemSolutionCreate, now time.Time) (int, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("problem investigation repository not initialised")
	}
	const query = `
		INSERT INTO problem_solutions
		    (problem_id, solution_type, solution_description, proposed_by, proposed_date,
		     status, priority, estimated_effort_hours, estimated_cost, risk_assessment,
		     approval_status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12)
		RETURNING id
	`
	var id int
	if err := r.db.QueryRowContext(ctx, query,
		row.ProblemID, row.SolutionType, row.SolutionDescription, row.ProposedBy, now,
		row.Status, row.Priority, row.EstimatedEffortHours, row.EstimatedCost, row.RiskAssessment,
		row.ApprovalStatus, now,
	).Scan(&id); err != nil {
		return 0, fmt.Errorf("create problem solution: %w", err)
	}
	return id, nil
}

// solutionUpdate 动态更新字段集合。
type solutionUpdate struct {
	SolutionType         *string
	SolutionDescription  *string
	Status               *string
	Priority             *string
	EstimatedEffortHours *int
	EstimatedCost        *float64
	RiskAssessment       *string
}

// UpdateProblemSolution 动态更新 solution 行，tenant 由子查询保证。
func (r *problemInvestigationRepository) UpdateProblemSolution(ctx context.Context, id, tenantID int, upd solutionUpdate, now time.Time) error {
	if r == nil || r.db == nil {
		return errors.New("problem investigation repository not initialised")
	}
	columns := []string{"updated_at = $1"}
	args := []interface{}{now}
	idx := 2
	if upd.SolutionType != nil {
		columns = append(columns, fmt.Sprintf("solution_type = $%d", idx))
		args = append(args, *upd.SolutionType)
		idx++
	}
	if upd.SolutionDescription != nil {
		columns = append(columns, fmt.Sprintf("solution_description = $%d", idx))
		args = append(args, *upd.SolutionDescription)
		idx++
	}
	if upd.Status != nil {
		columns = append(columns, fmt.Sprintf("status = $%d", idx))
		args = append(args, *upd.Status)
		idx++
	}
	if upd.Priority != nil {
		columns = append(columns, fmt.Sprintf("priority = $%d", idx))
		args = append(args, *upd.Priority)
		idx++
	}
	if upd.EstimatedEffortHours != nil {
		columns = append(columns, fmt.Sprintf("estimated_effort_hours = $%d", idx))
		args = append(args, *upd.EstimatedEffortHours)
		idx++
	}
	if upd.EstimatedCost != nil {
		columns = append(columns, fmt.Sprintf("estimated_cost = $%d", idx))
		args = append(args, *upd.EstimatedCost)
		idx++
	}
	if upd.RiskAssessment != nil {
		columns = append(columns, fmt.Sprintf("risk_assessment = $%d", idx))
		args = append(args, *upd.RiskAssessment)
		idx++
	}
	query := "UPDATE problem_solutions SET " + strings.Join(columns, ", ") +
		fmt.Sprintf(" WHERE id = $%d AND problem_id IN (SELECT id FROM problems WHERE tenant_id = $%d)", idx, idx+1)
	args = append(args, id, tenantID)
	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("update problem solution: %w", err)
	}
	return nil
}

// DeleteProblemSolution 删除 solution。
func (r *problemInvestigationRepository) DeleteProblemSolution(ctx context.Context, id, tenantID int) (bool, error) {
	if r == nil || r.db == nil {
		return false, errors.New("problem investigation repository not initialised")
	}
	const query = `
		DELETE FROM problem_solutions
		WHERE id = $1 AND problem_id IN (SELECT id FROM problems WHERE tenant_id = $2)
	`
	res, err := r.db.ExecContext(ctx, query, id, tenantID)
	if err != nil {
		return false, fmt.Errorf("delete problem solution: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected: %w", err)
	}
	return affected == 1, nil
}

// CountProblemSolution 检查 solution 是否存在。
func (r *problemInvestigationRepository) CountProblemSolution(ctx context.Context, id, tenantID int) (int, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("problem investigation repository not initialised")
	}
	const query = `
		SELECT COUNT(*) FROM problem_solutions
		WHERE id = $1 AND problem_id IN (SELECT id FROM problems WHERE tenant_id = $2)
	`
	var count int
	if err := r.db.QueryRowContext(ctx, query, id, tenantID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count solution: %w", err)
	}
	return count, nil
}

// ApproveSolution 一次完成审批状态 + 审批人 + 时间 三字段更新。
func (r *problemInvestigationRepository) ApproveSolution(ctx context.Context, id, tenantID, approverID int, approvalStatus string, now time.Time) error {
	if r == nil || r.db == nil {
		return errors.New("problem investigation repository not initialised")
	}
	const query = `
		UPDATE problem_solutions
		SET approval_status = $1, approved_by = $2, approval_date = $3, updated_at = $3
		WHERE id = $4 AND problem_id IN (SELECT id FROM problems WHERE tenant_id = $5)
	`
	if _, err := r.db.ExecContext(ctx, query, approvalStatus, approverID, now, id, tenantID); err != nil {
		return fmt.Errorf("approve solution: %w", err)
	}
	return nil
}

// SetSolutionStatus 更新 solution.status（审批通过后挂到待实施）。
func (r *problemInvestigationRepository) SetSolutionStatus(ctx context.Context, id, tenantID int, status string, now time.Time) error {
	if r == nil || r.db == nil {
		return errors.New("problem investigation repository not initialised")
	}
	const query = `
		UPDATE problem_solutions SET status = $1, updated_at = $2
		WHERE id = $3 AND problem_id IN (SELECT id FROM problems WHERE tenant_id = $4)
	`
	if _, err := r.db.ExecContext(ctx, query, status, now, id, tenantID); err != nil {
		return fmt.Errorf("set solution status: %w", err)
	}
	return nil
}

// ============================================================
// Read 路径：单行 / 多行读取（多表 JOIN + tenant 强制约束），直接返回 DTO。
// ============================================================

// confidenceFloatToLabel 把 DB 中的 float64 置信度映射回 DTO 字符串枚举。
//
// 历史数据可能填了精确数值（0.0~1.0），先用阈值映射到 low/medium/high；
// 任意低于 0.3 → low；0.3~0.7 区间 → medium；>= 0.7 → high。
func confidenceFloatToLabel(f *float64) dto.ConfidenceLevel {
	if f == nil {
		return ""
	}
	switch {
	case *f < 0.3:
		return dto.ConfidenceLow
	case *f < 0.7:
		return dto.ConfidenceMedium
	default:
		return dto.ConfidenceHigh
	}
}

// GetRootCauseAnalysis 单条根因分析 + analyst/reviewer 姓名 JOIN。
func (r *problemInvestigationRepository) GetRootCauseAnalysis(ctx context.Context, id, tenantID int) (*dto.RootCauseAnalysisResponse, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("problem investigation repository not initialised")
	}
	const query = `
		SELECT
		    rca.id, rca.problem_id, rca.analyst_id,
		    COALESCE(analyst.name, '') AS analyst_name,
		    rca.analysis_method, rca.root_cause_description,
		    rca.contributing_factors, rca.evidence, rca.confidence_level,
		    rca.analysis_date,
		    rca.reviewed_by,
		    COALESCE(reviewer.name, '') AS reviewer_name,
		    rca.review_date,
		    rca.created_at, rca.updated_at
		FROM problem_root_cause_analyses rca
		JOIN problems p ON rca.problem_id = p.id
		LEFT JOIN users analyst ON rca.analyst_id = analyst.id AND analyst.tenant_id = $2
		LEFT JOIN users reviewer ON rca.reviewed_by = reviewer.id AND reviewer.tenant_id = $2
		WHERE rca.id = $1 AND p.tenant_id = $2
	`
	out := &dto.RootCauseAnalysisResponse{}
	var confidence *float64
	var analystName, reviewerName string
	if err := r.db.QueryRowContext(ctx, query, id, tenantID).Scan(
		&out.ID, &out.ProblemID, &out.AnalystID, &analystName,
		&out.AnalysisMethod, &out.RootCauseDescription,
		&out.ContributingFactors, &out.Evidence, &confidence,
		&out.AnalysisDate,
		&out.ReviewedBy, &reviewerName, &out.ReviewDate,
		&out.CreatedAt, &out.UpdatedAt,
	); err != nil {
		return nil, err
	}
	out.AnalystName = analystName
	out.ConfidenceLevel = confidenceFloatToLabel(confidence)
	if out.ReviewedBy != nil {
		name := reviewerName
		out.ReviewedByName = &name
	}
	return out, nil
}

// GetProblemSolution 单条解决方案 + proposedBy/approvedBy 姓名 JOIN。
func (r *problemInvestigationRepository) GetProblemSolution(ctx context.Context, id, tenantID int) (*dto.ProblemSolutionResponse, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("problem investigation repository not initialised")
	}
	const query = `
		SELECT
		    ps.id, ps.problem_id, ps.solution_type, ps.solution_description,
		    ps.proposed_by, COALESCE(proposer.name, '') AS proposer_name,
		    ps.proposed_date,
		    ps.status, ps.priority,
		    ps.estimated_effort_hours, ps.estimated_cost, ps.risk_assessment,
		    ps.approval_status,
		    ps.approved_by, COALESCE(approver.name, '') AS approver_name,
		    ps.approval_date,
		    ps.created_at, ps.updated_at
		FROM problem_solutions ps
		JOIN problems p ON ps.problem_id = p.id
		LEFT JOIN users proposer ON ps.proposed_by = proposer.id AND proposer.tenant_id = $2
		LEFT JOIN users approver ON ps.approved_by = approver.id AND approver.tenant_id = $2
		WHERE ps.id = $1 AND p.tenant_id = $2
	`
	out := &dto.ProblemSolutionResponse{}
	var proposerName, approverName string
	if err := r.db.QueryRowContext(ctx, query, id, tenantID).Scan(
		&out.ID, &out.ProblemID, &out.SolutionType, &out.SolutionDescription,
		&out.ProposedBy, &proposerName, &out.ProposedDate,
		&out.Status, &out.Priority,
		&out.EstimatedEffortHours, &out.EstimatedCost, &out.RiskAssessment,
		&out.ApprovalStatus,
		&out.ApprovedBy, &approverName, &out.ApprovalDate,
		&out.CreatedAt, &out.UpdatedAt,
	); err != nil {
		return nil, err
	}
	out.ProposedByName = proposerName
	if out.ApprovedBy != nil {
		name := approverName
		out.ApprovedByName = &name
	}
	return out, nil
}

// GetProblemInvestigation 单条调查 + investigator 姓名 JOIN。
func (r *problemInvestigationRepository) GetProblemInvestigation(ctx context.Context, investigationID, tenantID int) (*dto.ProblemInvestigationResponse, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("problem investigation repository not initialised")
	}
	const query = `
		SELECT
		    pi.id, pi.problem_id, pi.investigator_id,
		    COALESCE(inv.name, '') AS investigator_name,
		    pi.status, pi.start_date, pi.estimated_completion_date,
		    pi.actual_completion_date, pi.investigation_summary,
		    pi.created_at, pi.updated_at
		FROM problem_investigations pi
		JOIN problems p ON pi.problem_id = p.id
		LEFT JOIN users inv ON pi.investigator_id = inv.id AND inv.tenant_id = $2
		WHERE pi.id = $1 AND p.tenant_id = $2
	`
	out := &dto.ProblemInvestigationResponse{}
	var investigatorName string
	if err := r.db.QueryRowContext(ctx, query, investigationID, tenantID).Scan(
		&out.ID, &out.ProblemID, &out.InvestigatorID, &investigatorName,
		&out.Status, &out.StartDate, &out.EstimatedCompletionDate,
		&out.ActualCompletionDate, &out.InvestigationSummary,
		&out.CreatedAt, &out.UpdatedAt,
	); err != nil {
		return nil, err
	}
	out.InvestigatorName = investigatorName
	return out, nil
}

// GetInvestigationStep 单条调查步骤 + assignee 姓名 JOIN。
func (r *problemInvestigationRepository) GetInvestigationStep(ctx context.Context, stepID, tenantID int) (*dto.InvestigationStepResponse, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("problem investigation repository not initialised")
	}
	const query = `
		SELECT
		    pis.id, pis.investigation_id, pis.step_number,
		    pis.step_title, pis.step_description, pis.status,
		    pis.assigned_to,
		    COALESCE(asg.name, '') AS assigned_name,
		    pis.start_date, pis.completion_date, pis.notes,
		    pis.created_at, pis.updated_at
		FROM problem_investigation_steps pis
		JOIN problem_investigations pi ON pis.investigation_id = pi.id
		JOIN problems p ON pi.problem_id = p.id
		LEFT JOIN users asg ON pis.assigned_to = asg.id AND asg.tenant_id = $2
		WHERE pis.id = $1 AND p.tenant_id = $2
	`
	out := &dto.InvestigationStepResponse{}
	var assignedName string
	if err := r.db.QueryRowContext(ctx, query, stepID, tenantID).Scan(
		&out.ID, &out.InvestigationID, &out.StepNumber,
		&out.StepTitle, &out.StepDescription, &out.Status,
		&out.AssignedTo, &assignedName,
		&out.StartDate, &out.CompletionDate, &out.Notes,
		&out.CreatedAt, &out.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if out.AssignedTo != nil {
		name := assignedName
		out.AssignedToName = &name
	}
	return out, nil
}

// ListInvestigationSteps 列出某调查下的全部步骤，按 step_number ASC。
func (r *problemInvestigationRepository) ListInvestigationSteps(ctx context.Context, investigationID, tenantID int) ([]*dto.InvestigationStepResponse, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("problem investigation repository not initialised")
	}
	const query = `
		SELECT
		    pis.id, pis.investigation_id, pis.step_number,
		    pis.step_title, pis.step_description, pis.status,
		    pis.assigned_to,
		    COALESCE(asg.name, '') AS assigned_name,
		    pis.start_date, pis.completion_date, pis.notes,
		    pis.created_at, pis.updated_at
		FROM problem_investigation_steps pis
		JOIN problem_investigations pi ON pis.investigation_id = pi.id
		JOIN problems p ON pi.problem_id = p.id
		LEFT JOIN users asg ON pis.assigned_to = asg.id AND asg.tenant_id = $2
		WHERE pis.investigation_id = $1 AND p.tenant_id = $2
		ORDER BY pis.step_number ASC, pis.id ASC
	`
	rows, err := r.db.QueryContext(ctx, query, investigationID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list investigation steps: %w", err)
	}
	defer rows.Close()
	out := make([]*dto.InvestigationStepResponse, 0)
	for rows.Next() {
		step := &dto.InvestigationStepResponse{}
		var assignedName string
		if err := rows.Scan(
			&step.ID, &step.InvestigationID, &step.StepNumber,
			&step.StepTitle, &step.StepDescription, &step.Status,
			&step.AssignedTo, &assignedName,
			&step.StartDate, &step.CompletionDate, &step.Notes,
			&step.CreatedAt, &step.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan investigation step: %w", err)
		}
		if step.AssignedTo != nil {
			name := assignedName
			step.AssignedToName = &name
		}
		out = append(out, step)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate investigation steps: %w", err)
	}
	return out, nil
}

// ListProblemSolutions 列出某 problem 下全部方案，按 created_at DESC。
func (r *problemInvestigationRepository) ListProblemSolutions(ctx context.Context, problemID, tenantID int) ([]*dto.ProblemSolutionResponse, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("problem investigation repository not initialised")
	}
	const query = `
		SELECT
		    ps.id, ps.problem_id, ps.solution_type, ps.solution_description,
		    ps.proposed_by, COALESCE(proposer.name, '') AS proposer_name,
		    ps.proposed_date,
		    ps.status, ps.priority,
		    ps.estimated_effort_hours, ps.estimated_cost, ps.risk_assessment,
		    ps.approval_status,
		    ps.approved_by, COALESCE(approver.name, '') AS approver_name,
		    ps.approval_date,
		    ps.created_at, ps.updated_at
		FROM problem_solutions ps
		JOIN problems p ON ps.problem_id = p.id
		LEFT JOIN users proposer ON ps.proposed_by = proposer.id AND proposer.tenant_id = $2
		LEFT JOIN users approver ON ps.approved_by = approver.id AND approver.tenant_id = $2
		WHERE ps.problem_id = $1 AND p.tenant_id = $2
		ORDER BY ps.created_at DESC, ps.id DESC
	`
	rows, err := r.db.QueryContext(ctx, query, problemID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list problem solutions: %w", err)
	}
	defer rows.Close()
	out := make([]*dto.ProblemSolutionResponse, 0)
	for rows.Next() {
		sol := &dto.ProblemSolutionResponse{}
		var proposerName, approverName string
		if err := rows.Scan(
			&sol.ID, &sol.ProblemID, &sol.SolutionType, &sol.SolutionDescription,
			&sol.ProposedBy, &proposerName, &sol.ProposedDate,
			&sol.Status, &sol.Priority,
			&sol.EstimatedEffortHours, &sol.EstimatedCost, &sol.RiskAssessment,
			&sol.ApprovalStatus,
			&sol.ApprovedBy, &approverName, &sol.ApprovalDate,
			&sol.CreatedAt, &sol.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan problem solution: %w", err)
		}
		sol.ProposedByName = proposerName
		if sol.ApprovedBy != nil {
			name := approverName
			sol.ApprovedByName = &name
		}
		out = append(out, sol)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate problem solutions: %w", err)
	}
	return out, nil
}