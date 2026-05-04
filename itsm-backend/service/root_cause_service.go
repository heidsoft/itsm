package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/rootcauseanalysis"
	"itsm-backend/ent/ticket"

	"go.uber.org/zap"
)

type RootCauseService struct {
	client  *ent.Client
	logger  *zap.SugaredLogger
	gateway *LLMGateway
}

func NewRootCauseService(client *ent.Client, logger *zap.SugaredLogger) *RootCauseService {
	return &RootCauseService{
		client: client,
		logger: logger,
	}
}

// SetGateway sets the LLM gateway for AI-powered analysis
func (s *RootCauseService) SetGateway(gateway *LLMGateway) {
	s.gateway = gateway
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
			"id":          rc.ID,
			"title":       rc.Title,
			"description": rc.Description,
			"confidence":  rc.Confidence,
			"category":    rc.Category,
			"status":      rc.Status,
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
				ID:        fmt.Sprintf("rc%d", i+1),
				CreatedAt: analysis.CreatedAt,
				UpdatedAt: analysis.UpdatedAt,
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

// performAnalysis 执行分析
// 当 LLM Gateway 可用时，使用 AI 进行根因分析；否则回退到基于规则的启发式分析
func (s *RootCauseService) performAnalysis(ticketEntity *ent.Ticket, tenantID int) []dto.TicketRootCauseResponse {
	rootCauses := make([]dto.TicketRootCauseResponse, 0)

	// 如果有 LLM Gateway，使用 AI 分析
	if s.gateway != nil {
		aiResults, err := s.performAIAnalysis(ticketEntity)
		if err == nil && len(aiResults) > 0 {
			return aiResults
		}
		// AI 分析失败，记录日志并继续使用启发式分析
		s.logger.Warnw("AI root cause analysis failed, falling back to heuristic", "error", err)
	}

	// 基于规则的启发式分析（原有逻辑）
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

// performAIAnalysis 使用 LLM 进行根因分析
func (s *RootCauseService) performAIAnalysis(ticketEntity *ent.Ticket) ([]dto.TicketRootCauseResponse, error) {
	ctx := context.Background()

	// 构建分析提示词
	prompt := fmt.Sprintf(`你是一个资深的 IT 运维专家，负责分析工单的根本原因。

工单信息：
- 工单编号：%s
- 标题：%s
- 描述：%s
- 优先级：%s
- 状态：%s
- 工单类型：%s
- 创建时间：%s

请分析这个工单，找出可能的根本原因。使用以下 JSON 格式返回结果（只返回 JSON，不要其他内容）：
{
  "root_causes": [
    {
      "title": "原因标题",
      "description": "详细描述",
      "confidence": 0.85,
      "category": "infrastructure|software|process|network|security|other"
    }
  ]
}

要求：
- 提供 1-3 个最可能的根本原因
- confidence 值为 0-1 之间的小数
- category 必须是指定的值之一`, ticketEntity.TicketNumber, ticketEntity.Title, ticketEntity.Description,
		ticketEntity.Priority, ticketEntity.Status, ticketEntity.Type, ticketEntity.CreatedAt.Format(time.RFC3339))

	messages := []LLMMessage{
		{Role: "system", Content: "你是一个专业的 IT 服务管理根因分析助手，擅长分析工单找出根本原因。"},
		{Role: "user", Content: prompt},
	}

	response, err := s.gateway.Chat(ctx, "", messages)
	if err != nil {
		return nil, fmt.Errorf("LLM analysis failed: %w", err)
	}

	// 解析 JSON 响应
	// 提取 JSON 部分（处理可能的 markdown 格式）
	jsonStr := response
	if idx := strings.Index(jsonStr, "{"); idx > 0 {
		jsonStr = jsonStr[idx:]
	}
	if idx := strings.LastIndex(jsonStr, "}"); idx >= 0 {
		jsonStr = jsonStr[:idx+1]
	}

	var result struct {
		RootCauses []struct {
			Title       string  `json:"title"`
			Description string  `json:"description"`
			Confidence  float64 `json:"confidence"`
			Category    string  `json:"category"`
		} `json:"root_causes"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	rootCauses := make([]dto.TicketRootCauseResponse, 0)
	for i, rc := range result.RootCauses {
		rootCauses = append(rootCauses, dto.TicketRootCauseResponse{
			ID:          fmt.Sprintf("rc%d", i+1),
			Title:       rc.Title,
			Description: rc.Description,
			Confidence:  rc.Confidence,
			Category:    rc.Category,
			Status:      "identified",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}

	return rootCauses, nil
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
