// Package prediction — 趋势预测导出 handler。
// 迁移自 controller/prediction_controller.go。
//
// 注：本包曾有一份 GetTrendPrediction，与 handlers/ai 的同名方法
// （/api/v1/ai/predictions 与 /api/v1/tickets/prediction/trend 的唯一接线者，
// service.PredictionService.GetTrendPrediction 的生产消费者）完全重复且从未被
// 任何路由引用（/prediction/export 是本包唯一接线的方法）——2026-10-18 清理删除，
// 消除「双 handler 同用例」形态。
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

// ExportPredictionReport POST /api/v1/tickets/prediction/export
func (h *Handler) ExportPredictionReport(ctx *gin.Context) {
	var req dto.TrendPredictionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		common.ParamErrorWithErr(ctx, err, "请求参数错误")
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
		common.RespondError(ctx, err, "导出预测报告失败")
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
