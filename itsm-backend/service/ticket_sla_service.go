package service

import (
	"context"
	"strconv"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/ent/sladefinition"
	"itsm-backend/ent/ticket"

	"go.uber.org/zap"
)

// TicketSLAServiceInterface 工单SLA服务接口
type TicketSLAServiceInterface interface {
	// GetTicketSLAInfo 获取工单SLA信息
	GetTicketSLAInfo(ctx context.Context, ticketID int, tenantID int) (*TicketSLAInfoResult, error)
	// GetOverdueTickets 获取逾期工单
	GetOverdueTickets(ctx context.Context, tenantID int) ([]*ent.Ticket, error)
	// GetTicketStats 获取工单统计
	GetTicketStats(ctx context.Context, tenantID int) (*TicketStats, error)
	// CalculateSLADeadline 计算SLA截止时间
	CalculateSLADeadline(ctx context.Context, tenantID int, ticketType, priority string) (*SLADeadlineResult, error)
	// CalculateSLADeadlineFromRequest 根据请求参数计算SLA截止时间（包含SLADefinitionID）
	CalculateSLADeadlineFromRequest(ctx context.Context, tenantID int, ticketType, priority string) (*SLADeadlineResult, error)
	// AdjustToBusinessHours 调整时间到工作时间
	AdjustToBusinessHours(t time.Time) time.Time
}

// TicketSLAInfoResult 工单SLA信息（计算结果）
type TicketSLAInfoResult struct {
	TicketID           int        `json:"ticketId"`
	TicketNumber       string     `json:"ticketNumber"`
	Priority           string     `json:"priority"`
	TicketType         string     `json:"ticketType"`
	ResponseDeadline   *time.Time `json:"responseDeadline"`
	ResolutionDeadline *time.Time `json:"resolutionDeadline"`
	ResponseTimeUsed   int        `json:"responseTimeUsed"`   // 分钟
	ResolutionTimeUsed int        `json:"resolutionTimeUsed"` // 分钟
	ResponseBreached   bool       `json:"responseBreached"`
	ResolutionBreached bool       `json:"resolutionBreached"`
	SLAStatus          string     `json:"slaStatus"` // ok, warning, breached
}

// SLADeadlineResult SLA截止时间计算结果
type SLADeadlineResult struct {
	SLADefinitionID    int
	ResponseDeadline   *time.Time
	ResolutionDeadline *time.Time
	BusinessHoursOnly  bool
}

// TicketStats 工单统计
type TicketStats struct {
	TotalTickets      int `json:"totalTickets"`
	OpenTickets       int `json:"openTickets"`
	InProgressTickets int `json:"inProgressTickets"`
	ResolvedTickets   int `json:"resolvedTickets"`
	ClosedTickets     int `json:"closedTickets"`
	OverdueTickets    int `json:"overdueTickets"`
	BreachedTickets   int `json:"breachedTickets"`
}

// TicketSLAService 工单SLA服务
type TicketSLAService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewTicketSLAService 创建工单SLA服务
func NewTicketSLAService(client *ent.Client, logger *zap.SugaredLogger) *TicketSLAService {
	return &TicketSLAService{
		client: client,
		logger: logger,
	}
}

// GetTicketSLAInfo 获取工单SLA信息
func (s *TicketSLAService) GetTicketSLAInfo(ctx context.Context, ticketID int, tenantID int) (*TicketSLAInfoResult, error) {
	// 查询工单
	t, err := s.client.Ticket.Query().
		Where(ticket.IDEQ(ticketID), ticket.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		s.logger.Errorw("Failed to find ticket", "ticketID", ticketID, "error", err)
		return nil, err
	}

	// 计算已用时间
	responseTimeUsed := int(time.Since(t.CreatedAt).Minutes())
	resolutionTimeUsed := int(time.Since(t.CreatedAt).Minutes())

	// 如果已有首次响应时间或解决时间，使用实际时间
	if !t.FirstResponseAt.IsZero() {
		responseTimeUsed = int(t.FirstResponseAt.Sub(t.CreatedAt).Minutes())
	}
	if !t.ResolvedAt.IsZero() {
		resolutionTimeUsed = int(t.ResolvedAt.Sub(t.CreatedAt).Minutes())
	}

	// 获取SLA定义
	slaDef, err := s.getSLADefinition(ctx, tenantID, t.Type, t.Priority)
	if err != nil {
		s.logger.Warnw("Failed to get SLA definition", "error", err)
		// 返回没有SLA信息的结果
		return &TicketSLAInfoResult{
			TicketID:           t.ID,
			TicketNumber:       t.TicketNumber,
			Priority:           t.Priority,
			TicketType:         t.Type,
			ResponseTimeUsed:   responseTimeUsed,
			ResolutionTimeUsed: resolutionTimeUsed,
			SLAStatus:          "unknown",
		}, nil
	}

	// 计算截止时间。
	// 阻断7/C-8 修复：统一读取 slaDef.BusinessHours 配置，与 CalculateSLADeadlineFromRequest
	// 使用同一口径，消除"建单落库调整 / 查询展示不调整"的两路径结论相反问题。
	businessHoursOnly := len(slaDef.BusinessHours) > 0
	var responseDeadline, resolutionDeadline *time.Time
	if slaDef.ResponseTime > 0 {
		respDeadline := s.calculateDeadline(t.CreatedAt, slaDef.ResponseTime, businessHoursOnly)
		responseDeadline = &respDeadline
	}
	if slaDef.ResolutionTime > 0 {
		resDeadline := s.calculateDeadline(t.CreatedAt, slaDef.ResolutionTime, businessHoursOnly)
		resolutionDeadline = &resDeadline
	}

	// 判断是否违规
	responseBreached := false
	resolutionBreached := false
	slaStatus := "ok"

	if responseDeadline != nil && time.Now().After(*responseDeadline) {
		responseBreached = true
		slaStatus = "breached"
	}

	if resolutionDeadline != nil && time.Now().After(*resolutionDeadline) {
		resolutionBreached = true
		slaStatus = "breached"
	}

	// 检查警告状态（默认30分钟警告）
	if !responseBreached && !resolutionBreached && responseDeadline != nil {
		timeLeft := time.Until(*responseDeadline)
		if timeLeft.Minutes() < 30 {
			slaStatus = "warning"
		}
	}

	return &TicketSLAInfoResult{
		TicketID:           t.ID,
		TicketNumber:       t.TicketNumber,
		Priority:           t.Priority,
		TicketType:         t.Type,
		ResponseDeadline:   responseDeadline,
		ResolutionDeadline: resolutionDeadline,
		ResponseTimeUsed:   responseTimeUsed,
		ResolutionTimeUsed: resolutionTimeUsed,
		ResponseBreached:   responseBreached,
		ResolutionBreached: resolutionBreached,
		SLAStatus:          slaStatus,
	}, nil
}

// GetOverdueTickets 获取逾期工单
func (s *TicketSLAService) GetOverdueTickets(ctx context.Context, tenantID int) ([]*ent.Ticket, error) {
	// 获取所有未关闭的工单
	tickets, err := s.client.Ticket.Query().
		Where(
			ticket.TenantID(tenantID),
			ticket.StatusNEQ(common.TicketStatusClosed),
			ticket.StatusNEQ(common.TicketStatusResolved),
		).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to query tickets", "error", err)
		return nil, err
	}

	// 筛选逾期工单
	var overdueTickets []*ent.Ticket
	now := time.Now()

	for _, t := range tickets {
		// 获取SLA定义
		slaDef, err := s.getSLADefinition(ctx, tenantID, t.Type, t.Priority)
		if err != nil {
			continue
		}

		if slaDef.ResolutionTime > 0 {
			deadline := s.calculateDeadline(t.CreatedAt, slaDef.ResolutionTime, false)
			if now.After(deadline) {
				overdueTickets = append(overdueTickets, t)
			}
		}
	}

	return overdueTickets, nil
}

// GetTicketStats 获取工单统计
func (s *TicketSLAService) GetTicketStats(ctx context.Context, tenantID int) (*TicketStats, error) {
	stats := &TicketStats{}

	// 统计总数
	total, err := s.client.Ticket.Query().
		Where(ticket.TenantID(tenantID)).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	stats.TotalTickets = total

	// 统计各状态数量
	openCount, err := s.client.Ticket.Query().
		Where(ticket.TenantID(tenantID), ticket.Status(common.TicketStatusOpen)).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	stats.OpenTickets = openCount

	inProgressCount, err := s.client.Ticket.Query().
		Where(ticket.TenantID(tenantID), ticket.Status(common.TicketStatusInProgress)).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	stats.InProgressTickets = inProgressCount

	resolvedCount, err := s.client.Ticket.Query().
		Where(ticket.TenantID(tenantID), ticket.Status(common.TicketStatusResolved)).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	stats.ResolvedTickets = resolvedCount

	closedCount, err := s.client.Ticket.Query().
		Where(ticket.TenantID(tenantID), ticket.Status(common.TicketStatusClosed)).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	stats.ClosedTickets = closedCount

	// 统计逾期工单
	overdueTickets, err := s.GetOverdueTickets(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	stats.OverdueTickets = len(overdueTickets)

	return stats, nil
}

// CalculateSLADeadline 计算SLA截止时间
func (s *TicketSLAService) CalculateSLADeadline(ctx context.Context, tenantID int, ticketType, priority string) (*SLADeadlineResult, error) {
	slaDef, err := s.getSLADefinition(ctx, tenantID, ticketType, priority)
	if err != nil {
		return nil, err
	}

	result := &SLADeadlineResult{}

	now := time.Now()

	if slaDef.ResponseTime > 0 {
		respDeadline := s.calculateDeadline(now, slaDef.ResponseTime, false)
		result.ResponseDeadline = &respDeadline
	}

	if slaDef.ResolutionTime > 0 {
		resDeadline := s.calculateDeadline(now, slaDef.ResolutionTime, false)
		result.ResolutionDeadline = &resDeadline
	}

	return result, nil
}

// getSLADefinition 获取SLA定义
func (s *TicketSLAService) getSLADefinition(ctx context.Context, tenantID int, ticketType, priority string) (*ent.SLADefinition, error) {
	// 尝试查找匹配的类型和优先级
	sla, err := s.client.SLADefinition.Query().
		Where(
			sladefinition.TenantID(tenantID),
			sladefinition.ServiceType(ticketType),
			sladefinition.Priority(priority),
			sladefinition.IsActive(true),
		).
		Only(ctx)
	if err == nil {
		return sla, nil
	}

	// 如果没有精确匹配，尝试只匹配类型
	sla, err = s.client.SLADefinition.Query().
		Where(
			sladefinition.TenantID(tenantID),
			sladefinition.ServiceType(ticketType),
			sladefinition.PriorityIsNil(),
			sladefinition.IsActive(true),
		).
		Only(ctx)
	if err == nil {
		return sla, nil
	}

	// 如果没有匹配的类型，返回默认SLA
	return &ent.SLADefinition{
		ResponseTime:   60,  // 默认1小时响应
		ResolutionTime: 480, // 默认8小时解决
	}, nil
}

// calculateDeadline 计算截止时间。
// 阻断7 修复：当 businessHoursOnly=true 时，必须只在工作时段内消耗 duration，
// 而非把"落在非工作时间的截止时刻"平移到最近的工作时段起点（旧 AdjustToBusinessHours 行为）。
// 旧算法：周六 10:00 + 480min = 周六 18:00 → 平移到周一 9:00（实际给了 0 分钟有效工作时间）。
// 新算法：从 startTime 开始，只在 9:00-18:00 内消耗 duration，跨日/跨周末跳过非工作时段。
func (s *TicketSLAService) calculateDeadline(startTime time.Time, durationMinutes int, businessHoursOnly bool) time.Time {
	if !businessHoursOnly || durationMinutes <= 0 {
		return startTime.Add(time.Duration(durationMinutes) * time.Minute)
	}
	return addBusinessMinutes(startTime, durationMinutes, defaultBusinessHoursConfig())
}

// AdjustToBusinessHours 调整到工作时间（公开方法，供外部调用）。
// 阻断7 说明：此方法保留向后兼容，仅用于"把某个时刻对齐到最近的工作时段起点"，
// 不能用于计算 SLA 截止时间（截止时间必须用 calculateDeadline/addBusinessMinutes）。
func (s *TicketSLAService) AdjustToBusinessHours(t time.Time) time.Time {
	return adjustToBusinessHoursStart(t, defaultBusinessHoursConfig())
}

// businessHoursConfig 业务时间配置。
// 默认：周一至周五 9:00-18:00，无节假日。
type businessHoursConfig struct {
	workDays    map[time.Weekday]bool // 工作日集合
	startHour   int                   // 工作时段起始小时（含），9 表示 09:00
	startMin    int                   // 工作时段起始分钟
	endHour     int                   // 工作时段结束小时（不含），18 表示 18:00
	endMin      int                   // 工作时段结束分钟
	holidays    map[string]bool       // 节假日集合，格式 "2006-01-02"
}

// defaultBusinessHoursConfig 返回默认业务时间配置（周一至周五 9:00-18:00）。
func defaultBusinessHoursConfig() businessHoursConfig {
	return businessHoursConfig{
		workDays: map[time.Weekday]bool{
			time.Monday: true, time.Tuesday: true, time.Wednesday: true,
			time.Thursday: true, time.Friday: true,
		},
		startHour: 9,
		endHour:   18,
		holidays:  map[string]bool{},
	}
}

// parseBusinessHoursConfig 从 SLADefinition.BusinessHours (map[string]interface{}) 解析配置。
// 配置格式参考 ent/schema/sla_policy.go 的 BusinessHoursConfig：
//   { "work_days": [1,2,3,4,5], "start_time": "09:00", "end_time": "18:00",
//     "time_zone": "Asia/Shanghai", "holiday_list": ["2026-01-01"] }
// 空或解析失败时返回默认配置（不阻断 SLA 计算）。
func parseBusinessHoursConfig(raw map[string]interface{}) businessHoursConfig {
	cfg := defaultBusinessHoursConfig()
	if len(raw) == 0 {
		return cfg
	}
	if days, ok := raw["work_days"].([]interface{}); ok && len(days) > 0 {
		cfg.workDays = map[time.Weekday]bool{}
		// work_days 用 1-7 表示周一到周日（与 time.Weekday 0=Sunday 不同）
		dayMap := map[int]time.Weekday{1: time.Monday, 2: time.Tuesday, 3: time.Wednesday,
			4: time.Thursday, 5: time.Friday, 6: time.Saturday, 7: time.Sunday}
		for _, d := range days {
			if dv, ok := d.(float64); ok {
				if wd, ok := dayMap[int(dv)]; ok {
					cfg.workDays[wd] = true
				}
			}
		}
	}
	parseHM := func(s string) (int, int) {
		parts := strings.Split(s, ":")
		if len(parts) != 2 {
			return -1, -1
		}
		h, e1 := strconv.Atoi(parts[0])
		m, e2 := strconv.Atoi(parts[1])
		if e1 != nil || e2 != nil {
			return -1, -1
		}
		return h, m
	}
	if st, ok := raw["start_time"].(string); ok {
		if h, m := parseHM(st); h >= 0 {
			cfg.startHour, cfg.startMin = h, m
		}
	}
	if et, ok := raw["end_time"].(string); ok {
		if h, m := parseHM(et); h >= 0 {
			cfg.endHour, cfg.endMin = h, m
		}
	}
	if holidays, ok := raw["holiday_list"].([]interface{}); ok {
		for _, h := range holidays {
			if hs, ok := h.(string); ok {
				cfg.holidays[hs] = true
			}
		}
	}
	return cfg
}

// isHoliday 判断给定日期是否为节假日。
func (c businessHoursConfig) isHoliday(t time.Time) bool {
	return c.holidays[t.Format("2006-01-02")]
}

// isWorkDay 判断给定日期是否为工作日（工作日集合 + 非节假日）。
func (c businessHoursConfig) isWorkDay(t time.Time) bool {
	if c.isHoliday(t) {
		return false
	}
	return c.workDays[t.Weekday()]
}

// workDayStart 返回 t 所在工作日的工时开始时刻。
func (c businessHoursConfig) workDayStart(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, c.startHour, c.startMin, 0, 0, t.Location())
}

// workDayEnd 返回 t 所在工作日的工时结束时刻。
func (c businessHoursConfig) workDayEnd(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, c.endHour, c.endMin, 0, 0, t.Location())
}

// nextWorkDayStart 返回 t 之后下一个工作日的工时开始时刻。
func (c businessHoursConfig) nextWorkDayStart(t time.Time) time.Time {
	next := t.AddDate(0, 0, 1)
	for !c.isWorkDay(next) {
		next = next.AddDate(0, 0, 1)
	}
	return c.workDayStart(next)
}

// adjustToBusinessHoursStart 把时刻 t 对齐到最近的工作时段起点。
// 若 t 已在工作时段内，返回 t；否则返回下一个工作时段起点。
func adjustToBusinessHoursStart(t time.Time, cfg businessHoursConfig) time.Time {
	// 跳过非工作日
	for !cfg.isWorkDay(t) {
		t = cfg.nextWorkDayStart(t)
	}
	dayStart := cfg.workDayStart(t)
	dayEnd := cfg.workDayEnd(t)
	if t.Before(dayStart) {
		return dayStart
	}
	if !t.Before(dayEnd) {
		// 当天工时已结束，跳到下一个工作日
		return cfg.nextWorkDayStart(t)
	}
	return t
}

// addBusinessMinutes 从 start 开始，只在工作时段内消耗 minutes 分钟，返回截止时刻。
// 阻断7 核心修复：正确排除非工作时段，而非把截止时刻平移到最近工时起点。
// 算法：
//  1. 若 start 在非工作时段，先把指针移到最近的工作时段起点。
//  2. 计算当天剩余工时；若 minutes <= 当天剩余工时，截止时刻 = 当前指针 + minutes。
//  3. 否则扣减当天剩余工时，跳到下一个工作日起点继续消耗，直到 minutes 耗尽。
func addBusinessMinutes(start time.Time, minutes int, cfg businessHoursConfig) time.Time {
	remaining := time.Duration(minutes) * time.Minute
	cursor := adjustToBusinessHoursStart(start, cfg)

	// 防御性上限：避免极端 minutes 导致死循环（最多循环 365 天）。
	for i := 0; i < 366 && remaining > 0; i++ {
		dayEnd := cfg.workDayEnd(cursor)
		available := dayEnd.Sub(cursor)
		if available <= 0 {
			// 理论上 adjustToBusinessHoursStart 已保证 cursor 在工时内，
			// 此分支仅为防御性兜底，避免死循环。
			cursor = cfg.nextWorkDayStart(cursor)
			continue
		}
		if remaining <= available {
			return cursor.Add(remaining)
		}
		remaining -= available
		cursor = cfg.nextWorkDayStart(cursor)
	}
	return cursor
}

// CalculateSLADeadlineFromRequest 根据请求参数计算SLA截止时间（包含SLADefinitionID）
func (s *TicketSLAService) CalculateSLADeadlineFromRequest(ctx context.Context, tenantID int, ticketType, priority string) (*SLADeadlineResult, error) {
	now := time.Now()

	// 确定service_type
	var serviceType string
	switch ticketType {
	case "incident":
		serviceType = "incident"
	case "service_request":
		serviceType = "service_request"
	case "change":
		serviceType = "change"
	default:
		// 默认为incident类型
		serviceType = "incident"
	}

	// 标准化优先级
	normalizedPriority := strings.ToLower(priority)
	if normalizedPriority == "urgent" {
		normalizedPriority = "critical"
	}

	// 从数据库查找匹配的SLA定义
	sla, err := s.client.SLADefinition.Query().
		Where(
			sladefinition.TenantID(tenantID),
			sladefinition.ServiceType(serviceType),
			sladefinition.Priority(normalizedPriority),
			sladefinition.IsActive(true),
		).
		First(ctx)
	if err != nil {
		// 如果找不到精确匹配，尝试查找默认SLA（不带优先级）
		if ent.IsNotFound(err) {
			sla, err = s.client.SLADefinition.Query().
				Where(
					sladefinition.TenantID(tenantID),
					sladefinition.ServiceType(serviceType),
					sladefinition.IsActive(true),
				).
				First(ctx)
		}

		if err != nil || sla == nil {
			// 如果还是找不到，返回默认值
			s.logger.Warnw("No SLA definition found, using defaults", "service_type", serviceType, "priority", normalizedPriority)
			return &SLADeadlineResult{
				SLADefinitionID:    0,
				ResponseDeadline:   toPointer(now.Add(8 * time.Hour)),
				ResolutionDeadline: toPointer(now.Add(24 * time.Hour)),
			}, nil
		}
	}

	// 计算截止时间（单位是分钟）。
	// 阻断7/C-8 修复：统一走 calculateDeadline，依据 sla.BusinessHours 决定是否启用工作时间口径。
	// 旧逻辑用 AdjustToBusinessHours 平移截止时刻，导致非工作时段被当作顺延而非排除，
	// 且与 GetTicketSLAInfo 路径使用不同口径，造成同一工单"是否违规"两路径结论相反。
	businessHoursOnly := len(sla.BusinessHours) > 0
	responseDeadline := s.calculateDeadline(now, sla.ResponseTime, businessHoursOnly)
	resolutionDeadline := s.calculateDeadline(now, sla.ResolutionTime, businessHoursOnly)

	return &SLADeadlineResult{
		SLADefinitionID:    sla.ID,
		ResponseDeadline:   &responseDeadline,
		ResolutionDeadline: &resolutionDeadline,
		BusinessHoursOnly:  businessHoursOnly,
	}, nil
}

// toPointer 返回指针（辅助函数）
func toPointer[T any](v T) *T {
	return &v
}
