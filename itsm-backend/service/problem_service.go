//go:build problem
// +build problem

package service

import (
	"context"
	"fmt"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/problem"
	"time"

	"go.uber.org/zap"
)

// ProblemService 问题管理服务
type ProblemService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewProblemService 创建问题管理服务
func NewProblemService(client *ent.Client, logger *zap.SugaredLogger) *ProblemService {
	return &ProblemService{
		client: client,
		logger: logger,
	}
}

// CreateProblem 创建问题
func (s *ProblemService) CreateProblem(ctx context.Context, req *dto.CreateProblemRequest, tenantID int) (*ent.Problem, error) {
	problem, err := s.client.Problem.Create().
		SetTitle(req.Title).
		SetDescription(req.Description).
		SetPriority(req.Priority).
		SetStatus("open").
		SetCategory(req.Category).
		SetRootCause(req.RootCause).
		SetImpact(req.Impact).
		SetTenantID(tenantID).
		SetCreatedBy(req.CreatedBy).
		Save(ctx)

	if err != nil {
		return nil, fmt.Errorf("创建问题失败: %v", err)
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

	return &dto.ListProblemsResponse{
		Problems: problems,
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
