package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type IncidentAlert struct{ ent.Schema }

func (IncidentAlert) Fields() []ent.Field {
	return []ent.Field{
		field.Int("incident_id").Comment("事件ID").Positive(),
		field.String("alert_type").Comment("告警类型").NotEmpty(),
		field.String("alert_name").Comment("告警名称").NotEmpty(),
		field.Text("message").Comment("告警消息").NotEmpty(),
		field.String("severity").Comment("严重程度").Default("medium"),
		field.String("status").Comment("状态").Default("active"),
		field.JSON("channels", []string{}).Comment("通知渠道").Optional(),
		field.JSON("recipients", []string{}).Comment("接收人").Optional(),
		field.Time("triggered_at").Comment("触发时间").Default(time.Now),
		field.Time("acknowledged_at").Comment("确认时间").Optional(),
		field.Time("resolved_at").Comment("解决时间").Optional(),
		field.Int("acknowledged_by").Comment("确认人ID").Optional(),
		field.JSON("metadata", map[string]interface{}{}).Comment("元数据").Optional(),
		field.Int("tenant_id").Comment("租户ID").Positive(),
		field.Time("created_at").Comment("创建时间").Default(time.Now),
		field.Time("updated_at").Comment("更新时间").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (IncidentAlert) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("incident", Incident.Type).
			Ref("incident_alerts").
			Field("incident_id").
			Unique().
			Required().
			Comment("事件"),
	}
}
