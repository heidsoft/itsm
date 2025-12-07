package controller

import (
	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

// PredictionController 趋势预测控制器
type PredictionController struct {
	predictionService *service.PredictionService
}

// NewPredictionController 创建趋势预测控制器实例
func NewPredictionController(predictionService *service.PredictionService) *PredictionController {
	return &PredictionController{
		predictionService: predictionService,
	}
}

// GetTrendPrediction 获取趋势预测
func (c *PredictionController) GetTrendPrediction(ctx *gin.Context) {
	var req dto.TrendPredictionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.Fail(ctx, common.ParamErrorCode, "请求参数错误: "+err.Error())
		return
	}

	tenantID, exists := ctx.Get("tenant_id")
	if !exists {
		common.Fail(ctx, common.UnauthorizedCode, "未授权访问: 租户信息缺失")
		return
	}

	response, err := c.predictionService.GetTrendPrediction(ctx.Request.Context(), &req, tenantID.(int))
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取趋势预测失败: "+err.Error())
		return
	}

	common.Success(ctx, response)
}

// ExportPredictionReport 导出预测报告
func (c *PredictionController) ExportPredictionReport(ctx *gin.Context) {
	var req dto.TrendPredictionRequest
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

	data, filename, err := c.predictionService.ExportPredictionReport(ctx.Request.Context(), &req, format, tenantID.(int))
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "导出预测报告失败: "+err.Error())
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

