package analytics

import (
	"context"

	"itsm-backend/dto"
)

// AnalyticsService 是 analytics 域对分析服务的窄契约，方法集等于本域所有
// 生产文件对 analyticsService 字段的真实调用面（ExportAnalytics / GetTicketStats）。
//
// 生产装配传入共享层具体指针即可满足本接口；值类型 DashboardViewer 已下沉到
// dto 包并由共享层以类型别名暴露，因此本文件无需引用共享层。
type AnalyticsService interface {
	ExportAnalytics(ctx context.Context, req *dto.DeepAnalyticsRequest, format string, tenantID int, viewer dto.DashboardViewer) ([]byte, string, error)
	GetTicketStats(ctx context.Context, tenantID int) (*dto.AnalyticsTicketStatsResponse, error)
}
