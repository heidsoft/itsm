package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// DiscoverySource holds the schema definition for the DiscoverySource entity.
type DiscoverySource struct {
	ent.Schema
}

// Fields of the DiscoverySource.
func (DiscoverySource) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			Comment("来源ID").
			NotEmpty().
			Immutable(),
		field.String("name").
			Comment("来源名称").
			NotEmpty(),
		field.String("source_type").
			Comment("来源类型（agent/api/import/manual）").
			NotEmpty(),
		field.String("provider").
			Comment("云厂商或私有云标识").
			Optional(),
		field.Int("cloud_account_id").
			Comment("租户内云账号记录ID").
			Optional(),
		field.JSON("service_codes", []string{}).
			Comment("发现服务范围").
			Optional(),
		field.JSON("regions", []string{}).
			Comment("发现地域范围").
			Optional(),
		field.String("schedule").
			Comment("调度表达式").
			Optional(),
		field.String("reconcile_policy").
			Comment("对账策略").
			Default("manual"),
		field.Int("stale_threshold").
			Comment("连续缺失多少次后进入退役候选").
			Default(3).
			NonNegative(),
		field.Time("last_success_at").
			Comment("最近一次完整成功发现时间").
			Optional(),
		field.Bool("enabled").
			Comment("是否启用").
			Default(true),
		field.String("description").
			Comment("描述").
			Optional(),
		field.Int("tenant_id").
			Comment("租户ID").
			Positive(), // 必填：存量数据已由 migrations/20260610_cmdb_tenant_id_backfill.sql 回填
		field.Time("created_at").
			Comment("创建时间").
			Default(time.Now),
		field.Time("updated_at").
			Comment("更新时间").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the DiscoverySource.
func (DiscoverySource) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("jobs", DiscoveryJob.Type),
	}
}

// Indexes of the DiscoverySource.
func (DiscoverySource) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id"),
		index.Fields("tenant_id", "cloud_account_id"),
		index.Fields("tenant_id", "name").Unique(),
	}
}
