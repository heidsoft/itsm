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

func (s *Service) GetAnalysisReport(ctx context.Context, ticketID int, tenantID int) (interface{}, error) {
	return s.rca.GetAnalysisReport(ctx, ticketID, tenantID)
}

// Analytics and Prediction

func (s *Service) GetDeepAnalytics(ctx context.Context, req *dto.DeepAnalyticsRequest, tenantID int) (interface{}, error) {
	return s.analytics.GetDeepAnalytics(ctx, req, tenantID)
}

func (s *Service) GetTrendPrediction(ctx context.Context, req *dto.TrendPredictionRequest, tenantID int) (interface{}, error) {
	return s.prediction.GetTrendPrediction(ctx, req, tenantID)
}

// Telemetry

func (s *Service) SaveFeedback(ctx context.Context, tenantID, userID int, requestID, kind, query, itemType string, itemID *int, useful bool, score *int, notes *string) error {
	return s.aiTelemetryService.SaveFeedback(ctx, tenantID, userID, requestID, kind, query, itemType, itemID, useful, score, notes)
}

func (s *Service) GetMetrics(ctx context.Context, tenantID int, lookbackDays int) (interface{}, error) {
	return s.aiTelemetryService.GetMetrics(ctx, tenantID, lookbackDays)
}
