package prediction

import (
	"context"

	"itsm-backend/dto"
)

// PredictionService 是 prediction 域对预测服务的窄契约，方法集等于本域所有
// 生产文件对 predictionService 字段的真实调用面（ExportPredictionReport）。
//
// /prediction/export 是本域唯一接线的路由；趋势预测主方法由 handlers/ai 消费，
// 不在本契约内。值类型 DashboardViewer 已下沉到 dto 包，本文件无需引用共享层。
type PredictionService interface {
	ExportPredictionReport(ctx context.Context, req *dto.TrendPredictionRequest, format string, tenantID int, viewer dto.DashboardViewer) ([]byte, string, error)
}
