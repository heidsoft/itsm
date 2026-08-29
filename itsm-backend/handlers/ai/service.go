package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"go.uber.org/zap"
)

type Service struct {
	repo               Repository
	logger             *zap.SugaredLogger
	rag                *service.RAGService
	llmGateway         *service.LLMGateway
	tools              *service.ToolRegistry
	queue              *service.ToolQueue
	analytics          *service.AnalyticsService
	prediction         *service.PredictionService
	slaForecastSkill   *service.SLAForecastSkill
	triageService      *service.TriageService
	rca                *service.RootCauseService
	aiTelemetryService *service.AITelemetryService
	// P2-6: ent client，用于复用 RBAC hasResourcePermission
	entClient *ent.Client
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

// SetLLMGateway wires an optional LLM gateway for streaming answers.
// Kept as a setter to avoid churning existing NewService call sites.
func (s *Service) SetLLMGateway(gateway *service.LLMGateway) {
	s.llmGateway = gateway
}

// SetEntClient wires the ent client for RBAC permission checks.
// P2-6: AI 工具执行前调用 hasResourcePermission 校验用户对工具 Resource/Action 的权限
func (s *Service) SetEntClient(client *ent.Client) {
	s.entClient = client
}

// Tool Methods

func (s *Service) ListTools() []service.ToolDefinition {
	if s.tools == nil {
		return nil
	}
	return s.tools.ListTools()
}

// ErrToolPermissionDenied 工具权限不足（P2-6 Gate 2）
var ErrToolPermissionDenied = fmt.Errorf("tool permission denied")

// ErrUnknownTool 未知工具（P2-6 Gate 2）
var ErrUnknownTool = fmt.Errorf("unknown tool")

// ExecuteTool 执行 AI 工具
// P2-6: 新增 userID 和 role 参数用于 Gate 2 RBAC 校验
//
// 校验分层：
//
//	Gate 1: 路由级 RBACMiddleware（/api/v1/agent/* 检查 ai:read/ai:write）— 调用前已完成
//	Gate 2: 工具级 RBAC（本方法）— 按 ToolDefinition.Resource/Action 校验
//	Gate 3: 审批流（写工具 !ReadOnly）— 由 NeedsApproval 处理
func (s *Service) ExecuteTool(ctx context.Context, userID, tenantID int, role, name string, args map[string]interface{}) (interface{}, int, error) {
	if s.tools == nil {
		return nil, 0, fmt.Errorf("tool registry not initialized")
	}

	// === P2-6 Gate 2: 工具级 RBAC 校验 ===
	permCheck := "skipped"
	permReason := ""
	allowed := true

	toolDef := s.tools.GetTool(name)
	if toolDef == nil {
		// 未知工具：记录 denied 审计，返回错误
		s.recordToolAudit(ctx, tenantID, userID, role, name, args, "denied", "unknown tool", "", nil, false)
		return nil, 0, ErrUnknownTool
	}

	if IsToolRBACEnabled() && s.entClient != nil && role != "" && role != "super_admin" {
		if middleware.HasResourcePermission(ctx, s.entClient, role, toolDef.Resource, toolDef.Action, tenantID) {
			permCheck = "passed"
		} else {
			permCheck = "denied"
			permReason = fmt.Sprintf("role=%s lacks %s:%s", role, toolDef.Resource, toolDef.Action)
			allowed = false
			s.logger.Warnw("AI tool RBAC denied",
				"user_id", userID, "tenant_id", tenantID, "role", role,
				"tool", name, "resource", toolDef.Resource, "action", toolDef.Action,
				"enforce", IsToolRBACEnforce())
		}
	}

	// 影子模式：只记录日志，不拦截；执行模式：拒绝请求
	if !allowed && IsToolRBACEnforce() {
		s.recordToolAudit(ctx, tenantID, userID, role, name, args, permCheck, permReason, "", nil, false)
		return nil, 0, fmt.Errorf("%w: %s", ErrToolPermissionDenied, permReason)
	}

	// Check if needs approval
	needsApproval := !toolDef.ReadOnly

	if !needsApproval {
		res, err := s.tools.Execute(ctx, tenantID, name, args)
		// 只读工具执行也记录审计（AGENTS.md: AI tool invocation must produce audit logs）
		// P2-6: 同步写入 RBAC 校验结果字段
		if err == nil {
			s.recordToolAudit(ctx, tenantID, userID, role, name, args, permCheck, permReason, "executed", nil, false)
		}
		return res, 0, err
	}

	// 写工具：创建 pending invocation，等待审批
	argsStr, _ := json.Marshal(args)
	inv, err := s.repo.CreateToolInvocation(ctx, &ToolInvocation{
		TenantID:         tenantID,
		ToolName:         name,
		Arguments:        string(argsStr),
		Status:           "pending",
		NeedsApproval:    true,
		ApprovalState:    "pending",
		UserID:           userID,
		PermissionCheck:  permCheck,
		PermissionReason: permReason,
		RoleSnapshot:     role,
	})
	if err != nil {
		return nil, 0, err
	}

	return nil, inv.ID, nil
}

// recordToolAudit 统一记录只读工具执行审计，包含 P2-6 RBAC 校验结果
func (s *Service) recordToolAudit(ctx context.Context, tenantID, userID int, role, toolName string, args map[string]interface{}, permCheck, permReason, status string, result *string, needsApproval bool) {
	argsStr, _ := json.Marshal(args)
	_, _ = s.repo.CreateToolInvocation(ctx, &ToolInvocation{
		TenantID:         tenantID,
		ToolName:         toolName,
		Arguments:        string(argsStr),
		Status:           status,
		NeedsApproval:    needsApproval,
		ApprovalState:    "auto",
		UserID:           userID,
		PermissionCheck:  permCheck,
		PermissionReason: permReason,
		RoleSnapshot:     role,
	})
}

func (s *Service) ApproveTool(ctx context.Context, id int, tenantID, userID int, approve bool, reason string) (string, error) {
	inv, err := s.repo.GetToolInvocation(ctx, id, tenantID)
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

// chatWritableTools 是允许注入聊天路径的写工具白名单。
// 这些工具经由 ExecuteTool 的审批流（创建 pending invocation + 入队等待人工审批），
// 绝不在聊天链路内直接落库，因此可安全暴露给 LLM 自主决策调用。
var chatWritableTools = map[string]bool{
	"create_ticket":      true,
	"update_ticket":      true,
	"create_ticket_type": true,
	// CMDB 本体链路：把工单挂到配置项（走同一审批流）
	"link_ticket_ci": true,
}

// ChatStream streams a RAG answer through onDelta while emitting sources
// separately via onSources. It also persists the resulting conversation and
// messages after the stream completes so history is preserved.
//
// P1-B: 注入按权限过滤后的只读工具。只有当前角色拥有 resource:action 权限的只读工具
// 才会进入聊天路径；写工具（!ReadOnly）绝不注入。模型发起的工具调用经由 execTool
// 复用 ExecuteTool 的 RBAC Gate 2 校验 + 审计记录，执行结果回填后继续流式生成。
func (s *Service) ChatStream(
	ctx context.Context,
	tenantID, userID int,
	role string,
	query string,
	limit int,
	convID int,
	onSources func([]map[string]any),
	onDelta func(string),
) (int, string, error) {
	s.logger.Infow("AI ChatStream", "query", query, "tenantID", tenantID, "convID", convID, "role", role)

	if s.rag == nil {
		return 0, "", fmt.Errorf("RAG service not initialized")
	}

	// 按权限过滤只读工具注入聊天路径；写工具仅放行经审批流的白名单（创建工单/更新工单/创建工单类型），
	// 它们经由 ExecuteTool 进入待审批队列，绝不在聊天链路内直接落地写库。
	var tools []service.LLMTool
	if s.tools != nil {
		for _, td := range s.tools.ListTools() {
			if !td.ReadOnly && !chatWritableTools[td.Name] {
				continue
			}
			if IsToolRBACEnabled() && s.entClient != nil && role != "" && role != "super_admin" {
				if !middleware.HasResourcePermission(ctx, s.entClient, role, td.Resource, td.Action, tenantID) {
					s.logger.Debugw("AI ChatStream: tool filtered by RBAC",
						"tool", td.Name, "resource", td.Resource, "action", td.Action, "role", role, "tenantID", tenantID)
					continue
				}
			}
			tools = append(tools, service.LLMTool{
				Name:        td.Name,
				Description: td.Description,
				Parameters:  td.ArgsSchema,
			})
		}
	}

	// 工具执行回调：复用 ExecuteTool 的 RBAC Gate 2 + 审计，与 agent 执行路径一致。
	// 写工具经审批流会返回 (res=nil, invID!=0, err=nil)，这里将其转译为结构化
	// approval_pending 提示，让 LLM 明确告知用户"操作已提交、待人工审批"。
	execTool := func(name string, args map[string]any) (any, error) {
		// 注入操作者身份，便于工单归属与审计（审批队列回放时据此归因）
		if _, ok := args["user_id"]; !ok {
			args["user_id"] = float64(userID)
		}
		if _, ok := args["requester_id"]; !ok {
			args["requester_id"] = float64(userID)
		}
		res, invID, err := s.ExecuteTool(ctx, userID, tenantID, role, name, args)
		if err != nil {
			return nil, err
		}
		if res == nil && invID != 0 {
			return map[string]any{
				"status":       "approval_pending",
				"tool":         name,
				"invocationId": invID,
				"message":      fmt.Sprintf("操作已提交，等待人工审批后执行（invocationId=%d）", invID),
			}, nil
		}
		return res, nil
	}

	var (
		captured strings.Builder
		sources  []map[string]any
	)

	wrappedSources := func(items []map[string]any) {
		sources = items
		if onSources != nil {
			onSources(items)
		}
	}
	wrappedDelta := func(delta string) {
		captured.WriteString(delta)
		if onDelta != nil {
			onDelta(delta)
		}
	}

	if err := s.rag.AskWithLLMStreamWithTools(ctx, tenantID, query, s.llmGateway, limit, tools, wrappedSources, wrappedDelta, execTool); err != nil {
		return 0, "", err
	}

	// Persist conversation and messages after the stream completes so we don't
	// leave partial messages if the client disconnects mid-stream.
	if convID == 0 {
		conv, err := s.repo.CreateConversation(ctx, &Conversation{
			Title:    "AI 对话",
			UserID:   userID,
			TenantID: tenantID,
		})
		if err == nil && conv != nil {
			convID = conv.ID
		}
	}
	if convID != 0 {
		_, _ = s.repo.CreateMessage(ctx, &Message{
			ConversationID: convID,
			Role:           "user",
			Content:        query,
		})
		payload := map[string]any{
			"answer":  captured.String(),
			"sources": sources,
		}
		buf, _ := json.Marshal(payload)
		_, _ = s.repo.CreateMessage(ctx, &Message{
			ConversationID: convID,
			Role:           "assistant",
			Content:        string(buf),
		})
	}

	return convID, captured.String(), nil
}

// Root Cause Analysis

func (s *Service) AnalyzeTicket(ctx context.Context, ticketID int, tenantID int) (interface{}, error) {
	return s.rca.AnalyzeTicket(ctx, ticketID, tenantID)
}

func (s *Service) AnalyzeIncident(ctx context.Context, incidentID int, tenantID int) (interface{}, error) {
	if s.rca == nil {
		return nil, service.ErrAIAnalysisUnavailable
	}
	return s.rca.AnalyzeIncident(ctx, incidentID, tenantID)
}

// CreateTicketByAI 通过 AI 解析自然语言描述，智能分析并返回工单创建建议
func (s *Service) CreateTicketByAI(ctx context.Context, description string, tenantID int) (map[string]interface{}, error) {
	s.logger.Infow("AI CreateTicketByAI", "tenantID", tenantID)

	// 使用 Triage 服务分析描述，提取分类和优先级
	if s.triageService != nil {
		result := s.triageService.Suggest(ctx, description, description)
		return map[string]interface{}{
			"suggested_title":    description,
			"suggested_category": result.Category,
			"suggested_priority": result.Priority,
			"confidence":         result.Confidence,
			"reasoning":          result.Explanation,
			"tenant_id":          tenantID,
			"status":             "draft",
			"message":            "AI 已分析描述，请确认后提交工单",
		}, nil
	}

	// fallback：基于关键词简单分析
	return map[string]interface{}{
		"suggested_title":    description,
		"suggested_category": "general",
		"suggested_priority": "medium",
		"confidence":         0.5,
		"reasoning":          "基于关键词分析",
		"tenant_id":          tenantID,
		"status":             "draft",
		"message":            "AI 服务不可用，已返回默认建议",
	}, nil
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
	if s.aiTelemetryService == nil {
		return fmt.Errorf("AI telemetry service not initialized")
	}
	return s.aiTelemetryService.SaveFeedback(ctx, tenantID, userID, requestID, kind, query, itemType, itemID, useful, score, notes)
}

func (s *Service) GetMetrics(ctx context.Context, tenantID int, lookbackDays int) (interface{}, error) {
	if s.aiTelemetryService == nil {
		return nil, fmt.Errorf("AI telemetry service not initialized")
	}
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

// Evaluate delegates to AITelemetryService.Evaluate (nil-safe for unit tests).
func (s *Service) Evaluate(ctx context.Context, tenantID int, days int) (*service.AIEvaluationReport, error) {
	if s.aiTelemetryService == nil {
		return &service.AIEvaluationReport{GeneratedAt: time.Now().Format(time.RFC3339), LookbackDays: days}, nil
	}
	return s.aiTelemetryService.Evaluate(ctx, tenantID, days)
}

// ListAuditLogs delegates to AITelemetryService.ListAuditLogs (nil-safe for unit tests).
func (s *Service) ListAuditLogs(ctx context.Context, tenantID, page, pageSize int, kind string, days int) ([]service.AIAuditEntry, int, error) {
	if s.aiTelemetryService == nil {
		return []service.AIAuditEntry{}, 0, nil
	}
	return s.aiTelemetryService.ListAuditLogs(ctx, tenantID, page, pageSize, kind, days)
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
