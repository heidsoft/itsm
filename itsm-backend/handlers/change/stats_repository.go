package change

import (
	"context"
	"errors"

	"itsm-backend/ent"
	"itsm-backend/ent/change"
)

// statsRepository 将变更单状态分布聚合从 EntRepository 拆出来。
// 目的：避免 EntRepository 混入 SQL/聚合两条路径，并按租户隔离的硬性约束
// 把"统计"的查询责任收敛到一处。Ent GroupBy + Count 在 PG 和 SQLite 上都可运行。
type statsRepository struct {
	client *ent.Client
}

func newStatsRepository(client *ent.Client) *statsRepository {
	if client == nil {
		return nil
	}
	return &statsRepository{client: client}
}

// countsByStatus 是 GroupBy Scan 的目标结构：status 列 + Count 列。
type countsByStatus struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

// GetStats 一次性 GROUP BY 返回 tenant 下所有状态计数，
// 再在 Go 侧把 status 折叠到 canonical 的 Stats 字段。
//
// 折叠规则：
//   - pending / pending_review 折叠到 Pending（pending_review 是 seed 别名）
//   - 其余状态一一映射
//   - 未出现状态保持 0
//
// 安全关键：所有 SQL 都绑定 tenantID，未传或 ≤0 直接返回错误而非返回全表。
func (r *statsRepository) GetStats(ctx context.Context, tenantID int) (*Stats, error) {
	if tenantID <= 0 {
		return nil, errors.New("stats repository requires tenantID")
	}
	stats := &Stats{}

	total, err := r.client.Change.Query().
		Where(change.TenantIDEQ(tenantID)).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	stats.Total = total

	rows := []countsByStatus{}
	if err := r.client.Change.Query().
		Where(change.TenantIDEQ(tenantID)).
		GroupBy(change.FieldStatus).
		Aggregate(ent.Count()).
		Scan(ctx, &rows); err != nil {
		return nil, err
	}
	for _, b := range rows {
		switch b.Status {
		case "draft":
			stats.Draft = b.Count
		case "pending":
			stats.Pending += b.Count
		case "pending_review":
			stats.Pending += b.Count
		case "approved":
			stats.Approved = b.Count
		case "scheduled":
			stats.Scheduled = b.Count
		case "in_progress":
			stats.InProgress = b.Count
		case "completed":
			stats.Completed = b.Count
		case "failed":
			stats.Failed = b.Count
		case "rolled_back":
			stats.RolledBack = b.Count
		case "rejected":
			stats.Rejected = b.Count
		case "cancelled":
			stats.Cancelled = b.Count
		}
	}
	return stats, nil
}
