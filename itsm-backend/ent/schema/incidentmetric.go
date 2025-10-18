package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type IncidentMetric struct{ ent.Schema }

func (IncidentMetric) Fields() []ent.Field {
	return []ent.Field{
		field.Int("incident_id").Comment("事件ID").Positive(),
		field.String("metric_type").Comment("指标类型").NotEmpty(),
		field.String("metric_name").Comment("指标名称").NotEmpty(),
		field.Float("metric_value").Comment("指标值"),
		field.String("unit").Comment("单位").Optional(),
		field.Time("measured_at").Comment("测量时间").Default(time.Now),
		field.JSON("tags", map[string]string{}).Comment("标签").Optional(),
		field.JSON("metadata", map[string]interface{}{}).Comment("元数据").Optional(),
		field.Int("tenant_id").Comment("租户ID").Positive(),
		field.Time("created_at").Comment("创建时间").Default(time.Now),
		field.Time("updated_at").Comment("更新时间").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (IncidentMetric) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("incident", Incident.Type).
			Ref("incident_metrics").
			Field("incident_id").
			Unique().
			Required().
			Comment("事件"),
	}
}
