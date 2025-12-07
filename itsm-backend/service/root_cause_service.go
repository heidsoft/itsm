package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/rootcauseanalysis"
	"itsm-backend/ent/ticket"

	"go.uber.org/zap"
)

type RootCauseService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

func NewRootCauseService(client *ent.Client, logger *zap.SugaredLogger) *RootCauseService {
	return &RootCauseService{
		client: client,
		logger: logger,
	}
}

// AnalyzeTicket 执行根因分析
func (s *RootCauseService) AnalyzeTicket(ctx context.Context, ticketID int, tenantID int) (*dto.TicketRootCauseAnalysisResponse, error) {
	s.logger.Infow("Analyzing ticket root cause", "ticket_id", ticketID, "tenant_id", tenantID)

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

	// 执行根因分析（简化版，实际需要更复杂的算法）
	rootCauses := s.performAnalysis(ticketEntity, tenantID)

	// 保存分析结果
	rootCausesMap := make([]map[string]interface{}, len(rootCauses))
	for i, rc := range rootCauses {
		rootCausesMap[i] = map[string]interface{}{
			"id":             rc.ID,
			"title":          rc.Title,
			"description":    rc.Description,
			"confidence":     rc.Confidence,
			"category":       rc.Category,
			"status":         rc.Status,
		}
	}

	analysis, err := s.client.RootCauseAnalysis.Create().
		SetTicketID(ticketID).
		SetTicketNumber(ticketEntity.TicketNumber).
		SetTicketTitle(ticketEntity.Title).
		SetAnalysisDate(time.Now().Format("2006-01-02")).
		SetRootCauses(rootCausesMap).
		SetAnalysisSummary(fmt.Sprintf("系统自动分析识别出%d个可能的根本原因", len(rootCauses))).
		SetConfidenceScore(s.calculateAverageConfidence(rootCauses)).
		SetAnalysisMethod("automatic").
		SetTenantID(tenantID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to save root cause analysis", "error", err)
		return nil, fmt.Errorf("failed to save analysis: %w", err)
	}

	// 构建响应
	response := &dto.TicketRootCauseAnalysisResponse{
		TicketID:        ticketID,
		TicketNumber:    ticketEntity.TicketNumber,
		TicketTitle:     ticketEntity.Title,
		AnalysisDate:    analysis.AnalysisDate,
		RootCauses:      rootCauses,
		AnalysisSummary: analysis.AnalysisSummary,
		ConfidenceScore: analysis.ConfidenceScore,
		AnalysisMethod:  analysis.AnalysisMethod,
		GeneratedAt:     analysis.CreatedAt,
	}

	return response, nil
}

// GetAnalysisReport 获取分析报告
func (s *RootCauseService) GetAnalysisReport(ctx context.Context, ticketID int, tenantID int) (*dto.TicketRootCauseAnalysisResponse, error) {
	s.logger.Infow("Getting root cause analysis report", "ticket_id", ticketID, "tenant_id", tenantID)

	// 获取最新的分析记录
	analysis, err := s.client.RootCauseAnalysis.Query().
		Where(
			rootcauseanalysis.TicketIDEQ(ticketID),
			rootcauseanalysis.TenantIDEQ(tenantID),
		).
		Order(ent.Desc(rootcauseanalysis.FieldCreatedAt)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("analysis not found, please run analysis first")
		}
		return nil, fmt.Errorf("failed to get analysis: %w", err)
	}

	// 转换根因数据
	rootCauses := make([]dto.TicketRootCauseResponse, 0)
	if analysis.RootCauses != nil {
		for i, rcMap := range analysis.RootCauses {
			rc := dto.TicketRootCauseResponse{
				ID:          fmt.Sprintf("rc%d", i+1),
				CreatedAt:   analysis.CreatedAt,
				UpdatedAt:   analysis.UpdatedAt,
			}
			
			if title, ok := rcMap["title"].(string); ok {
				rc.Title = title
			}
			if desc, ok := rcMap["description"].(string); ok {
				rc.Description = desc
			}
			if conf, ok := rcMap["confidence"].(float64); ok {
				rc.Confidence = conf
			}
			if cat, ok := rcMap["category"].(string); ok {
				rc.Category = cat
			}
			if status, ok := rcMap["status"].(string); ok {
				rc.Status = status
			}
			
			rootCauses = append(rootCauses, rc)
		}
	}

	response := &dto.TicketRootCauseAnalysisResponse{
		TicketID:        ticketID,
		TicketNumber:    analysis.TicketNumber,
		TicketTitle:     analysis.TicketTitle,
		AnalysisDate:    analysis.AnalysisDate,
		RootCauses:      rootCauses,
		AnalysisSummary: analysis.AnalysisSummary,
		ConfidenceScore: analysis.ConfidenceScore,
		AnalysisMethod:  analysis.AnalysisMethod,
		GeneratedAt:     analysis.CreatedAt,
	}

	return response, nil
}

// performAnalysis 执行分析（简化版）
func (s *RootCauseService) performAnalysis(ticketEntity *ent.Ticket, tenantID int) []dto.TicketRootCauseResponse {
	rootCauses := make([]dto.TicketRootCauseResponse, 0)

	// 基于工单信息生成模拟根因
	if ticketEntity.Priority == "critical" || ticketEntity.Priority == "high" {
		rootCauses = append(rootCauses, dto.TicketRootCauseResponse{
			ID:          "rc1",
			Title:       "系统资源不足",
			Description: "检测到系统在高峰期资源使用率过高",
			Confidence:  0.85,
			Category:    "infrastructure",
			Status:      "identified",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}

	if ticketEntity.Status == "open" && time.Since(ticketEntity.CreatedAt) > 24*time.Hour {
		rootCauses = append(rootCauses, dto.TicketRootCauseResponse{
			ID:          "rc2",
			Title:       "响应时间过长",
			Description: "工单创建后超过24小时未得到响应",
			Confidence:  0.75,
			Category:    "process",
			Status:      "identified",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}

	return rootCauses
}

// calculateAverageConfidence 计算平均置信度
func (s *RootCauseService) calculateAverageConfidence(rootCauses []dto.TicketRootCauseResponse) float64 {
	if len(rootCauses) == 0 {
		return 0
	}
	total := 0.0
	for _, rc := range rootCauses {
		total += rc.Confidence
	}
	return total / float64(len(rootCauses))
}

// ConfirmRootCause 确认根因
func (s *RootCauseService) ConfirmRootCause(ctx context.Context, ticketID int, rootCauseID string, tenantID int) error {
	s.logger.Infow("Confirming root cause", "ticket_id", ticketID, "root_cause_id", rootCauseID, "tenant_id", tenantID)

	// 获取最新的分析记录
	analysis, err := s.client.RootCauseAnalysis.Query().
		Where(
			rootcauseanalysis.TicketIDEQ(ticketID),
			rootcauseanalysis.TenantIDEQ(tenantID),
		).
		Order(ent.Desc(rootcauseanalysis.FieldCreatedAt)).
		First(ctx)
	if err != nil {
		return fmt.Errorf("analysis not found: %w", err)
	}

	// 更新根因状态为confirmed
	if analysis.RootCauses != nil {
		rootCauses := analysis.RootCauses
		for i, rcMap := range rootCauses {
			if id, ok := rcMap["id"].(string); ok && id == rootCauseID {
				rcMap["status"] = "confirmed"
				rcMap["confirmed_at"] = time.Now().Format(time.RFC3339)
				rootCauses[i] = rcMap
				break
			}
		}

		// 更新分析记录
		_, err = s.client.RootCauseAnalysis.Update().
			Where(rootcauseanalysis.IDEQ(analysis.ID)).
			SetRootCauses(rootCauses).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		if err != nil {
			s.logger.Errorw("Failed to update root cause status", "error", err)
			return fmt.Errorf("failed to update root cause status: %w", err)
		}
	}

	return nil
}

// ResolveRootCause 标记根因为已解决
func (s *RootCauseService) ResolveRootCause(ctx context.Context, ticketID int, rootCauseID string, tenantID int) error {
	s.logger.Infow("Resolving root cause", "ticket_id", ticketID, "root_cause_id", rootCauseID, "tenant_id", tenantID)

	// 获取最新的分析记录
	analysis, err := s.client.RootCauseAnalysis.Query().
		Where(
			rootcauseanalysis.TicketIDEQ(ticketID),
			rootcauseanalysis.TenantIDEQ(tenantID),
		).
		Order(ent.Desc(rootcauseanalysis.FieldCreatedAt)).
		First(ctx)
	if err != nil {
		return fmt.Errorf("analysis not found: %w", err)
	}

	// 更新根因状态为resolved
	if analysis.RootCauses != nil {
		rootCauses := analysis.RootCauses
		for i, rcMap := range rootCauses {
			if id, ok := rcMap["id"].(string); ok && id == rootCauseID {
				rcMap["status"] = "resolved"
				rcMap["resolved_at"] = time.Now().Format(time.RFC3339)
				rootCauses[i] = rcMap
				break
			}
		}

		// 更新分析记录
		_, err = s.client.RootCauseAnalysis.Update().
			Where(rootcauseanalysis.IDEQ(analysis.ID)).
			SetRootCauses(rootCauses).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		if err != nil {
			s.logger.Errorw("Failed to update root cause status", "error", err)
			return fmt.Errorf("failed to update root cause status: %w", err)
		}
	}

	return nil
}

