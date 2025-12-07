package service

import (
	"context"
	"fmt"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/ticket"

	"go.uber.org/zap"
)

type TicketDependencyService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

func NewTicketDependencyService(client *ent.Client, logger *zap.SugaredLogger) *TicketDependencyService {
	return &TicketDependencyService{
		client: client,
		logger: logger,
	}
}

// AnalyzeDependencyImpact 分析依赖关系影响
func (s *TicketDependencyService) AnalyzeDependencyImpact(ctx context.Context, ticketID int, action string, newStatus *string, tenantID int) (*dto.RelationImpactAnalysis, error) {
	s.logger.Infow("Analyzing dependency impact", "ticket_id", ticketID, "action", action, "tenant_id", tenantID)

	// 获取工单信息
	ticketEntity, err := s.client.Ticket.Query().
		Where(
			ticket.IDEQ(ticketID),
			ticket.TenantIDEQ(tenantID),
		).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("ticket not found: %w", err)
	}

	// 获取所有相关工单（通过parent_ticket_id）
	relatedTickets, err := s.client.Ticket.Query().
		Where(
			ticket.ParentTicketIDEQ(ticketID),
			ticket.TenantIDEQ(tenantID),
		).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get related tickets", "error", err)
		return nil, fmt.Errorf("failed to get related tickets: %w", err)
	}

	// 分析影响
	impact := &dto.RelationImpactAnalysis{
		TicketID:        ticketID,
		TicketNumber:    ticketEntity.TicketNumber,
		TicketTitle:     ticketEntity.Title,
		Action:          action,
		AffectedCount:   len(relatedTickets),
		Warnings:        []string{},
		Recommendations: []string{},
	}

	// 根据action分析影响
	switch action {
	case "close":
		// 检查是否有未完成的子工单
		for _, related := range relatedTickets {
			if related.Status != "closed" && related.Status != "resolved" {
				impact.Warnings = append(impact.Warnings,
					fmt.Sprintf("子工单 %s (%s) 尚未完成", related.TicketNumber, related.Title))
				impact.AffectedTickets = append(impact.AffectedTickets, dto.AffectedTicketInfo{
					ID:          related.ID,
					Number:      related.TicketNumber,
					Title:       related.Title,
					Status:      related.Status,
					ImpactType:  "blocked",
					Description: "父工单关闭可能导致此工单无法继续",
				})
			}
		}
		if len(impact.Warnings) > 0 {
			impact.Recommendations = append(impact.Recommendations,
				"建议先完成或取消所有子工单后再关闭父工单")
		}

	case "delete":
		// 删除操作影响更大
		for _, related := range relatedTickets {
			impact.Warnings = append(impact.Warnings,
				fmt.Sprintf("子工单 %s (%s) 将失去父工单关联", related.TicketNumber, related.Title))
			impact.AffectedTickets = append(impact.AffectedTickets, dto.AffectedTicketInfo{
				ID:          related.ID,
				Number:      related.TicketNumber,
				Title:       related.Title,
				Status:      related.Status,
				ImpactType:  "orphaned",
				Description: "父工单删除后此工单将变为孤立工单",
			})
		}
		impact.Recommendations = append(impact.Recommendations,
			"删除操作不可逆，请谨慎操作")
		impact.Recommendations = append(impact.Recommendations,
			"建议先处理所有相关工单后再删除")

	case "change_status":
		if newStatus != nil {
			// 检查状态变更对依赖工单的影响
			if *newStatus == "closed" || *newStatus == "resolved" {
				for _, related := range relatedTickets {
					if related.Status == "open" || related.Status == "in_progress" {
						impact.Warnings = append(impact.Warnings,
							fmt.Sprintf("子工单 %s (%s) 可能受到影响", related.TicketNumber, related.Title))
						impact.AffectedTickets = append(impact.AffectedTickets, dto.AffectedTicketInfo{
							ID:          related.ID,
							Number:      related.TicketNumber,
							Title:       related.Title,
							Status:      related.Status,
							ImpactType:  "status_change",
							Description: fmt.Sprintf("父工单状态变更为 %s 可能影响此工单", *newStatus),
						})
					}
				}
			}
		}
	}

	// 计算风险等级
	if len(impact.Warnings) == 0 {
		impact.RiskLevel = "low"
	} else if len(impact.Warnings) <= 2 {
		impact.RiskLevel = "medium"
	} else {
		impact.RiskLevel = "high"
	}

	return impact, nil
}
