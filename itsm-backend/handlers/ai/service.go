package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/ticket"
	"itsm-backend/metrics"
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
	analytics          *service.AnalyticsService
	prediction         *service.PredictionService
	slaForecastSkill   *service.SLAForecastSkill
	triageService      *service.TriageService
	rca                *service.RootCauseService
	aiTelemetryService *service.AITelemetryService
	// SummarizeService 装配路径；摘要优先走它，不再绕行 rca.SummarizeTicket。
	summarizeService *service.SummarizeService
	// ent client，用于复用 RBAC hasResourcePermission
	entClient *ent.Client
	// auditLedger 服务端 AI 决策底账写口（窄接口，生产实现是 AITelemetryService）。
	// nil 时分诊不落底账，响应不携带 auditId（降级不阻塞）。
	auditLedger AuditLedgerWriter
}

// AuditLedgerWriter 是服务端 AI 决策底账的窄接口：产出 AI 建议时自动落一条
// item_type='ai_audit' 行（accepted=NULL 未决策），返回行 ID 供操作者经
// PATCH /ai/audit-logs/:id 翻转采纳/拒绝。生产实现 *service.AITelemetryService。
type AuditLedgerWriter interface {
	RecordDecisionAudit(ctx context.Context, tenantID, userID int, reqID, scenario, inputRef, promptVersion, model string, confidence float64, suggestion map[string]interface{}) (int64, error)
}

// 编译期断言：生产实现满足窄接口。
var _ AuditLedgerWriter = (*service.AITelemetryService)(nil)

func NewService(
	repo Repository,
	logger *zap.SugaredLogger,
	rag *service.RAGService,
	tools *service.ToolRegistry,
	analytics *service.AnalyticsService,
	prediction *service.PredictionService,
	slaForecastSkill *service.SLAForecastSkill,
	triageService *service.TriageService,
	rca *service.RootCauseService,
	aiTelemetryService *service.AITelemetryService,
) *Service {
	s := &Service{
		repo:               repo,
		logger:             logger,
		rag:                rag,
		tools:              tools,
		analytics:          analytics,
		prediction:         prediction,
		slaForecastSkill:   slaForecastSkill,
		triageService:      triageService,
		rca:                rca,
		aiTelemetryService: aiTelemetryService,
	}
	// 遥测服务就绪时默认作为底账写口；测试可用 SetAuditLedger 覆盖为 mock。
	if aiTelemetryService != nil {
		s.auditLedger = aiTelemetryService
	}
	return s
}

// SetAuditLedger 注入/替换决策底账写口（测试用；bootstrap 走 NewService 默认装配）。
func (s *Service) SetAuditLedger(l AuditLedgerWriter) {
	s.auditLedger = l
}

// SetLLMGateway wires an optional LLM gateway for streaming answers.
// Kept as a setter to avoid churning existing NewService call sites.
func (s *Service) SetLLMGateway(gateway *service.LLMGateway) {
	s.llmGateway = gateway
}

// SetSummarizeService wires SummarizeService。
// SummarizeTicket 优先走 SummarizeService.SummarizeWithMetadata，失败兜底回 rca。
func (s *Service) SetSummarizeService(svc *service.SummarizeService) {
	s.summarizeService = svc
}

// SetEntClient wires the ent client for RBAC permission checks.
// AI 工具执行前调用 hasResourcePermission 校验用户对工具 Resource/Action 的权限
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

// ListToolsForTenant 返回按租户动态化的工具清单（list_cis 的 ci_type 枚举来自租户 CIType 表）。
func (s *Service) ListToolsForTenant(ctx context.Context, tenantID int) []service.ToolDefinition {
	if s.tools == nil {
		return nil
	}
	return s.tools.ListToolsForTenant(ctx, tenantID)
}

// ErrToolPermissionDenied 工具权限不足
var ErrToolPermissionDenied = fmt.Errorf("tool permission denied")

// ErrUnknownTool 未知工具
var ErrUnknownTool = fmt.Errorf("unknown tool")

var ErrToolUnavailable = fmt.Errorf("tool authorization dependencies unavailable")

// ErrRAGUnavailable RAG 检索服务未装配。此前 rag 为 nil 时 Chat 直接解引用
// 崩溃；现在返回错误，由 handler 的降级分支接住（返回空结果而不是 500）。
var ErrRAGUnavailable = fmt.Errorf("rag retrieval service unavailable")

// ExecuteTool 执行 AI 工具
// userID 和 role 用于 Gate 2 RBAC 校验
//
// 校验分层：
//
//	Gate 1: 路由级 RBACMiddleware（/api/v1/agent/* 检查 ai:read/ai:write）— 调用前已完成
//	Gate 2: 工具级 RBAC（本方法）— 按 ToolDefinition.Resource/Action 校验
//	Gate 3: 审批流（写工具 !ReadOnly）— 由 NeedsApproval 处理
func (s *Service) ExecuteTool(ctx context.Context, userID, tenantID int, role, name string, args map[string]interface{}) (interface{}, int, error) {
	if userID <= 0 || tenantID <= 0 {
		return nil, 0, ErrToolPermissionDenied
	}
	if s.tools == nil || s.entClient == nil {
		return nil, 0, ErrToolUnavailable
	}

	// Gate 2: 工具级 RBAC 校验
	permCheck := "passed"
	permReason := ""
	allowed := true

	toolDef := s.tools.GetTool(name)
	if toolDef == nil {
		// 未知工具：记录 denied 审计，返回错误
		s.recordToolAudit(ctx, tenantID, userID, role, name, args, "denied", "unknown tool", "", nil, false)
		return nil, 0, ErrUnknownTool
	}

	// 工具定义完整性校验：Resource/Action 必须显式声明。
	// 漏配时 fail-closed，且审计里写明是配置缺陷而非用户越权——
	// 否则空值会在 middleware.AuthorizeResource 的通配匹配里产生意外放行
	// （该入口也已显式拒绝空值，这里是更早、更可诊断的一道）。
	if toolDef.Resource == "" || toolDef.Action == "" {
		reason := fmt.Sprintf("tool %s missing resource/action declaration", name)
		s.logger.Errorw("AI tool definition incomplete",
			"tool", name, "resource", toolDef.Resource, "action", toolDef.Action)
		s.recordToolAudit(ctx, tenantID, userID, role, name, args, "denied", reason, "", nil, false)
		return nil, 0, fmt.Errorf("%w: %s", ErrToolPermissionDenied, reason)
	}

	{
		if role != "" && middleware.HasResourcePermission(ctx, s.entClient, role, toolDef.Resource, toolDef.Action, tenantID) {
			permCheck = "passed"
		} else {
			permCheck = "denied"
			permReason = fmt.Sprintf("role=%s lacks %s:%s", role, toolDef.Resource, toolDef.Action)
			allowed = false
			s.logger.Warnw("AI tool RBAC denied",
				"user_id", userID, "tenant_id", tenantID, "role", role,
				"tool", name, "resource", toolDef.Resource, "action", toolDef.Action,
				"enforce", true)
		}
	}

	// Authorization is mandatory; feature flags must never bypass it.
	if !allowed {
		s.recordToolAudit(ctx, tenantID, userID, role, name, args, permCheck, permReason, "", nil, false)
		return nil, 0, fmt.Errorf("%w: %s", ErrToolPermissionDenied, permReason)
	}

	// Check if needs approval
	needsApproval := !toolDef.ReadOnly

	if !needsApproval {
		res, err := s.tools.Execute(ctx, tenantID, name, args)
		// 只读工具执行也记录审计（AGENTS.md: AI tool invocation must produce audit logs）
		// 同步写入 RBAC 校验结果字段
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

// recordToolAudit 统一记录只读工具执行审计，包含 RBAC 校验结果
//
// 审计写入失败不能用 _, _ = 吞掉，走结构化错误日志 +
// itsm_ai_persist_errors_total{operation="create_tool_invocation"} 计数器埋点，
// 保证 DB 抖动时审计丢失可观测、可告警。
func (s *Service) recordToolAudit(ctx context.Context, tenantID, userID int, role, toolName string, args map[string]interface{}, permCheck, permReason, status string, result *string, needsApproval bool) {
	argsStr, _ := json.Marshal(args)
	if _, err := s.repo.CreateToolInvocation(ctx, &ToolInvocation{
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
	}); err != nil {
		s.logger.Errorw("AI tool audit persistence failed",
			"operation", "create_tool_invocation",
			"tenant_id", tenantID,
			"user_id", userID,
			"tool_name", toolName,
			"approval_state", "auto",
			"error", err,
		)
		metrics.AIPersistErrors.WithLabelValues(
			"create_tool_invocation",
			"",
			strconv.Itoa(tenantID),
		).Inc()
	}
}

// ApproveTool 审批一次工具调用。
// 审批状态与待执行任务在同一事务内落库（见 Repository.DecideToolInvocation），
// 因此通过审批不会再因为进程退出或队列满而丢失执行。
func (s *Service) ApproveTool(ctx context.Context, id int, tenantID, userID int, approve bool, reason string) (string, error) {
	if _, err := s.repo.DecideToolInvocation(ctx, id, tenantID, userID, approve, reason); err != nil {
		return "", err
	}
	if approve {
		return "approved", nil
	}
	return "rejected", nil
}

// Chat and RAG

func (s *Service) Chat(ctx context.Context, tenantID, userID int, query string, limit int, convID int) (interface{}, int, error) {
	s.logger.Infow("AI Chat", "query", query, "tenantID", tenantID)

	// RAG 未装配时返回错误而不是 panic：此前直接 s.rag.Ask(...) 在 rag 为 nil
	// 时解引用崩溃（nil receiver 上的字段访问）。生产装配一定会注入，但装配
	// 缺口不该以 500 + 崩溃的形式暴露——handler 已有降级分支可接住错误。
	if s.rag == nil {
		return nil, 0, ErrRAGUnavailable
	}

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
		// Chat 路径持久化失败必须可见。两类消息分别打点，避免一次失败掩盖另一类错误。
		if _, err := s.repo.CreateMessage(ctx, &Message{
			ConversationID: convID,
			Role:           "user",
			Content:        query,
		}); err != nil {
			s.logger.Errorw("AI chat persist user message failed",
				"operation", "create_message",
				"tenant_id", tenantID,
				"user_id", userID,
				"conversation_id", convID,
				"role", "user",
				"error", err,
			)
			metrics.AIPersistErrors.WithLabelValues(
				"create_message",
				"user",
				strconv.Itoa(tenantID),
			).Inc()
		}
		payload, _ := json.Marshal(items)
		if _, err := s.repo.CreateMessage(ctx, &Message{
			ConversationID: convID,
			Role:           "assistant",
			Content:        string(payload),
		}); err != nil {
			s.logger.Errorw("AI chat persist assistant message failed",
				"operation", "create_message",
				"tenant_id", tenantID,
				"user_id", userID,
				"conversation_id", convID,
				"role", "assistant",
				"error", err,
			)
			metrics.AIPersistErrors.WithLabelValues(
				"create_message",
				"assistant",
				strconv.Itoa(tenantID),
			).Inc()
		}
	}

	return items, convID, nil
}

// ragAskPromptVersion 是 RAG 问答的检索口径版本：换 embedding 模型、换召回
// 策略（top-k / 阈值 / 融合权重）都必须换它，否则底账里新旧口径的答案混在一起，
// 事后无法解释「为什么那批答案质量明显不同」。
const ragAskPromptVersion = "rag-ask-v1"

// ChatResult 是带检索溯源的问答结果。
//
// Model 是**检索模式**（hybrid/vector/keyword）而不是生成模型：RAG 问答的答案
// 片段来自知识库检索，不经过生成模型。把它写成某个模型名就是伪造归因。
// Confidence 是命中片段里的最高分（真实检索分，0 表示无命中）。
type ChatResult struct {
	Answers        interface{} `json:"answers"`
	ConversationID int         `json:"conversationId"`
	AuditID        int64       `json:"auditId,omitempty"`
	PromptVersion  string      `json:"promptVersion"`
	Model          string      `json:"model"`
	Confidence     float64     `json:"confidence"`
}

// ChatWithTrace 在 Chat 之上补检索溯源并落服务端决策底账。
//
// 为什么单独一个方法而不是改 Chat 的签名：Chat 的 (answers, convID, error)
// 三元组被 handler、skill、测试多处使用，改动面大；溯源是新增能力，
// 独立入口不影响既有调用。
//
// 底账照落但 confidence 是真实检索分：无命中时为 0——「问了但库里没有」是最
// 需要被记录的一类情况，否则它会被当成「AI 答得不好」而不是「知识库缺内容」。
func (s *Service) ChatWithTrace(ctx context.Context, tenantID, userID int, query string, limit, convID int) (*ChatResult, error) {
	answers, cid, err := s.Chat(ctx, tenantID, userID, query, limit, convID)
	if err != nil {
		return nil, err
	}

	mode := "unknown"
	if s.rag != nil {
		mode = s.rag.Mode()
	}
	confidence := topRetrievalScore(answers)

	out := &ChatResult{
		Answers:        answers,
		ConversationID: cid,
		PromptVersion:  ragAskPromptVersion,
		Model:          mode,
		Confidence:     confidence,
	}
	if s.auditLedger == nil {
		return out, nil
	}
	auditID, ledgerErr := s.auditLedger.RecordDecisionAudit(ctx, tenantID, userID,
		fmt.Sprintf("chat_%d_%d", tenantID, time.Now().UnixNano()),
		"chat", fmt.Sprintf("query:%s", truncateForRef(query, 120)),
		ragAskPromptVersion, mode, confidence,
		map[string]interface{}{
			"retrievalMode": mode,
			"hitCount":      hitCount(answers),
		},
	)
	if ledgerErr != nil {
		s.logger.Warnw("Chat: audit ledger write failed (best-effort)", "error", ledgerErr)
		return out, nil
	}
	out.AuditID = auditID
	return out, nil
}

// topRetrievalScore 取命中片段里的最高分作为检索置信度。
// 取不到（无命中 / 结构不符）时返回 0——「没有可信命中」是实情，不是缺失。
func topRetrievalScore(answers interface{}) float64 {
	items, ok := answers.([]map[string]any)
	if !ok {
		return 0
	}
	var best float64
	for _, it := range items {
		if sc, ok := it["score"].(float64); ok && sc > best {
			best = sc
		}
	}
	return best
}

func hitCount(answers interface{}) int {
	if items, ok := answers.([]map[string]any); ok {
		return len(items)
	}
	return 0
}

// truncateForRef 把用户输入截断后作为底账的 inputRef：原样入库会让一条超长
// query 占满底账字段，截断到 120 字符足够定位又不至于膨胀。
func truncateForRef(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
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
	// CMDB 关系写操作：与上面同一审批流 + Gate2 RBAC（resource=ci_relationship, action=write）
	// LLM 在聊天中建/删 CI 关系必须走人工审批，与原生 UI 路径行为一致。
	"create_ci_relationship": true,
	"delete_ci_relationship": true,
}

// ChatStream streams a RAG answer through onDelta while emitting sources
// separately via onSources. It also persists the resulting conversation and
// messages after the stream completes so history is preserved.
//
// 注入按权限过滤后的只读工具。只有当前角色拥有 resource:action 权限的只读工具
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
	//
	// Provider 能力闸门：部分 provider（如 MiniMax Anthropic 兼容接口）仅实现了
	// Chat，未实现 ChatStreamWithTools。此时 LLMGateway 会退化为普通 ChatStream，
	// 但 System Prompt 依然告诉模型“必须调用工具”，导致模型在文本中假装调用
	// (“正在调用 list_tickets 工具...”) 但永远拿不到结果。检测 provider
	// 能力，不支持时不注入 tools，让 RAG 层退化为纯知识库问答（合规）。
	var tools []service.LLMTool
	providerSupportsTools := s.llmGateway != nil && s.llmGateway.SupportsToolCalling()
	if s.tools != nil && providerSupportsTools {
		// 按租户动态化工具参数（list_cis 的 ci_type 枚举来自租户 CIType 表）
		for _, td := range s.tools.ListToolsForTenant(ctx, tenantID) {
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
		// LLM 自报的 user_id/requester_id 不可信，一律以认证上下文覆盖。
		args["user_id"] = float64(userID)
		args["requester_id"] = float64(userID)
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
		// ChatStream 路径持久化失败同样必须可见，与 Chat 路径共用同一计数器。
		// 流式响应已经写回客户端；如果 DB 写入失败而日志被吞，
		// 用户会看到回复但刷新后历史丢失，难以排查。
		if _, err := s.repo.CreateMessage(ctx, &Message{
			ConversationID: convID,
			Role:           "user",
			Content:        query,
		}); err != nil {
			s.logger.Errorw("AI chatstream persist user message failed",
				"operation", "create_message",
				"tenant_id", tenantID,
				"user_id", userID,
				"conversation_id", convID,
				"role", "user",
				"error", err,
			)
			metrics.AIPersistErrors.WithLabelValues(
				"create_message",
				"user",
				strconv.Itoa(tenantID),
			).Inc()
		}
		payload := map[string]any{
			"answer":  captured.String(),
			"sources": sources,
		}
		buf, _ := json.Marshal(payload)
		if _, err := s.repo.CreateMessage(ctx, &Message{
			ConversationID: convID,
			Role:           "assistant",
			Content:        string(buf),
		}); err != nil {
			s.logger.Errorw("AI chatstream persist assistant message failed",
				"operation", "create_message",
				"tenant_id", tenantID,
				"user_id", userID,
				"conversation_id", convID,
				"role", "assistant",
				"error", err,
			)
			metrics.AIPersistErrors.WithLabelValues(
				"create_message",
				"assistant",
				strconv.Itoa(tenantID),
			).Inc()
		}
	}

	return convID, captured.String(), nil
}

func (s *Service) ListConversations(ctx context.Context, tenantID, userID int) ([]*Conversation, error) {
	return s.repo.ListConversations(ctx, tenantID, userID)
}

func (s *Service) GetConversation(ctx context.Context, id, tenantID int) (*Conversation, error) {
	return s.repo.GetConversation(ctx, id, tenantID)
}

func (s *Service) GetConversationMessages(ctx context.Context, convID, tenantID int) ([]*Message, error) {
	_, err := s.repo.GetConversation(ctx, convID, tenantID)
	if err != nil {
		return nil, err
	}
	return s.repo.GetMessages(ctx, convID)
}

func (s *Service) DeleteConversation(ctx context.Context, id, tenantID int) error {
	return s.repo.DeleteConversation(ctx, id, tenantID)
}

// Root Cause Analysis

func (s *Service) AnalyzeTicket(ctx context.Context, ticketID int, tenantID int) (interface{}, error) {
	return s.rca.AnalyzeTicket(ctx, ticketID, tenantID)
}

func (s *Service) AnalyzeTicketWithAudit(ctx context.Context, ticketID int, tenantID int, userID int) (interface{}, error) {
	result, err := s.rca.AnalyzeTicket(ctx, ticketID, tenantID)
	if err != nil {
		return nil, err
	}
	prompt := fmt.Sprintf("AnalyzeTicket ticketID=%d", ticketID)
	confidence, promptVersion, _ := aiResultTrace(result)
	s.persistAIResult(ctx, "rca", tenantID, userID, prompt+" promptVersion="+promptVersion, result, "", 0, 0, &confidence)
	return result, nil
}

func (s *Service) AnalyzeIncident(ctx context.Context, incidentID int, tenantID int) (interface{}, error) {
	if s.rca == nil {
		return nil, service.ErrAIAnalysisUnavailable
	}
	return s.rca.AnalyzeIncident(ctx, incidentID, tenantID)
}

func (s *Service) AnalyzeIncidentWithAudit(ctx context.Context, incidentID int, tenantID int, userID int) (interface{}, error) {
	result, err := s.AnalyzeIncident(ctx, incidentID, tenantID)
	if err != nil {
		return nil, err
	}
	prompt := fmt.Sprintf("AnalyzeIncident incidentID=%d", incidentID)
	confidence, promptVersion, _ := aiResultTrace(result)
	s.persistAIResult(ctx, "incident_impact", tenantID, userID, prompt+" promptVersion="+promptVersion, result, "", 0, 0, &confidence)
	return result, nil
}

// aiResultTrace 从 RCA/事件分析响应中提取置信度与 prompt 版本，供持久化与底账使用。
// 未知类型返回 ok=false（confidence=0，不落置信度语义）。
func aiResultTrace(result interface{}) (confidence float64, promptVersion string, ok bool) {
	switch r := result.(type) {
	case *dto.TicketRootCauseAnalysisResponse:
		return r.ConfidenceScore, r.PromptVersion, true
	case *dto.IncidentAIAnalysisResponse:
		return r.ConfidenceScore, r.PromptVersion, true
	default:
		return 0, "", false
	}
}

// CreateTicketByAI 通过 AI 解析自然语言描述，智能分析并返回工单创建建议
func (s *Service) CreateTicketByAI(ctx context.Context, description string, tenantID int) (map[string]interface{}, error) {
	s.logger.Infow("AI CreateTicketByAI", "tenantID", tenantID)

	// 使用 Triage 服务分析描述，提取分类和优先级
	if s.triageService != nil {
		result := s.triageService.SuggestForTenant(ctx, description, description, tenantID)
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

// defaultSummaryMaxLen 是摘要的目标字数上限。
const defaultSummaryMaxLen = 200

// SummarizeTicket B9: AI 工单总结。
//
// 此前这里只委托 rca，而 SetSummarizeService 的注释却写着「优先走
// SummarizeService.SummarizeWithMetadata」——装配与实现不一致：summarizeService
// 被注入后从未被调用（死装配）。后果是摘要永远走 rca 那条路，而 rca 的默认
// 产物是**字段拼接**（method=fallback），没有任何字段能区分「模型生成的摘要」
// 与「拼出来的原文」。这就是能力矩阵里「默认摘要仍是原文拼接」的由来。
//
// 现在按注释真正接上：SummarizeService 产出 method/model/promptVersion 三件套
// （llm / direct / fallback 如实区分），失败再兜底 rca。两条路径都落服务端 AI
// 决策底账并带出 auditId，操作者可经 PATCH /ai/audit-logs/:id 翻转采纳/拒绝。
//
// userID 为 0 表示无用户上下文（如 skill 调用），底账照落（user_id=0 是实情，
// 不是缺失），不因拿不到用户就放弃溯源。
func (s *Service) SummarizeTicket(ctx context.Context, ticketID int, tenantID int, userID int) (interface{}, error) {
	if s.summarizeService != nil && s.entClient != nil {
		text, err := s.buildTicketSummaryText(ctx, ticketID, tenantID)
		if err != nil {
			s.logger.Warnw("SummarizeTicket: 组装摘要输入失败，兜底 rca", "error", err, "ticketID", ticketID)
		} else {
			meta, mErr := s.summarizeService.SummarizeWithMetadata(ctx, text, defaultSummaryMaxLen)
			if mErr != nil || meta == nil {
				s.logger.Warnw("SummarizeTicket: SummarizeService 失败，兜底 rca", "error", mErr, "ticketID", ticketID)
			} else {
				return s.withSummaryAudit(ctx, ticketID, tenantID, userID, meta.Summary, meta.Method, meta.PromptVersion, meta.Model, meta.Confidence), nil
			}
		}
	}
	return s.rca.SummarizeTicket(ctx, ticketID, tenantID)
}

// buildTicketSummaryText 组装喂给摘要模型的工单文本。
//
// 与 rca 里的那份拼接是两处独立实现：rca 拼的是「fallback 展示文案」（人读的
// 一行状态描述），这里拼的是「模型输入」（结构化字段）。两者目的不同，不共用——
// 把 fallback 文案当模型输入会让模型复述状态字段而不是概括问题。
func (s *Service) buildTicketSummaryText(ctx context.Context, ticketID, tenantID int) (string, error) {
	if s.entClient == nil {
		return "", fmt.Errorf("ent client not initialized")
	}
	tk, err := s.entClient.Ticket.Query().
		Where(ticket.IDEQ(ticketID), ticket.TenantIDEQ(tenantID)).
		Only(ctx)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("工单号：")
	b.WriteString(tk.TicketNumber)
	b.WriteString("\n标题：")
	b.WriteString(tk.Title)
	if tk.Description != "" {
		b.WriteString("\n描述：")
		b.WriteString(tk.Description)
	}
	b.WriteString("\n状态：")
	b.WriteString(tk.Status)
	b.WriteString("\n优先级：")
	b.WriteString(tk.Priority)
	return b.String(), nil
}

// withSummaryAudit 给摘要结果补溯源字段并落服务端决策底账。
//
// confidence 对摘要恒为 0（生成类产出没有自评分数，硬填就是编造），底账的
// notes 里显式标注 confidenceNotApplicable + method，让事后看底账的人知道
// 「这条记录的 0 是不适用，不是模型给了零分」。method=direct/fallback 时也照落：
// 恰恰是这些「不是模型产出」的摘要最需要被记录——否则采纳率会把它当成 AI 表现。
func (s *Service) withSummaryAudit(ctx context.Context, ticketID, tenantID, userID int, summary, method, promptVersion, model string, confidence float64) map[string]interface{} {
	resp := map[string]interface{}{
		"ticketId":      ticketID,
		"summary":       summary,
		"method":        method,
		"promptVersion": promptVersion,
		"model":         model,
		"confidence":    confidence,
		"generatedAt":   time.Now().Format(time.RFC3339),
	}
	if s.auditLedger == nil {
		return resp
	}
	auditID, err := s.auditLedger.RecordDecisionAudit(ctx, tenantID, userID,
		fmt.Sprintf("summary_%d_%d", tenantID, time.Now().UnixNano()),
		"summary", fmt.Sprintf("ticket:%d", ticketID),
		promptVersion, model, confidence,
		map[string]interface{}{
			"method":                  method,
			"confidenceNotApplicable": true,
			"summaryChars":            len([]rune(summary)),
		},
	)
	if err != nil {
		s.logger.Warnw("SummarizeTicket: audit ledger write failed (best-effort)", "error", err)
		return resp
	}
	resp["auditId"] = auditID
	return resp
}

func (s *Service) GetAnalysisReport(ctx context.Context, ticketID int, tenantID int) (interface{}, error) {
	return s.rca.GetAnalysisReport(ctx, ticketID, tenantID)
}

// Analytics and Prediction

// analyticsMetricVersion 是深度分析的统计口径版本。
const analyticsMetricVersion = "deep-analytics-v1"

// GetDeepAnalytics 深度分析。
//
// 它是 SQL 聚合，不是 AI 生成——所以这里只补口径版本与 method="aggregate"，
// **不补 model**。此前能力矩阵把 analytics 与 chat/摘要并列为「缺溯源字段」，
// 但给聚合结果加一个 model 字段只会制造假溯源：客户端会以为这是某模型的建议，
// 而实际没有任何模型参与。诚实标注「这是聚合，口径版本是 X」才是它该有的溯源。
func (s *Service) GetDeepAnalytics(ctx context.Context, req *dto.DeepAnalyticsRequest, tenantID int) (interface{}, error) {
	res, err := s.analytics.GetDeepAnalytics(ctx, req, tenantID)
	if err != nil {
		return nil, err
	}
	if res != nil {
		res.MetricVersion = analyticsMetricVersion
		res.Method = "aggregate"
	}
	return res, nil
}

// slaForecastPromptVersion 是 SLA 趋势预测的口径版本。
//
// 预测值会被拿去做容量与风险决策，「这组数按什么口径算的」必须可追：换了
// 预测算法或平滑参数而不换版本号，底账里新旧口径的预测就无法区分。
const slaForecastPromptVersion = "sla-forecast-v1"

func (s *Service) GetTrendPrediction(ctx context.Context, req *dto.TrendPredictionRequest, tenantID int, userID int) (interface{}, error) {
	// 前端不带 timeRange 时默认为「过去 6 个月 → 今天」。
	// 否则 req.TimeRange[0/1] 会 panic,与 prediction_service.go 行为对齐。
	if len(req.TimeRange) != 2 {
		now := time.Now()
		req.TimeRange = []string{
			now.AddDate(0, -6, 0).Format("2006-01-02"),
			now.Format("2006-01-02"),
		}
	}

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
			// slaForecastSkill 的 PredictionPoint 用
			// "predicted" 字段，与 dto.TrendPredictionResponse.Predictions 契约
			// "predictedValue" 不一致。出口做一层转译，保留 SLAForecastSkill 的
			// 全部增强字段（insights / seasonality / trend / anomalyDates），
			// 同时让前端 SLATrendPredictionCard 的 predictedValue 字段拿到真值。
			model, confidence := "", 0.0
			if output != nil {
				model, confidence = output.Model, output.Confidence
			}
			auditID := s.recordForecastAudit(ctx, tenantID, userID,
				fmt.Sprintf("trend:%s", req.PredictionType), model, confidence)
			return normalizeForecastOutput(output, slaForecastPromptVersion, auditID), nil
		}
		// Fall back to legacy prediction service on error
		s.logger.Warnw("SLAForecastSkill failed, falling back to legacy", "error", err)
	}

	res, err := s.prediction.GetTrendPrediction(ctx, req, tenantID)
	if err != nil {
		return nil, err
	}
	// 兜底路同样补溯源：legacy 预测服务产出的数同样会被拿去决策，
	// 没有口径版本就等于有一批预测来源不明。
	if res != nil {
		res.PromptVersion = slaForecastPromptVersion
		res.AuditID = s.recordForecastAudit(ctx, tenantID, userID,
			fmt.Sprintf("trend:%s", req.PredictionType), res.Model, res.Confidence)
	}
	return res, nil
}

// recordForecastAudit 落一条预测决策底账并返回行 ID；底账不可用时返回 0
// （best-effort：预测结果照常返回，只是这次没有底账 ID）。
//
// 预测与分诊同源地落 ai_audit：采纳/拒绝统一走 PATCH /ai/audit-logs/:id，
// 不再由前端自报。confidence 直接取 ForecastOutput 的真实计算值——它与分诊
// 不同，预测本来就有可计算的置信度（不是生成类产出）。
func (s *Service) recordForecastAudit(ctx context.Context, tenantID, userID int, inputRef, model string, confidence float64) int64 {
	if s.auditLedger == nil {
		return 0
	}
	auditID, err := s.auditLedger.RecordDecisionAudit(ctx, tenantID, userID,
		fmt.Sprintf("forecast_%d_%d", tenantID, time.Now().UnixNano()),
		"forecast", inputRef, slaForecastPromptVersion, model, confidence,
		map[string]interface{}{"model": model},
	)
	if err != nil {
		s.logger.Warnw("forecast: audit ledger write failed (best-effort)", "error", err)
		return 0
	}
	return auditID
}

// normalizeForecastOutput 把 SLAForecastSkill.ForecastOutput 转译为 dto 契约
// （predictions[i].predictedValue 而非 predicted）。同时保留 SLAForecastSkill
// 的 AI 增强字段（insights/seasonality/trend/anomalyDates）作为额外元数据。
func normalizeForecastOutput(output *service.ForecastOutput, promptVersion string, auditID int64) interface{} {
	if output == nil {
		return output
	}
	type predictionDTO struct {
		Date           string  `json:"date"`
		PredictedValue float64 `json:"predictedValue"`
		LowerBound     float64 `json:"lowerBound"`
		UpperBound     float64 `json:"upperBound"`
		Confidence     float64 `json:"confidence"`
		Anomaly        bool    `json:"anomaly"`
	}
	type forecastDTO struct {
		Predictions  []predictionDTO `json:"predictions"`
		Confidence   float64         `json:"confidence"`
		Model        string          `json:"model"`
		GeneratedAt  string          `json:"generatedAt"`
		Insights     string          `json:"insights"`
		Seasonality  map[string]bool `json:"seasonality"`
		Trend        string          `json:"trend"`
		AnomalyDates []string        `json:"anomalyDates"`
		// 溯源：口径版本 + 服务端决策底账 ID（0 = 底账不可用，如实不填）
		PromptVersion string `json:"promptVersion,omitempty"`
		AuditID       int64  `json:"auditId,omitempty"`
	}
	points := make([]predictionDTO, 0, len(output.Predictions))
	for _, p := range output.Predictions {
		points = append(points, predictionDTO{
			Date:           p.Date.Format("2006-01-02"),
			PredictedValue: p.Predicted,
			LowerBound:     p.LowerBound,
			UpperBound:     p.UpperBound,
			Confidence:     p.Confidence,
			Anomaly:        p.Anomaly,
		})
	}
	return forecastDTO{
		Predictions:   points,
		Confidence:    output.Confidence,
		Model:         output.Model,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Insights:      output.Insights,
		Seasonality:   output.Seasonality,
		Trend:         output.Trend,
		AnomalyDates:  output.AnomalyDates,
		PromptVersion: promptVersion,
		AuditID:       auditID,
	}
}

// GetForecastInsights returns AI-generated insights for SLA forecasting
func (s *Service) GetForecastInsights(ctx context.Context, req *dto.TrendPredictionRequest, tenantID int, userID int) (interface{}, error) {
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

	// Return insights-focused response（同样带溯源字段：洞察会被直接展示给
	// 管理者并据此做决策，没有口径版本就没有办法解释「为什么上个月这么说、
	// 这个月又那么说」）。
	auditID := s.recordForecastAudit(ctx, tenantID, userID,
		fmt.Sprintf("insights:%s", req.PredictionType), output.Model, output.Confidence)
	return map[string]interface{}{
		"confidence":    output.Confidence,
		"model":         output.Model,
		"insights":      output.Insights,
		"seasonality":   output.Seasonality,
		"trend":         output.Trend,
		"anomaly_dates": output.AnomalyDates,
		"promptVersion": slaForecastPromptVersion,
		"auditId":       auditID,
	}, nil
}

// Telemetry

// SaveFeedback delegates to AITelemetryService.
// accepted (*bool) for ai_audit accept/reject decisions to be first-class
// (was overloaded onto useful).
func (s *Service) SaveFeedback(ctx context.Context, tenantID, userID int, requestID, kind, query, itemType string, itemID *int, useful bool, score *int, notes *string, accepted *bool) error {
	if s.aiTelemetryService == nil {
		return fmt.Errorf("AI telemetry service not initialized")
	}
	return s.aiTelemetryService.SaveFeedback(ctx, tenantID, userID, requestID, kind, query, itemType, itemID, useful, score, notes, accepted)
}

// UpdateAuditAccepted flips accepted on an ai_feedbacks row whose item_type
// is 'ai_audit'. Returns service.ErrFeedbackNotFound when the row is
// missing, belongs to another tenant, or is not an audit row.
func (s *Service) UpdateAuditAccepted(ctx context.Context, tenantID int, id int64, accepted bool) error {
	if s.aiTelemetryService == nil {
		return fmt.Errorf("AI telemetry service not initialized")
	}
	return s.aiTelemetryService.UpdateAuditAccepted(ctx, tenantID, id, accepted)
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
		return nil, fmt.Errorf("RAG 服务未配置")
	}

	results, err := s.rag.Ask(ctx, tenantID, query, limit)
	if err != nil {
		return nil, fmt.Errorf("RAG 搜索失败: %w", err)
	}

	return results, nil
}

// TriageTicket provides ticket classification and recommendations.
//
// 移除内联关键词 fallback，统一委托
// service/triage_service.go（Guidance → LLM → keyword 三级链）。
// triageService 不可用时返回 unclassified + confidence=0，不再维护第二套分类枚举。
//
// 置信度/溯源强制化：结果携带 promptVersion/model 溯源字段，并自动落一条服务端
// AI 决策底账（item_type='ai_audit'，accepted=NULL 未决策）。响应携带 auditId，
// 前端采纳/拒绝直接 PATCH /ai/audit-logs/:id 翻转底账，不再需要伪造 model
// 二次 POST 自报。底账写失败不阻塞建议返回（best-effort），仅去掉 auditId。
func (s *Service) TriageTicket(ctx context.Context, tenantID int, userID int, title, description, category, priority string) (interface{}, error) {
	s.logger.Infow("Ticket Triage", "title", title, "tenantID", tenantID)

	if s.triageService != nil {
		// 用租户版 SuggestForTenant：此前调 Suggest 内部硬编码 tenant=1，
		// Guidance sidecar 拿到的租户上下文是错的。
		result := s.triageService.SuggestForTenant(ctx, title, description, tenantID)
		response := map[string]interface{}{
			"title":         title,
			"description":   description,
			"promptVersion": result.PromptVersion,
			"model":         result.Model,
			"suggestions": map[string]interface{}{
				"category":   result.Category,
				"priority":   result.Priority,
				"confidence": result.Confidence,
				"reasoning":  result.Explanation,
				"urgency":    s.determineUrgency(result.Priority),
			},
		}
		if s.auditLedger != nil {
			auditID, ledgerErr := s.auditLedger.RecordDecisionAudit(ctx, tenantID, userID,
				fmt.Sprintf("triage_%d_%d", tenantID, time.Now().UnixNano()),
				"triage", fmt.Sprintf("ticket:%s", title),
				result.PromptVersion, result.Model, result.Confidence,
				map[string]interface{}{
					"category": result.Category,
					"priority": result.Priority,
				},
			)
			if ledgerErr != nil {
				s.logger.Warnw("TriageTicket: audit ledger write failed (best-effort)", "error", ledgerErr)
			} else {
				response["auditId"] = auditID
			}
		}
		return response, nil
	}

	return map[string]interface{}{
		"title":       title,
		"description": description,
		"suggestions": map[string]interface{}{
			"category":   "unclassified",
			"priority":   "unclassified",
			"confidence": 0,
			"reasoning":  "triage service unavailable",
			"urgency":    "medium",
		},
	}, nil
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

// ExposeResultTraceForTest 供包外测试验证 aiResultTrace 的类型提取逻辑。
func ExposeResultTraceForTest(result interface{}) (float64, string, bool) {
	return aiResultTrace(result)
}

// parseDate parses date string in YYYY-MM-DD format
func parseDate(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Now()
	}
	return t
}

// persistAIResult serializes and saves an AI analysis result to the database.
// Errors are logged but not propagated — analysis results should still be returned
// to the caller even if persistence fails.
func (s *Service) persistAIResult(ctx context.Context, analysisType string, tenantID, userID int, prompt string, result interface{}, model string, latencyMs, totalTokens int, confidenceScore *float64) {
	if s.repo == nil {
		return
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		s.logger.Warnw("persistAIResult: marshal failed", "error", err)
		return
	}
	record := &AIAnalysisResult{
		TenantID:      tenantID,
		UserID:        userID,
		AnalysisType:  analysisType,
		RequestPrompt: prompt,
		ResultJSON:    string(resultJSON),
		Model:         model,
		Degraded:      false,
	}
	if latencyMs > 0 {
		record.LatencyMs = latencyMs
	}
	if totalTokens > 0 {
		record.TotalTokens = totalTokens
	}
	if confidenceScore != nil {
		record.ConfidenceScore = *confidenceScore
	}
	_, err = s.repo.SaveAIAnalysisResult(ctx, record)
	if err != nil {
		s.logger.Warnw("persistAIResult: save failed", "error", err)
	}
}

// ListAIAnalysisResults returns analysis history for the tenant.
func (s *Service) ListAIAnalysisResults(ctx context.Context, tenantID int, analysisType string, limit int) ([]*AIAnalysisResult, error) {
	return s.repo.ListAIAnalysisResults(ctx, tenantID, analysisType, limit)
}

// GetAIAnalysisResult returns a single analysis result by ID.
func (s *Service) GetAIAnalysisResult(ctx context.Context, id int, tenantID int) (*AIAnalysisResult, error) {
	return s.repo.GetAIAnalysisResult(ctx, id, tenantID)
}

// DeleteAIAnalysisResult deletes an analysis result record.
func (s *Service) DeleteAIAnalysisResult(ctx context.Context, id int, tenantID int) error {
	return s.repo.DeleteAIAnalysisResult(ctx, id, tenantID)
}
