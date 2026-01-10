package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// DiscoveryJob holds the schema definition for the DiscoveryJob entity.
type DiscoveryJob struct {
	ent.Schema
}

// Fields of the DiscoveryJob.
func (DiscoveryJob) Fields() []ent.Field {
	return []ent.Field{
		field.String("source_id").
			Comment("发现源ID").
			NotEmpty(),
		field.String("status").
			Comment("任务状态（pending/running/success/failed）").
			Default("pending"),
		field.Time("started_at").
			Comment("开始时间").
			Optional(),
		field.Time("finished_at").
			Comment("结束时间").
			Optional(),
		field.JSON("summary", map[string]interface{}{}).
			Comment("任务摘要").
			Optional(),
		field.Int("tenant_id").
			Comment("租户ID").
			Positive(),
		field.Time("created_at").
			Comment("创建时间").
			Default(time.Now),
		field.Time("updated_at").
			Comment("更新时间").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the DiscoveryJob.
func (DiscoveryJob) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("source", DiscoverySource.Type).
			Ref("jobs").
			Unique().
			Field("source_id").
			Required(),
		edge.To("results", DiscoveryResult.Type),
	}
}

// Indexes of the DiscoveryJob.
func (DiscoveryJob) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id"),
		index.Fields("source_id"),
		index.Fields("status"),
	}
}
