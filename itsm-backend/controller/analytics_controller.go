package controller

import (
	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

// AnalyticsController 数据分析控制器
type AnalyticsController struct {
	analyticsService *service.AnalyticsService
}

// NewAnalyticsController 创建数据分析控制器实例
func NewAnalyticsController(analyticsService *service.AnalyticsService) *AnalyticsController {
	return &AnalyticsController{
		analyticsService: analyticsService,
	}
}

// GetDeepAnalytics 获取深度分析数据
func (c *AnalyticsController) GetDeepAnalytics(ctx *gin.Context) {
	var req dto.DeepAnalyticsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}

	response, err := c.analyticsService.GetDeepAnalytics(ctx.Request.Context(), &req, tenantID.(int))
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取分析数据失败: "+err.Error())
		return
	}

	common.Success(ctx, response)
}

// ExportAnalytics 导出分析数据
func (c *AnalyticsController) ExportAnalytics(ctx *gin.Context) {
	var req dto.DeepAnalyticsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
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

	data, filename, err := c.analyticsService.ExportAnalytics(ctx.Request.Context(), &req, format, tenantID.(int))
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "导出分析数据失败: "+err.Error())
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

