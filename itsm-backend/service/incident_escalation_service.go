package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/incident"
)

// IncidentEscalationService 事件升级服务
type IncidentEscalationService struct {
	client *ent.Client
}

// NewIncidentEscalationService 创建事件升级服务
func NewIncidentEscalationService(client *ent.Client) *IncidentEscalationService {
	return &IncidentEscalationService{client: client}
}

// CreateEscalationRule 创建升级规则
func (s *IncidentEscalationService) CreateEscalationRule(ctx context.Context, input dto.CreateIncidentEscalationRuleRequest) (*ent.IncidentEscalationRule, error) {
	build := s.client.IncidentEscalationRule.Create().
		SetName(input.Name).
		SetDescription(input.Description).
		SetTriggerType(input.TriggerType).
		SetEscalationLevel(input.EscalationLevel).
		SetTriggerMinutes(input.TriggerMinutes).
		SetTargetAssigneeType(input.TargetAssigneeType).
		SetAutoEscalate(input.AutoEscalate).
		SetNotificationConfig(input.NotificationConfig).
		SetIsActive(input.IsActive).
		SetTenantID(input.TenantID)

	if input.FromStatus != nil && *input.FromStatus != "" {
		build.SetFromStatus(*input.FromStatus)
	}
	if input.ToStatus != nil && *input.ToStatus != "" {
		build.SetToStatus(*input.ToStatus)
	}
	if input.TargetAssigneeID != nil && *input.TargetAssigneeID > 0 {
		build.SetTargetAssigneeID(*input.TargetAssigneeID)
	}
	if input.TargetGroup != nil && *input.TargetGroup != "" {
		build.SetTargetGroup(*input.TargetGroup)
	}
	if input.PriorityMatch != nil && *input.PriorityMatch != "" {
		build.SetPriorityMatch(*input.PriorityMatch)
	}
	if input.CategoryMatch != nil && *input.CategoryMatch != "" {
		build.SetCategoryMatch(*input.CategoryMatch)
	}

	return build.Save(ctx)
}

// GetEscalationRuleByID 获取升级规则
func (s *IncidentEscalationService) GetEscalationRuleByID(ctx context.Context, id int) (*ent.IncidentEscalationRule, error) {
	return s.client.IncidentEscalationRule.Get(ctx, id)
}

// QueryEscalationRules 查询升级规则列表
func (s *IncidentEscalationService) QueryEscalationRules(ctx context.Context, tenantID int) ([]*ent.IncidentEscalationRule, error) {
	// Simple query - get all and filter
	all, err := s.client.IncidentEscalationRule.Query().All(ctx)
	if err != nil {
		return nil, err
	}

	var result []*ent.IncidentEscalationRule
	for _, rule := range all {
		if rule.TenantID == tenantID {
			result = append(result, rule)
		}
	}
	return result, nil
}

// UpdateEscalationRule 更新升级规则
func (s *IncidentEscalationService) UpdateEscalationRule(ctx context.Context, id int, input dto.UpdateIncidentEscalationRuleRequest) (*ent.IncidentEscalationRule, error) {
	update := s.client.IncidentEscalationRule.UpdateOneID(id)
	if input.Name != nil {
		update.SetName(*input.Name)
	}
	if input.Description != nil {
		update.SetDescription(*input.Description)
	}
	if input.TriggerType != nil {
		update.SetTriggerType(*input.TriggerType)
	}
	if input.EscalationLevel != nil {
		update.SetEscalationLevel(*input.EscalationLevel)
	}
	if input.TriggerMinutes != nil {
		update.SetTriggerMinutes(*input.TriggerMinutes)
	}
	if input.FromStatus != nil {
		update.SetFromStatus(*input.FromStatus)
	}
	if input.ToStatus != nil {
		update.SetToStatus(*input.ToStatus)
	}
	if input.TargetAssigneeType != nil {
		update.SetTargetAssigneeType(*input.TargetAssigneeType)
	}
	if input.TargetAssigneeID != nil {
		update.SetTargetAssigneeID(*input.TargetAssigneeID)
	}
	if input.TargetGroup != nil {
		update.SetTargetGroup(*input.TargetGroup)
	}
	if input.AutoEscalate != nil {
		update.SetAutoEscalate(*input.AutoEscalate)
	}
	if input.NotificationConfig != nil {
		update.SetNotificationConfig(*input.NotificationConfig)
	}
	if input.IsActive != nil {
		update.SetIsActive(*input.IsActive)
	}
	if input.PriorityMatch != nil {
		update.SetPriorityMatch(*input.PriorityMatch)
	}
	if input.CategoryMatch != nil {
		update.SetCategoryMatch(*input.CategoryMatch)
	}
	return update.Save(ctx)
}

// DeleteEscalationRule 删除升级规则
func (s *IncidentEscalationService) DeleteEscalationRule(ctx context.Context, id int) error {
	return s.client.IncidentEscalationRule.DeleteOneID(id).Exec(ctx)
}

// CheckAndEscalate 检查事件是否需要升级
func (s *IncidentEscalationService) CheckAndEscalate(ctx context.Context, incidentID int) (*ent.Incident, error) {
	incidentEnt, err := s.client.Incident.Get(ctx, incidentID)
	if err != nil {
		return nil, err
	}

	// 获取匹配的升级规则
	rules, err := s.getMatchingRules(ctx, incidentEnt)
	if err != nil {
		return nil, err
	}

	// 检查是否需要升级
	for _, rule := range rules {
		shouldEscalate, err := s.shouldEscalate(ctx, incidentEnt, rule)
		if err != nil {
			return nil, err
		}

		if shouldEscalate {
			return s.escalateIncident(ctx, incidentEnt, rule)
		}
	}

	return incidentEnt, nil
}

// getMatchingRules 获取匹配的升级规则
func (s *IncidentEscalationService) getMatchingRules(ctx context.Context, incidentEnt *ent.Incident) ([]*ent.IncidentEscalationRule, error) {
	all, err := s.client.IncidentEscalationRule.Query().All(ctx)
	if err != nil {
		return nil, err
	}

	var matchedRules []*ent.IncidentEscalationRule
	for _, rule := range all {
		if rule.TenantID != incidentEnt.TenantID || !rule.IsActive {
			continue
		}

		// 匹配优先级
		if rule.PriorityMatch != "" && rule.PriorityMatch != incidentEnt.Priority {
			continue
		}
		// 匹配分类
		if rule.CategoryMatch != "" && rule.CategoryMatch != incidentEnt.Category {
			continue
		}
		matchedRules = append(matchedRules, rule)
	}

	return matchedRules, nil
}

// shouldEscalate 检查是否应该升级
func (s *IncidentEscalationService) shouldEscalate(ctx context.Context, incidentEnt *ent.Incident, rule *ent.IncidentEscalationRule) (bool, error) {
	switch rule.TriggerType {
	case "time_based":
		// 基于时间升级
		elapsedMinutes := time.Since(incidentEnt.DetectedAt).Minutes()
		return elapsedMinutes >= float64(rule.TriggerMinutes), nil
	case "sla_breach":
		// 基于SLA违规升级
		// TODO: 检查SLA状态
		return false, nil
	case "manual":
		// 手动升级
		return false, nil
	}
	return false, nil
}

// escalateIncident 执行事件升级
func (s *IncidentEscalationService) escalateIncident(ctx context.Context, incidentEnt *ent.Incident, rule *ent.IncidentEscalationRule) (*ent.Incident, error) {
	// 更新事件
	update := incidentEnt.Update().
		SetEscalatedAt(time.Now()).
		SetEscalationLevel(rule.EscalationLevel)

	if rule.ToStatus != "" {
		update.SetStatus(rule.ToStatus)
	}

	if rule.TargetAssigneeID > 0 {
		update.SetAssigneeID(rule.TargetAssigneeID)
	}

	updatedIncident, err := update.Save(ctx)
	if err != nil {
		return nil, err
	}

	// 发送升级通知
	if rule.NotificationConfig != nil {
		s.sendEscalationNotification(ctx, updatedIncident, rule)
	}

	// 记录升级日志
	_, err = s.client.IncidentEvent.Create().
		SetIncidentID(incidentEnt.ID).
		SetEventType("escalation").
		SetDescription(fmt.Sprintf("事件升级到L%d: %s", rule.EscalationLevel, rule.Name)).
		SetCreatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return updatedIncident, nil
}

// sendEscalationNotification 发送升级通知
func (s *IncidentEscalationService) sendEscalationNotification(ctx context.Context, incidentEnt *ent.Incident, rule *ent.IncidentEscalationRule) {
	// TODO: 实现通知发送逻辑
	// 可以调用 notification service 发送邮件/短信/站内信
}

// ProcessEscalations 批量处理升级检查
// 由定时任务调用
func (s *IncidentEscalationService) ProcessEscalations(ctx context.Context, tenantID int) error {
	// 查询所有待处理的事件
	incidents, err := s.client.Incident.Query().
		Where(
			incident.TenantIDEQ(tenantID),
			incident.StatusIn("new", "investigating"),
		).
		All(ctx)
	if err != nil {
		return err
	}

	for _, incidentEnt := range incidents {
		_, err := s.CheckAndEscalate(ctx, incidentEnt.ID)
		if err != nil {
			// 记录错误但继续处理其他事件
			fmt.Printf("Error processing escalation for incident %d: %v\n", incidentEnt.ID, err)
		}
	}

	return nil
}

// GetEscalationHistory 获取事件升级历史
func (s *IncidentEscalationService) GetEscalationHistory(ctx context.Context, incidentID int) ([]*ent.IncidentEvent, error) {
	all, err := s.client.IncidentEvent.Query().All(ctx)
	if err != nil {
		return nil, err
	}

	var result []*ent.IncidentEvent
	for _, event := range all {
		if event.IncidentID == incidentID && event.EventType == "escalation" {
			result = append(result, event)
		}
	}
	return result, nil
}
