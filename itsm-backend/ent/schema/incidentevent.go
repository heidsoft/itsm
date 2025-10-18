package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type IncidentEvent struct{ ent.Schema }

func (IncidentEvent) Fields() []ent.Field {
	return []ent.Field{
		field.Int("incident_id").Comment("事件ID").Positive(),
		field.String("event_type").Comment("事件类型").NotEmpty(),
		field.String("event_name").Comment("事件名称").NotEmpty(),
		field.Text("description").Comment("事件描述").Optional(),
		field.String("status").Comment("状态").Default("active"),
		field.String("severity").Comment("严重程度").Default("medium"),
		field.JSON("data", map[string]interface{}{}).Comment("事件数据").Optional(),
		field.Time("occurred_at").Comment("发生时间").Default(time.Now),
		field.Int("user_id").Comment("操作用户ID").Optional(),
		field.String("source").Comment("事件来源").Default("system"),
		field.JSON("metadata", map[string]interface{}{}).Comment("元数据").Optional(),
		field.Int("tenant_id").Comment("租户ID").Positive(),
		field.Time("created_at").Comment("创建时间").Default(time.Now),
		field.Time("updated_at").Comment("更新时间").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (IncidentEvent) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("incident", Incident.Type).
			Ref("incident_events").
			Field("incident_id").
			Unique().
			Required().
			Comment("事件"),
	}
}
