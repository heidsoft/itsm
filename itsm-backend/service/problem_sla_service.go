package service

import (
	"context"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/problem"
	"itsm-backend/ent/sladefinition"

	"go.uber.org/zap"
)

// ProblemSLAService 问题SLA服务
type ProblemSLAService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewProblemSLAService 创建问题SLA服务
func NewProblemSLAService(client *ent.Client, logger *zap.SugaredLogger) *ProblemSLAService {
	return &ProblemSLAService{client: client, logger: logger}
}

// ProblemSLAInfoResult 问题SLA信息
type ProblemSLAInfoResult struct {
	ProblemID         int        `json:"problemId"`
	Priority          string     `json:"priority"`
	ResponseDeadline  *time.Time `json:"responseDeadline,omitempty"`
	ResolutionDeadline *time.Time `json:"resolutionDeadline,omitempty"`
	ResponseTimeUsed  int        `json:"responseTimeUsed"`   // 分钟
	ResolutionTimeUsed int       `json:"resolutionTimeUsed"` // 分钟
	ResponseBreached  bool       `json:"responseBreached"`
	ResolutionBreached bool      `json:"resolutionBreached"`
	SLAStatus         string     `json:"slaStatus"` // ok, warning, breached
}

// ProblemSLADeadlineResult 问题SLA截止时间计算结果
type ProblemSLADeadlineResult struct {
	ResponseDeadline   time.Time `json:"responseDeadline"`
	ResolutionDeadline time.Time `json:"resolutionDeadline"`
}

// 默认SLA时间 (分钟)
const (
	DefaultProblemResponseMinutes   = 240  // 4小时
	DefaultProblemResolutionMinutes = 1440 // 24小时
	WarningThresholdRatio           = 0.8  // 80%时进入warning
)

// CalculateSLADeadline 计算SLA截止时间
func (s *ProblemSLAService) CalculateSLADeadline(ctx context.Context, tenantID int, priority string) (*ProblemSLADeadlineResult, error) {
	responseMinutes, resolutionMinutes := s.getSLATimes(ctx, tenantID, priority)

	now := time.Now()
	return &ProblemSLADeadlineResult{
		ResponseDeadline:   now.Add(time.Duration(responseMinutes) * time.Minute),
		ResolutionDeadline: now.Add(time.Duration(resolutionMinutes) * time.Minute),
	}, nil
}

// GetProblemSLAInfo 获取问题SLA信息
func (s *ProblemSLAService) GetProblemSLAInfo(ctx context.Context, problemID int, tenantID int) (*ProblemSLAInfoResult, error) {
	p, err := s.client.Problem.Query().
		Where(problem.ID(problemID), problem.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	result := &ProblemSLAInfoResult{
		ProblemID: p.ID,
		Priority:  p.Priority,
	}

	now := time.Now()

	if p.ResponseDeadline != nil {
		result.ResponseDeadline = p.ResponseDeadline
		result.ResponseTimeUsed = int(now.Sub(p.CreatedAt).Minutes())
		if p.FirstResponseAt != nil {
			result.ResponseTimeUsed = int(p.FirstResponseAt.Sub(p.CreatedAt).Minutes())
		}
		result.ResponseBreached = now.After(*p.ResponseDeadline) && p.FirstResponseAt == nil
	}

	if p.ResolutionDeadline != nil {
		result.ResolutionDeadline = p.ResolutionDeadline
		if p.ResolvedAt != nil {
			result.ResolutionTimeUsed = int(p.ResolvedAt.Sub(p.CreatedAt).Minutes())
		} else {
			result.ResolutionTimeUsed = int(now.Sub(p.CreatedAt).Minutes())
		}
		result.ResolutionBreached = now.After(*p.ResolutionDeadline) && p.ResolvedAt == nil
	}

	// 计算SLA状态
	result.SLAStatus = "ok"
	if result.ResponseBreached || result.ResolutionBreached {
		result.SLAStatus = "breached"
	} else if s.isWarning(p, now) {
		result.SLAStatus = "warning"
	}

	return result, nil
}

// GetOverdueProblems 获取SLA逾期的问题
func (s *ProblemSLAService) GetOverdueProblems(ctx context.Context, tenantID int) ([]*ent.Problem, error) {
	now := time.Now()
	return s.client.Problem.Query().
		Where(
			problem.TenantID(tenantID),
			problem.StatusNEQ("closed"),
			problem.Or(
				problem.And(
					problem.ResponseDeadlineNotNil(),
					problem.ResponseDeadlineLT(now),
					problem.FirstResponseAtIsNil(),
				),
				problem.And(
					problem.ResolutionDeadlineNotNil(),
					problem.ResolutionDeadlineLT(now),
					problem.ResolvedAtIsNil(),
				),
			),
		).
		All(ctx)
}

// getSLATimes 获取SLA时间定义
func (s *ProblemSLAService) getSLATimes(ctx context.Context, tenantID int, priority string) (responseMinutes int, resolutionMinutes int) {
	responseMinutes = DefaultProblemResponseMinutes
	resolutionMinutes = DefaultProblemResolutionMinutes

	// 1. 精确匹配 (service_type + priority)
	defs, err := s.client.SLADefinition.Query().
		Where(
			sladefinition.TenantID(tenantID),
			sladefinition.ServiceType("problem"),
			sladefinition.Priority(priority),
		).
		All(ctx)
	if err == nil && len(defs) > 0 {
		d := defs[0]
		if d.ResponseTime > 0 {
			responseMinutes = d.ResponseTime
		}
		if d.ResolutionTime > 0 {
			resolutionMinutes = d.ResolutionTime
		}
		return
	}

	// 2. 类型默认 (只有 service_type)
	defs, err = s.client.SLADefinition.Query().
		Where(
			sladefinition.TenantID(tenantID),
			sladefinition.ServiceType("problem"),
		).
		All(ctx)
	if err == nil && len(defs) > 0 {
		d := defs[0]
		if d.ResponseTime > 0 {
			responseMinutes = d.ResponseTime
		}
		if d.ResolutionTime > 0 {
			resolutionMinutes = d.ResolutionTime
		}
	}

	return
}

// isWarning 判断是否接近SLA截止时间
func (s *ProblemSLAService) isWarning(p *ent.Problem, now time.Time) bool {
	if p.ResponseDeadline != nil && p.FirstResponseAt == nil {
		total := p.ResponseDeadline.Sub(p.CreatedAt)
		remaining := p.ResponseDeadline.Sub(now)
		if remaining > 0 && remaining.Seconds() < total.Seconds()*WarningThresholdRatio {
			return true
		}
	}
	if p.ResolutionDeadline != nil && p.ResolvedAt == nil {
		total := p.ResolutionDeadline.Sub(p.CreatedAt)
		remaining := p.ResolutionDeadline.Sub(now)
		if remaining > 0 && remaining.Seconds() < total.Seconds()*WarningThresholdRatio {
			return true
		}
	}
	return false
}
