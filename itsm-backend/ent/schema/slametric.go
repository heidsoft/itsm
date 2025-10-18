package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type SLAMetric struct{ ent.Schema }

func (SLAMetric) Fields() []ent.Field {
	return []ent.Field{
		field.Int("sla_definition_id").Comment("SLA定义ID").Positive(),
		field.String("metric_type").Comment("指标类型").NotEmpty(),
		field.String("metric_name").Comment("指标名称").NotEmpty(),
		field.Float("metric_value").Comment("指标值"),
		field.String("unit").Comment("单位").Optional(),
		field.Time("measurement_time").Comment("测量时间").Default(time.Now),
		field.JSON("metadata", map[string]interface{}{}).Comment("元数据").Optional(),
		field.Int("tenant_id").Comment("租户ID").Positive(),
		field.Time("created_at").Comment("创建时间").Default(time.Now),
		field.Time("updated_at").Comment("更新时间").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (SLAMetric) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("sla_definition", SLADefinition.Type).
			Ref("metrics").
			Field("sla_definition_id").
			Unique().
			Required().
			Comment("SLA定义"),
	}
}
