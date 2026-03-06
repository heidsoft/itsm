package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
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

// QuerySLAPolicies 查询SLA策略列表
func (s *SLAPolicyService) QuerySLAPolicies(ctx context.Context, tenantID int) ([]*ent.SLAPolicy, error) {
	all, err := s.client.SLAPolicy.Query().All(ctx)
	if err != nil {
		return nil, err
	}

	var result []*ent.SLAPolicy
	for _, p := range all {
		if p.TenantID == tenantID {
			result = append(result, p)
		}
	}

	// Sort by priority score descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].PriorityScore > result[j].PriorityScore
	})

	return result, nil
}

// UpdateSLAPolicy 更新SLA策略
func (s *SLAPolicyService) UpdateSLAPolicy(ctx context.Context, id int, input dto.UpdateSLAPolicyRequest) (*ent.SLAPolicy, error) {
	update := s.client.SLAPolicy.UpdateOneID(id)
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
	return update.Save(ctx)
}

// DeleteSLAPolicy 删除SLA策略
func (s *SLAPolicyService) DeleteSLAPolicy(ctx context.Context, id int) error {
	return s.client.SLAPolicy.DeleteOneID(id).Exec(ctx)
}

// MatchSLAPolicy 根据工单属性匹配最优SLA策略
// 匹配优先级: customer_tier + ticket_type + priority > ticket_type + priority > priority > default
func (s *SLAPolicyService) MatchSLAPolicy(ctx context.Context, tenantID int, ticketType, priority, customerTier string) (*ent.SLAPolicy, error) {
	// 查询所有激活的策略
	policies, err := s.QuerySLAPolicies(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	if len(policies) == 0 {
		return nil, fmt.Errorf("no active SLA policy found")
	}

	// 过滤激活的策略
	var activePolicies []*ent.SLAPolicy
	for _, p := range policies {
		if p.IsActive {
			activePolicies = append(activePolicies, p)
		}
	}

	// 过滤匹配的策略
	var matchedPolicies []*ent.SLAPolicy
	for _, p := range activePolicies {
		if s.matchPolicy(p, ticketType, priority, customerTier) {
			matchedPolicies = append(matchedPolicies, p)
		}
	}

	if len(matchedPolicies) == 0 {
		// 如果没有精确匹配，返回优先级最低的默认策略
		return s.getDefaultPolicy(activePolicies)
	}

	// 按优先级分数排序，返回最高的
	sort.Slice(matchedPolicies, func(i, j int) bool {
		return matchedPolicies[i].PriorityScore > matchedPolicies[j].PriorityScore
	})

	return matchedPolicies[0], nil
}

// matchPolicy 检查策略是否匹配
func (s *SLAPolicyService) matchPolicy(p *ent.SLAPolicy, ticketType, priority, customerTier string) bool {
	// 完全匹配
	if p.CustomerTier != "" && p.TicketType != "" && p.Priority != "" {
		if p.CustomerTier == customerTier && p.TicketType == ticketType && p.Priority == priority {
			return true
		}
	}

	// 匹配 ticket_type + priority
	if p.TicketType != "" && p.Priority != "" && p.CustomerTier == "" {
		if p.TicketType == ticketType && p.Priority == priority {
			return true
		}
	}

	// 匹配 priority
	if p.Priority != "" && p.TicketType == "" && p.CustomerTier == "" {
		if p.Priority == priority {
			return true
		}
	}

	return false
}

// getDefaultPolicy 获取默认策略
func (s *SLAPolicyService) getDefaultPolicy(policies []*ent.SLAPolicy) (*ent.SLAPolicy, error) {
	if len(policies) == 0 {
		return nil, fmt.Errorf("no policies available")
	}
	// 返回优先级最低的
	sort.Slice(policies, func(i, j int) bool {
		return policies[i].PriorityScore < policies[j].PriorityScore
	})
	return policies[0], nil
}

// CalculateSLAExpireTime 计算SLA到期时间
// 考虑业务时间配置
func (s *SLAPolicyService) CalculateSLAExpireTime(ctx context.Context, policy *ent.SLAPolicy, startTime time.Time) time.Time {
	if policy.BusinessHours == nil || len(policy.BusinessHours) == 0 {
		// 无业务时间配置，使用24小时制
		return startTime.Add(time.Duration(policy.ResolutionTimeMinutes) * time.Minute)
	}

	// 简化实现：暂不处理复杂的业务时间计算
	// TODO: 实现完整的业务时间计算逻辑
	return startTime.Add(time.Duration(policy.ResolutionTimeMinutes) * time.Minute)
}

// GetSLAComplianceRate 获取SLA合规率
func (s *SLAPolicyService) GetSLAComplianceRate(ctx context.Context, tenantID int, startDate, endDate time.Time) (float64, error) {
	// 查询指定时间范围内的SLA违规情况
	violations, err := s.client.SLAViolation.Query().Count(ctx)
	if err != nil {
		return 0, err
	}

	// 过滤租户
	if err == nil {
		allViolations, _ := s.client.SLAViolation.Query().All(ctx)
		violations = 0
		for _, v := range allViolations {
			if v.TenantID == tenantID && v.CreatedAt.After(startDate) && v.CreatedAt.Before(endDate) {
				violations++
			}
		}
	}

	// 查询总工单数
	totalTickets := 0
	allTickets, err := s.client.Ticket.Query().All(ctx)
	if err == nil {
		for _, t := range allTickets {
			if t.TenantID == tenantID && t.CreatedAt.After(startDate) && t.CreatedAt.Before(endDate) {
				totalTickets++
			}
		}
	}

	if totalTickets == 0 {
		return 100.0, nil
	}

	return float64(totalTickets-violations) / float64(totalTickets) * 100, nil
}
