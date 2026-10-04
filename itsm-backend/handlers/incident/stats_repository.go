package incident

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"

	"itsm-backend/common"
)

// incidentStatsRepository 负责 incidents 表的租户全量标量聚合。
//
// 说明：这里的聚合依赖 PostgreSQL 专有语法：
//   - COUNT(*) FILTER (WHERE ...) — 聚合过滤
//   - EXTRACT(EPOCH FROM (resolved_at - created_at)) — 时间差转秒
//
// 这些语法没有 Ent 等价的「跨方言」表达，因此封装到本文件：
//  1. 把 SQL 字符串集中到一处，未来加方言兼容只需改这里；
//  2. 强制 tenant_id 谓词、deleted_at 守卫（与 incident schema 软删语义一致）；
//  3. 调用方只看到面向对象方法 GetStats(ctx, tenantID)。
//
// 报表读模型（窗口分布、每日趋势、窗口平均时长）不在本文件，见 EntRepository.GetReport：
// 那部分刻意用 Ent 表达，以便在 SQLite 测试库上验证口径。
type incidentStatsRepository struct {
	db *sql.DB
}

func newIncidentStatsRepository(db *sql.DB) *incidentStatsRepository {
	if db == nil {
		return nil
	}
	return &incidentStatsRepository{db: db}
}

// 事件状态词表（status 是无约束字符串列，只能靠常量对齐，改动必须同步
// common.IncidentStatus*、ent schema 注释与前端 constants/incident.ts）。
//
// incidentOpenStatuses 是「未终结」集合。SQL 曾经写 status IN ('open','in_progress')，
// 而 'open' 从来不是事件域的合法取值（新建事件落库是 'new'），于是 new/acknowledged/
// assigned/triaged/escalated/on_hold 全部漏计，openIncidents 长期接近 0。
// incidentResolvedStatuses 是「已解决+已关闭」集合，与 Resolve/Close 迁移目标一致。
var (
	incidentOpenStatuses = []string{
		common.IncidentStatusNew,
		common.IncidentStatusAcknowledged,
		common.IncidentStatusAssigned,
		common.IncidentStatusTriaged,
		common.IncidentStatusInProgress,
		common.IncidentStatusEscalated,
		common.IncidentStatusOnHold,
	}
	incidentResolvedStatuses = []string{
		common.IncidentStatusResolved,
		common.IncidentStatusClosed,
	}
)

// IsOpenIncidentStatus 判断状态是否属于「未终结」集合。生产聚合走 SQL，
// Go 侧折叠（测试替身、调用方自检）走这里，保证词表只有一个维护点。
func IsOpenIncidentStatus(status string) bool {
	return slices.Contains(incidentOpenStatuses, status)
}

// IsResolvedIncidentStatus 判断状态是否属于「已解决/已关闭」集合。
func IsResolvedIncidentStatus(status string) bool {
	return slices.Contains(incidentResolvedStatuses, status)
}

// buildStatsQuery 组装租户隔离的标量聚合 SQL 与绑定参数。
//
// 状态取值一律走占位符绑定，不把词表拼进 SQL 文本：词表只有一个维护点，
// 也避免任何取值里的引号影响语句结构。优先级保持字面量，因为
// ent/schema/incident.go 对 priority 有枚举校验，取值不可能漂移。
func buildStatsQuery(tenantID int) (string, []any) {
	args := []any{tenantID}
	ordinal := 2
	inClause := func(values []string) string {
		parts := make([]string, 0, len(values))
		for _, value := range values {
			parts = append(parts, fmt.Sprintf("$%d", ordinal))
			args = append(args, value)
			ordinal++
		}
		return strings.Join(parts, ", ")
	}

	openIn := inClause(incidentOpenStatuses)
	resolvedIn := inClause(incidentResolvedStatuses)
	query := fmt.Sprintf(`
		SELECT
		  COUNT(*) FILTER (WHERE TRUE) AS total,
		  COUNT(*) FILTER (WHERE status IN (%s)) AS open,
		  COUNT(*) FILTER (WHERE priority = 'critical') AS critical,
		  COUNT(*) FILTER (WHERE priority = 'high') AS major,
		  COUNT(*) FILTER (WHERE status IN (%s)) AS resolved,
		  COALESCE(AVG(EXTRACT(EPOCH FROM (resolved_at - created_at)) / 60.0)
		           FILTER (WHERE status IN (%s) AND resolved_at IS NOT NULL),
		           0)::int AS avg_minutes
		FROM incidents
		WHERE tenant_id = $1 AND deleted_at IS NULL
	`, openIn, resolvedIn, resolvedIn)
	return query, args
}

// GetStats 一次性返回 IncidentStats 所需的全部聚合指标。
//   - total/open/critical/major/resolved: COUNT(*) FILTER 一次完成
//   - avgResolutionTime: 已解决/已关闭事件的平均 (resolved_at - created_at) 分钟数；
//     未解决事件排除；空集通过 COALESCE 返回 0 而非 NULL。
func (r *incidentStatsRepository) GetStats(ctx context.Context, tenantID int) (*IncidentStats, error) {
	if tenantID <= 0 {
		return nil, fmt.Errorf("incident stats repository requires tenantID")
	}
	query, args := buildStatsQuery(tenantID)

	stats := &IncidentStats{}
	row := r.db.QueryRowContext(ctx, query, args...)
	if err := row.Scan(
		&stats.TotalIncidents,
		&stats.OpenIncidents,
		&stats.CriticalIncidents,
		&stats.MajorIncidents,
		&stats.ResolvedIncidents,
		&stats.AvgResolutionTime,
	); err != nil {
		return nil, fmt.Errorf("scan incident stats: %w", err)
	}
	return stats, nil
}
