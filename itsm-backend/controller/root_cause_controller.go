package controller

import (
	"strconv"

	"itsm-backend/common"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

// RootCauseController 根因分析控制器
type RootCauseController struct {
	rootCauseService *service.RootCauseService
}

// NewRootCauseController 创建根因分析控制器实例
func NewRootCauseController(rootCauseService *service.RootCauseService) *RootCauseController {
	return &RootCauseController{
		rootCauseService: rootCauseService,
	}
}

// AnalyzeTicket 执行根因分析
func (c *RootCauseController) AnalyzeTicket(ctx *gin.Context) {
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

	response, err := c.rootCauseService.AnalyzeTicket(ctx.Request.Context(), ticketID, tenantID.(int))
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "执行根因分析失败: "+err.Error())
		return
	}

	common.Success(ctx, response)
}

// GetAnalysisReport 获取分析报告
func (c *RootCauseController) GetAnalysisReport(ctx *gin.Context) {
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

	response, err := c.rootCauseService.GetAnalysisReport(ctx.Request.Context(), ticketID, tenantID.(int))
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取分析报告失败: "+err.Error())
		return
	}

	common.Success(ctx, response)
}

// ConfirmRootCause 确认根因
func (c *RootCauseController) ConfirmRootCause(ctx *gin.Context) {
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

	err = c.rootCauseService.ConfirmRootCause(ctx.Request.Context(), ticketID, rootCauseID, tenantID.(int))
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "确认根因失败: "+err.Error())
		return
	}

	common.Success(ctx, map[string]string{"message": "根因已确认"})
}

// ResolveRootCause 标记根因为已解决
func (c *RootCauseController) ResolveRootCause(ctx *gin.Context) {
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

	err = c.rootCauseService.ResolveRootCause(ctx.Request.Context(), ticketID, rootCauseID, tenantID.(int))
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "标记根因为已解决失败: "+err.Error())
		return
	}

	common.Success(ctx, map[string]string{"message": "根因已标记为已解决"})
}

