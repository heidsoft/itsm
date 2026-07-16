package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/predicate"
	"itsm-backend/ent/problem"
	"itsm-backend/ent/processinstance"
	"itsm-backend/ent/user"

	"go.uber.org/zap"
)

// ProblemService 问题管理服务
type ProblemService struct {
	knownErrorService     *KnownErrorService
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
func (s *ProblemService) SetKnownErrorService(kes *KnownErrorService) {
	s.knownErrorService = kes
}

func (s *ProblemService) SetProcessTriggerService(triggerService ProcessTriggerServiceInterface) {
	s.processTriggerService = triggerService
}

// CreateProblem 创建问题
func (s *ProblemService) CreateProblem(ctx context.Context, req *dto.CreateProblemRequest, createdBy, tenantID int) (*dto.ProblemResponse, error) {
	s.logger.Infow("Creating problem", "title", req.Title, "tenant_id", tenantID, "created_by", createdBy)
	if err := validateProblemInput(req.Title, req.Priority); err != nil {
		return nil, err
	}
	creatorExists, err := s.client.User.Query().
		Where(user.IDEQ(createdBy), user.TenantIDEQ(tenantID), user.ActiveEQ(true)).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("验证问题创建人失败: %w", err)
	}
	if !creatorExists {
		return nil, fmt.Errorf("问题创建人不存在或不可用")
	}

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
		s.logger.Errorw("Failed to create problem", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("创建问题失败: %v", err)
	}

	s.logger.Infow("Problem created successfully", "id", problem.ID, "tenant_id", tenantID)

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

	return dto.ToProblemResponse(problem), nil
}

// GetProblem 获取问题详情
func (s *ProblemService) GetProblem(ctx context.Context, id int, tenantID int) (*dto.ProblemResponse, error) {
	s.logger.Infow("Getting problem", "id", id, "tenant_id", tenantID)

	problem, err := s.client.Problem.Query().
		Where(problem.ID(id), problem.TenantID(tenantID), problem.DeletedAtIsNil()).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			s.logger.Warnw("Problem not found", "id", id, "tenant_id", tenantID)
			return nil, fmt.Errorf("问题不存在")
		}
		s.logger.Errorw("Failed to get problem", "error", err, "id", id)
		return nil, fmt.Errorf("获取问题失败: %v", err)
	}

	return dto.ToProblemResponse(problem), nil
}

// ListProblems 获取问题列表
func (s *ProblemService) ListProblems(ctx context.Context, req *dto.ListProblemsRequest, tenantID int) (*dto.ListProblemsResponse, error) {
	s.logger.Infow("Listing problems", "tenant_id", tenantID, "page", req.Page, "page_size", req.PageSize)

	// 只查询未删除的问题
	query := s.client.Problem.Query().Where(
		problem.TenantID(tenantID),
		problem.DeletedAtIsNil(),
	)

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
	if pageSize > 200 {
		pageSize = 200
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
	dtoProblems := dto.ToProblemResponseList(problems)

	return &dto.ListProblemsResponse{
		Problems: dtoProblems,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// UpdateProblem 更新问题
func (s *ProblemService) UpdateProblem(ctx context.Context, id int, req *dto.UpdateProblemRequest, tenantID int) (*dto.ProblemResponse, error) {
	s.logger.Infow("Updating problem", "id", id, "tenant_id", tenantID)

	// 检查问题是否存在
	existing, err := s.client.Problem.Query().
		Where(problem.ID(id), problem.TenantID(tenantID), problem.DeletedAtIsNil()).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			s.logger.Warnw("Problem not found", "id", id, "tenant_id", tenantID)
			return nil, fmt.Errorf("问题不存在")
		}
		s.logger.Errorw("Failed to get problem for update", "error", err, "id", id)
		return nil, fmt.Errorf("获取问题失败: %v", err)
	}

	// 构建更新字段
	update := existing.Update()

	if req.Title != nil {
		update.SetTitle(*req.Title)
	}
	if req.Description != nil {
		update.SetDescription(*req.Description)
	}
	if req.Priority != nil {
		if !isValidProblemPriorityValue(*req.Priority) {
			return nil, fmt.Errorf("invalid problem priority: %s", *req.Priority)
		}
		update.SetPriority(*req.Priority)
	}
	if req.Status != nil {
		if existing.Status != *req.Status && !isValidProblemStatusTransition(existing.Status, *req.Status) {
			s.logger.Warnw("Invalid problem status transition", "id", id, "from", existing.Status, "to", *req.Status)
			return nil, fmt.Errorf("invalid problem status transition: %s -> %s", existing.Status, *req.Status)
		}
		update.SetStatus(*req.Status)
		// 如果状态变更为resolved，设置解决时间
		if *req.Status == "resolved" {
			update.SetResolvedAt(time.Now())
		}
		// 如果状态变更为closed，设置关闭时间
		if *req.Status == "closed" {
			update.SetClosedAt(time.Now())
		}
	}
	if req.Category != nil {
		update.SetCategory(*req.Category)
	}
	if req.RootCause != nil {
		update.SetRootCause(*req.RootCause)
	}
	if req.Impact != nil {
		update.SetImpact(*req.Impact)
	}
	update.SetUpdatedAt(time.Now())

	updatedProblem, err := update.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update problem", "error", err, "id", id, "tenant_id", tenantID)
		return nil, fmt.Errorf("更新问题失败: %v", err)
	}

	s.logger.Infow("Problem updated successfully", "id", id, "tenant_id", tenantID)
	return dto.ToProblemResponse(updatedProblem), nil
}

// DeleteProblem 删除问题（软删除）
func (s *ProblemService) DeleteProblem(ctx context.Context, id int, tenantID int) error {
	s.logger.Infow("Deleting problem", "id", id, "tenant_id", tenantID)

	// 使用软删除
	_, err := s.client.Problem.UpdateOneID(id).
		Where(problem.TenantID(tenantID), problem.DeletedAtIsNil()).
		SetDeletedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to delete problem", "error", err, "id", id, "tenant_id", tenantID)
		return fmt.Errorf("删除问题失败: %v", err)
	}

	s.logger.Infow("Problem deleted successfully", "id", id, "tenant_id", tenantID)
	return nil
}

// GetProblemStats 获取问题统计
func (s *ProblemService) GetProblemStats(ctx context.Context, tenantID int) (*dto.ProblemStatsResponse, error) {
	count := func(predicates ...predicate.Problem) (int, error) {
		return s.client.Problem.Query().
			Where(append([]predicate.Problem{problem.TenantIDEQ(tenantID), problem.DeletedAtIsNil()}, predicates...)...).
			Count(ctx)
	}
	total, err := count()
	if err != nil {
		return nil, err
	}

	open, err := count(problem.StatusEQ("open"))
	if err != nil {
		return nil, err
	}

	inProgress, err := count(problem.StatusIn("investigating", "in_progress"))
	if err != nil {
		return nil, err
	}

	resolved, err := count(problem.StatusEQ("resolved"))
	if err != nil {
		return nil, err
	}

	closed, err := count(problem.StatusEQ("closed"))
	if err != nil {
		return nil, err
	}

	highPriority, err := count(problem.PriorityIn("high", "critical"))
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
	p, err := s.client.Problem.Query().
		Where(
			problem.ID(problemID),
			problem.TenantID(tenantID),
		).
		Only(ctx)
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

	s.logger.Infow(
		"Workflow triggered for problem",
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

// isValidProblemStatusTransition checks problem status transitions
// state machine:
//
//	open          -> investigating, identified, resolved, closed
//	investigating -> identified, resolved, closed
//	identified    -> resolved, closed, investigating
//	resolved      -> closed, investigating (reopen)
//	closed        -> terminal
func isValidProblemStatusTransition(currentStatus, newStatus string) bool {
	if currentStatus == newStatus {
		return true
	}
	validTransitions := map[string][]string{
		"open":          {"investigating", "identified", "resolved", "closed"},
		"investigating": {"identified", "resolved", "closed"},
		"identified":    {"resolved", "closed", "investigating"},
		"resolved":      {"closed", "investigating"},
		"closed":        {},
		"in_progress":   {"identified", "resolved", "closed"},
	}
	allowed, ok := validTransitions[currentStatus]
	if !ok {
		return false
	}
	for _, st := range allowed {
		if st == newStatus {
			return true
		}
	}
	return false
}

func validateProblemInput(title, priority string) error {
	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("问题标题不能为空")
	}
	if !isValidProblemPriorityValue(priority) {
		return fmt.Errorf("invalid problem priority: %s", priority)
	}
	return nil
}

func isValidProblemPriorityValue(priority string) bool {
	switch priority {
	case "low", "medium", "high", "critical":
		return true
	default:
		return false
	}
}

// CreateKnownErrorFromProblem 从问题创建已知错误
func (s *ProblemService) CreateKnownErrorFromProblem(ctx context.Context, problemID int, createdBy int, req *dto.KEDBCreateRequest) (*dto.KEDBResponse, error) {
	s.logger.Infow("Creating known error from problem", "problemID", problemID, "createdBy", createdBy)
	if s.knownErrorService == nil {
		return nil, fmt.Errorf("known error service is not configured")
	}
	creator, err := s.client.User.Query().
		Where(user.IDEQ(createdBy), user.ActiveEQ(true)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("creator not found")
		}
		return nil, fmt.Errorf("failed to validate creator: %w", err)
	}

	// 获取问题
	problem, err := s.client.Problem.Query().
		Where(problem.IDEQ(problemID), problem.TenantIDEQ(creator.TenantID), problem.DeletedAtIsNil()).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("problem not found")
		}
		return nil, fmt.Errorf("failed to get problem: %w", err)
	}

	// 构建创建已知错误的请求
	createReq := dto.KEDBCreateRequest{
		Title:            problem.Title,
		Description:      problem.Description,
		RootCause:        problem.RootCause,
		Category:         problem.Category,
		Severity:         mapPriorityToSeverity(problem.Priority),
		AffectedProducts: []string{},
		AffectedCIs:      []string{},
		Keywords:         []string{problem.Title},
		ProblemID:        &problemID,
	}

	// 如果用户传入了自定义请求，覆盖默认值
	if req != nil {
		if req.Title != "" {
			createReq.Title = req.Title
		}
		if req.Description != "" {
			createReq.Description = req.Description
		}
		if req.Symptoms != "" {
			createReq.Symptoms = req.Symptoms
		}
		if req.RootCause != "" {
			createReq.RootCause = req.RootCause
		}
		if req.Workaround != "" {
			createReq.Workaround = req.Workaround
		}
		if req.Resolution != "" {
			createReq.Resolution = req.Resolution
		}
		if req.Category != "" {
			createReq.Category = req.Category
		}
		if req.Severity != "" {
			createReq.Severity = req.Severity
		}
		if len(req.AffectedProducts) > 0 {
			createReq.AffectedProducts = req.AffectedProducts
		}
		if len(req.AffectedCIs) > 0 {
			createReq.AffectedCIs = req.AffectedCIs
		}
		if len(req.Keywords) > 0 {
			createReq.Keywords = req.Keywords
		}
	}

	// 验证必填字段
	if createReq.Title == "" {
		return nil, fmt.Errorf("title is required")
	}

	// 创建已知错误 - 需要先转换为CreateKnownErrorRequest
	// 注意：这里KnownErrorService需要的是CreateKnownErrorRequest，我先手动映射
	knownError, err := s.knownErrorService.CreateKnownError(ctx, dto.CreateKnownErrorRequest{
		Title:            createReq.Title,
		Description:      createReq.Description,
		Symptoms:         createReq.Symptoms,
		RootCause:        createReq.RootCause,
		Workaround:       createReq.Workaround,
		Resolution:       createReq.Resolution,
		Status:           dto.KnownErrorStatusDraft, // 默认是草稿状态，需要审批
		Category:         createReq.Category,
		Severity:         createReq.Severity,
		AffectedProducts: createReq.AffectedProducts,
		AffectedCIs:      createReq.AffectedCIs,
		Keywords:         createReq.Keywords,
		CreatedBy:        createdBy,
		TenantID:         problem.TenantID,
	})
	if err != nil {
		s.logger.Errorw("Failed to create known error from problem", "error", err)
		return nil, fmt.Errorf("failed to create known error: %w", err)
	}

	// 关联已知错误到问题
	err = s.knownErrorService.LinkKnownErrorToProblem(ctx, knownError.ID, problemID, problem.TenantID)
	if err != nil {
		if deleteErr := s.client.KnownError.DeleteOneID(knownError.ID).Exec(ctx); deleteErr != nil {
			s.logger.Errorw("Failed to compensate orphan known error", "knownErrorID", knownError.ID, "error", deleteErr)
		}
		return nil, fmt.Errorf("failed to link known error to problem: %w", err)
	}

	// 转换为响应
	return toKEDBResponse(knownError), nil
}

// mapPriorityToSeverity 将问题优先级映射为已知错误严重程度
func mapPriorityToSeverity(priority string) string {
	switch priority {
	case "critical":
		return dto.KnownErrorSeverityCritical
	case "high":
		return dto.KnownErrorSeverityHigh
	case "medium":
		return dto.KnownErrorSeverityMedium
	case "low":
		return dto.KnownErrorSeverityLow
	default:
		return dto.KnownErrorSeverityMedium
	}
}

// toKEDBResponse 将ent.KnownError转换为dto.KEDBResponse
func toKEDBResponse(ke *ent.KnownError) *dto.KEDBResponse {
	return &dto.KEDBResponse{
		ID:               ke.ID,
		Title:            ke.Title,
		Description:      ke.Description,
		Symptoms:         ke.Symptoms,
		RootCause:        ke.RootCause,
		Workaround:       ke.Workaround,
		Resolution:       ke.Resolution,
		Status:           ke.Status,
		Category:         ke.Category,
		Severity:         ke.Severity,
		AffectedProducts: ke.AffectedProducts,
		AffectedCIs:      ke.AffectedCis,
		Keywords:         ke.Keywords,
		OccurrenceCount:  ke.OccurrenceCount,
		CreatedBy:        ke.CreatedBy,
		TenantID:         ke.TenantID,
		CreatedAt:        ke.CreatedAt,
		UpdatedAt:        ke.UpdatedAt,
	}
}
