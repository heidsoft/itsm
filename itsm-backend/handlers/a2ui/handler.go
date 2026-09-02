// Package a2ui — A2UI 工单表单 handler.
// 迁移自 controller/a2ui_ticket_controller.go，保持原有 API 契约不变。
package a2ui

import (
	"net/http"
	"strings"

	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler struct {
	a2uiService *service.A2UITicketService
	logger      *zap.SugaredLogger
}

func NewHandler(a2uiService *service.A2UITicketService, logger *zap.SugaredLogger) *Handler {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &Handler{
		a2uiService: a2uiService,
		logger:      logger,
	}
}

type GenerateFormRequest struct {
	Intent    string `json:"intent" binding:"required"`
	SurfaceID string `json:"surfaceId"`
}

type GenerateFormResponse struct {
	Code     int      `json:"code"`
	Message  string   `json:"message"`
	Messages []string `json:"messages"`
}

type HandleActionRequest struct {
	Action    string                 `json:"action" binding:"required"`
	SurfaceID string                 `json:"surfaceId" binding:"required"`
	Context   map[string]interface{} `json:"context"`
}

type HandleActionResponse struct {
	Code     int      `json:"code"`
	Message  string   `json:"message"`
	Messages []string `json:"messages"`
	Success  bool     `json:"success"`
}

type ParseTicketIntentRequest struct {
	Description string `json:"description" binding:"required"`
}

// RegisterRoutes 注册 /api/v1/a2ui 路由组.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	if rg == nil {
		return
	}
	a2ai := rg.Group("/a2ui")
	{
		a2ai.POST("/ticket/form", h.GenerateForm)
		a2ai.POST("/ticket/action", h.HandleAction)
		// B9: 前端实际调用的别名 POST /api/v1/a2ui/tickets
		a2ai.POST("/tickets", h.ParseTicketIntent)
	}
}

// GenerateForm POST /api/v1/a2ui/ticket/form
func (h *Handler) GenerateForm(ctx *gin.Context) {
	var req GenerateFormRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    1001,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	surfaceID := req.SurfaceID
	if surfaceID == "" {
		surfaceID = "ticket-create"
	}

	messages, err := h.a2uiService.GenerateFormMessages(ctx.Request.Context(), req.Intent, surfaceID)
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

// HandleAction POST /api/v1/a2ui/ticket/action
func (h *Handler) HandleAction(ctx *gin.Context) {
	var req HandleActionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    1001,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	messages, err := h.a2uiService.HandleUserAction(ctx.Request.Context(), req.Action, req.SurfaceID, req.Context)
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

// ParseTicketIntent POST /api/v1/a2ui/tickets
func (h *Handler) ParseTicketIntent(ctx *gin.Context) {
	var req ParseTicketIntentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    1001,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	result, err := h.a2uiService.ParseTicketIntent(ctx.Request.Context(), req.Description)
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
