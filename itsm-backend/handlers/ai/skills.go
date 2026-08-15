package ai

// Sprint C — Skill Registry v1：builtin Skill 实现。
//
// 设计约束：
//   - 已有 AI endpoint 行为零回归，本期不替换 handler 主体逻辑。
//   - 每个内置 Skill 是 *Service 方法的薄包装 + 元数据（manifest/tags/permissions）
//     + 指标自动累计。
//   - 输入采用 map[string]any，由 Skill.Validate 解析为强类型并委派给对应 Service 方法。
//   - 协作者通过 internal/bootstrap/app.go 的 RegisterBuiltinSkills() 注入。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"itsm-backend/dto"
	"itsm-backend/service"
)

// SkillInputError 输入校验失败的标准化错误。
type SkillInputError struct {
	Field   string
	Reason  string
	Wrapped error
}

func (e *SkillInputError) Error() string {
	if e.Wrapped != nil {
		return fmt.Sprintf("skill input error field=%s: %s: %v", e.Field, e.Reason, e.Wrapped)
	}
	return fmt.Sprintf("skill input error field=%s: %s", e.Field, e.Reason)
}

func (e *SkillInputError) Unwrap() error { return e.Wrapped }

// helper：从 map 中按 key 取出 string，缺值时返回 SkillInputError。
func inputString(input map[string]any, key string, required bool) (string, error) {
	v, ok := input[key]
	if !ok || v == nil {
		if required {
			return "", &SkillInputError{Field: key, Reason: "required field missing"}
		}
		return "", nil
	}
	s, ok := v.(string)
	if !ok {
		return "", &SkillInputError{Field: key, Reason: "expected string type"}
	}
	return s, nil
}

// helper：从 map 中按 key 取出 int，缺值/类型错时按 defaultOrError 返回。
func inputInt(input map[string]any, key string, defaultVal int) (int, error) {
	v, ok := input[key]
	if !ok || v == nil {
		return defaultVal, nil
	}
	switch x := v.(type) {
	case int:
		return x, nil
	case int32:
		return int(x), nil
	case int64:
		return int(x), nil
	case float64:
		return int(x), nil
	case json.Number:
		n, err := x.Int64()
		if err != nil {
			return 0, &SkillInputError{Field: key, Reason: "expected integer", Wrapped: err}
		}
		return int(n), nil
	case string:
		n, err := strconv.Atoi(x)
		if err != nil {
			return 0, &SkillInputError{Field: key, Reason: "expected integer", Wrapped: err}
		}
		return n, nil
	}
	return 0, &SkillInputError{Field: key, Reason: "expected integer"}
}

// ----------------------------------------------------------------------------
// TriageSkill —— ai.triage
// ----------------------------------------------------------------------------

// TriageSkillInput 是 ai.triage Skill 的规整输入。
type TriageSkillInput struct {
	TenantID    int    `json:"tenantId"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Priority    string `json:"priority"`
}

// TriageSkill 是 ai.triage 的 Skill 包装。
type TriageSkill struct {
	*service.BaseSkill
	svc *Service
}

func NewTriageSkill(svc *Service) *TriageSkill {
	b := service.NewBaseSkill(
		"ai.triage",
		"AI 智能分诊",
		"v1",
		"ga",
		[]string{"ai", "triage", "ticket", "llm", "ga"},
		[]string{"ai:read"},
		[]string{"ticket.classify", "ticket.suggest_priority"},
	).WithProvider("itsm-backend").
		WithAuthor("itsm-backend").
		WithDescription("使用 LLM 对工单标题和描述做智能分类与优先级建议")
	return &TriageSkill{BaseSkill: b, svc: svc}
}

// Manifest 复用 BaseSkill 字段并附加输入/输出 schema 与 long description。
func (s *TriageSkill) Manifest() service.SkillManifest {
	m := s.BuildManifest()
	m.LongDescription = "TriageSkill 接受工单的标题与描述，返回建议的 category/priority/confidence/reasoning。" +
		" 内部委托给 service.TriageService 走 LLM 推理；LLM 不可用时降级为关键词匹配。"
	m.InputSchema = map[string]any{
		"type":     "object",
		"required": []string{"title", "description", "tenantId"},
		"properties": map[string]any{
			"tenantId":    map[string]any{"type": "integer", "description": "租户 ID（必填）"},
			"title":       map[string]any{"type": "string", "description": "工单标题"},
			"description": map[string]any{"type": "string", "description": "工单描述"},
			"category":    map[string]any{"type": "string", "description": "已知的类别，可选"},
			"priority":    map[string]any{"type": "string", "enum": []any{"low", "medium", "high", "critical"}},
		},
	}
	m.OutputSchema = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"suggestions": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"category":   map[string]any{"type": "string"},
					"priority":   map[string]any{"type": "string"},
					"confidence": map[string]any{"type": "number"},
					"reasoning":  map[string]any{"type": "string"},
					"urgency":    map[string]any{"type": "string"},
				},
			},
		},
	}
	return m
}

func (s *TriageSkill) Validate(input interface{}) error {
	in, err := coerceToMap(input)
	if err != nil {
		return err
	}
	if _, err := inputInt(in, "tenantId", 0); err != nil {
		return err
	}
	title, err := inputString(in, "title", true)
	if err != nil {
		return err
	}
	if title == "" {
		return &SkillInputError{Field: "title", Reason: "must not be empty"}
	}
	if _, err := inputString(in, "description", true); err != nil {
		return err
	}
	return nil
}

func (s *TriageSkill) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	in, _ := coerceToMap(input)
	tenantID, _ := inputInt(in, "tenantId", 0)
	title, _ := inputString(in, "title", false)
	description, _ := inputString(in, "description", false)
	category, _ := inputString(in, "category", false)
	priority, _ := inputString(in, "priority", false)
	if s.svc == nil {
		return nil, errors.New("triage skill: underlying service is nil")
	}
	return s.svc.TriageTicket(ctx, tenantID, title, description, category, priority)
}

// ----------------------------------------------------------------------------
// ChatSkill —— ai.chat
// ----------------------------------------------------------------------------

type ChatSkillInput struct {
	TenantID int    `json:"tenantId"`
	UserID   int    `json:"userId"`
	Query    string `json:"query"`
	Limit    int    `json:"limit"`
	ConvID   int    `json:"conversationId"`
}

type ChatSkill struct {
	*service.BaseSkill
	svc *Service
}

func NewChatSkill(svc *Service) *ChatSkill {
	b := service.NewBaseSkill(
		"ai.chat",
		"AI 知识对话",
		"v1",
		"ga",
		[]string{"ai", "chat", "rag", "llm", "ga"},
		[]string{"ai:read"},
		[]string{"rag.query", "conversation.persist"},
	).WithProvider("itsm-backend").
		WithAuthor("itsm-backend").
		WithDescription("基于知识库的 RAG 对话：检索 + 大模型生成，保持对话历史")
	return &ChatSkill{BaseSkill: b, svc: svc}
}

func (s *ChatSkill) Manifest() service.SkillManifest {
	m := s.BuildManifest()
	m.LongDescription = "ChatSkill 接受 user 的自然语言 query，调用 RAGService 检索知识库并由 LLM 整合答案；" +
		"对话历史会自动持久化到 conversations/messages 表。"
	m.InputSchema = map[string]any{
		"type":     "object",
		"required": []string{"tenantId", "userId", "query"},
		"properties": map[string]any{
			"tenantId":       map[string]any{"type": "integer"},
			"userId":         map[string]any{"type": "integer"},
			"query":          map[string]any{"type": "string"},
			"limit":          map[string]any{"type": "integer", "default": 5},
			"conversationId": map[string]any{"type": "integer", "description": "已有会话 ID；缺省自动新建"},
		},
	}
	m.OutputSchema = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"sources":        map[string]any{"type": "array"},
			"conversationId": map[string]any{"type": "integer"},
		},
	}
	return m
}

func (s *ChatSkill) Validate(input interface{}) error {
	in, err := coerceToMap(input)
	if err != nil {
		return err
	}
	if _, err := inputInt(in, "tenantId", 0); err != nil {
		return err
	}
	if _, err := inputInt(in, "userId", 0); err != nil {
		return err
	}
	q, err := inputString(in, "query", true)
	if err != nil {
		return err
	}
	if q == "" {
		return &SkillInputError{Field: "query", Reason: "must not be empty"}
	}
	return nil
}

func (s *ChatSkill) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	in, _ := coerceToMap(input)
	tenantID, _ := inputInt(in, "tenantId", 0)
	userID, _ := inputInt(in, "userId", 0)
	query, _ := inputString(in, "query", false)
	limit, _ := inputInt(in, "limit", 5)
	convID, _ := inputInt(in, "conversationId", 0)
	if s.svc == nil {
		return nil, errors.New("chat skill: underlying service is nil")
	}
	items, newConv, err := s.svc.Chat(ctx, tenantID, userID, query, limit, convID)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"sources":        items,
		"conversationId": newConv,
	}, nil
}

// ----------------------------------------------------------------------------
// KnowledgeSearchSkill —— ai.knowledge_search
// ----------------------------------------------------------------------------

type KnowledgeSearchSkillInput struct {
	TenantID   int    `json:"tenantId"`
	Query      string `json:"query"`
	SearchType string `json:"searchType"`
	Limit      int    `json:"limit"`
}

type KnowledgeSearchSkill struct {
	*service.BaseSkill
	svc *Service
}

func NewKnowledgeSearchSkill(svc *Service) *KnowledgeSearchSkill {
	b := service.NewBaseSkill(
		"ai.knowledge_search",
		"知识库检索",
		"v1",
		"ga",
		[]string{"ai", "rag", "search", "knowledge", "ga"},
		[]string{"ai:read"},
		[]string{"rag.query", "vector.similarity_search"},
	).WithProvider("itsm-backend").
		WithAuthor("itsm-backend").
		WithDescription("RAG 知识库语义检索，仅检索不生成")
	return &KnowledgeSearchSkill{BaseSkill: b, svc: svc}
}

func (s *KnowledgeSearchSkill) Manifest() service.SkillManifest {
	return s.BuildManifest()
}

func (s *KnowledgeSearchSkill) Validate(input interface{}) error {
	in, err := coerceToMap(input)
	if err != nil {
		return err
	}
	if _, err := inputInt(in, "tenantId", 0); err != nil {
		return err
	}
	q, err := inputString(in, "query", true)
	if err != nil {
		return err
	}
	if q == "" {
		return &SkillInputError{Field: "query", Reason: "must not be empty"}
	}
	return nil
}

func (s *KnowledgeSearchSkill) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	in, _ := coerceToMap(input)
	tenantID, _ := inputInt(in, "tenantId", 0)
	query, _ := inputString(in, "query", false)
	searchType, _ := inputString(in, "searchType", false)
	limit, _ := inputInt(in, "limit", 10)
	if s.svc == nil {
		return nil, errors.New("knowledge search skill: underlying service is nil")
	}
	return s.svc.SearchKnowledge(ctx, tenantID, query, searchType, limit)
}

// ----------------------------------------------------------------------------
// SummarizeSkill —— ai.summarize
// ----------------------------------------------------------------------------

type SummarizeSkillInput struct {
	TenantID int `json:"tenantId"`
	TicketID int `json:"ticketId"`
}

type SummarizeSkill struct {
	*service.BaseSkill
	svc *Service
}

func NewSummarizeSkill(svc *Service) *SummarizeSkill {
	b := service.NewBaseSkill(
		"ai.summarize",
		"工单摘要",
		"v1",
		"ga",
		[]string{"ai", "summarize", "ticket", "llm", "ga"},
		[]string{"ai:read"},
		[]string{"ticket.summarize"},
	).WithProvider("itsm-backend").
		WithAuthor("itsm-backend").
		WithDescription("基于工单上下文生成可读的摘要描述")
	return &SummarizeSkill{BaseSkill: b, svc: svc}
}

func (s *SummarizeSkill) Manifest() service.SkillManifest { return s.BuildManifest() }

func (s *SummarizeSkill) Validate(input interface{}) error {
	in, err := coerceToMap(input)
	if err != nil {
		return err
	}
	if _, err := inputInt(in, "tenantId", 0); err != nil {
		return err
	}
	if _, err := inputInt(in, "ticketId", 0); err != nil {
		return err
	}
	return nil
}

func (s *SummarizeSkill) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	in, _ := coerceToMap(input)
	tenantID, _ := inputInt(in, "tenantId", 0)
	ticketID, _ := inputInt(in, "ticketId", 0)
	if s.svc == nil {
		return nil, errors.New("summarize skill: underlying service is nil")
	}
	return s.svc.SummarizeTicket(ctx, ticketID, tenantID)
}

// ----------------------------------------------------------------------------
// AnalyzeSkill —— ai.analyze
// ----------------------------------------------------------------------------

type AnalyzeSkillInput struct {
	TenantID int `json:"tenantId"`
	TicketID int `json:"ticketId"`
}

type AnalyzeSkill struct {
	*service.BaseSkill
	svc *Service
}

func NewAnalyzeSkill(svc *Service) *AnalyzeSkill {
	b := service.NewBaseSkill(
		"ai.analyze",
		"工单根因分析",
		"v1",
		"ga",
		[]string{"ai", "analyze", "rca", "ticket", "llm", "ga"},
		[]string{"ai:read"},
		[]string{"ticket.rca", "ticket.classify"},
	).WithProvider("itsm-backend").
		WithAuthor("itsm-backend").
		WithDescription("基于 LLM 的工单根因分析（RCA）")
	return &AnalyzeSkill{BaseSkill: b, svc: svc}
}

func (s *AnalyzeSkill) Manifest() service.SkillManifest { return s.BuildManifest() }

func (s *AnalyzeSkill) Validate(input interface{}) error {
	in, err := coerceToMap(input)
	if err != nil {
		return err
	}
	if _, err := inputInt(in, "tenantId", 0); err != nil {
		return err
	}
	if _, err := inputInt(in, "ticketId", 0); err != nil {
		return err
	}
	return nil
}

func (s *AnalyzeSkill) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	in, _ := coerceToMap(input)
	tenantID, _ := inputInt(in, "tenantId", 0)
	ticketID, _ := inputInt(in, "ticketId", 0)
	if s.svc == nil {
		return nil, errors.New("analyze skill: underlying service is nil")
	}
	return s.svc.AnalyzeTicket(ctx, ticketID, tenantID)
}

// ----------------------------------------------------------------------------
// AnalyticsSkill —— ai.analytics (pilot)
// ----------------------------------------------------------------------------

type AnalyticsSkill struct {
	*service.BaseSkill
	svc *Service
}

func NewAnalyticsSkill(svc *Service) *AnalyticsSkill {
	b := service.NewBaseSkill(
		"ai.analytics",
		"深度数据分析",
		"v1",
		"pilot",
		[]string{"ai", "analytics", "report", "llm", "pilot"},
		[]string{"report:read"},
		[]string{"report.deep_analytics"},
	).WithProvider("itsm-backend").
		WithAuthor("itsm-backend").
		WithDescription("多维度工单数据深度分析与图表生成（pilot）")
	return &AnalyticsSkill{BaseSkill: b, svc: svc}
}

func (s *AnalyticsSkill) Manifest() service.SkillManifest {
	m := s.BuildManifest()
	m.InputSchema = map[string]any{
		"type":     "object",
		"required": []string{"tenantId", "dimensions", "metrics", "chartType", "timeRange"},
		"properties": map[string]any{
			"tenantId":   map[string]any{"type": "integer"},
			"dimensions": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"metrics":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"chartType":  map[string]any{"enum": []any{"line", "bar", "pie", "area", "table"}},
			"timeRange":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		},
	}
	return m
}

func (s *AnalyticsSkill) Validate(input interface{}) error {
	in, err := coerceToMap(input)
	if err != nil {
		return err
	}
	if _, err := inputInt(in, "tenantId", 0); err != nil {
		return err
	}
	dims, _ := in["dimensions"]
	if dims == nil {
		return &SkillInputError{Field: "dimensions", Reason: "required"}
	}
	metrics, _ := in["metrics"]
	if metrics == nil {
		return &SkillInputError{Field: "metrics", Reason: "required"}
	}
	chartType, err := inputString(in, "chartType", true)
	if err != nil {
		return err
	}
	if chartType == "" {
		return &SkillInputError{Field: "chartType", Reason: "required"}
	}
	return nil
}

func (s *AnalyticsSkill) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	in, _ := coerceToMap(input)
	tenantID, _ := inputInt(in, "tenantId", 0)
	req := &dto.DeepAnalyticsRequest{
		ChartType: stringOrEmpty(in, "chartType"),
	}
	if dims, ok := in["dimensions"].([]any); ok {
		for _, d := range dims {
			if s, ok := d.(string); ok {
				req.Dimensions = append(req.Dimensions, s)
			}
		}
	}
	if metrics, ok := in["metrics"].([]any); ok {
		for _, m := range metrics {
			if s, ok := m.(string); ok {
				req.Metrics = append(req.Metrics, s)
			}
		}
	}
	if tr, ok := in["timeRange"].([]any); ok {
		for _, t := range tr {
			if s, ok := t.(string); ok {
				req.TimeRange = append(req.TimeRange, s)
			}
		}
	}
	if f, ok := in["filters"].(map[string]any); ok {
		req.Filters = f
	}
	if g, ok := in["groupBy"].(string); ok {
		req.GroupBy = &g
	}
	page, _ := inputInt(in, "page", 1)
	req.Page = page
	pageSize, _ := inputInt(in, "pageSize", 20)
	req.PageSize = pageSize
	if s.svc == nil {
		return nil, errors.New("analytics skill: underlying service is nil")
	}
	return s.svc.GetDeepAnalytics(ctx, req, tenantID)
}

// ----------------------------------------------------------------------------
// TrendPredictionSkill —— ai.trend_prediction (pilot)
// ----------------------------------------------------------------------------

type TrendPredictionSkill struct {
	*service.BaseSkill
	svc *Service
}

func NewTrendPredictionSkill(svc *Service) *TrendPredictionSkill {
	b := service.NewBaseSkill(
		"ai.trend_prediction",
		"趋势预测",
		"v1",
		"pilot",
		[]string{"ai", "prediction", "forecast", "llm", "pilot"},
		[]string{"report:read"},
		[]string{"report.trend_forecast"},
	).WithProvider("itsm-backend").
		WithAuthor("itsm-backend").
		WithDescription("基于历史工单的趋势与 SLA 风险预测（pilot，优先使用 SLAForecastSkill）")
	return &TrendPredictionSkill{BaseSkill: b, svc: svc}
}

func (s *TrendPredictionSkill) Manifest() service.SkillManifest { return s.BuildManifest() }

func (s *TrendPredictionSkill) Validate(input interface{}) error {
	in, err := coerceToMap(input)
	if err != nil {
		return err
	}
	if _, err := inputInt(in, "tenantId", 0); err != nil {
		return err
	}
	if _, err := inputString(in, "predictionType", true); err != nil {
		return err
	}
	if _, err := inputString(in, "timeRange", true); err != nil {
		return err
	}
	return nil
}

func (s *TrendPredictionSkill) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	in, _ := coerceToMap(input)
	tenantID, _ := inputInt(in, "tenantId", 0)
	predType, _ := inputString(in, "predictionType", false)
	model, _ := inputString(in, "model", false)
	req := &dto.TrendPredictionRequest{
		PredictionType: predType,
		Model:          model,
	}
	if tr, ok := in["timeRange"].([]any); ok {
		for _, t := range tr {
			if s, ok := t.(string); ok {
				req.TimeRange = append(req.TimeRange, s)
			}
		}
	}
	if f, ok := in["filters"].(map[string]any); ok {
		req.Filters = f
	}
	if s.svc == nil {
		return nil, errors.New("trend_prediction skill: underlying service is nil")
	}
	return s.svc.GetTrendPrediction(ctx, req, tenantID)
}

// ----------------------------------------------------------------------------
// CreateTicketSkill —— ai.create_ticket (pilot)
// ----------------------------------------------------------------------------

type CreateTicketSkill struct {
	*service.BaseSkill
	svc *Service
}

func NewCreateTicketSkill(svc *Service) *CreateTicketSkill {
	b := service.NewBaseSkill(
		"ai.create_ticket",
		"AI 创建工单",
		"v1",
		"pilot",
		[]string{"ai", "ticket", "create", "llm", "pilot"},
		[]string{"ticket:create"},
		[]string{"ticket.create_draft"},
	).WithProvider("itsm-backend").
		WithAuthor("itsm-backend").
		WithDescription("从自然语言描述生成工单草稿：自动建议分类与优先级（pilot）")
	return &CreateTicketSkill{BaseSkill: b, svc: svc}
}

func (s *CreateTicketSkill) Manifest() service.SkillManifest { return s.BuildManifest() }

func (s *CreateTicketSkill) Validate(input interface{}) error {
	in, err := coerceToMap(input)
	if err != nil {
		return err
	}
	if _, err := inputInt(in, "tenantId", 0); err != nil {
		return err
	}
	desc, err := inputString(in, "description", true)
	if err != nil {
		return err
	}
	if len(desc) < 4 {
		return &SkillInputError{Field: "description", Reason: "must have at least 4 chars"}
	}
	return nil
}

func (s *CreateTicketSkill) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	in, _ := coerceToMap(input)
	tenantID, _ := inputInt(in, "tenantId", 0)
	desc, _ := inputString(in, "description", false)
	if s.svc == nil {
		return nil, errors.New("create_ticket skill: underlying service is nil")
	}
	return s.svc.CreateTicketByAI(ctx, desc, tenantID)
}

// ----------------------------------------------------------------------------
// AgentToolSkill —— ai.agent_tool (pilot)
// ----------------------------------------------------------------------------

type AgentToolSkillInput struct {
	TenantID int                    `json:"tenantId"`
	UserID   int                    `json:"userId"`
	Role     string                 `json:"role"`
	Name     string                 `json:"name"`
	Args     map[string]interface{} `json:"args"`
}

type AgentToolSkill struct {
	*service.BaseSkill
	svc *Service
}

func NewAgentToolSkill(svc *Service) *AgentToolSkill {
	b := service.NewBaseSkill(
		"ai.agent_tool",
		"AI 工具执行",
		"v1",
		"pilot",
		[]string{"ai", "tool", "agent", "rbac", "pilot"},
		[]string{"ai:read"},
		[]string{"tool.execute", "tool.audit"},
	).WithProvider("itsm-backend").
		WithAuthor("itsm-backend").
		WithDescription("通过统一入口执行 AI 注册工具（含 RBAC 校验与审计记录；pilot）")
	return &AgentToolSkill{BaseSkill: b, svc: svc}
}

func (s *AgentToolSkill) Manifest() service.SkillManifest {
	m := s.BuildManifest()
	m.LongDescription = "AgentToolSkill 接受 tool 名称与参数，调用 Service.ExecuteTool，" +
		"其内部已实现 Gate 2 工具级 RBAC + 写工具的审批队列逻辑。"
	m.InputSchema = map[string]any{
		"type":     "object",
		"required": []string{"tenantId", "userId", "role", "name"},
		"properties": map[string]any{
			"tenantId": map[string]any{"type": "integer"},
			"userId":   map[string]any{"type": "integer"},
			"role":     map[string]any{"type": "string"},
			"name":     map[string]any{"type": "string"},
			"args":     map[string]any{"type": "object"},
		},
	}
	return m
}

func (s *AgentToolSkill) Validate(input interface{}) error {
	in, err := coerceToMap(input)
	if err != nil {
		return err
	}
	if _, err := inputInt(in, "tenantId", 0); err != nil {
		return err
	}
	if _, err := inputInt(in, "userId", 0); err != nil {
		return err
	}
	if _, err := inputString(in, "role", true); err != nil {
		return err
	}
	name, err := inputString(in, "name", true)
	if err != nil {
		return err
	}
	if name == "" {
		return &SkillInputError{Field: "name", Reason: "must not be empty"}
	}
	return nil
}

func (s *AgentToolSkill) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	in, _ := coerceToMap(input)
	tenantID, _ := inputInt(in, "tenantId", 0)
	userID, _ := inputInt(in, "userId", 0)
	role, _ := inputString(in, "role", false)
	name, _ := inputString(in, "name", false)
	args, _ := in["args"].(map[string]interface{})
	if s.svc == nil {
		return nil, errors.New("agent_tool skill: underlying service is nil")
	}
	res, _, err := s.svc.ExecuteTool(ctx, userID, tenantID, role, name, args)
	if err != nil {
		return nil, err
	}
	if res == nil {
		// 写工具 pending 状态：返回观察 ID 占位，避免 caller 拿到 nil 时误以为完成
		return map[string]any{"status": "approval_pending"}, nil
	}
	return map[string]any{"result": res}, nil
}

// ----------------------------------------------------------------------------
// MetricsSkill —— ai.metrics（只读：返回指标聚合）
// ----------------------------------------------------------------------------

type MetricsSkill struct {
	*service.BaseSkill
	svc *Service
}

func NewMetricsSkill(svc *Service) *MetricsSkill {
	b := service.NewBaseSkill(
		"ai.metrics",
		"AI 指标聚合",
		"v1",
		"ga",
		[]string{"ai", "metrics", "audit", "ga"},
		[]string{"ai:read"},
		[]string{"audit.metrics_query"},
	).WithProvider("itsm-backend").
		WithAuthor("itsm-backend").
		WithDescription("AI 使用指标聚合（反馈量/成功率/调用次数/延迟）")
	return &MetricsSkill{BaseSkill: b, svc: svc}
}

func (s *MetricsSkill) Manifest() service.SkillManifest { return s.BuildManifest() }

func (s *MetricsSkill) Validate(input interface{}) error {
	in, err := coerceToMap(input)
	if err != nil {
		return err
	}
	if _, err := inputInt(in, "tenantId", 0); err != nil {
		return err
	}
	return nil
}

func (s *MetricsSkill) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	in, _ := coerceToMap(input)
	tenantID, _ := inputInt(in, "tenantId", 0)
	lookbackDays, _ := inputInt(in, "lookbackDays", 7)
	if s.svc == nil {
		return nil, errors.New("metrics skill: underlying service is nil")
	}
	return s.svc.GetMetrics(ctx, tenantID, lookbackDays)
}

// ----------------------------------------------------------------------------
// FeedbackSkill —— ai.feedback（保存反馈）
// ----------------------------------------------------------------------------

type FeedbackSkillInput struct {
	TenantID  int     `json:"tenantId"`
	UserID    int     `json:"userId"`
	RequestID string  `json:"requestId"`
	Kind      string  `json:"kind"`
	Query     string  `json:"query"`
	ItemType  string  `json:"itemType"`
	ItemID    *int    `json:"itemId,omitempty"`
	Useful    bool    `json:"useful"`
	Score     *int    `json:"score,omitempty"`
	Notes     *string `json:"notes,omitempty"`
}

type FeedbackSkill struct {
	*service.BaseSkill
	svc *Service
}

func NewFeedbackSkill(svc *Service) *FeedbackSkill {
	b := service.NewBaseSkill(
		"ai.feedback",
		"AI 反馈收集",
		"v1",
		"ga",
		[]string{"ai", "feedback", "telemetry", "ga"},
		[]string{"ai:write"},
		[]string{"telemetry.feedback_record"},
	).WithProvider("itsm-backend").
		WithAuthor("itsm-backend").
		WithDescription("收集对 AI 输出的有用性反馈，用于评估模型质量")
	return &FeedbackSkill{BaseSkill: b, svc: svc}
}

func (s *FeedbackSkill) Manifest() service.SkillManifest { return s.BuildManifest() }

func (s *FeedbackSkill) Validate(input interface{}) error {
	in, err := coerceToMap(input)
	if err != nil {
		return err
	}
	if _, err := inputInt(in, "tenantId", 0); err != nil {
		return err
	}
	if _, err := inputInt(in, "userId", 0); err != nil {
		return err
	}
	if _, err := inputString(in, "requestId", true); err != nil {
		return err
	}
	if _, err := inputString(in, "kind", true); err != nil {
		return err
	}
	if _, err := inputString(in, "itemType", true); err != nil {
		return err
	}
	return nil
}

func (s *FeedbackSkill) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	in, _ := coerceToMap(input)
	tenantID, _ := inputInt(in, "tenantId", 0)
	userID, _ := inputInt(in, "userId", 0)
	requestID, _ := inputString(in, "requestId", false)
	kind, _ := inputString(in, "kind", false)
	query, _ := inputString(in, "query", false)
	itemType, _ := inputString(in, "itemType", false)
	var itemID *int
	if v, ok := in["itemId"]; ok && v != nil {
		switch n := v.(type) {
		case int:
			itemID = &n
		case float64:
			ii := int(n)
			itemID = &ii
		}
	}
	var score *int
	if v, ok := in["score"]; ok && v != nil {
		switch n := v.(type) {
		case int:
			score = &n
		case float64:
			sn := int(n)
			score = &sn
		}
	}
	var notes *string
	if v, ok := in["notes"]; ok && v != nil {
		if str, ok := v.(string); ok {
			notes = &str
		}
	}
	useful, _ := in["useful"].(bool)
	if s.svc == nil {
		return nil, errors.New("feedback skill: underlying service is nil")
	}
	if err := s.svc.SaveFeedback(ctx, tenantID, userID, requestID, kind, query, itemType, itemID, useful, score, notes); err != nil {
		return nil, err
	}
	return map[string]any{"recorded": true}, nil
}

// ----------------------------------------------------------------------------
// helpers
// ----------------------------------------------------------------------------

// coerceToMap 兼容 *struct 与 map[string]any 两种形式的输入。
// 如果 input 为 nil，返回空 map（让 Validate 之后捕捉必填项缺失）。
func coerceToMap(input interface{}) (map[string]any, error) {
	switch v := input.(type) {
	case nil:
		return map[string]any{}, nil
	case map[string]any:
		return v, nil
	default:
		// 尝试 JSON 反序列化作为兜底（struct → map）
		raw, err := json.Marshal(input)
		if err != nil {
			return nil, &SkillInputError{Field: "input", Reason: "unsupported input type", Wrapped: err}
		}
		var out map[string]any
		if err := json.Unmarshal(raw, &out); err != nil {
			return nil, &SkillInputError{Field: "input", Reason: "input is not a JSON object", Wrapped: err}
		}
		return out, nil
	}
}

func stringOrEmpty(in map[string]any, key string) string {
	s, _ := in[key].(string)
	return s
}
