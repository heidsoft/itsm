package problem

import (
	"context"
	"fmt"
	"strings"
	"time"

	"itsm-backend/ent"
	"itsm-backend/handlers/common/datascope"

	"go.uber.org/zap"
)

type Service struct {
	repo   Repository
	logger *zap.SugaredLogger
}

// IsNotFound keeps persistence error classification behind the domain service.
func (s *Service) IsNotFound(err error) bool {
	return ent.IsNotFound(err)
}

func NewService(repo Repository, logger *zap.SugaredLogger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

func (s *Service) Create(ctx context.Context, tenantID int, p *Problem) (*Problem, error) {
	if strings.TrimSpace(p.Title) == "" {
		return nil, fmt.Errorf("problem title is required")
	}
	if !isValidProblemPriority(p.Priority) {
		return nil, fmt.Errorf("invalid problem priority: %s", p.Priority)
	}
	p.Title = strings.TrimSpace(p.Title)
	p.Status = "open"
	p.TenantID = tenantID
	return s.repo.Create(ctx, p)
}

func (s *Service) Get(ctx context.Context, id int, tenantID int) (*Problem, error) {
	return s.repo.Get(ctx, id, tenantID)
}

func (s *Service) GetWithAssociations(ctx context.Context, id int, tenantID int) (*Problem, error) {
	return s.repo.GetWithAssociations(ctx, id, tenantID)
}

func (s *Service) AddAssociations(ctx context.Context, tenantID, problemID int, relatedType string, relatedIDs []int) error {
	relatedIDs = uniquePositiveIDs(relatedIDs)
	if len(relatedIDs) == 0 {
		return fmt.Errorf("at least one related id is required")
	}
	return s.repo.AddAssociations(ctx, tenantID, problemID, relatedType, relatedIDs)
}

func (s *Service) RemoveAssociation(ctx context.Context, tenantID, problemID int, relatedType string, relatedID int) error {
	if relatedID <= 0 {
		return fmt.Errorf("invalid related id")
	}
	return s.repo.RemoveAssociation(ctx, tenantID, problemID, relatedType, relatedID)
}

// List 列出问题单。推广 ticket 的 DataScope 行级权限：
// 管理角色可见全租户，其余角色仅可见本人创建或分配给自己的问题单。
// currentUserID/currentRole 由 handler 从鉴权中间件注入的 user_id/role 取得。
func (s *Service) List(ctx context.Context, tenantID int, page, size int, filters map[string]interface{}, currentUserID int, currentRole string) ([]*Problem, int, error) {
	dataScope := datascope.DataScopeAll
	if !datascope.IsDataScopeAllRole(currentRole) {
		dataScope = datascope.DataScopeOwnedOrAssigned
	}
	return s.repo.List(ctx, tenantID, page, size, filters, dataScope, currentUserID)
}

func (s *Service) Update(ctx context.Context, tenantID int, id int, p *Problem) (*Problem, error) {
	existing, err := s.repo.Get(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	// Update fields if they are set (non-zero/non-empty check in Handler or here)
	// Here assuming 'p' contains only fields to update usually, but domain entity isn't partial.
	// We merge changes here.
	if p.Title != "" {
		existing.Title = p.Title
	}
	if p.Description != "" {
		existing.Description = p.Description
	}
	if p.Status != "" {
		if !isValidProblemStatusTransition(existing.Status, p.Status) {
			return nil, fmt.Errorf("invalid problem status transition: %s -> %s", existing.Status, p.Status)
		}
		// P1 修复：先捕获前态再修改，时间戳分支才能正确判定"是否首次进入"。
		// 旧实现在状态写入后才比对 `existing.Status != "resolved"`，此时两侧永远相等，
		// 导致 ResolvedAt 从未被设置、MTTR 永远为 0、closed_at 派生也走错分支。
		prevStatus := existing.Status
		existing.Status = p.Status
		now := time.Now()
		switch p.Status {
		case "resolved":
			if prevStatus != "resolved" {
				existing.ResolvedAt = &now
			}
			existing.ClosedAt = nil
		case "closed":
			existing.ClosedAt = &now
		case "investigating":
			if prevStatus == "resolved" || prevStatus == "closed" {
				existing.ResolvedAt = nil
				existing.ClosedAt = nil
			}
		}
	}
	if p.Priority != "" {
		if !isValidProblemPriority(p.Priority) {
			return nil, fmt.Errorf("invalid problem priority: %s", p.Priority)
		}
		existing.Priority = p.Priority
	}
	if p.Category != "" {
		existing.Category = p.Category
	}
	if p.RootCause != "" {
		existing.RootCause = p.RootCause
	}
	if p.Workaround != "" {
		existing.Workaround = p.Workaround
	}
	if p.Resolution != "" {
		existing.Resolution = p.Resolution
	}
	if p.Impact != "" {
		existing.Impact = p.Impact
	}
	if p.AssigneeID != nil {
		existing.AssigneeID = p.AssigneeID
	}

	return s.repo.Update(ctx, existing)
}

// InvestigateProblem starts the investigation lifecycle for a problem.
func (s *Service) InvestigateProblem(ctx context.Context, tenantID, id int) (*Problem, error) {
	return s.Update(ctx, tenantID, id, &Problem{Status: "investigating"})
}

// UpdateRootCause records the confirmed root cause.
func (s *Service) UpdateRootCause(ctx context.Context, tenantID, id int, rootCause string) (*Problem, error) {
	rootCause = strings.TrimSpace(rootCause)
	if rootCause == "" {
		return nil, fmt.Errorf("rootCause is required")
	}
	return s.Update(ctx, tenantID, id, &Problem{RootCause: rootCause})
}

// UpdateSolution records a workaround and/or final resolution.
func (s *Service) UpdateSolution(ctx context.Context, tenantID, id int, workaround, resolution string) (*Problem, error) {
	workaround = strings.TrimSpace(workaround)
	resolution = strings.TrimSpace(resolution)
	if workaround == "" && resolution == "" {
		return nil, fmt.Errorf("solution, workaround or resolution is required")
	}
	return s.Update(ctx, tenantID, id, &Problem{Workaround: workaround, Resolution: resolution})
}

// CloseProblem closes a problem and optionally records its final resolution.
func (s *Service) CloseProblem(ctx context.Context, tenantID, id int, resolution string) (*Problem, error) {
	return s.Update(ctx, tenantID, id, &Problem{
		Status:     "closed",
		Resolution: strings.TrimSpace(resolution),
	})
}

func isValidProblemPriority(priority string) bool {
	switch priority {
	case "low", "medium", "high", "critical":
		return true
	default:
		return false
	}
}

func isValidProblemStatusTransition(current, next string) bool {
	if current == next {
		return true
	}
	transitions := map[string]map[string]struct{}{
		"open":          {"investigating": {}, "identified": {}, "resolved": {}},
		"investigating": {"identified": {}, "resolved": {}},
		"identified":    {"investigating": {}, "resolved": {}},
		"resolved":      {"investigating": {}, "closed": {}},
		"closed":        {},
		// 兼容存量 in_progress 数据，仅允许进入规范状态。
		"in_progress": {"identified": {}, "resolved": {}},
	}
	allowed, ok := transitions[current]
	if !ok {
		return false
	}
	_, ok = allowed[next]
	return ok
}

func uniquePositiveIDs(ids []int) []int {
	seen := make(map[int]struct{}, len(ids))
	result := make([]int, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func (s *Service) Delete(ctx context.Context, id int, tenantID int) error {
	return s.repo.Delete(ctx, id, tenantID)
}

func (s *Service) GetStats(ctx context.Context, tenantID int) (*ProblemStats, error) {
	return s.repo.GetStats(ctx, tenantID)
}

// GetTrend computes problem trend data for the given period.
func (s *Service) GetTrend(ctx context.Context, tenantID int, startDate, endDate time.Time) (*ProblemTrendData, error) {
	problems, err := s.repo.GetAllForAnalytics(ctx, tenantID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	total := len(problems)
	resolved := 0
	open := 0
	categoryMap := make(map[string]int)
	priorityMap := make(map[string]int)
	monthlyMap := make(map[string]*MonthlyCount)
	var totalResolutionHours float64
	resolvedCount := 0

	for _, p := range problems {
		if p.Category != "" {
			categoryMap[p.Category]++
		}
		if p.Priority != "" {
			priorityMap[p.Priority]++
		}
		month := p.CreatedAt.Format("2006-01")
		mc, ok := monthlyMap[month]
		if !ok {
			mc = &MonthlyCount{Month: month}
			monthlyMap[month] = mc
		}
		mc.Count++
		switch p.Status {
		case "resolved", "closed":
			resolved++
			mc.Resolved++
			if p.ResolvedAt != nil {
				hours := p.ResolvedAt.Sub(p.CreatedAt).Hours()
				totalResolutionHours += hours
				resolvedCount++
			}
		case "open", "investigating", "identified", "in_progress":
			open++
			mc.Open++
		}
	}

	resolutionRate := 0.0
	if total > 0 {
		resolutionRate = float64(resolved) / float64(total) * 100
	}
	avgResolutionHours := 0.0
	if resolvedCount > 0 {
		avgResolutionHours = totalResolutionHours / float64(resolvedCount)
	}

	// Build top categories
	topCategories := make([]CategoryCount, 0, len(categoryMap))
	for cat, cnt := range categoryMap {
		topCategories = append(topCategories, CategoryCount{Category: cat, Count: cnt})
	}
	for i := 0; i < len(topCategories); i++ {
		for j := i + 1; j < len(topCategories); j++ {
			if topCategories[j].Count > topCategories[i].Count {
				topCategories[i], topCategories[j] = topCategories[j], topCategories[i]
			}
		}
	}
	if len(topCategories) > 5 {
		topCategories = topCategories[:5]
	}

	// Build monthly trend sorted
	monthlyTrend := make([]MonthlyCount, 0, len(monthlyMap))
	for _, mc := range monthlyMap {
		monthlyTrend = append(monthlyTrend, *mc)
	}
	for i := 0; i < len(monthlyTrend); i++ {
		for j := i + 1; j < len(monthlyTrend); j++ {
			if monthlyTrend[j].Month < monthlyTrend[i].Month {
				monthlyTrend[i], monthlyTrend[j] = monthlyTrend[j], monthlyTrend[i]
			}
		}
	}

	trendDirection := "stable"
	if len(monthlyTrend) >= 2 {
		last := monthlyTrend[len(monthlyTrend)-1].Count
		prev := monthlyTrend[len(monthlyTrend)-2].Count
		if last > prev {
			trendDirection = "increasing"
		} else if last < prev {
			trendDirection = "decreasing"
		}
	}

	return &ProblemTrendData{
		Period:                 startDate.Format("2006-01-02") + " ~ " + endDate.Format("2006-01-02"),
		TotalProblems:          total,
		ResolvedProblems:       resolved,
		OpenProblems:           open,
		ResolutionRate:         resolutionRate,
		AvgResolutionTimeHours: avgResolutionHours,
		CategoryBreakdown:      categoryMap,
		PriorityBreakdown:      priorityMap,
		TrendDirection:         trendDirection,
		TopCategories:          topCategories,
		MonthlyTrend:           monthlyTrend,
	}, nil
}

// GetHotspot computes problem hotspot data for the given period.
func (s *Service) GetHotspot(ctx context.Context, tenantID int, startDate, endDate time.Time) (*ProblemHotspotData, error) {
	problems, err := s.repo.GetAllForAnalytics(ctx, tenantID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	categoryMap := make(map[string]int)
	priorityMap := make(map[string]int)
	for _, p := range problems {
		if p.Category != "" {
			categoryMap[p.Category]++
		}
		if p.Priority != "" {
			priorityMap[p.Priority]++
		}
	}

	hotspots := make([]string, 0)
	avgPerCategory := 0.0
	if len(categoryMap) > 0 {
		totalCount := 0
		for cat, cnt := range categoryMap {
			totalCount += cnt
			if cnt > 1 {
				hotspots = append(hotspots, cat)
			}
		}
		avgPerCategory = float64(totalCount) / float64(len(categoryMap))
	}

	return &ProblemHotspotData{
		PeriodStart:       startDate.Format("2006-01-02"),
		PeriodEnd:         endDate.Format("2006-01-02"),
		CategoryBreakdown: categoryMap,
		PriorityBreakdown: priorityMap,
		Hotspots:          hotspots,
		AvgPerCategory:    avgPerCategory,
	}, nil
}
