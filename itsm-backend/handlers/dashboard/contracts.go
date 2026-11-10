package dashboard

import (
	"context"

	"itsm-backend/dto"
)

// 工作台域的共享层能力窄契约。方法集等于 handlers/dashboard 全部生产文件
// 对工作台服务与服务质量聚合器的调用并集；签名只引用 dto 值类型与标准库类型，
// 因此本文件（含注释）不出现共享层包路径。具体服务指针由装配层注入，
// 编译期断言在 internal/layering/domain_contracts_test.go。

// DashboardOverviewProvider 工作台概览、趋势与统计的只读能力。
type DashboardOverviewProvider interface {
	GetDashboardOverview(ctx context.Context, tenantID int, viewer dto.DashboardViewer) (*dto.DashboardOverviewData, error)
	GetDashboardOverviewStats(ctx context.Context, tenantID int) (*dto.DashboardOverviewStats, error)
	// GetDashboardData 只被 migrate.go 的三个兼容路由调用，但仍属本域调用并集。
	GetDashboardData(ctx context.Context, tenantID int) (*dto.DashboardResponse, error)
	GetSystemStats(ctx context.Context) (*dto.SystemStatsResponse, error)
	GetTicketTrend(ctx context.Context, tenantID, days int) ([]dto.TicketTrendData, error)
	GetUserStats(ctx context.Context, tenantID int) (*dto.UserStatsResponse, error)
}

// ServiceQualityReportBuilder 管理者统一服务质量视图聚合能力。
// 未装配时端点显式 503，不做静默降级。
type ServiceQualityReportBuilder interface {
	Build(ctx context.Context, tenantID int, viewer dto.DashboardViewer, days int) (*dto.ServiceQualityReport, error)
}
