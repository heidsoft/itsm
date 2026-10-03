package change

import (
	"context"
	"errors"
	"sort"

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

// countsByType 是 GroupBy(type) Scan 的目标结构。
type countsByType struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

// changeTypeOrder 是类型分布的固定展示顺序，与 dto.ChangeType 的取值一致。
// ent 的 type 是无约束字符串字段，因此清单外的历史值按字典序追加而不是丢弃，
// 保证报表总数与 total 可对账，且输出顺序确定、不会产生抖动。
var changeTypeOrder = []string{"standard", "normal", "emergency"}

// orderChangeTypes 把 GROUP BY type 的原始行折叠成固定顺序的 ByType。
func orderChangeTypes(rows []countsByType) []TypeCount {
	counts := make(map[string]int, len(rows))
	for _, row := range rows {
		counts[row.Type] += row.Count
	}

	ordered := make([]TypeCount, 0, len(counts))
	for _, typ := range changeTypeOrder {
		if count, ok := counts[typ]; ok {
			ordered = append(ordered, TypeCount{Type: typ, Count: count})
			delete(counts, typ)
		}
	}
	unknown := make([]string, 0, len(counts))
	for typ := range counts {
		unknown = append(unknown, typ)
	}
	sort.Strings(unknown)
	for _, typ := range unknown {
		ordered = append(ordered, TypeCount{Type: typ, Count: counts[typ]})
	}
	return ordered
}

// GetStats 用两次 GROUP BY 返回 tenant 下的状态计数与类型计数，
// 再在 Go 侧把 status 折叠到 canonical 的 Stats 字段。
//
// 折叠规则：
//   - pending / pending_review / submitted 折叠到 Pending
//     （pending_review 是 seed 别名；submitted 是枚举化之前写入过的历史值，
//     按 service/change_service.go 的读规则只允许读、不允许再写，
//     因此统计口径必须与"按待审批过滤"一致，否则 total 与状态计数无法对账）
//   - 其余状态一一映射
//   - 未出现状态保持 0
//   - ByType 只包含真实存在的类型，按 changeTypeOrder 排序
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
		case "pending", "pending_review", "submitted":
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
		case "closed":
			stats.Closed = b.Count
		}
	}

	typeRows := []countsByType{}
	if err := r.client.Change.Query().
		Where(change.TenantIDEQ(tenantID)).
		GroupBy(change.FieldType).
		Aggregate(ent.Count()).
		Scan(ctx, &typeRows); err != nil {
		return nil, err
	}
	stats.ByType = orderChangeTypes(typeRows)
	return stats, nil
}
