package ai

import (
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
		common.Fail(c, http.StatusBadRequest, err.Error())
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
		common.Fail(c, http.StatusForbidden, "tool requires approval; write tools are not enabled in agent v1")
		return
	}

	if req.Args == nil {
		req.Args = map[string]interface{}{}
	}

	res, _, err := h.svc.ExecuteTool(c.Request.Context(), tenantID, req.Name, req.Args)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
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
		ConversationID int    `json:"conversation_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")

	answers, convID, err := h.svc.Chat(c.Request.Context(), tenantID, userID, req.Query, req.Limit, req.ConversationID)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
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
		common.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	tenantID := c.GetInt("tenant_id")
	res, err := h.svc.GetDeepAnalytics(c.Request.Context(), &req, tenantID)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, res)
}

// GetTrendPrediction handles POST /api/v1/ai/predictions
func (h *Handler) GetTrendPrediction(c *gin.Context) {
	var req dto.TrendPredictionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	tenantID := c.GetInt("tenant_id")
	res, err := h.svc.GetTrendPrediction(c.Request.Context(), &req, tenantID)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, res)
}

// AnalyzeTicket handles POST /api/v1/ai/tickets/:id/analyze
func (h *Handler) AnalyzeTicket(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantID := c.GetInt("tenant_id")

	res, err := h.svc.AnalyzeTicket(c.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, res)
}

// SummarizeTicket handles GET /api/v1/ai/tickets/:id/summary
// B9: AI 工单总结 - 优先用 LLM，fallback 用字段拼接
func (h *Handler) SummarizeTicket(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	tenantID := c.GetInt("tenant_id")

	if id <= 0 {
		common.Fail(c, http.StatusBadRequest, "invalid ticket id")
		return
	}

	summary, err := h.svc.SummarizeTicket(c.Request.Context(), id, tenantID)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, summary)
}

// SaveFeedback handles POST /api/v1/ai/feedback
func (h *Handler) SaveFeedback(c *gin.Context) {
	var req struct {
		Kind     string  `json:"kind" binding:"required"`
		Query    string  `json:"query"`
		ItemType *string `json:"item_type"`
		ItemID   *int    `json:"item_id"`
		Useful   bool    `json:"useful" binding:"required"`
		Score    *int    `json:"score"`
		Notes    *string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, err.Error())
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
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, gin.H{"message": "Feedback saved"})
}

// KnowledgeSearch handles POST /api/v1/ai/knowledge/search - RAG search over knowledge base
func (h *Handler) KnowledgeSearch(c *gin.Context) {
	var req struct {
		Query string `json:"query" binding:"required"`
		Limit int    `json:"limit"`
		Type  string `json:"type"` // kb|incident
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		tenantID = 1 // default tenant
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 5
	}

	// Use the service's RAG search capability
	result, err := h.svc.SearchKnowledge(c.Request.Context(), tenantID, req.Query, req.Type, limit)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, result)
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
		common.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		tenantID = 1 // default tenant
	}

	result, err := h.svc.TriageTicket(c.Request.Context(), tenantID, req.Title, req.Description, req.Category, req.Priority)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(c, result)
}
