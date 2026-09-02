// Package prediction — 趋势预测 handler.
// 迁移自 controller/prediction_controller.go，保持原有 API 契约不变。
package prediction

import (
	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler 持有 PredictionService 依赖.
type Handler struct {
	predictionService *service.PredictionService
	logger            *zap.SugaredLogger
}

// NewHandler 构造 prediction Handler.
func NewHandler(predictionService *service.PredictionService, logger *zap.SugaredLogger) *Handler {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &Handler{
		predictionService: predictionService,
		logger:            logger,
	}
}

// GetTrendPrediction POST /api/v1/tickets/prediction/trend
func (h *Handler) GetTrendPrediction(ctx *gin.Context) {
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

	response, err := h.predictionService.GetTrendPrediction(ctx.Request.Context(), &req, tenantID.(int))
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "获取趋势预测失败: "+err.Error())
		return
	}

	common.Success(ctx, response)
}

// ExportPredictionReport POST /api/v1/tickets/prediction/export
func (h *Handler) ExportPredictionReport(ctx *gin.Context) {
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

	data, filename, err := h.predictionService.ExportPredictionReport(ctx.Request.Context(), &req, format, tenantID.(int))
	if err != nil {
		common.Fail(ctx, common.InternalErrorCode, "导出预测报告失败: "+err.Error())
		return
	}

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
