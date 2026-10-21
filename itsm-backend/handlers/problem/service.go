package problem

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/internal/datascope"

	"go.uber.org/zap"
)

type Service struct {
	repo      Repository
	validator AssigneeValidator
	logger    *zap.SugaredLogger
}

// AssigneeValidator 校验受指人属于同一租户且处于在职可用状态。
// 实现由 bootstrap 注入（当前复用 service.TicketService.ValidateAssignee，
// 与 ticket 域同一实现）——同一校验用例不复制第二份实现；
// placeholders 为 nil 时受指人引用必须 fail-closed，禁止跨租户关系注入。
type AssigneeValidator interface {
	ValidateAssignee(ctx context.Context, assigneeID, tenantID int) error
}

// IsNotFound keeps persistence error classification behind the domain service.
func (s *Service) IsNotFound(err error) bool {
	return ent.IsNotFound(err)
}

func NewService(repo Repository, validator AssigneeValidator, logger *zap.SugaredLogger) *Service {
	return &Service{
		repo:      repo,
		validator: validator,
		logger:    logger,
	}
}

// validateAssignee 统一受指人同租户/在职校验入口。
func (s *Service) validateAssignee(ctx context.Context, assigneeID, tenantID int) error {
	if assigneeID <= 0 {
		return fmt.Errorf("受理人 ID 无效")
	}
	if s.validator == nil {
		return fmt.Errorf("受理人校验器未配置，拒绝写入受指人引用")
	}
	if err := s.validator.ValidateAssignee(ctx, assigneeID, tenantID); err != nil {
		return fmt.Errorf("处理人不存在或不可用: %w", err)
	}
	return nil
}

func (s *Service) Create(ctx context.Context, tenantID int, p *Problem) (*Problem, error) {
	if strings.TrimSpace(p.Title) == "" {
		return nil, fmt.Errorf("problem title is required")
	}
	if !isValidProblemPriority(p.Priority) {
		return nil, fmt.Errorf("invalid problem priority: %s", p.Priority)
	}
	// AssigneeID 可指向任意租户的用户：写入前必须校验同租户 +在职，
	// 否则问题单（含根因等敏感内容）可以被静默指给别家租户员工。
	if p.AssigneeID != nil && *p.AssigneeID > 0 {
		if err := s.validateAssignee(ctx, *p.AssigneeID, tenantID); err != nil {
			return nil, err
		}
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

// LoadUserNames 委托 repo 批量加载 user 显示名映射。查询失败仅记日志并返回空映射，
// 由调用方决定是否降级（一般继续返回带 ID 的响应，让前端降级展示）。
func (s *Service) LoadUserNames(ctx context.Context, tenantID int, ids []int) map[int]string {
	m, err := s.repo.LoadUserNames(ctx, tenantID, ids)
	if err != nil {
		s.logger.Warnw("LoadUserNames failed", "error", err, "tenant_id", tenantID)
		return map[int]string{}
	}
	return m
}

func (s *Service) Update(ctx context.Context, tenantID int, id int, p *Problem, actorID int, actorRole string) (*Problem, error) {
	existing, err := s.repo.Get(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}
	// 写权限 ⊆ 读权限：普通角色仅可修改本人创建或受理的问题单。
	if !datascope.CanWriteResource(actorID, actorRole, existing.CreatedBy, existing.AssigneeID) {
		return nil, common.NewForbiddenError("无权限修改该问题单：仅创建人、受理人或管理员可操作")
	}

	// Update fields if they are set (non-zero/non-empty check in Handler or here)
	// Here assuming 'p' contains only fields to update usually, but domain entity isn't partial.
	// We merge changes here.
	// CAS 预期：读取时的状态。service 校验与仓储写入之间的并发窗口里若状态被
	// 推进（如另一请求已完成 resolved→closed），条件更新未命中 → 409 冲突，
	// 绝不允许用本次校验结论覆盖推进后的状态。
	expectedStatus := existing.Status

	if p.Title != "" {
		existing.Title = p.Title
	}
	if p.Description != "" {
		existing.Description = p.Description
	}
	if p.Status != "" {
		if !isValidProblemStatusTransition(existing.Status, p.Status) {
			// 业务状态机违规必须用 BusinessError(409) 表达，否则 handler 会误判为
			// 内部错误返回 500，前端会把合法的流程防护当成服务器故障。
			return nil, common.NewBusinessError(
				common.ConflictCode,
				"当前问题状态不允许此操作",
				fmt.Sprintf("%s -> %s", existing.Status, p.Status),
			)
		}
		// 先捕获前态再修改，时间戳分支才能正确判定"是否首次进入"。
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
			return nil, common.NewBusinessError(
				common.ParamErrorCode,
				"无效的问题优先级",
				p.Priority,
			)
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
		if *p.AssigneeID > 0 && existing.AssigneeID != nil && *existing.AssigneeID == *p.AssigneeID {
			// 与当前受理人相同：跳过校验，允许仅变更其他字段。
		} else {
			// AssigneeID 可指向任意租户的用户，必须与 Create 同强度的同租户校验。
			if err := s.validateAssignee(ctx, *p.AssigneeID, tenantID); err != nil {
				return nil, err
			}
		}
		existing.AssigneeID = p.AssigneeID
	}

	// 关闭是唯一需要「证据」的状态跃迁：必须已有解决方案与根因结论。
	// 放在字段合并之后校验，因为 CloseProblem 会在同一次调用里写入 resolution。
	if existing.Status == "closed" {
		if err := validateCloseEvidence(existing); err != nil {
			return nil, err
		}
	}

	result, err := s.repo.Update(ctx, existing, expectedStatus)
	if err != nil {
		if errors.Is(err, ErrProblemStatusMoved) {
			return nil, common.NewBusinessError(
				common.ConflictCode,
				"问题单状态已被并发修改，请刷新后重试",
				fmt.Sprintf("期望状态 '%s'", expectedStatus),
			)
		}
		return nil, err
	}
	return result, nil
}

// InvestigateProblem starts the investigation lifecycle for a problem.
func (s *Service) InvestigateProblem(ctx context.Context, tenantID, id, actorID int, actorRole string) (*Problem, error) {
	return s.Update(ctx, tenantID, id, &Problem{Status: "investigating"}, actorID, actorRole)
}

// UpdateRootCause records the confirmed root cause.
func (s *Service) UpdateRootCause(ctx context.Context, tenantID, id, actorID int, actorRole string, rootCause string) (*Problem, error) {
	rootCause = strings.TrimSpace(rootCause)
	if rootCause == "" {
		return nil, fmt.Errorf("rootCause is required")
	}
	return s.Update(ctx, tenantID, id, &Problem{RootCause: rootCause}, actorID, actorRole)
}

// UpdateSolution records a workaround and/or final resolution.
func (s *Service) UpdateSolution(ctx context.Context, tenantID, id, actorID int, actorRole string, workaround, resolution string) (*Problem, error) {
	workaround = strings.TrimSpace(workaround)
	resolution = strings.TrimSpace(resolution)
	if workaround == "" && resolution == "" {
		return nil, fmt.Errorf("solution, workaround or resolution is required")
	}
	return s.Update(ctx, tenantID, id, &Problem{Workaround: workaround, Resolution: resolution}, actorID, actorRole)
}

// CloseProblem closes a problem and optionally records its final resolution.
func (s *Service) CloseProblem(ctx context.Context, tenantID, id, actorID int, actorRole string, resolution string) (*Problem, error) {
	return s.Update(ctx, tenantID, id, &Problem{
		Status:     "closed",
		Resolution: strings.TrimSpace(resolution),
	}, actorID, actorRole)
}

// validateCloseEvidence 落实「问题关闭前必须明确根因和解决方案」这一商业不变量。
//
// 旧实现允许 investigating -> resolved -> closed 全程不填任何分析结论，
// 结果 KEDB 里堆满「已关闭但什么都没有」的问题单，既不能复用也不能审计。
// 根因确实无法确定时，要求把判定依据写进根因字段（例如「无法确定：日志已轮转」），
// 这样「无法确定」本身也是可审计的结论，而不是留空。
func validateCloseEvidence(p *Problem) error {
	if strings.TrimSpace(p.Resolution) == "" {
		return common.NewBusinessError(
			common.ConflictCode,
			"问题关闭前必须填写解决方案",
			"resolution is required before closing a problem",
		)
	}
	if strings.TrimSpace(p.RootCause) == "" {
		return common.NewBusinessError(
			common.ConflictCode,
			"问题关闭前必须明确根因；若无法确定，请在根因中写明判定依据",
			"root_cause is required before closing a problem",
		)
	}
	return nil
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

// Delete 删除问题单，受行级写权限约束（写权限 ⊆ 读权限）。
func (s *Service) Delete(ctx context.Context, id int, tenantID int, actorID int, actorRole string) error {
	existing, err := s.repo.Get(ctx, id, tenantID)
	if err != nil {
		return err
	}
	if !datascope.CanWriteResource(actorID, actorRole, existing.CreatedBy, existing.AssigneeID) {
		return common.NewForbiddenError("无权限删除该问题单：仅创建人、受理人或管理员可操作")
	}
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
