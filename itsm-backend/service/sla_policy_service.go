package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/slapolicy"
	"itsm-backend/ent/slaviolation"
	"itsm-backend/ent/ticket"
)

// SLAPolicyService SLA策略服务
type SLAPolicyService struct {
	client *ent.Client
}

// NewSLAPolicyService 创建SLA策略服务
func NewSLAPolicyService(client *ent.Client) *SLAPolicyService {
	return &SLAPolicyService{client: client}
}

// CreateSLAPolicy 创建SLA策略
func (s *SLAPolicyService) CreateSLAPolicy(ctx context.Context, input dto.CreateSLAPolicyRequest) (*ent.SLAPolicy, error) {
	build := s.client.SLAPolicy.Create().
		SetName(input.Name).
		SetResponseTimeMinutes(input.ResponseTimeMinutes).
		SetResolutionTimeMinutes(input.ResolutionTimeMinutes).
		SetExcludeWeekends(input.ExcludeWeekends).
		SetExcludeHolidays(input.ExcludeHolidays).
		SetEscalationRules(input.EscalationRules).
		SetIsActive(input.IsActive).
		SetPriorityScore(input.PriorityScore).
		SetTenantID(input.TenantID)

	if input.Description != "" {
		build.SetDescription(input.Description)
	}
	if input.CustomerTier != nil && *input.CustomerTier != "" {
		build.SetCustomerTier(*input.CustomerTier)
	}
	if input.TicketType != nil && *input.TicketType != "" {
		build.SetTicketType(*input.TicketType)
	}
	if input.Priority != nil && *input.Priority != "" {
		build.SetPriority(*input.Priority)
	}
	if input.BusinessHours != nil {
		build.SetBusinessHours(input.BusinessHours)
	}

	return build.Save(ctx)
}

// GetSLAPolicyByID 根据ID获取SLA策略
func (s *SLAPolicyService) GetSLAPolicyByID(ctx context.Context, id int) (*ent.SLAPolicy, error) {
	return s.client.SLAPolicy.Get(ctx, id)
}

// GetSLAPolicyByIDForTenant 根据ID和租户获取SLA策略
func (s *SLAPolicyService) GetSLAPolicyByIDForTenant(ctx context.Context, id, tenantID int) (*ent.SLAPolicy, error) {
	return s.client.SLAPolicy.Query().
		Where(slapolicy.ID(id), slapolicy.TenantID(tenantID)).
		Only(ctx)
}

// QuerySLAPolicies 查询SLA策略列表
func (s *SLAPolicyService) QuerySLAPolicies(ctx context.Context, tenantID int) ([]*ent.SLAPolicy, error) {
	return s.client.SLAPolicy.Query().
		Where(slapolicy.TenantID(tenantID)).
		Order(slapolicy.ByPriorityScore(sql.OrderDesc())).
		All(ctx)
}

// UpdateSLAPolicy 更新SLA策略
func (s *SLAPolicyService) UpdateSLAPolicy(ctx context.Context, id int, input dto.UpdateSLAPolicyRequest) (*ent.SLAPolicy, error) {
	update := s.client.SLAPolicy.UpdateOneID(id)
	s.applyUpdateFields(update, input)
	return update.Save(ctx)
}

// UpdateSLAPolicyForTenant 按租户约束更新SLA策略
func (s *SLAPolicyService) UpdateSLAPolicyForTenant(ctx context.Context, id, tenantID int, input dto.UpdateSLAPolicyRequest) (*ent.SLAPolicy, error) {
	update := s.client.SLAPolicy.UpdateOneID(id).Where(slapolicy.TenantID(tenantID))
	s.applyUpdateFields(update, input)
	return update.Save(ctx)
}

func (s *SLAPolicyService) applyUpdateFields(update *ent.SLAPolicyUpdateOne, input dto.UpdateSLAPolicyRequest) {
	if input.Name != nil {
		update.SetName(*input.Name)
	}
	if input.Description != nil {
		update.SetDescription(*input.Description)
	}
	if input.CustomerTier != nil {
		update.SetCustomerTier(*input.CustomerTier)
	}
	if input.TicketType != nil {
		update.SetTicketType(*input.TicketType)
	}
	if input.Priority != nil {
		update.SetPriority(*input.Priority)
	}
	if input.ResponseTimeMinutes != nil {
		update.SetResponseTimeMinutes(*input.ResponseTimeMinutes)
	}
	if input.ResolutionTimeMinutes != nil {
		update.SetResolutionTimeMinutes(*input.ResolutionTimeMinutes)
	}
	if input.BusinessHours != nil {
		update.SetBusinessHours(*input.BusinessHours)
	}
	if input.ExcludeWeekends != nil {
		update.SetExcludeWeekends(*input.ExcludeWeekends)
	}
	if input.ExcludeHolidays != nil {
		update.SetExcludeHolidays(*input.ExcludeHolidays)
	}
	if input.EscalationRules != nil {
		update.SetEscalationRules(*input.EscalationRules)
	}
	if input.IsActive != nil {
		update.SetIsActive(*input.IsActive)
	}
	if input.PriorityScore != nil {
		update.SetPriorityScore(*input.PriorityScore)
	}
}

// DeleteSLAPolicy 删除SLA策略
func (s *SLAPolicyService) DeleteSLAPolicy(ctx context.Context, id int) error {
	return s.client.SLAPolicy.DeleteOneID(id).Exec(ctx)
}

// DeleteSLAPolicyForTenant 按租户约束删除SLA策略
func (s *SLAPolicyService) DeleteSLAPolicyForTenant(ctx context.Context, id, tenantID int) error {
	return s.client.SLAPolicy.DeleteOneID(id).Where(slapolicy.TenantID(tenantID)).Exec(ctx)
}

// MatchSLAPolicy 根据工单属性匹配最优SLA策略
// 匹配优先级: customer_tier + ticket_type + priority > ticket_type + priority > priority > default
func (s *SLAPolicyService) MatchSLAPolicy(ctx context.Context, tenantID int, ticketType, priority, customerTier string) (*ent.SLAPolicy, error) {
	policies, err := s.QuerySLAPolicies(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	if len(policies) == 0 {
		return nil, fmt.Errorf("no active SLA policy found")
	}

	var activePolicies []*ent.SLAPolicy
	for _, p := range policies {
		if p.IsActive {
			activePolicies = append(activePolicies, p)
		}
	}

	var matchedPolicies []*ent.SLAPolicy
	for _, p := range activePolicies {
		if s.matchPolicy(p, ticketType, priority, customerTier) {
			matchedPolicies = append(matchedPolicies, p)
		}
	}

	if len(matchedPolicies) == 0 {
		return s.getDefaultPolicy(activePolicies)
	}

	// 先按匹配精确度排序，再用优先级分数打破同级策略冲突。
	sort.Slice(matchedPolicies, func(i, j int) bool {
		leftSpecificity := s.matchSpecificity(matchedPolicies[i], ticketType, priority, customerTier)
		rightSpecificity := s.matchSpecificity(matchedPolicies[j], ticketType, priority, customerTier)
		if leftSpecificity != rightSpecificity {
			return leftSpecificity > rightSpecificity
		}
		return matchedPolicies[i].PriorityScore > matchedPolicies[j].PriorityScore
	})

	return matchedPolicies[0], nil
}

func (s *SLAPolicyService) matchPolicy(p *ent.SLAPolicy, ticketType, priority, customerTier string) bool {
	if p.CustomerTier != "" && p.TicketType != "" && p.Priority != "" {
		if p.CustomerTier == customerTier && p.TicketType == ticketType && p.Priority == priority {
			return true
		}
	}

	if p.TicketType != "" && p.Priority != "" && p.CustomerTier == "" {
		if p.TicketType == ticketType && p.Priority == priority {
			return true
		}
	}

	if p.Priority != "" && p.TicketType == "" && p.CustomerTier == "" {
		if p.Priority == priority {
			return true
		}
	}

	return false
}

func (s *SLAPolicyService) matchSpecificity(p *ent.SLAPolicy, ticketType, priority, customerTier string) int {
	if p.CustomerTier != "" && p.TicketType != "" && p.Priority != "" &&
		p.CustomerTier == customerTier && p.TicketType == ticketType && p.Priority == priority {
		return 3
	}

	if p.CustomerTier == "" && p.TicketType != "" && p.Priority != "" &&
		p.TicketType == ticketType && p.Priority == priority {
		return 2
	}

	if p.CustomerTier == "" && p.TicketType == "" && p.Priority != "" && p.Priority == priority {
		return 1
	}

	return 0
}

// getDefaultPolicy 获取默认策略
func (s *SLAPolicyService) getDefaultPolicy(policies []*ent.SLAPolicy) (*ent.SLAPolicy, error) {
	if len(policies) == 0 {
		return nil, fmt.Errorf("no policies available")
	}
	sort.Slice(policies, func(i, j int) bool {
		return policies[i].PriorityScore < policies[j].PriorityScore
	})
	return policies[0], nil
}

// CalculateSLAExpireTime 计算SLA到期时间
// 考虑业务时间配置
func (s *SLAPolicyService) CalculateSLAExpireTime(ctx context.Context, policy *ent.SLAPolicy, startTime time.Time) time.Time {
	_ = ctx
	if policy == nil || policy.ResolutionTimeMinutes <= 0 {
		return startTime
	}

	windowStart, windowEnd, hasWindow := extractBusinessHoursWindow(policy.BusinessHours)
	if !hasWindow && !policy.ExcludeWeekends && !policy.ExcludeHolidays {
		return startTime.Add(time.Duration(policy.ResolutionTimeMinutes) * time.Minute)
	}

	if !hasWindow {
		windowStart = 0
		windowEnd = 24 * 60
	}

	current := alignToBusinessWindow(startTime, windowStart, windowEnd, policy.ExcludeWeekends)
	remaining := policy.ResolutionTimeMinutes

	for remaining > 0 {
		current = alignToBusinessWindow(current, windowStart, windowEnd, policy.ExcludeWeekends)
		dayEnd := minutesFromMidnight(current)
		availableToday := windowEnd - dayEnd
		if availableToday <= 0 {
			current = nextBusinessDayStart(current, windowStart, policy.ExcludeWeekends)
			continue
		}
		if remaining <= availableToday {
			return current.Add(time.Duration(remaining) * time.Minute)
		}
		remaining -= availableToday
		current = nextBusinessDayStart(current, windowStart, policy.ExcludeWeekends)
	}

	return current
}

func extractBusinessHoursWindow(businessHours map[string]interface{}) (int, int, bool) {
	if len(businessHours) == 0 {
		return 0, 0, false
	}

	start, okStart := parseClockMinutes(businessHours["start"])
	end, okEnd := parseClockMinutes(businessHours["end"])
	if !okStart || !okEnd || end <= start {
		return 0, 0, false
	}

	return start, end, true
}

func parseClockMinutes(value interface{}) (int, bool) {
	raw, ok := value.(string)
	if !ok {
		return 0, false
	}
	parts := strings.Split(strings.TrimSpace(raw), ":")
	if len(parts) != 2 {
		return 0, false
	}
	hour, errHour := strconv.Atoi(parts[0])
	minute, errMinute := strconv.Atoi(parts[1])
	if errHour != nil || errMinute != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, false
	}
	return hour*60 + minute, true
}

func alignToBusinessWindow(current time.Time, windowStart, windowEnd int, excludeWeekends bool) time.Time {
	current = moveToWorkingDay(current, excludeWeekends)

	minutes := minutesFromMidnight(current)
	if minutes < windowStart {
		return setMinutesFromMidnight(current, windowStart)
	}
	if minutes >= windowEnd {
		return nextBusinessDayStart(current, windowStart, excludeWeekends)
	}
	return current
}

func nextBusinessDayStart(current time.Time, windowStart int, excludeWeekends bool) time.Time {
	next := setMinutesFromMidnight(current, windowStart).AddDate(0, 0, 1)
	next = moveToWorkingDay(next, excludeWeekends)
	return setMinutesFromMidnight(next, windowStart)
}

func moveToWorkingDay(current time.Time, excludeWeekends bool) time.Time {
	if !excludeWeekends {
		return current
	}
	for current.Weekday() == time.Saturday || current.Weekday() == time.Sunday {
		current = current.AddDate(0, 0, 1)
	}
	return current
}

func minutesFromMidnight(value time.Time) int {
	return value.Hour()*60 + value.Minute()
}

func setMinutesFromMidnight(base time.Time, minutes int) time.Time {
	hour := minutes / 60
	minute := minutes % 60
	return time.Date(base.Year(), base.Month(), base.Day(), hour, minute, 0, 0, base.Location())
}

// GetSLAComplianceRate 获取SLA合规率
func (s *SLAPolicyService) GetSLAComplianceRate(ctx context.Context, tenantID int, startDate, endDate time.Time) (float64, error) {
	var violatedTickets []struct {
		TicketID int `json:"ticket_id"` // json tag matches SQL column for ent GroupBy().Scan()
	}
	if err := s.client.SLAViolation.Query().
		Where(
			slaviolation.TenantID(tenantID),
			slaviolation.CreatedAtGTE(startDate),
			slaviolation.CreatedAtLTE(endDate),
			slaviolation.HasTicketWith(
				ticket.TenantID(tenantID),
				ticket.CreatedAtGTE(startDate),
				ticket.CreatedAtLTE(endDate),
			),
		).
		GroupBy(slaviolation.FieldTicketID).
		Scan(ctx, &violatedTickets); err != nil {
		return 0, err
	}

	totalTickets, err := s.client.Ticket.Query().
		Where(
			ticket.TenantID(tenantID),
			ticket.CreatedAtGTE(startDate),
			ticket.CreatedAtLTE(endDate),
		).
		Count(ctx)
	if err != nil {
		return 0, err
	}

	if totalTickets == 0 {
		return 100.0, nil
	}

	violations := len(violatedTickets)
	return float64(totalTickets-violations) / float64(totalTickets) * 100, nil
}
