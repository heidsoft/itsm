package ai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// ListTools handles GET /api/v1/agent/tools
func (h *Handler) ListTools(c *gin.Context) {
	tools := h.svc.ListTools()
	common.Success(c, gin.H{"tools": tools})
}

// ExecuteTool handles POST /api/v1/agent/tools/execute
func (h *Handler) ExecuteTool(c *gin.Context) {
	var req struct {
		Name string                 `json:"name" binding:"required"`
		Args map[string]interface{} `json:"args"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		tenantID = 1
	}

	readOnly := false
	for _, t := range h.svc.ListTools() {
		if t.Name == req.Name {
			readOnly = t.ReadOnly
			break
		}
	}
	if !readOnly {
		common.Fail(c, common.ForbiddenCode, "tool requires approval; write tools are not enabled in agent v1")
		return
	}

	if req.Args == nil {
		req.Args = map[string]interface{}{}
	}

	res, _, err := h.svc.ExecuteTool(c.Request.Context(), tenantID, req.Name, req.Args)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{
		"status":       "success",
		"summary":      "tool executed",
		"next_actions": []string{"If the result is incomplete, refine args and retry."},
		"artifacts":    []string{},
		"data":         res,
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
		common.Fail(c, common.ParamErrorCode, err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")

	answers, convID, err := h.svc.Chat(c.Request.Context(), tenantID, userID, req.Query, req.Limit, req.ConversationID)
	if err != nil {
		// RAG 失败时降级处理：返回空结果而非 500 错误，避免前端崩溃
		h.svc.logger.Warnw("AI Chat RAG 检索失败，返回降级响应", "error", err, "tenantID", tenantID)
		common.Success(c, gin.H{
			"answers":         []interface{}{},
			"conversation_id": 0,
			"degraded":        true,
			"message":         "AI 服务暂时不可用，请稍后重试",
		})
		return
	}

	common.Success(c, gin.H{
		"answers":         answers,
		"conversation_id": convID,
	})
}

// GetDeepAnalytics handles POST /api/v1/ai/analytics
func (h *Handler) GetDeepAnalytics(c *gin.Context) {
	var req dto.DeepAnalyticsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, err.Error())
		return
	}
	tenantID := c.GetInt("tenant_id")
	res, err := h.svc.GetDeepAnalytics(c.Request.Context(), &req, tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(c, res)
}

// GetTrendPrediction handles POST /api/v1/ai/predictions
func (h *Handler) GetTrendPrediction(c *gin.Context) {
	var req dto.TrendPredictionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, err.Error())
		return
	}
	tenantID := c.GetInt("tenant_id")
	res, err := h.svc.GetTrendPrediction(c.Request.Context(), &req, tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
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
		common.Fail(c, common.InternalErrorCode, err.Error())
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
		common.Fail(c, common.ParamErrorCode, err.Error())
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
		common.Fail(c, common.InternalErrorCode, err.Error())
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
		common.Fail(c, common.ParamErrorCode, err.Error())
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
		"prompt_version": req.PromptVersion,
		"model":          req.Model,
		"confidence":     req.Confidence,
		"suggestion":     req.Suggestion,
		"notes":          req.Notes,
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
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}

	common.Success(c, gin.H{
		"request_id":     requestID,
		"scenario":       req.Scenario,
		"input_ref":      req.InputRef,
		"prompt_version": req.PromptVersion,
		"model":          req.Model,
		"confidence":     req.Confidence,
		"accepted":       req.Accepted,
	})
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
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(c, metrics)
}

// KnowledgeSearch handles POST /api/v1/ai/knowledge/search - RAG search over knowledge base
func (h *Handler) KnowledgeSearch(c *gin.Context) {
	var req struct {
		Query string `json:"query" binding:"required"`
		Limit int    `json:"limit"`
		Type  string `json:"type"` // kb|incident
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 5
	}

	// Use the service's RAG search capability
	result, err := h.svc.SearchKnowledge(c.Request.Context(), tenantID, req.Query, req.Type, limit)
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
		common.Fail(c, common.ParamErrorCode, err.Error())
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
		common.Fail(c, common.ParamErrorCode, "参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		tenantID = req.TenantID
	}
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}

	// 调用 AI 分析描述，返回工单创建建议
	result, err := h.svc.CreateTicketByAI(c.Request.Context(), req.Description, tenantID)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "AI 工单创建失败: "+err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}

// SummarizeTicketPost handles POST /api/v1/ai/tickets/:id/summarize
// POST 版本的工单总结，复用 GET SummarizeTicket 逻辑
func (h *Handler) SummarizeTicketPost(c *gin.Context) {
	h.SummarizeTicket(c)
}
