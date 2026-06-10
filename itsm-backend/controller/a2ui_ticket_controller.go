package controller

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"itsm-backend/service"
)

// A2UITicketController handles A2UI-based ticket form operations
type A2UITicketController struct {
	a2uiService *service.A2UITicketService
}

// NewA2UITicketController creates a new A2UI ticket controller
func NewA2UITicketController(a2uiSvc *service.A2UITicketService) *A2UITicketController {
	return &A2UITicketController{a2uiService: a2uiSvc}
}

// GenerateFormRequest represents the request to generate an A2UI form
type GenerateFormRequest struct {
	Intent    string `json:"intent" binding:"required"`
	SurfaceID string `json:"surfaceId"`
}

// GenerateFormResponse represents the response with A2UI messages
type GenerateFormResponse struct {
	Code     int      `json:"code"`
	Message  string   `json:"message"`
	Messages []string `json:"messages"`
}

// HandleActionRequest represents a user action on the form
type HandleActionRequest struct {
	Action    string                 `json:"action" binding:"required"`
	SurfaceID string                 `json:"surfaceId" binding:"required"`
	Context   map[string]interface{} `json:"context"`
}

// HandleActionResponse represents the response after handling an action
type HandleActionResponse struct {
	Code     int      `json:"code"`
	Message  string   `json:"message"`
	Messages []string `json:"messages"`
	Success  bool     `json:"success"`
}

// GenerateForm generates an A2UI form based on user intent
// POST /api/v1/a2ui/ticket/form
func (c *A2UITicketController) GenerateForm(ctx *gin.Context) {
	var req GenerateFormRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    1001,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 生成 surface ID（如果未提供）
	surfaceID := req.SurfaceID
	if surfaceID == "" {
		surfaceID = "ticket-create"
	}

	// 调用服务生成表单消息
	messages, err := c.a2uiService.GenerateFormMessages(ctx.Request.Context(), req.Intent, surfaceID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    5001,
			"message": "生成表单失败: " + err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, GenerateFormResponse{
		Code:     0,
		Message:  "success",
		Messages: messages,
	})
}

// HandleAction handles user actions on the A2UI form
// POST /api/v1/a2ui/ticket/action
func (c *A2UITicketController) HandleAction(ctx *gin.Context) {
	var req HandleActionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    1001,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 处理用户操作
	messages, err := c.a2uiService.HandleUserAction(ctx.Request.Context(), req.Action, req.SurfaceID, req.Context)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    5001,
			"message": "处理操作失败: " + err.Error(),
		})
		return
	}

	success := len(messages) == 0 || !strings.Contains(strings.Join(messages, ""), "error")

	ctx.JSON(http.StatusOK, HandleActionResponse{
		Code:     0,
		Message:  "success",
		Messages: messages,
		Success:  success,
	})
}

// RegisterRoutes registers the A2UI routes
func (c *A2UITicketController) RegisterRoutes(r *gin.RouterGroup) {
	a2ai := r.Group("/a2ui")
	{
		a2ai.POST("/ticket/form", c.GenerateForm)
		a2ai.POST("/ticket/action", c.HandleAction)
		// B9: 前端实际调用的别名 POST /api/v1/a2ui/tickets
		a2ai.POST("/tickets", c.ParseTicketIntent)
	}
}

// ParseTicketIntentRequest 前端调用 /a2ui/tickets 时的入参
type ParseTicketIntentRequest struct {
	Description string `json:"description" binding:"required"`
}

// ParseTicketIntent B9: 解析自然语言描述，返回结构化字段供前端预填表单
func (c *A2UITicketController) ParseTicketIntent(ctx *gin.Context) {
	var req ParseTicketIntentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    1001,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	result, err := c.a2uiService.ParseTicketIntent(ctx.Request.Context(), req.Description)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    5001,
			"message": "解析工单意图失败: " + err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}
