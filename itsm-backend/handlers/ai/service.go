package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/service"

	"go.uber.org/zap"
)

type Service struct {
	repo               Repository
	logger             *zap.SugaredLogger
	rag                *service.RAGService
	tools              *service.ToolRegistry
	queue              *service.ToolQueue
	analytics          *service.AnalyticsService
	prediction         *service.PredictionService
	slaForecastSkill   *service.SLAForecastSkill
	triageService      *service.TriageService
	rca                *service.RootCauseService
	aiTelemetryService *service.AITelemetryService
}

func NewService(
	repo Repository,
	logger *zap.SugaredLogger,
	rag *service.RAGService,
	tools *service.ToolRegistry,
	queue *service.ToolQueue,
	analytics *service.AnalyticsService,
	prediction *service.PredictionService,
	slaForecastSkill *service.SLAForecastSkill,
	triageService *service.TriageService,
	rca *service.RootCauseService,
	aiTelemetryService *service.AITelemetryService,
) *Service {
	return &Service{
		repo:               repo,
		logger:             logger,
		rag:                rag,
		tools:              tools,
		queue:              queue,
		analytics:          analytics,
		prediction:         prediction,
		slaForecastSkill:   slaForecastSkill,
		triageService:      triageService,
		rca:                rca,
		aiTelemetryService: aiTelemetryService,
	}
}

// Tool Methods

func (s *Service) ListTools() []service.ToolDefinition {
	if s.tools == nil {
		return nil
	}
	return s.tools.ListTools()
}

func (s *Service) ExecuteTool(ctx context.Context, tenantID int, name string, args map[string]interface{}) (interface{}, int, error) {
	if s.tools == nil {
		return nil, 0, fmt.Errorf("tool registry not initialized")
	}

	// Check if needs approval
	needsApproval := false
	for _, t := range s.tools.ListTools() {
		if t.Name == name {
			needsApproval = !t.ReadOnly
			break
		}
	}

	if !needsApproval {
		res, err := s.tools.Execute(ctx, tenantID, name, args)
		return res, 0, err
	}

	// Create pending invocation
	argsStr, _ := json.Marshal(args)
	inv, err := s.repo.CreateToolInvocation(ctx, &ToolInvocation{
		ToolName:      name,
		Arguments:     string(argsStr),
		Status:        "pending",
		NeedsApproval: true,
		ApprovalState: "pending",
	})
	if err != nil {
		return nil, 0, err
	}

	return nil, inv.ID, nil
}

func (s *Service) ApproveTool(ctx context.Context, id int, tenantID, userID int, approve bool, reason string) (string, error) {
	inv, err := s.repo.GetToolInvocation(ctx, id)
	if err != nil {
		return "", err
	}

	if !approve {
		inv.ApprovalState = "rejected"
		inv.ApprovalReason = reason
		_, err = s.repo.UpdateToolInvocation(ctx, inv)
		return "rejected", err
	}

	inv.ApprovalState = "approved"
	inv.ApprovedBy = userID
	now := time.Now()
	inv.ApprovedAt = &now
	_, err = s.repo.UpdateToolInvocation(ctx, inv)
	if err != nil {
		return "", err
	}

	if s.queue != nil {
		s.queue.Enqueue(service.ToolJob{
			InvocationID: inv.ID,
			TenantID:     tenantID,
		})
	}

	return "approved", nil
}

// Chat and RAG

func (s *Service) Chat(ctx context.Context, tenantID, userID int, query string, limit int, convID int) (interface{}, int, error) {
	s.logger.Infow("AI Chat", "query", query, "tenantID", tenantID)

	items, err := s.rag.Ask(ctx, tenantID, query, limit)
	if err != nil {
		return nil, 0, err
	}

	// Persist conversation
	if convID == 0 {
		conv, err := s.repo.CreateConversation(ctx, &Conversation{
			Title:    "AI 对话",
			UserID:   userID,
			TenantID: tenantID,
		})
		if err == nil {
			convID = conv.ID
		}
	}

	if convID != 0 {
		_, _ = s.repo.CreateMessage(ctx, &Message{
			ConversationID: convID,
			Role:           "user",
			Content:        query,
		})
		payload, _ := json.Marshal(items)
		_, _ = s.repo.CreateMessage(ctx, &Message{
			ConversationID: convID,
			Role:           "assistant",
			Content:        string(payload),
		})
	}

	return items, convID, nil
}

// Root Cause Analysis

func (s *Service) AnalyzeTicket(ctx context.Context, ticketID int, tenantID int) (interface{}, error) {
	return s.rca.AnalyzeTicket(ctx, ticketID, tenantID)
}

// SummarizeTicket B9: AI 工单总结 - 委托 RootCauseService 处理（有 ent 访问）
func (s *Service) SummarizeTicket(ctx context.Context, ticketID int, tenantID int) (interface{}, error) {
	return s.rca.SummarizeTicket(ctx, ticketID, tenantID)
}

func (s *Service) GetAnalysisReport(ctx context.Context, ticketID int, tenantID int) (interface{}, error) {
	return s.rca.GetAnalysisReport(ctx, ticketID, tenantID)
}

// Analytics and Prediction

func (s *Service) GetDeepAnalytics(ctx context.Context, req *dto.DeepAnalyticsRequest, tenantID int) (interface{}, error) {
	return s.analytics.GetDeepAnalytics(ctx, req, tenantID)
}

func (s *Service) GetTrendPrediction(ctx context.Context, req *dto.TrendPredictionRequest, tenantID int) (interface{}, error) {
	// Try to use AI-Native SLAForecastSkill first
	if s.slaForecastSkill != nil {
		input := &service.ForecastInput{
			TenantID:  tenantID,
			StartDate: parseDate(req.TimeRange[0]),
			EndDate:   parseDate(req.TimeRange[1]),
			Metrics:   []string{req.PredictionType},
		}
		output, err := s.slaForecastSkill.Execute(ctx, input)
		if err == nil {
			return output, nil
		}
		// Fall back to legacy prediction service on error
		s.logger.Warnw("SLAForecastSkill failed, falling back to legacy", "error", err)
	}
	return s.prediction.GetTrendPrediction(ctx, req, tenantID)
}

// GetForecastInsights returns AI-generated insights for SLA forecasting
func (s *Service) GetForecastInsights(ctx context.Context, req *dto.TrendPredictionRequest, tenantID int) (interface{}, error) {
	if s.slaForecastSkill == nil {
		return nil, fmt.Errorf("SLAForecastSkill not initialized")
	}

	input := &service.ForecastInput{
		TenantID:  tenantID,
		StartDate: parseDate(req.TimeRange[0]),
		EndDate:   parseDate(req.TimeRange[1]),
		Metrics:   []string{req.PredictionType},
	}

	output, err := s.slaForecastSkill.Execute(ctx, input)
	if err != nil {
		return nil, err
	}

	// Return insights-focused response
	return map[string]interface{}{
		"confidence":    output.Confidence,
		"model":         output.Model,
		"insights":      output.Insights,
		"seasonality":   output.Seasonality,
		"trend":         output.Trend,
		"anomaly_dates": output.AnomalyDates,
	}, nil
}

// Telemetry

func (s *Service) SaveFeedback(ctx context.Context, tenantID, userID int, requestID, kind, query, itemType string, itemID *int, useful bool, score *int, notes *string) error {
	return s.aiTelemetryService.SaveFeedback(ctx, tenantID, userID, requestID, kind, query, itemType, itemID, useful, score, notes)
}

func (s *Service) GetMetrics(ctx context.Context, tenantID int, lookbackDays int) (interface{}, error) {
	return s.aiTelemetryService.GetMetrics(ctx, tenantID, lookbackDays)
}

// SearchKnowledge handles RAG search over knowledge base
func (s *Service) SearchKnowledge(ctx context.Context, tenantID int, query string, searchType string, limit int) (interface{}, error) {
	s.logger.Infow("Knowledge Search", "query", query, "type", searchType, "tenantID", tenantID)

	if s.rag == nil {
		// Fallback to basic search if RAG is not available
		return []map[string]interface{}{}, nil
	}

	results, err := s.rag.Ask(ctx, tenantID, query, limit)
	if err != nil {
		s.logger.Warnw("RAG search failed", "error", err)
		return []map[string]interface{}{}, nil
	}

	return results, nil
}

// TriageTicket provides ticket classification and recommendations using LLM
func (s *Service) TriageTicket(ctx context.Context, tenantID int, title, description, category, priority string) (interface{}, error) {
	s.logger.Infow("Ticket Triage with LLM", "title", title, "tenantID", tenantID)

	// Use LLM-powered TriageService if available
	if s.triageService != nil {
		result := s.triageService.Suggest(ctx, title, description)
		return map[string]interface{}{
			"title":       title,
			"description": description,
			"suggestions": map[string]interface{}{
				"category":   result.Category,
				"priority":   result.Priority,
				"confidence": result.Confidence,
				"reasoning":  result.Explanation,
				"urgency":    s.determineUrgency(result.Priority),
			},
		}, nil
	}

	// Fallback to keyword-based classification
	result := map[string]interface{}{
		"title":       title,
		"description": description,
		"suggestions": make(map[string]interface{}),
	}

	suggestedCategory := category
	suggestedPriority := priority
	suggestedUrgency := "medium"

	titleLower := title
	if len(titleLower) > 0 {
		switch {
		case containsAny(titleLower, "网络", "网速", "连接", "wifi", "网络"):
			suggestedCategory = "network"
		case containsAny(titleLower, "软件", "应用", "系统", "程序", "app"):
			suggestedCategory = "software"
		case containsAny(titleLower, "硬件", "电脑", "设备", "服务器", "hardware"):
			suggestedCategory = "hardware"
		case containsAny(titleLower, "账号", "密码", "权限", "登录", "access"):
			suggestedCategory = "access"
		case containsAny(titleLower, "打印机", "打印", "print"):
			suggestedCategory = "printer"
		case containsAny(titleLower, "邮箱", "邮件", "email", "outlook"):
			suggestedCategory = "email"
		default:
			suggestedCategory = "general"
		}
	}

	descLower := description
	if containsAny(descLower, "紧急", "严重", "无法工作", "critical", "urgent", "emergency") {
		suggestedPriority = "critical"
		suggestedUrgency = "high"
	} else if containsAny(descLower, "重要", "影响工作", "high", "important") {
		suggestedPriority = "high"
		suggestedUrgency = "high"
	} else if containsAny(descLower, "不紧急", "low", "minor") {
		suggestedPriority = "low"
		suggestedUrgency = "low"
	}

	suggestions := make(map[string]interface{})
	suggestions["category"] = suggestedCategory
	suggestions["priority"] = suggestedPriority
	suggestions["urgency"] = suggestedUrgency
	suggestions["confidence"] = 0.7
	suggestions["reasoning"] = "Based on keyword analysis"

	result["suggestions"] = suggestions

	return result, nil
}

func (s *Service) determineUrgency(priority string) string {
	switch priority {
	case "critical":
		return "high"
	case "high":
		return "high"
	case "medium":
		return "medium"
	case "low":
		return "low"
	default:
		return "medium"
	}
}

// containsAny checks if string contains any of the keywords
func containsAny(s string, keywords ...string) bool {
	for _, kw := range keywords {
		if len(s) >= len(kw) {
			for i := 0; i <= len(s)-len(kw); i++ {
				if len(s[i:i+len(kw)]) >= len(kw) {
					// Simple case-insensitive check
					sub := ""
					for j := 0; j < len(kw); j++ {
						if i+j < len(s) {
							c := s[i+j]
							if c >= 'A' && c <= 'Z' {
								c = c + 32
							}
							sub += string(c)
						}
					}
					if sub == kw {
						return true
					}
				}
			}
		}
	}
	return false
}

// parseDate parses date string in YYYY-MM-DD format
func parseDate(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Now()
	}
	return t
}
