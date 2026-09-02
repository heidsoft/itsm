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

// GetDeepAnalytics 获取深度分析数据
// @Summary 获取深度分析数据
// @Description 获取深度分析数据，支持多维度聚合
// @Tags 数据分析
// @Accept json
// @Produce json
// @Param request body dto.DeepAnalyticsRequest true "分析请求"
// @Success 200 {object} common.Response
// @Router /api/v1/analytics/deep [post]
func (h *Handler) GetDeepAnalytics(ctx *gin.Context) {
	var req dto.DeepAnalyticsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误")
		return
	}

	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}

	response, err := h.analyticsService.GetDeepAnalytics(ctx.Request.Context(), &req, tenantID.(int))
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取分析数据失败")
		return
	}

	common.Success(ctx, response)
}

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
