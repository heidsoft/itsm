package ai

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/handlers/common/knowledgeaccess"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// ListTools handles GET /api/v1/agent/tools
// P2-6: 按 ToolDefinition.Resource/Action 过滤，仅返回当前角色有权限的工具
func (h *Handler) ListTools(c *gin.Context) {
	allTools := h.svc.ListTools()

	role := c.GetString("role")
	tenantID := c.GetInt("tenant_id")

	// Flag 未开启或无 ent client 时，返回全部工具（兼容历史行为）
	if !IsToolRBACEnabled() || h.svc.entClient == nil || role == "" || role == "super_admin" {
		common.Success(c, gin.H{"tools": allTools})
		return
	}

	visible := make([]service.ToolDefinition, 0, len(allTools))
	for _, t := range allTools {
		if middleware.HasResourcePermission(c.Request.Context(), h.svc.entClient, role, t.Resource, t.Action, tenantID) {
			visible = append(visible, t)
		}
	}
	common.Success(c, gin.H{"tools": visible})
}

// ExecuteTool handles POST /api/v1/agent/tools/execute
func (h *Handler) ExecuteTool(c *gin.Context) {
	var req struct {
		Name string                 `json:"name" binding:"required"`
		Args map[string]interface{} `json:"args"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	// P2-6: 校验工具存在性；写工具（!ReadOnly）交由 ExecuteTool 统一走审批流
	// （创建 pending invocation + 入队等待人工审批），与聊天路径行为一致。
	toolDef := h.svc.tools.GetTool(req.Name)
	if toolDef == nil {
		common.Fail(c, common.UnknownToolCode, "unknown tool: "+req.Name)
		return
	}

	if req.Args == nil {
		req.Args = map[string]interface{}{}
	}

	userID := c.GetInt("user_id")
	role := c.GetString("role")

	res, _, err := h.svc.ExecuteTool(c.Request.Context(), userID, tenantID, role, req.Name, req.Args)
	if err != nil {
		// P2-6: 区分权限拒绝与未知工具的错误码
		if errors.Is(err, ErrToolPermissionDenied) {
			common.FailWithErr(c, err, "操作失败")
			return
		}
		if errors.Is(err, ErrUnknownTool) {
			common.FailWithErr(c, err, "操作失败")
			return
		}
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{
		"status":      "success",
		"summary":     "tool executed",
		"nextActions": []string{"If the result is incomplete, refine args and retry."},
		"artifacts":   []string{},
		"data":        res,
	})
}

// Chat handles POST /api/v1/ai/chat
func (h *Handler) Chat(c *gin.Context) {
	var req struct {
		Query          string `json:"query" binding:"required"`
		Limit          int    `json:"limit"`
		ConversationID int    `json:"conversationId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")
	role := c.GetString("role")

	// 注入知识访问者身份：RAG 检索据此做分类级可见性过滤（L0 权限边界）。
	// 不注入则按匿名处理，已纳管的受限分类一律不可见（fail-closed）。
	chatCtx := knowledgeaccess.WithViewer(c.Request.Context(), knowledgeaccess.Viewer{UserID: userID, Role: role})

	answers, convID, err := h.svc.Chat(chatCtx, tenantID, userID, req.Query, req.Limit, req.ConversationID)
	if err != nil {
		// RAG 失败时降级处理：返回空结果而非 500 错误，避免前端崩溃
		h.svc.logger.Warnw("AI Chat RAG 检索失败，返回降级响应", "error", err, "tenantID", tenantID)
		common.Success(c, gin.H{
			"answers":        []interface{}{},
			"conversationId": 0,
			"degraded":       true,
			"message":        "AI 服务暂时不可用，请稍后重试",
		})
		return
	}

	common.Success(c, gin.H{
		"answers":        answers,
		"conversationId": convID,
	})
}

// ChatStream handles POST /api/v1/ai/chat/stream and emits Server-Sent Events.
// Events:
//   - event: sources        data: [{objectType,id,title,snippet,score,...}]
//   - event: delta          data: {"content": "..."}
//   - event: done           data: {"conversationId": <id>}
//   - event: error          data: {"message": "..."}
func (h *Handler) ChatStream(c *gin.Context) {
	var req struct {
		Query          string `json:"query" binding:"required"`
		Limit          int    `json:"limit"`
		ConversationID int    `json:"conversationId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")
	role := c.GetString("role")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	// SSE headers. Nginx-friendly: X-Accel-Buffering:no disables proxy buffering.
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		// Streaming not supported: fall back to a normal chat response so the
		// client still gets an answer.
		answers, convID, err := h.svc.Chat(
			knowledgeaccess.WithViewer(c.Request.Context(), knowledgeaccess.Viewer{UserID: userID, Role: role}),
			tenantID, userID, req.Query, req.Limit, req.ConversationID)
		if err != nil {
			common.FailWithErr(c, err, "操作失败")
			return
		}
		common.Success(c, gin.H{"answers": answers, "conversationId": convID})
		return
	}

	writeEvent := func(event string, payload interface{}) {
		buf, err := json.Marshal(payload)
		if err != nil {
			return
		}
		// Multi-line data payloads must be prefixed on each line, but json.Marshal
		// produces a single line, so a single data: prefix is sufficient.
		_, _ = fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, string(buf))
		flusher.Flush()
	}

	onSources := func(items []map[string]any) {
		writeEvent("sources", items)
	}
	onDelta := func(delta string) {
		writeEvent("delta", map[string]string{"content": delta})
	}

	// 注入访问者身份：AI 助手主链路，RAG 据此做知识分类可见性过滤（L0 权限边界）
	convID, _, err := h.svc.ChatStream(
		knowledgeaccess.WithViewer(c.Request.Context(), knowledgeaccess.Viewer{UserID: userID, Role: role}),
		tenantID, userID, role, req.Query, req.Limit, req.ConversationID, onSources, onDelta)
	if err != nil {
		h.svc.logger.Warnw("AI ChatStream 失败", "error", err, "tenantID", tenantID)
		writeEvent("error", map[string]string{"message": err.Error()})
		return
	}
	writeEvent("done", map[string]int{"conversationId": convID})
}

// GetDeepAnalytics handles POST /api/v1/ai/analytics
func (h *Handler) GetDeepAnalytics(c *gin.Context) {
	var req dto.DeepAnalyticsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	tenantID := c.GetInt("tenant_id")
	res, err := h.svc.GetDeepAnalytics(c.Request.Context(), &req, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, res)
}

// GetTrendPrediction handles POST /api/v1/ai/predictions
func (h *Handler) GetTrendPrediction(c *gin.Context) {
	var req dto.TrendPredictionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}
	tenantID := c.GetInt("tenant_id")
	res, err := h.svc.GetTrendPrediction(c.Request.Context(), &req, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, res)
}

// AnalyzeTicket handles POST /api/v1/ai/tickets/:id/analyze
func (h *Handler) AnalyzeTicket(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		common.Fail(c, common.ParamErrorCode, "invalid ticket id")
		return
	}
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	res, err := h.svc.AnalyzeTicket(c.Request.Context(), id, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, res)
}

// AnalyzeIncident handles POST /api/v1/ai/incidents/:id/analyze.
func (h *Handler) AnalyzeIncident(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.Fail(c, common.ParamErrorCode, "invalid incident id")
		return
	}
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	res, err := h.svc.AnalyzeIncident(c.Request.Context(), id, tenantID)
	if err != nil {
		if errors.Is(err, service.ErrIncidentNotFound) {
			common.NotFoundWithErr(c, err, "事件不存在")
			return
		}
		if errors.Is(err, service.ErrAIAnalysisUnavailable) {
			common.Fail(c, common.ServiceUnavailableCode, "AI 事件分析服务尚未就绪")
			return
		}
		common.FailWithErr(c, err, "事件分析失败")
		return
	}
	common.Success(c, res)
}

// SummarizeTicket handles GET /api/v1/ai/tickets/:id/summary
// B9: AI 工单总结 - 优先用 LLM，fallback 用字段拼接
func (h *Handler) SummarizeTicket(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		common.Fail(c, common.ParamErrorCode, "invalid ticket id")
		return
	}
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	summary, err := h.svc.SummarizeTicket(c.Request.Context(), id, tenantID)
	if err != nil {
		h.svc.logger.Warnw("AI摘要失败，返回降级响应", "error", err, "ticketID", id)
		common.Success(c, gin.H{
			"degraded": true,
			"message":  "AI 摘要服务暂时不可用，请稍后重试",
			"summary":  "",
		})
		return
	}
	common.Success(c, summary)
}

// SaveFeedback handles POST /api/v1/ai/feedback
func (h *Handler) SaveFeedback(c *gin.Context) {
	var req struct {
		Kind     string  `json:"kind" binding:"required"`
		Query    string  `json:"query"`
		ItemType *string `json:"itemType"`
		ItemID   *int    `json:"itemId"`
		Useful   bool    `json:"useful" binding:"required"`
		Score    *int    `json:"score"`
		Notes    *string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")
	requestID := c.GetString("request_id")
	if requestID == "" {
		requestID = fmt.Sprintf("req_%d_%d", time.Now().Unix(), userID)
	}

	var itemTypeVal string
	if req.ItemType != nil {
		itemTypeVal = *req.ItemType
	}

	err := h.svc.SaveFeedback(c.Request.Context(), tenantID, userID, requestID, req.Kind, req.Query, itemTypeVal, req.ItemID, req.Useful, req.Score, req.Notes)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, gin.H{"message": "Feedback saved"})
}

// RecordAudit handles POST /api/v1/ai/audit.
// It records the GA AI trace contract without allowing the AI to auto-apply high-risk actions.
func (h *Handler) RecordAudit(c *gin.Context) {
	var req struct {
		Scenario      string                 `json:"scenario" binding:"required"`
		InputRef      string                 `json:"inputRef" binding:"required"`
		PromptVersion string                 `json:"promptVersion"`
		Model         string                 `json:"model"`
		Confidence    float64                `json:"confidence"`
		Suggestion    map[string]interface{} `json:"suggestion" binding:"required"`
		Accepted      bool                   `json:"accepted"`
		Notes         string                 `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	userID := c.GetInt("user_id")
	requestID := c.GetString("request_id")
	if requestID == "" {
		requestID = fmt.Sprintf("ai_audit_%d_%d", time.Now().Unix(), userID)
	}

	notePayload := map[string]interface{}{
		"promptVersion": req.PromptVersion,
		"model":         req.Model,
		"confidence":    req.Confidence,
		"suggestion":    req.Suggestion,
		"notes":         req.Notes,
	}
	noteBytes, _ := json.Marshal(notePayload)
	note := string(noteBytes)
	itemType := "ai_audit"
	score := int(req.Confidence * 100)
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	if err := h.svc.SaveFeedback(c.Request.Context(), tenantID, userID, requestID, req.Scenario, req.InputRef, itemType, nil, req.Accepted, &score, &note); err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}

	common.Success(c, gin.H{
		"requestId":     requestID,
		"scenario":      req.Scenario,
		"inputRef":      req.InputRef,
		"promptVersion": req.PromptVersion,
		"model":         req.Model,
		"confidence":    req.Confidence,
		"accepted":      req.Accepted,
	})
}

// GetEvaluation handles GET /api/v1/ai/evaluation.
// 输出 AI 评估报告：按场景的有用率、置信度校准、平台级 LLM 成功率/延迟。
func (h *Handler) GetEvaluation(c *gin.Context) {
	days := queryInt(c, "days", 30)
	tenantID := c.GetInt("tenant_id")
	report, err := h.svc.Evaluate(c.Request.Context(), tenantID, days)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, report)
}

// GetAuditLogs handles GET /api/v1/ai/audit-logs.
// 分页查询 AI 审计记录（item_type='ai_audit'），可按场景 kind 过滤。
func (h *Handler) GetAuditLogs(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "pageSize", 20)
	days := queryInt(c, "days", 90)
	kind := c.Query("kind")
	tenantID := c.GetInt("tenant_id")
	entries, total, err := h.svc.ListAuditLogs(c.Request.Context(), tenantID, page, pageSize, kind, days)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, gin.H{"items": entries, "total": total, "page": page, "pageSize": pageSize})
}

// queryInt 解析正整数查询参数，非法/缺失时回退到默认值。
func queryInt(c *gin.Context, key string, def int) int {
	if v := c.Query(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

// GetMetrics handles GET /api/v1/ai/metrics.
func (h *Handler) GetMetrics(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	lookbackDays := 7
	if daysStr := c.Query("days"); daysStr != "" {
		if days, err := strconv.Atoi(daysStr); err == nil && days > 0 && days <= 365 {
			lookbackDays = days
		}
	}
	metrics, err := h.svc.GetMetrics(c.Request.Context(), tenantID, lookbackDays)
	if err != nil {
		common.FailWithErr(c, err, "操作失败")
		return
	}
	common.Success(c, metrics)
}

// KnowledgeSearch handles POST /api/v1/ai/rag/search - RAG search over knowledge base
func (h *Handler) KnowledgeSearch(c *gin.Context) {
	var req struct {
		Query string `json:"query" binding:"required"`
		Limit int    `json:"limit"`
		Type  string `json:"type"` // kb|incident
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	userID := c.GetInt("user_id")
	role := c.GetString("role")

	limit := req.Limit
	if limit <= 0 {
		limit = 5
	}

	// Use the service's RAG search capability
	// 注入访问者身份，使知识分类可见性过滤生效（L0 权限边界）
	result, err := h.svc.SearchKnowledge(
		knowledgeaccess.WithViewer(c.Request.Context(), knowledgeaccess.Viewer{UserID: userID, Role: role}),
		tenantID, req.Query, req.Type, limit)
	if err != nil {
		h.svc.logger.Warnw("AI知识搜索失败，返回降级响应", "error", err, "tenantID", tenantID)
		common.Success(c, gin.H{
			"results":  []interface{}{},
			"degraded": true,
			"message":  "AI 搜索服务暂时不可用，请稍后重试",
		})
		return
	}
	common.Success(c, gin.H{
		"results":  result,
		"degraded": false,
	})
}

// Triage handles POST /api/v1/ai/triage - Ticket classification and recommendation
func (h *Handler) Triage(c *gin.Context) {
	var req struct {
		Title       string `json:"title" binding:"required"`
		Description string `json:"description"`
		Category    string `json:"category"`
		Priority    string `json:"priority"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BindValidationError(c, err, "请求参数错误")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	result, err := h.svc.TriageTicket(c.Request.Context(), tenantID, req.Title, req.Description, req.Category, req.Priority)
	if err != nil {
		h.svc.logger.Warnw("AI分诊失败，返回降级响应", "error", err, "tenantID", tenantID)
		common.Success(c, gin.H{
			"title":       req.Title,
			"description": req.Description,
			"suggestions": map[string]interface{}{},
			"degraded":    true,
			"message":     "AI 分诊服务暂时不可用，请稍后重试",
		})
		return
	}
	common.Success(c, result)
}

// CreateTicketByAI handles POST /api/v1/ai/ticket/create
// 通过 AI 解析自然语言描述，智能分析描述并返回工单创建建议
func (h *Handler) CreateTicketByAI(c *gin.Context) {
	var req struct {
		Description string `json:"description" binding:"required"`
		TenantID    int    `json:"tenantId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(c, err, "请求参数错误")
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	// 调用 AI 分析描述，返回工单创建建议
	result, err := h.svc.CreateTicketByAI(c.Request.Context(), req.Description, tenantID)
	if err != nil {
		common.FailWithErr(c, err, "AI ticket creation failed")
		return
	}
	c.JSON(http.StatusOK, result)
}

// GetToolInvocation handles GET /api/v1/agent/tools/:id
// 查询工具执行记录（跨租户隔离）
func (h *Handler) GetToolInvocation(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		common.Fail(c, common.ParamErrorCode, "invalid invocation id")
		return
	}
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	inv, err := h.svc.repo.GetToolInvocation(c.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(c, common.NotFoundCode, "invocation not found")
		return
	}
	common.Success(c, gin.H{
		"id":             inv.ID,
		"status":         inv.Status,
		"result":         inv.Result,
		"error":          inv.Error,
		"needsApproval":  inv.NeedsApproval,
		"approvalState":  inv.ApprovalState,
		"requestId":      inv.RequestID,
		"createdAt":      inv.CreatedAt,
		"conversationId": inv.ConversationID,
		"toolName":       inv.ToolName,
	})
}

// ApproveTool handles POST /api/v1/agent/tools/:id/approve
// 审批危险工具执行请求（RBAC 由路由中间件强制）
func (h *Handler) ApproveTool(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		common.Fail(c, common.ParamErrorCode, "invalid invocation id")
		return
	}
	var req struct {
		Approve bool   `json:"approve"`
		Reason  string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "请求参数错误")
		return
	}
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	userID := c.GetInt("user_id")

	state, err := h.svc.ApproveTool(c.Request.Context(), id, tenantID, userID, req.Approve, req.Reason)
	if err != nil {
		common.Fail(c, common.NotFoundCode, "invocation not found or operation failed")
		return
	}
	common.Success(c, gin.H{"invocationId": id, "approvalState": state})
}
