// Package a2ui — A2UI 工单表单 handler.
// 迁移自 controller/a2ui_ticket_controller.go，保持原有 API 契约不变。
package a2ui

import (
	"errors"
	"net/http"

	"itsm-backend/common"
	"itsm-backend/middleware"
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
	// A2UI 属 AI 生成面：写操作按 ai:write 授权
	a2ai := rg.Group("/a2ui")
	{
		a2ai.POST("/ticket/form", middleware.RequirePermission("ai", "write"), h.GenerateForm)
		a2ai.POST("/ticket/action", middleware.RequirePermission("ai", "write"), h.HandleAction)
		// B9: 前端实际调用的别名 POST /api/v1/a2ui/tickets
		a2ai.POST("/tickets", middleware.RequirePermission("ai", "write"), h.ParseTicketIntent)
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
		// B11（UPGRADE §1.48）：错误响应不透传内部 err.Error()。
		h.logger.Errorw("a2ui generate form failed", "intent", req.Intent, "error", err)
		common.Fail(ctx, common.InternalErrorCode, "生成表单失败")
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
		// B11（UPGRADE §1.48）：submit 是未接线能力 → 显式 501/NotImplementedCode，
		// 不再返回伪造的 TKT 工单号假成功（缺陷修复：443 行原实现）。
		if errors.Is(err, service.ErrA2UISubmitNotImplemented) {
			common.Fail(ctx, common.NotImplementedCode, "A2UI 工单提交通道尚未接入，暂不支持提交，请前往工单创建页手动提交")
			return
		}
		// 其余意外错误：日志记原始错误，响应不透传 err.Error()。
		h.logger.Errorw("a2ui handle action failed", "action", req.Action, "error", err)
		common.Fail(ctx, common.InternalErrorCode, "处理操作失败")
		return
	}

	// B11：移除「响应拼串里是否含 error 子串」的嗅探启发式——success 由真实
	// 执行结果决定（到达此处的调用均已成功执行）。
	ctx.JSON(http.StatusOK, HandleActionResponse{
		Code:     0,
		Message:  "success",
		Messages: messages,
		Success:  true,
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
		// B11（UPGRADE §1.48）：错误响应不透传内部 err.Error()。
		h.logger.Errorw("a2ui parse intent failed", "error", err)
		common.Fail(ctx, common.InternalErrorCode, "解析工单意图失败")
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}
