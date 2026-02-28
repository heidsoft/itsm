package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/problem"
	"itsm-backend/ent/processinstance"

	"go.uber.org/zap"
)

// ProblemService 问题管理服务
type ProblemService struct {
	client                *ent.Client
	logger                *zap.SugaredLogger
	processTriggerService ProcessTriggerServiceInterface
}

// NewProblemService 创建问题管理服务
func NewProblemService(client *ent.Client, logger *zap.SugaredLogger) *ProblemService {
	return &ProblemService{
		client: client,
		logger: logger,
	}
}

// SetProcessTriggerService 设置流程触发服务
func (s *ProblemService) SetProcessTriggerService(triggerService ProcessTriggerServiceInterface) {
	s.processTriggerService = triggerService
}

// CreateProblem 创建问题
func (s *ProblemService) CreateProblem(ctx context.Context, req *dto.CreateProblemRequest, createdBy, tenantID int) (*ent.Problem, error) {
	problem, err := s.client.Problem.Create().
		SetTitle(req.Title).
		SetDescription(req.Description).
		SetPriority(req.Priority).
		SetStatus("open").
		SetCategory(req.Category).
		SetRootCause(req.RootCause).
		SetImpact(req.Impact).
		SetTenantID(tenantID).
		SetCreatedBy(createdBy).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建问题失败: %v", err)
	}

	// 触发BPMN工作流（异步执行，不阻塞问题创建）
	if s.processTriggerService != nil {
		go func() {
			workflowCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := s.triggerWorkflowForProblem(workflowCtx, problem.ID, tenantID); err != nil {
				s.logger.Warnw("Failed to trigger workflow for problem", "error", err, "problem_id", problem.ID)
			}
		}()
	}

	return problem, nil
}

// GetProblem 获取问题详情
func (s *ProblemService) GetProblem(ctx context.Context, id int, tenantID int) (*ent.Problem, error) {
	problem, err := s.client.Problem.Query().
		Where(problem.ID(id), problem.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取问题失败: %v", err)
	}

	return problem, nil
}

// ListProblems 获取问题列表
func (s *ProblemService) ListProblems(ctx context.Context, req *dto.ListProblemsRequest, tenantID int) (*dto.ListProblemsResponse, error) {
	query := s.client.Problem.Query().Where(problem.TenantID(tenantID))

	// 应用过滤条件
	if req.Status != "" {
		query = query.Where(problem.StatusEQ(req.Status))
	}
	if req.Priority != "" {
		query = query.Where(problem.PriorityEQ(req.Priority))
	}
	if req.Category != "" {
		query = query.Where(problem.CategoryEQ(req.Category))
	}
	if req.Keyword != "" {
		query = query.Where(problem.Or(
			problem.TitleContains(req.Keyword),
			problem.DescriptionContains(req.Keyword),
		))
	}

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取问题总数失败: %v", err)
	}

	// 分页
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	problems, err := query.
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Order(ent.Desc(problem.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取问题列表失败: %v", err)
	}

	// map ent -> dto
	dtoProblems := make([]dto.ProblemResponse, 0, len(problems))
	for _, p := range problems {
		item := dto.ProblemResponse{
			ID:          p.ID,
			Title:       p.Title,
			Description: p.Description,
			Status:      p.Status,
			Priority:    p.Priority,
			Category:    p.Category,
			RootCause:   p.RootCause,
			Impact:      p.Impact,
			CreatedBy:   p.CreatedBy,
			TenantID:    p.TenantID,
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
		}
		if p.AssigneeID != 0 {
			val := p.AssigneeID
			item.AssigneeID = &val
		}
		dtoProblems = append(dtoProblems, item)
	}

	return &dto.ListProblemsResponse{
		Problems: dtoProblems,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// UpdateProblem 更新问题
func (s *ProblemService) UpdateProblem(ctx context.Context, id int, req *dto.UpdateProblemRequest, tenantID int) (*ent.Problem, error) {
	problem, err := s.client.Problem.UpdateOneID(id).
		SetTitle(req.Title).
		SetDescription(req.Description).
		SetPriority(req.Priority).
		SetStatus(req.Status).
		SetCategory(req.Category).
		SetRootCause(req.RootCause).
		SetImpact(req.Impact).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("更新问题失败: %v", err)
	}

	return problem, nil
}

// DeleteProblem 删除问题
func (s *ProblemService) DeleteProblem(ctx context.Context, id int, tenantID int) error {
	err := s.client.Problem.DeleteOneID(id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("删除问题失败: %v", err)
	}

	return nil
}

// GetProblemStats 获取问题统计
func (s *ProblemService) GetProblemStats(ctx context.Context, tenantID int) (*dto.ProblemStatsResponse, error) {
	query := s.client.Problem.Query().Where(problem.TenantID(tenantID))

	total, err := query.Count(ctx)
	if err != nil {
		return nil, err
	}

	open, err := query.Where(problem.StatusEQ("open")).Count(ctx)
	if err != nil {
		return nil, err
	}

	inProgress, err := query.Where(problem.StatusEQ("in_progress")).Count(ctx)
	if err != nil {
		return nil, err
	}

	resolved, err := query.Where(problem.StatusEQ("resolved")).Count(ctx)
	if err != nil {
		return nil, err
	}

	closed, err := query.Where(problem.StatusEQ("closed")).Count(ctx)
	if err != nil {
		return nil, err
	}

	highPriority, err := query.Where(problem.PriorityEQ("high")).Count(ctx)
	if err != nil {
		return nil, err
	}

	return &dto.ProblemStatsResponse{
		Total:        total,
		Open:         open,
		InProgress:   inProgress,
		Resolved:     resolved,
		Closed:       closed,
		HighPriority: highPriority,
	}, nil
}

// triggerWorkflowForProblem 为问题触发工作流
func (s *ProblemService) triggerWorkflowForProblem(ctx context.Context, problemID int, tenantID int) error {
	// 获取问题信息
	p, err := s.client.Problem.Get(ctx, problemID)
	if err != nil {
		return fmt.Errorf("failed to get problem: %w", err)
	}

	// 构建流程变量
	variables := map[string]interface{}{
		"problem_id":  p.ID,
		"title":       p.Title,
		"description": p.Description,
		"priority":    p.Priority,
		"status":      p.Status,
		"category":    p.Category,
		"impact":      p.Impact,
		"root_cause":  p.RootCause,
		"created_by":  p.CreatedBy,
	}

	// 触发问题管理流程
	triggerReq := &dto.ProcessTriggerRequest{
		BusinessType:         dto.BusinessTypeProblem,
		BusinessID:           problemID,
		ProcessDefinitionKey: "problem_management_flow",
		Variables:            variables,
		TriggeredBy:          fmt.Sprintf("%d", p.CreatedBy),
		TriggeredAt:          time.Now(),
		TenantID:             tenantID,
	}

	resp, err := s.processTriggerService.TriggerProcess(ctx, triggerReq)
	if err != nil {
		return fmt.Errorf("failed to trigger workflow: %w", err)
	}

	s.logger.Infow("Workflow triggered for problem",
		"problem_id", problemID,
		"process_instance_id", resp.ProcessInstanceID,
	)

	return nil
}

// GetWorkflowStatus 获取问题关联的流程状态
func (s *ProblemService) GetWorkflowStatus(ctx context.Context, problemID int, tenantID int) (*dto.ProcessTriggerResponse, error) {
	businessKey := fmt.Sprintf("problem:%d", problemID)

	processInstance, err := s.client.ProcessInstance.Query().
		Where(
			processinstance.BusinessKey(businessKey),
			processinstance.TenantID(tenantID),
		).
		WithDefinition().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("未找到问题关联的流程实例")
		}
		return nil, fmt.Errorf("查询流程实例失败: %w", err)
	}

	processDefName := ""
	if processInstance.Edges.Definition != nil {
		processDefName = processInstance.Edges.Definition.Name
	}

	return &dto.ProcessTriggerResponse{
		ProcessInstanceID:     processInstance.ID,
		ProcessDefinitionKey:  processInstance.ProcessDefinitionKey,
		ProcessDefinitionName: processDefName,
		BusinessKey:           processInstance.BusinessKey,
		Status:                s.mapProcessStatus(processInstance.Status),
		CurrentActivityID:     processInstance.CurrentActivityID,
		CurrentActivityName:   processInstance.CurrentActivityName,
		StartTime:             processInstance.StartTime,
		EndTime:               &processInstance.EndTime,
	}, nil
}

// mapProcessStatus 映射流程状态
func (s *ProblemService) mapProcessStatus(status string) dto.ProcessStatus {
	switch status {
	case "running", "active":
		return dto.ProcessStatusRunning
	case "completed":
		return dto.ProcessStatusCompleted
	case "suspended":
		return dto.ProcessStatusSuspended
	case "terminated", "cancelled":
		return dto.ProcessStatusTerminated
	default:
		return dto.ProcessStatusPending
	}
}
