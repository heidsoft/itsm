package analytics

import (
	"context"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for analytics
type Handler struct {
	analyticsService *service.AnalyticsService
}

// NewHandler creates a new analytics handler
func NewHandler(analyticsService *service.AnalyticsService) *Handler {
	return &Handler{analyticsService: analyticsService}
}

// GetDeepAnalytics 已删除（2026-10-18 P1 收口）：它是未接线的双实现——
// 深度分析只有 AI 域一个 HTTP 所有者（`POST /ai/analytics` 与
// `POST /tickets/analytics/deep` 都走 handlers/ai → service.AnalyticsService），
// 本方法无任何路由引用，却通过 swagger 注解把幽灵路由 `/api/v1/analytics/deep`
// 宣传进 API 文档。业务规则所有者仍是 service.AnalyticsService（单一实现未动）。

// ExportAnalytics 导出分析数据
// @Summary 导出分析数据
// @Description 导出分析数据为CSV格式
// @Tags 数据分析
// @Accept json
// @Produce json
// @Param request body dto.DeepAnalyticsRequest true "导出请求"
// @Success 200 {object} common.Response
// @Router /api/v1/analytics/export [post]
func (h *Handler) ExportAnalytics(ctx *gin.Context) {
	var req dto.DeepAnalyticsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误")
		return
	}

	format := ctx.DefaultQuery("format", "csv")
	if format != "csv" && format != "excel" && format != "pdf" {
		common.Fail(ctx, common.ParamErrorCode, "不支持的导出格式，支持: csv, excel, pdf")
		return
	}

	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}

	exportCtx, cancel := context.WithTimeout(ctx.Request.Context(), 60*time.Second)
	defer cancel()
	data, filename, err := h.analyticsService.ExportAnalytics(exportCtx, &req, format, tenantID.(int))
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "导出分析数据失败")
		return
	}

	// 设置响应头
	contentType := "application/octet-stream"
	switch format {
	case "csv":
		contentType = "text/csv"
	case "excel":
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case "pdf":
		contentType = "application/pdf"
	}

	ctx.Header("Content-Type", contentType)
	ctx.Header("Content-Disposition", "attachment; filename="+filename)
	ctx.Data(200, contentType, data)
}

// GetTicketAnalytics 获取工单分析概览
// @Summary 获取工单分析概览
// @Description 聚合工单的 status / priority 分布与 30 天趋势
// @Tags 数据分析
// @Produce json
// @Success 200 {object} common.Response
// @Router /api/v1/analytics/tickets [get]
func (h *Handler) GetTicketAnalytics(ctx *gin.Context) {
	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}
	stats, err := h.analyticsService.GetTicketStats(ctx.Request.Context(), tenantID.(int))
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取工单分析失败")
		return
	}
	common.Success(ctx, stats)
}
