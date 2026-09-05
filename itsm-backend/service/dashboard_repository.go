package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"itsm-backend/database"
)

// dashboardRepository 封装仪表盘复杂聚合 raw SQL。
//
// 之所以单独抽出这些方法而不是改用 Ent：
//  1. dashboard 的 AVG(...) FILTER (...) / EXTRACT(EPOCH FROM ...) / 跨月 CTE 聚合
//     在当前 Ent 生成器中没有等价的能力，硬迁会牺牲可读性并产生大量 generated 噪声。
//  2. 单一职责：service 负责业务编排和租户/权限校验；raw SQL 集中在 repository，
//     schema 变化只影响本文件，便于演进为可审计的复杂聚合出口。
//  3. 与本目录其他 repository（change/incident/ticket/problem_investigation）保持
//     同样的封装形态，避免 controller/service 中夹杂裸 SQL。
//
// 所有方法都强制 tenant_id 谓词（fail-closed），并在 db 为 nil 时返回明确错误，
// 避免生产环境误退化为 nil QueryRowContext panic。
type dashboardRepository struct {
	db *sql.DB
}

// newDashboardRepository 构造 dashboard 聚合仓储。
// 允许传入 nil（测试场景或尚未初始化的全局 rawDB），调用方在拿到 nil 仓储时
// 应当决定是否走降级路径；仓储自身方法在 nil 状态下会返回 error。
func newDashboardRepository(db *sql.DB) *dashboardRepository {
	return &dashboardRepository{db: db}
}

// requireDB 仓储方法内部前置校验：db 必须非空。
func (r *dashboardRepository) requireDB() error {
	if r == nil || r.db == nil {
		return errors.New("dashboard repository: 数据库连接未初始化")
	}
	return nil
}

// AvgResponseAndResolutionHours 计算当前租户工单的平均首次响应时长和平均解决时长（小时）。
// 对应原 dashboard_service.go 中 GetDashboardOverviewStats 的 raw SQL。
//
// 入参：
//   - ctx：请求上下文（必须携带租户信息之外的取消/超时信号）
//   - tenantID：租户主键；缺失或非法将返回 0 值与 nil error（空集场景），不应跨租户查询
//
// 返回：avgResp（小时，可能为 0）、avgRes（小时，可能为 0）、error
func (r *dashboardRepository) AvgResponseAndResolutionHours(ctx context.Context, tenantID int) (avgResp, avgRes float64, err error) {
	if err := r.requireDB(); err != nil {
		return 0, 0, err
	}
	if tenantID <= 0 {
		return 0, 0, errors.New("dashboard repository: tenantID 必须为正整数")
	}

	const query = `
		SELECT
			COALESCE(AVG(EXTRACT(EPOCH FROM (first_response_at - created_at)) / 3600.0)
				FILTER (WHERE first_response_at IS NOT NULL), 0),
			COALESCE(AVG(EXTRACT(EPOCH FROM (resolved_at - created_at)) / 3600.0)
				FILTER (WHERE resolved_at IS NOT NULL), 0)
		FROM tickets
		WHERE tenant_id = $1 AND deleted_at IS NULL
	`
	if _, err = database.WithTenantSQL(ctx, r.db, tenantID, func(q database.SQLExecutor) (struct{}, error) {
		return struct{}{}, q.QueryRowContext(ctx, query, tenantID).Scan(&avgResp, &avgRes)
	}); err != nil {
		return 0, 0, err
	}
	return avgResp, avgRes, nil
}

// KPIAvgAndSLAScope 计算本月/上月平均响应与解决时长 + 本月 SLA 达成 scope。
// 对应原 dashboard_service.go 中 getKPIMetrics 的 CTE raw SQL。
//
// 入参：
//   - ctx：请求上下文
//   - tenantID：租户主键
//   - thisMonthStart：本月窗口起点（含）
//   - lastMonthStart：上月窗口起点（含），用于本月/上月对比及 SLA scope 起点
//
// 返回：
//   - avgRespNow / avgResNow：本月平均首次响应 / 解决时长（小时）
//   - avgRespPrev / avgResPrev：上月平均首次响应 / 解决时长（小时）
//   - totalSLATickets / metSLATickets：本月参与 SLA 考核的工单数 / 达成数
//
// 时间窗口语义：current_month = [thisMonthStart, +∞)、prev_month =
// [lastMonthStart, thisMonthStart)、sla_scope = [lastMonthStart, +∞)。
// 与原 SQL 完全一致，迁移到本仓储后口径不变。
func (r *dashboardRepository) KPIAvgAndSLAScope(
	ctx context.Context,
	tenantID int,
	thisMonthStart, lastMonthStart time.Time,
) (
	avgRespNow, avgResNow, avgRespPrev, avgResPrev float64,
	totalSLATickets, metSLATickets int,
	err error,
) {
	if err := r.requireDB(); err != nil {
		return 0, 0, 0, 0, 0, 0, err
	}
	if tenantID <= 0 {
		return 0, 0, 0, 0, 0, 0, errors.New("dashboard repository: tenantID 必须为正整数")
	}
	if thisMonthStart.IsZero() || lastMonthStart.IsZero() {
		return 0, 0, 0, 0, 0, 0, errors.New("dashboard repository: 时间窗口不能为零值")
	}
	if !lastMonthStart.Before(thisMonthStart) {
		return 0, 0, 0, 0, 0, 0, errors.New("dashboard repository: lastMonthStart 必须早于 thisMonthStart")
	}

	const query = `
		WITH current_month AS (
			SELECT id, first_response_at, resolved_at, created_at
			FROM tickets
			WHERE tenant_id = $1 AND deleted_at IS NULL AND created_at >= $2
		),
		prev_month AS (
			SELECT id, first_response_at, resolved_at, created_at
			FROM tickets
			WHERE tenant_id = $1 AND deleted_at IS NULL AND created_at >= $3 AND created_at < $2
		),
		sla_scope AS (
			SELECT id, first_response_at, resolved_at, created_at, sla_response_deadline, sla_resolution_deadline
			FROM tickets
			WHERE tenant_id = $1 AND deleted_at IS NULL AND created_at >= $3
		)
		SELECT
			COALESCE((SELECT AVG(EXTRACT(EPOCH FROM (first_response_at - created_at)) / 3600.0)
			           FROM current_month WHERE first_response_at IS NOT NULL), 0),
			COALESCE((SELECT AVG(EXTRACT(EPOCH FROM (resolved_at - created_at)) / 3600.0)
			           FROM current_month WHERE resolved_at IS NOT NULL), 0),
			COALESCE((SELECT AVG(EXTRACT(EPOCH FROM (first_response_at - created_at)) / 3600.0)
			           FROM prev_month WHERE first_response_at IS NOT NULL), 0),
			COALESCE((SELECT AVG(EXTRACT(EPOCH FROM (resolved_at - created_at)) / 3600.0)
			           FROM prev_month WHERE resolved_at IS NOT NULL), 0),
			(SELECT COUNT(*) FROM sla_scope),
			(SELECT COUNT(*) FROM sla_scope WHERE (
				sla_response_deadline IS NULL OR first_response_at <= sla_response_deadline
			) AND (
				sla_resolution_deadline IS NULL OR resolved_at <= sla_resolution_deadline
			))
	`
	if _, err = database.WithTenantSQL(ctx, r.db, tenantID, func(q database.SQLExecutor) (struct{}, error) {
		return struct{}{}, q.QueryRowContext(ctx, query, tenantID, thisMonthStart, lastMonthStart).Scan(
			&avgRespNow, &avgResNow, &avgRespPrev, &avgResPrev, &totalSLATickets, &metSLATickets,
		)
	}); err != nil {
		return 0, 0, 0, 0, 0, 0, err
	}
	return avgRespNow, avgResNow, avgRespPrev, avgResPrev, totalSLATickets, metSLATickets, nil
}

// PreviousSLAScopeCount 计算上月 SLA 达成 scope，用于计算 SLA 达成率环比变化。
// 对应原 dashboard_service.go 中 getKPIMetrics 第二段 raw SQL。
//
// 入参：
//   - ctx：请求上下文
//   - tenantID：租户主键
//   - lastMonthStart：上月窗口起点（含）
//   - thisMonthStart：本月窗口起点（不含），用于圈定上月窗口上界
//
// 返回：上月参与 SLA 考核的工单数 / 达成数。当窗口为空时返回 0,0,nil。
func (r *dashboardRepository) PreviousSLAScopeCount(
	ctx context.Context,
	tenantID int,
	lastMonthStart, thisMonthStart time.Time,
) (totalSLATickets, metSLATickets int, err error) {
	if err := r.requireDB(); err != nil {
		return 0, 0, err
	}
	if tenantID <= 0 {
		return 0, 0, errors.New("dashboard repository: tenantID 必须为正整数")
	}
	if thisMonthStart.IsZero() || lastMonthStart.IsZero() {
		return 0, 0, errors.New("dashboard repository: 时间窗口不能为零值")
	}
	if !lastMonthStart.Before(thisMonthStart) {
		return 0, 0, errors.New("dashboard repository: lastMonthStart 必须早于 thisMonthStart")
	}

	const query = `
		SELECT COUNT(*), COUNT(*) FILTER (
			WHERE (sla_response_deadline IS NULL OR first_response_at <= sla_response_deadline)
			AND (sla_resolution_deadline IS NULL OR resolved_at <= sla_resolution_deadline)
		)
		FROM tickets
		WHERE tenant_id = $1 AND deleted_at IS NULL AND created_at >= $2 AND created_at < $3
	`
	if _, err = database.WithTenantSQL(ctx, r.db, tenantID, func(q database.SQLExecutor) (struct{}, error) {
		return struct{}{}, q.QueryRowContext(ctx, query, tenantID, lastMonthStart, thisMonthStart).Scan(&totalSLATickets, &metSLATickets)
	}); err != nil {
		return 0, 0, err
	}
	return totalSLATickets, metSLATickets, nil
}
