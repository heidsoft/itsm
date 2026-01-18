package controller

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"go.uber.org/zap"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

type AIController struct {
	rag                *service.RAGService
	tools              *service.ToolRegistry
	client             *ent.Client
	queue              *service.ToolQueue
	embedder           service.Embedder
	vectors            *service.VectorStore
	triage             *service.TriageService
	summary            *service.SummarizeService
	aiTelemetryService *service.AITelemetryService
	logger             *zap.Logger
}

func NewAIController(rag *service.RAGService, client *ent.Client, aiTelemetryService *service.AITelemetryService, gateway *service.LLMGateway, logger *zap.Logger) *AIController {
	return &AIController{
		rag:                rag,
		client:             client,
		triage:             service.NewTriageService(gateway, logger),
		summary:            service.NewSummarizeService(gateway, logger),
		aiTelemetryService: aiTelemetryService,
		logger:             logger,
	}
}

func (a *AIController) SetToolRegistry(tr *service.ToolRegistry) { a.tools = tr }
func (a *AIController) SetToolQueue(q *service.ToolQueue)        { a.queue = q }
func (a *AIController) SetEmbeddingResources(e service.Embedder, vs *service.VectorStore) {
	a.embedder = e
	a.vectors = vs
}

type SearchRequest struct {
	Query string `json:"query" binding:"required"`
	Limit int    `json:"limit"`
	Type  string `json:"type"` // kb|incident|*
}

// Search RAG over KB or incidents
func (a *AIController) Search(c *gin.Context) {
	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "参数错误: "+err.Error())
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
	if req.Type == "incident" && a.vectors != nil && a.embedder != nil {
		items, err := service.SimilarIncidents(c.Request.Context(), a.vectors, a.embedder, tenantID, req.Query, limit)
		if err != nil {
			common.Fail(c, common.InternalErrorCode, err.Error())
			return
		}
		common.Success(c, gin.H{"answers": items})
		return
	}
	// default to KB RAG
	items, err := a.rag.Ask(c.Request.Context(), tenantID, req.Query, limit)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(c, gin.H{"answers": items})
}

type TriageRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
}

// Triage intelligent suggestion
func (a *AIController) Triage(c *gin.Context) {
	var req TriageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "参数错误: "+err.Error())
		return
	}
	res := a.triage.Suggest(c.Request.Context(), req.Title, req.Description)
	common.Success(c, res)
}

type SummarizeRequest struct {
	Text   string `json:"text" binding:"required"`
	MaxLen int    `json:"max_len"`
}

// Summarize text for internal/external updates
func (a *AIController) Summarize(c *gin.Context) {
	var req SummarizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "参数错误: "+err.Error())
		return
	}
	sum, err := a.summary.Summarize(c.Request.Context(), req.Text, req.MaxLen)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "摘要生成失败: "+err.Error())
		return
	}
	common.Success(c, gin.H{"summary": sum})
}

type SimilarIncidentsRequest struct {
	Query string `json:"query" binding:"required"`
	Limit int    `json:"limit"`
}

func (a *AIController) SimilarIncidents(c *gin.Context) {
	var req SimilarIncidentsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "参数错误: "+err.Error())
		return
	}
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	if a.vectors == nil || a.embedder == nil {
		common.Fail(c, common.InternalErrorCode, "Vector 搜索未初始化")
		return
	}
	if req.Limit <= 0 {
		req.Limit = 5
	}
	items, err := service.SimilarIncidents(c.Request.Context(), a.vectors, a.embedder, tenantID, req.Query, req.Limit)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(c, gin.H{"incidents": items})
}

type ChatRequest struct {
	Query          string `json:"query" binding:"required"`
	Limit          int    `json:"limit"`
	ConversationID int    `json:"conversation_id"`
}

// Chat godoc
// @Summary      AI 问答（RAG）
// @Description  基于知识库的检索增强问答
// @Tags         ai
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  ChatRequest  true  "查询参数"
// @Success      200   {object}  map[string]interface{}
// @Router       /ai/chat [post]
func (a *AIController) Chat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "参数错误: "+err.Error())
		return
	}
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		// fallback from header (for testing)
		if s := c.GetHeader("X-Tenant-ID"); s != "" {
			if id, err := strconv.Atoi(s); err == nil {
				tenantID = id
			}
		}
	}
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	items, err := a.rag.Ask(c.Request.Context(), tenantID, req.Query, req.Limit)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	// persist conversation/messages
	convID := req.ConversationID
	if convID == 0 {
		conv, err := a.client.Conversation.Create().
			SetTenantID(tenantID).
			SetUserID(c.GetInt("user_id")).
			SetTitle("AI 对话").
			Save(c.Request.Context())
		if err == nil {
			convID = conv.ID
		}
	}
	if convID != 0 {
		_ = a.client.Message.Create().
			SetConversationID(convID).
			SetRole("user").
			SetContent(req.Query).
			SetRequestID(c.GetString("request_id")).
			Exec(c.Request.Context())
		payload, _ := json.Marshal(items)
		_ = a.client.Message.Create().
			SetConversationID(convID).
			SetRole("assistant").
			SetContent(string(payload)).
			SetRequestID(c.GetString("request_id")).
			Exec(c.Request.Context())
	}
	common.Success(c, gin.H{"answers": items, "conversation_id": convID})
}

// Tools godoc
// @Summary      工具清单
// @Description  列出可用工具
// @Tags         ai
// @Produce      json
// @Security     BearerAuth
// @Success      200   {object}  map[string]interface{}
// @Router       /ai/tools [get]
func (a *AIController) Tools(c *gin.Context) {
	if a.tools == nil {
		common.Fail(c, common.InternalErrorCode, "tool registry not initialized")
		return
	}
	common.Success(c, a.tools.ListTools())
}

type ToolExecRequest struct {
	Name string                 `json:"name" binding:"required"`
	Args map[string]interface{} `json:"args"`
}

// ExecuteTool godoc
// @Summary      执行工具
// @Description  只读工具直接执行，危险工具生成审批记录
// @Tags         ai
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  ToolExecRequest  true  "执行参数"
// @Success      200   {object}  map[string]interface{}
// @Router       /ai/tools/execute [post]
func (a *AIController) ExecuteTool(c *gin.Context) {
	if a.tools == nil {
		common.Fail(c, common.InternalErrorCode, "tool registry not initialized")
		return
	}
	var req ToolExecRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "参数错误: "+err.Error())
		return
	}
	tenantID := c.GetInt("tenant_id")
	// check tool definition to decide approval
	needsApproval := false
	for _, t := range a.tools.ListTools() {
		if t.Name == req.Name {
			needsApproval = !t.ReadOnly
			break
		}
	}
	if !needsApproval {
		result, err := a.tools.Execute(c.Request.Context(), tenantID, req.Name, req.Args)
		if err != nil {
			common.Fail(c, common.InternalErrorCode, err.Error())
			return
		}
		common.Success(c, result)
		return
	}
	// Danger tools require admin to request/approve; record who requested
	if role := c.GetString("role"); role == "" {
		common.Fail(c, common.ForbiddenCode, "无角色信息")
		return
	}
	// create ToolInvocation pending approval
	argsStr, _ := json.Marshal(req.Args)
	inv, err := a.client.ToolInvocation.Create().
		SetConversationID(0).
		SetToolName(req.Name).
		SetArguments(string(argsStr)).
		SetStatus("pending").
		SetRequestID(c.GetString("request_id")).
		SetNeedsApproval(true).
		SetApprovalState("pending").
		Save(c.Request.Context())
	if err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(c, gin.H{"invocation_id": inv.ID, "approval_state": inv.ApprovalState})
}

type ApproveRequest struct {
	Approve bool   `json:"approve"`
	Reason  string `json:"reason"`
}

// ApproveTool godoc
// @Summary      审批工具执行
// @Description  管理员审批危险工具执行请求
// @Tags         ai
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path  int            true  "invocation id"
// @Param        body  body  ApproveRequest true  "审批参数"
// @Success      200   {object}  map[string]interface{}
// @Router       /ai/tools/{id}/approve [post]
func (a *AIController) ApproveTool(c *gin.Context) {
	idStr := c.Param("id")
	var body ApproveRequest
	_ = c.ShouldBindJSON(&body)
	// load invocation
	id, _ := strconv.Atoi(idStr)
	inv, err := a.client.ToolInvocation.Get(c.Request.Context(), id)
	if err != nil {
		common.Fail(c, common.ParamErrorCode, "invocation not found")
		return
	}
	// RBAC: only admin can approve dangerous tool invocations
	if role := c.GetString("role"); role != "admin" {
		common.Fail(c, common.ForbiddenCode, "仅管理员可审批")
		return
	}
	if !body.Approve {
		_, _ = a.client.ToolInvocation.UpdateOneID(inv.ID).SetApprovalState("rejected").SetApprovalReason(body.Reason).Save(c.Request.Context())
		common.Success(c, gin.H{"invocation_id": inv.ID, "approval_state": "rejected"})
		return
	}
	// approve and enqueue
	_, _ = a.client.ToolInvocation.UpdateOneID(inv.ID).SetApprovalState("approved").SetApprovedBy(c.GetInt("user_id")).Save(c.Request.Context())
	if a.queue != nil {
		a.queue.Enqueue(service.ToolJob{InvocationID: inv.ID, TenantID: c.GetInt("tenant_id"), RequestID: c.GetString("request_id")})
	}
	common.Success(c, gin.H{"invocation_id": inv.ID, "approval_state": "approved"})
}

// GetToolInvocation godoc
// @Summary      查询工具执行状态
// @Description  根据ID查询工具执行记录状态与结果
// @Tags         ai
// @Produce      json
// @Security     BearerAuth
// @Param        id    path  int  true  "invocation id"
// @Success      200   {object}  map[string]interface{}
// @Router       /ai/tools/{id} [get]
func (a *AIController) GetToolInvocation(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	inv, err := a.client.ToolInvocation.Get(c.Request.Context(), id)
	if err != nil {
		common.Fail(c, common.NotFoundCode, "invocation not found")
		return
	}
	common.Success(c, gin.H{
		"id":              inv.ID,
		"status":          inv.Status,
		"result":          inv.Result,
		"error":           inv.Error,
		"needs_approval":  inv.NeedsApproval,
		"approval_state":  inv.ApprovalState,
		"request_id":      inv.RequestID,
		"created_at":      inv.CreatedAt,
		"conversation_id": inv.ConversationID,
		"tool_name":       inv.ToolName,
	})
}

type RunEmbedRequest struct {
	Limit int `json:"limit"`
}

// RunEmbed 触发一次嵌入流水线（按当前租户）
func (a *AIController) RunEmbed(c *gin.Context) {
	var req RunEmbedRequest
	_ = c.ShouldBindJSON(&req)
	if req.Limit <= 0 {
		req.Limit = 50
	}
	tenantID := c.GetInt("tenant_id")
	if tenantID == 0 {
		common.Fail(c, common.AuthFailedCode, "租户信息缺失")
		return
	}
	if a.embedder == nil || a.vectors == nil {
		common.Fail(c, common.InternalErrorCode, "Embedding resources not initialized")
		return
	}
	pipeline := service.NewEmbeddingPipeline(a.client, a.embedder, nil, a.vectors)
	if err := pipeline.RunOnce(c.Request.Context(), tenantID, req.Limit); err != nil {
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	common.Success(c, gin.H{"status": "ok", "embedded": req.Limit})
}

// FeedbackRequest represents the request body for AI feedback
type FeedbackRequest struct {
	Kind     string  `json:"kind" binding:"required"`
	Query    string  `json:"query"`
	ItemType *string `json:"item_type"`
	ItemID   *int    `json:"item_id"`
	Useful   bool    `json:"useful" binding:"required"`
	Score    *int    `json:"score"`
	Notes    *string `json:"notes"`
}

// SaveFeedback saves user feedback on AI suggestions
func (a *AIController) SaveFeedback(c *gin.Context) {
	var req FeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, "参数错误: "+err.Error())
		return
	}

	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")
	requestID := c.GetString("request_id")

	if requestID == "" {
		requestID = fmt.Sprintf("req_%d_%d", time.Now().Unix(), userID)
	}

	// Handle pointer types for optional fields
	var itemTypeVal string
	if req.ItemType != nil {
		itemTypeVal = *req.ItemType
	}

	err := a.aiTelemetryService.SaveFeedback(c.Request.Context(), tenantID, userID, requestID, req.Kind, req.Query, itemTypeVal, req.ItemID, req.Useful, req.Score, req.Notes)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "Failed to save feedback")
		return
	}

	common.Success(c, gin.H{"message": "Feedback saved successfully"})
}

// GetMetrics retrieves AI usage metrics
func (a *AIController) GetMetrics(c *gin.Context) {
	tenantID := c.GetInt("tenant_id")

	lookbackDays := 7 // default to 7 days
	if daysStr := c.Query("days"); daysStr != "" {
		if days, err := strconv.Atoi(daysStr); err == nil && days > 0 && days <= 365 {
			lookbackDays = days
		}
	}

	metrics, err := a.aiTelemetryService.GetMetrics(c.Request.Context(), tenantID, lookbackDays)
	if err != nil {
		common.Fail(c, common.InternalErrorCode, "Failed to get metrics")
		return
	}

	common.Success(c, metrics)
}
