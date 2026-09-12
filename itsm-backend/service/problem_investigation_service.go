package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/knowledgearticle"

	"go.uber.org/zap"
)

// ProblemInvestigationService 问题调查服务
type ProblemInvestigationService struct {
	db     *sql.DB
	client *ent.Client
	logger *zap.SugaredLogger
	repo   *problemInvestigationRepository
	users  *userLookupRepository
}

// NewProblemInvestigationService 创建问题调查服务
func NewProblemInvestigationService(db *sql.DB, client *ent.Client, logger *zap.SugaredLogger) *ProblemInvestigationService {
	return &ProblemInvestigationService{
		db:     db,
		client: client,
		logger: logger,
		repo:   newProblemInvestigationRepository(db),
		users:  newUserLookupRepository(db),
	}
}

// SetUserLookupRepository 注入用户查询仓储（避免循环依赖）
func (s *ProblemInvestigationService) SetUserLookupRepository(repo *userLookupRepository) {
	s.users = repo
}

func (s *ProblemInvestigationService) CreateKnowledgeArticle(ctx context.Context, tenantID, authorID int, req *dto.CreateProblemKnowledgeArticleRequest) (*ent.KnowledgeArticle, error) {
	return s.client.KnowledgeArticle.Create().
		SetTitle(req.ArticleTitle).
		SetContent(req.ArticleContent).
		SetCategory(req.ArticleType).
		SetAuthorID(authorID).
		SetTags(strings.Join(req.Tags, ",")).
		SetTenantID(tenantID).
		Save(ctx)
}

func (s *ProblemInvestigationService) ListKnowledgeArticles(ctx context.Context, tenantID int) ([]*ent.KnowledgeArticle, error) {
	return s.client.KnowledgeArticle.Query().
		Where(knowledgearticle.TenantIDEQ(tenantID), knowledgearticle.DeletedAtIsNil()).
		Order(ent.Desc(knowledgearticle.FieldCreatedAt)).
		All(ctx)
}

// problemInvestigationStatusTransitions 与 handlers/problem 的状态机保持一致。
// 调查服务联动问题状态时必须经过该校验，禁止绕过状态机直写。
var problemInvestigationStatusTransitions = map[string]map[string]struct{}{
	"open":          {"investigating": {}, "identified": {}, "resolved": {}},
	"investigating": {"identified": {}, "resolved": {}},
	"identified":    {"investigating": {}, "resolved": {}},
	"resolved":      {"investigating": {}, "closed": {}},
	"closed":        {},
	// 兼容存量 in_progress 数据，仅允许进入规范状态。
	"in_progress": {"identified": {}, "resolved": {}},
}

// transitionProblemStatus 校验并联动问题状态（带状态机校验 + 乐观并发防护）
//
// 返回错误当且仅当目标转换非法（例如 closed 终态复活）。
// SQL 带 AND status = $current 防止检查与更新之间的竞态。
func (s *ProblemInvestigationService) transitionProblemStatus(ctx context.Context, problemID int, tenantID int, targetStatus string) error {
	currentStatus, err := s.repo.CurrentProblemStatus(ctx, problemID, tenantID)
	if err != nil {
		return fmt.Errorf("查询问题状态失败: %v", err)
	}

	if currentStatus == targetStatus {
		return nil
	}
	allowed, ok := problemInvestigationStatusTransitions[currentStatus]
	if !ok {
		return fmt.Errorf("问题当前状态 %q 非法，拒绝状态联动", currentStatus)
	}
	if _, ok := allowed[targetStatus]; !ok {
		return fmt.Errorf("问题状态 %q 不允许转换为 %q", currentStatus, targetStatus)
	}

	updated, err := s.repo.UpdateProblemStatusWithGuard(ctx, problemID, tenantID, currentStatus, targetStatus, time.Now())
	if err != nil {
		return fmt.Errorf("更新问题状态失败: %v", err)
	}
	if !updated {
		// 并发竞争：状态已被其他请求改变，按失败处理由调用方决定是否重试
		return fmt.Errorf("问题状态已被并发修改，请刷新后重试")
	}
	return nil
}

// GetRootCauseAnalysis 获取根本原因分析
func (s *ProblemInvestigationService) GetRootCauseAnalysis(ctx context.Context, id int, tenantID int) (*dto.RootCauseAnalysisResponse, error) {
	analysis, err := s.repo.GetRootCauseAnalysis(ctx, id, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("根因分析不存在")
		}
		return nil, fmt.Errorf("查询根因分析失败: %v", err)
	}
	return analysis, nil
}

// GetProblemSolution 获取解决方案
func (s *ProblemInvestigationService) GetProblemSolution(ctx context.Context, id int, tenantID int) (*dto.ProblemSolutionResponse, error) {
	solution, err := s.repo.GetProblemSolution(ctx, id, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("解决方案不存在")
		}
		return nil, fmt.Errorf("查询解决方案失败: %v", err)
	}
	return solution, nil
}

// CreateProblemInvestigation 创建问题调查
func (s *ProblemInvestigationService) CreateProblemInvestigation(ctx context.Context, req *dto.CreateProblemInvestigationRequest, tenantID int) (*dto.ProblemInvestigationResponse, error) {
	// 检查问题是否存在
	if _, ok, err := s.repo.FetchProblemTitle(ctx, req.ProblemID, tenantID); err != nil {
		return nil, fmt.Errorf("查询问题失败: %v", err)
	} else if !ok {
		return nil, fmt.Errorf("问题不存在")
	}

	// 检查是否已存在调查记录
	exists, err := s.repo.InvestigationExistsByProblem(ctx, req.ProblemID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("查询调查记录失败: %v", err)
	}
	if exists {
		return nil, fmt.Errorf("该问题已存在调查记录")
	}

	// 创建调查记录
	var summaryPtr *string
	if req.InvestigationSummary != "" {
		summaryPtr = &req.InvestigationSummary
	}
	investigationID, err := s.repo.CreateInvestigation(ctx, req.ProblemID, req.InvestigatorID, req.EstimatedCompletionDate, summaryPtr, time.Now())
	if err != nil {
		return nil, fmt.Errorf("创建问题调查失败: %v", err)
	}

	// 获取调查者姓名
	investigatorName := "未知用户"
	if name, ok, err := s.users.GetUserName(ctx, req.InvestigatorID, tenantID); err == nil && ok {
		investigatorName = name
	}

	// 联动问题状态为"调查中"（经状态机校验，closed 终态拒绝联动；调查记录仍创建）
	if err := s.transitionProblemStatus(ctx, req.ProblemID, tenantID, "investigating"); err != nil {
		s.logger.Warnw("Failed to transition problem status to investigating",
			"problem_id", req.ProblemID, "error", err)
	}

	return &dto.ProblemInvestigationResponse{
		ID:                      investigationID,
		ProblemID:               req.ProblemID,
		InvestigatorID:          req.InvestigatorID,
		InvestigatorName:        investigatorName,
		Status:                  dto.InvestigationStatusInProgress,
		StartDate:               time.Now(),
		EstimatedCompletionDate: req.EstimatedCompletionDate,
		InvestigationSummary:    summaryPtr,
		CreatedAt:               time.Now(),
		UpdatedAt:               time.Now(),
	}, nil
}

// GetProblemInvestigation 获取问题调查详情
func (s *ProblemInvestigationService) GetProblemInvestigation(ctx context.Context, investigationID, tenantID int) (*dto.ProblemInvestigationResponse, error) {
	investigation, err := s.repo.GetProblemInvestigation(ctx, investigationID, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("问题调查不存在")
		}
		return nil, fmt.Errorf("获取问题调查失败: %v", err)
	}

	return investigation, nil
}

// UpdateProblemInvestigation 更新问题调查
func (s *ProblemInvestigationService) UpdateProblemInvestigation(ctx context.Context, investigationID int, req *dto.UpdateProblemInvestigationRequest, tenantID int) (*dto.ProblemInvestigationResponse, error) {
	// 检查调查记录是否存在
	investigation, err := s.GetProblemInvestigation(ctx, investigationID, tenantID)
	if err != nil {
		return nil, err
	}

	upd := investigationUpdate{
		Status:                  statusStringPtr(req.Status),
		EstimatedCompletionDate: req.EstimatedCompletionDate,
		ActualCompletionDate:    req.ActualCompletionDate,
		InvestigationSummary:    req.InvestigationSummary,
	}
	if err := s.repo.UpdateInvestigation(ctx, investigationID, upd, int64(tenantID), time.Now().Unix()); err != nil {
		return nil, fmt.Errorf("更新问题调查失败: %v", err)
	}

	// 如果状态更新为完成，联动问题状态为"已解决"（经状态机校验，closed 终态不可复活）
	if req.Status != nil && *req.Status == dto.InvestigationStatusCompleted {
		if err := s.transitionProblemStatus(ctx, investigation.ProblemID, tenantID, "resolved"); err != nil {
			s.logger.Warnw("Failed to transition problem status to resolved",
				"problem_id", investigation.ProblemID, "error", err)
		}
	}

	// 返回更新后的调查记录
	return s.GetProblemInvestigation(ctx, investigationID, tenantID)
}

// CreateInvestigationStep 创建调查步骤
func (s *ProblemInvestigationService) CreateInvestigationStep(ctx context.Context, req *dto.CreateInvestigationStepRequest, tenantID int) (*dto.InvestigationStepResponse, error) {
	// 检查调查记录是否存在
	problemID, err := s.repo.FetchInvestigationProblemID(ctx, req.InvestigationID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("查询调查记录失败: %v", err)
	}
	if problemID == 0 {
		return nil, fmt.Errorf("调查记录不存在")
	}

	// 创建调查步骤
	var notesPtr *string
	if req.Notes != "" {
		notesPtr = &req.Notes
	}
	stepID, err := s.repo.CreateInvestigationStep(ctx, &InvestigationStepCreate{
		InvestigationID: req.InvestigationID,
		StepNumber:      req.StepNumber,
		StepTitle:       req.StepTitle,
		StepDescription: req.StepDescription,
		AssignedTo:      req.AssignedTo,
		Notes:           notesPtr,
	}, time.Now())
	if err != nil {
		return nil, fmt.Errorf("创建调查步骤失败: %v", err)
	}

	// 获取分配人员姓名
	var assignedToName *string
	if req.AssignedTo != nil {
		if name, ok, err := s.users.GetUserName(ctx, *req.AssignedTo, tenantID); err == nil && ok {
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
		Notes:           notesPtr,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}, nil
}

// UpdateInvestigationStep 更新调查步骤
func (s *ProblemInvestigationService) UpdateInvestigationStep(ctx context.Context, stepID int, req *dto.UpdateInvestigationStepRequest, tenantID int) (*dto.InvestigationStepResponse, error) {
	// 检查步骤是否存在
	investigationID, err := s.repo.FetchStepInvestigationID(ctx, stepID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("查询调查步骤失败: %v", err)
	}
	if investigationID == 0 {
		return nil, fmt.Errorf("调查步骤不存在")
	}

	upd := stepUpdate{
		StepTitle:       req.StepTitle,
		StepDescription: req.StepDescription,
		Status:          stepStatusStringPtr(req.Status),
		AssignedTo:      req.AssignedTo,
		StartDate:       req.StartDate,
		CompletionDate:  req.CompletionDate,
		Notes:           req.Notes,
	}
	if err := s.repo.UpdateStep(ctx, stepID, upd, time.Now()); err != nil {
		return nil, fmt.Errorf("更新调查步骤失败: %v", err)
	}

	// 返回更新后的步骤
	return s.getInvestigationStep(ctx, stepID, tenantID)
}

// getInvestigationStep 获取调查步骤详情
func (s *ProblemInvestigationService) getInvestigationStep(ctx context.Context, stepID, tenantID int) (*dto.InvestigationStepResponse, error) {
	step, err := s.repo.GetInvestigationStep(ctx, stepID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("获取调查步骤失败: %v", err)
	}
	return step, nil
}

// CreateRootCauseAnalysis 创建根本原因分析
func (s *ProblemInvestigationService) CreateRootCauseAnalysis(ctx context.Context, req *dto.CreateRootCauseAnalysisRequest, tenantID int) (*dto.RootCauseAnalysisResponse, error) {
	// 检查问题是否存在
	if _, ok, err := s.repo.FetchProblemTitle(ctx, req.ProblemID, tenantID); err != nil {
		return nil, fmt.Errorf("查询问题失败: %v", err)
	} else if !ok {
		return nil, fmt.Errorf("问题不存在")
	}

	// 检查是否已存在根本原因分析
	exists, err := s.repo.RootCauseExistsByProblem(ctx, req.ProblemID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("查询根因分析失败: %v", err)
	}
	if exists {
		return nil, fmt.Errorf("该问题已存在根本原因分析")
	}

	// 创建根本原因分析
	confidence, err := parseConfidenceLevel(string(req.ConfidenceLevel))
	if err != nil {
		return nil, fmt.Errorf("置信度解析失败: %v", err)
	}
	var contributing *string
	if req.ContributingFactors != "" {
		contributing = &req.ContributingFactors
	}
	var evidence *string
	if req.Evidence != "" {
		evidence = &req.Evidence
	}
	analysisID, err := s.repo.CreateRootCauseAnalysis(ctx, &RootCauseAnalysisCreate{
		ProblemID:            req.ProblemID,
		AnalystID:            req.AnalystID,
		AnalysisMethod:       req.AnalysisMethod,
		RootCauseDescription: req.RootCauseDescription,
		ContributingFactors:  contributing,
		Evidence:             evidence,
		ConfidenceLevel:      confidence,
	}, time.Now())
	if err != nil {
		return nil, fmt.Errorf("创建根本原因分析失败: %v", err)
	}

	// 获取分析师姓名
	analystName := "未知用户"
	if name, ok, err := s.users.GetUserName(ctx, req.AnalystID, tenantID); err == nil && ok {
		analystName = name
	}

	return &dto.RootCauseAnalysisResponse{
		ID:                   analysisID,
		ProblemID:            req.ProblemID,
		AnalystID:            req.AnalystID,
		AnalystName:          analystName,
		AnalysisMethod:       req.AnalysisMethod,
		RootCauseDescription: req.RootCauseDescription,
		ContributingFactors:  contributing,
		Evidence:             evidence,
		ConfidenceLevel:      req.ConfidenceLevel,
		AnalysisDate:         time.Now(),
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}, nil
}

// CreateProblemSolution 创建问题解决方案
func (s *ProblemInvestigationService) CreateProblemSolution(ctx context.Context, req *dto.CreateProblemSolutionRequest, tenantID int) (*dto.ProblemSolutionResponse, error) {
	// 检查问题是否存在
	if _, ok, err := s.repo.FetchProblemTitle(ctx, req.ProblemID, tenantID); err != nil {
		return nil, fmt.Errorf("查询问题失败: %v", err)
	} else if !ok {
		return nil, fmt.Errorf("问题不存在")
	}

	var riskAssessment *string
	if req.RiskAssessment != "" {
		riskAssessment = &req.RiskAssessment
	}
	solutionID, err := s.repo.CreateProblemSolution(ctx, &ProblemSolutionCreate{
		ProblemID:            req.ProblemID,
		SolutionType:         string(req.SolutionType),
		SolutionDescription:  req.SolutionDescription,
		ProposedBy:           req.ProposedBy,
		Status:               string(dto.SolutionStatusProposed),
		Priority:             req.Priority,
		EstimatedEffortHours: req.EstimatedEffortHours,
		EstimatedCost:        req.EstimatedCost,
		RiskAssessment:       riskAssessment,
		ApprovalStatus:       "pending",
	}, time.Now())
	if err != nil {
		return nil, fmt.Errorf("创建问题解决方案失败: %v", err)
	}

	// 获取提议者姓名
	proposedByName := "未知用户"
	if name, ok, err := s.users.GetUserName(ctx, req.ProposedBy, tenantID); err == nil && ok {
		proposedByName = name
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
		RiskAssessment:       riskAssessment,
		ApprovalStatus:       "pending",
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}, nil
}

// GetProblemInvestigationSummary 获取问题调查摘要
func (s *ProblemInvestigationService) GetProblemInvestigationSummary(ctx context.Context, problemID, tenantID int) (*dto.ProblemInvestigationSummaryResponse, error) {
	// 检查问题是否存在
	if _, ok, err := s.repo.FetchProblemTitle(ctx, problemID, tenantID); err != nil {
		return nil, fmt.Errorf("查询问题失败: %v", err)
	} else if !ok {
		return nil, fmt.Errorf("问题不存在")
	}

	summary := &dto.ProblemInvestigationSummaryResponse{
		// 空数组而非 null，避免前端对 steps/solutions 等字段做空值防御时崩溃
		Steps:             []*dto.InvestigationStepResponse{},
		Solutions:         []*dto.ProblemSolutionResponse{},
		Implementations:   []*dto.SolutionImplementationResponse{},
		Relationships:     []*dto.ProblemRelationshipResponse{},
		KnowledgeArticles: []*dto.ProblemKnowledgeArticleResponse{},
	}

	// 获取调查记录
	if investigationID, ok, err := s.repo.FetchInvestigationIDByProblem(ctx, problemID, tenantID); err == nil && ok {
		if investigation, err := s.GetProblemInvestigation(ctx, investigationID, tenantID); err == nil {
			summary.Investigation = investigation
		}
	}

	// 获取调查步骤
	if summary.Investigation != nil {
		steps, err := s.repo.ListInvestigationSteps(ctx, summary.Investigation.ID, tenantID)
		if err == nil {
			summary.Steps = steps
		}
	}

	// 获取根本原因分析
	if id, ok, err := s.repo.FetchRootCauseAnalysisIDByProblem(ctx, problemID, tenantID); err == nil && ok {
		if analysis, err := s.repo.GetRootCauseAnalysis(ctx, id, tenantID); err == nil {
			summary.RootCauseAnalysis = analysis
		}
	}

	// 获取解决方案
	solutions, err := s.repo.ListProblemSolutions(ctx, problemID, tenantID)
	if err == nil {
		summary.Solutions = solutions
	}

	return summary, nil
}

// UpdateRootCauseAnalysis 更新根本原因分析
func (s *ProblemInvestigationService) UpdateRootCauseAnalysis(ctx context.Context, id int, req *dto.UpdateRootCauseAnalysisRequest, tenantID int) (*dto.RootCauseAnalysisResponse, error) {
	s.logger.Infow("Updating root cause analysis", "id", id, "tenant_id", tenantID)

	// 检查根因分析是否存在
	count, err := s.repo.CountRootCauseAnalysis(ctx, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("查询根因分析失败: %v", err)
	}
	if count == 0 {
		return nil, fmt.Errorf("根因分析不存在")
	}

	var confidence *float64
	if req.ConfidenceLevel != nil {
		confidence, err = parseConfidenceLevel(string(*req.ConfidenceLevel))
		if err != nil {
			return nil, fmt.Errorf("置信度解析失败: %v", err)
		}
	}

	upd := rootCauseUpdate{
		AnalysisMethod:       req.AnalysisMethod,
		RootCauseDescription: req.RootCauseDescription,
		ContributingFactors:  req.ContributingFactors,
		Evidence:             req.Evidence,
		ConfidenceLevel:      confidence,
		ReviewedBy:           req.ReviewedBy,
	}
	if err := s.repo.UpdateRootCauseAnalysis(ctx, id, tenantID, upd, time.Now()); err != nil {
		return nil, fmt.Errorf("更新根因分析失败: %v", err)
	}

	// 获取更新后的数据
	return s.GetRootCauseAnalysis(ctx, id, tenantID)
}

// DeleteRootCauseAnalysis 删除根本原因分析
func (s *ProblemInvestigationService) DeleteRootCauseAnalysis(ctx context.Context, id int, tenantID int) error {
	s.logger.Infow("Deleting root cause analysis", "id", id, "tenant_id", tenantID)

	deleted, err := s.repo.DeleteRootCauseAnalysis(ctx, id, tenantID)
	if err != nil {
		return fmt.Errorf("删除根因分析失败: %v", err)
	}
	if !deleted {
		return fmt.Errorf("根因分析不存在")
	}

	return nil
}

// UpdateProblemSolution 更新解决方案
func (s *ProblemInvestigationService) UpdateProblemSolution(ctx context.Context, id int, req *dto.UpdateProblemSolutionRequest, tenantID int) (*dto.ProblemSolutionResponse, error) {
	s.logger.Infow("Updating problem solution", "id", id, "tenant_id", tenantID)

	// 检查解决方案是否存在
	count, err := s.repo.CountProblemSolution(ctx, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("查询解决方案失败: %v", err)
	}
	if count == 0 {
		return nil, fmt.Errorf("解决方案不存在")
	}

	upd := solutionUpdate{
		SolutionType:         solutionTypeStringPtr(req.SolutionType),
		SolutionDescription:  req.SolutionDescription,
		Status:               solutionStatusStringPtr(req.Status),
		Priority:             req.Priority,
		EstimatedEffortHours: req.EstimatedEffortHours,
		EstimatedCost:        req.EstimatedCost,
		RiskAssessment:       req.RiskAssessment,
	}
	if err := s.repo.UpdateProblemSolution(ctx, id, tenantID, upd, time.Now()); err != nil {
		return nil, fmt.Errorf("更新解决方案失败: %v", err)
	}

	// 获取更新后的数据
	return s.GetProblemSolution(ctx, id, tenantID)
}

// DeleteProblemSolution 删除解决方案
func (s *ProblemInvestigationService) DeleteProblemSolution(ctx context.Context, id int, tenantID int) error {
	s.logger.Infow("Deleting problem solution", "id", id, "tenant_id", tenantID)

	deleted, err := s.repo.DeleteProblemSolution(ctx, id, tenantID)
	if err != nil {
		return fmt.Errorf("删除解决方案失败: %v", err)
	}
	if !deleted {
		return fmt.Errorf("解决方案不存在")
	}

	return nil
}

// ApproveSolution 审批解决方案
func (s *ProblemInvestigationService) ApproveSolution(ctx context.Context, id int, approverID int, approved bool, comment string, tenantID int) (*dto.ProblemSolutionResponse, error) {
	s.logger.Infow("Approving solution", "id", id, "approver_id", approverID, "approved", approved)

	// 检查解决方案是否存在
	count, err := s.repo.CountProblemSolution(ctx, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("查询解决方案失败: %v", err)
	}
	if count == 0 {
		return nil, fmt.Errorf("解决方案不存在")
	}

	now := time.Now()
	approvalStatus := "rejected"
	if approved {
		approvalStatus = "approved"
	}

	if err := s.repo.ApproveSolution(ctx, id, tenantID, approverID, approvalStatus, now); err != nil {
		return nil, fmt.Errorf("更新审批状态失败: %v", err)
	}

	// 如果批准，更新解决方案状态为待实施
	if approved {
		if err := s.repo.SetSolutionStatus(ctx, id, tenantID, string(dto.SolutionStatusPendingImplementation), now); err != nil {
			s.logger.Warnw("Failed to update solution status after approval", "error", err)
		}
	}

	return s.GetProblemSolution(ctx, id, tenantID)
}

// ============================================================
// 内部工具：把强类型别名转 string/*，方便传进 repository
// ============================================================

func statusStringPtr(s *dto.ProblemInvestigationStatus) *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}

func stepStatusStringPtr(s *dto.ProblemInvestigationStepStatus) *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}

func solutionTypeStringPtr(s *dto.SolutionType) *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}

func solutionStatusStringPtr(s *dto.SolutionStatus) *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}

// parseConfidenceLevel 把字符串置信度（"high" / "0.9" 等）归一为 0-1 的 float64。
//
// 历史数据里 ConfidenceLevel 是枚举字符串，但数据库列是 float64，
// 这里既支持直接是数字，也支持 "low/medium/high" 这种枚举（映射到 0.3/0.6/0.9）。
func parseConfidenceLevel(raw string) (*float64, error) {
	if raw == "" {
		return nil, nil
	}
	if f, err := strconv.ParseFloat(raw, 64); err == nil {
		if f < 0 || f > 1 {
			return nil, fmt.Errorf("置信度超出 [0,1] 范围: %v", f)
		}
		return &f, nil
	}
	switch strings.ToLower(raw) {
	case "low":
		v := 0.3
		return &v, nil
	case "medium":
		v := 0.6
		return &v, nil
	case "high":
		v := 0.9
		return &v, nil
	}
	return nil, fmt.Errorf("无法解析置信度: %q", raw)
}
