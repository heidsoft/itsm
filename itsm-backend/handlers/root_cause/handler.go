// Package root_cause — 根因分析 handler.
// 迁移自 controller/root_cause_controller.go，保持原有 API 契约不变。
package root_cause

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler 持有 RootCauseService 依赖.
type Handler struct {
	rootCauseService *service.RootCauseService
	logger          *zap.SugaredLogger
}

// NewHandler 构造 root_cause Handler.
func NewHandler(rootCauseService *service.RootCauseService, logger *zap.SugaredLogger) *Handler {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &Handler{
		rootCauseService: rootCauseService,
		logger:          logger,
	}
}

// AnalyzeTicket POST /api/v1/tickets/:id/root-cause (via AI handler)
func (h *Handler) AnalyzeTicket(ctx *gin.Context) {
	ticketIDStr := ctx.Param("id")
	ticketID, err := strconv.Atoi(ticketIDStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的工单ID: "+err.Error())
		return
	}

	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}

	response, err := h.rootCauseService.AnalyzeTicket(ctx.Request.Context(), ticketID, tenantID.(int))
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "执行根因分析失败: "+err.Error())
		return
	}

	common.Success(ctx, response)
}

// GetAnalysisReport GET /api/v1/tickets/:id/root-cause/report
func (h *Handler) GetAnalysisReport(ctx *gin.Context) {
	ticketIDStr := ctx.Param("id")
	ticketID, err := strconv.Atoi(ticketIDStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的工单ID: "+err.Error())
		return
	}

	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}

	response, err := h.rootCauseService.GetAnalysisReport(ctx.Request.Context(), ticketID, tenantID.(int))
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取分析报告失败: "+err.Error())
		return
	}

	common.Success(ctx, response)
}

// ConfirmRootCause PUT /api/v1/tickets/:id/root-cause/:rootCauseId/confirm
func (h *Handler) ConfirmRootCause(ctx *gin.Context) {
	ticketIDStr := ctx.Param("id")
	ticketID, err := strconv.Atoi(ticketIDStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的工单ID: "+err.Error())
		return
	}

	rootCauseID := ctx.Param("rootCauseId")
	if rootCauseID == "" {
		common.Fail(ctx, common.ParamErrorCode, "根因ID不能为空")
		return
	}

	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}

	err = h.rootCauseService.ConfirmRootCause(ctx.Request.Context(), ticketID, rootCauseID, tenantID.(int))
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "确认根因失败: "+err.Error())
		return
	}

	common.Success(ctx, map[string]string{"message": "根因已确认"})
}

// ResolveRootCause PUT /api/v1/tickets/:id/root-cause/:rootCauseId/resolve
func (h *Handler) ResolveRootCause(ctx *gin.Context) {
	ticketIDStr := ctx.Param("id")
	ticketID, err := strconv.Atoi(ticketIDStr)
	if err != nil {
		common.Fail(ctx, common.ParamErrorCode, "无效的工单ID: "+err.Error())
		return
	}

	rootCauseID := ctx.Param("rootCauseId")
	if rootCauseID == "" {
		common.Fail(ctx, common.ParamErrorCode, "根因ID不能为空")
		return
	}

	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}

	err = h.rootCauseService.ResolveRootCause(ctx.Request.Context(), ticketID, rootCauseID, tenantID.(int))
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "标记根因为已解决失败: "+err.Error())
		return
	}

	common.Success(ctx, map[string]string{"message": "根因已标记为已解决"})
}
