package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type SLAViolation struct{ ent.Schema }

func (SLAViolation) Fields() []ent.Field {
	return []ent.Field{
		field.Int("sla_definition_id").Comment("SLA定义ID").Positive(),
		field.Int("ticket_id").Comment("工单ID").Positive(),
		field.String("violation_type").Comment("违规类型").NotEmpty(),
		field.Time("violation_time").Comment("违规时间").Default(time.Now),
		field.Text("description").Comment("违规描述").Optional(),
		field.String("severity").Comment("严重程度").Default("medium"),
		field.Bool("is_resolved").Comment("是否已解决").Default(false),
		field.Time("resolved_at").Comment("解决时间").Optional(),
		field.Text("resolution_notes").Comment("解决说明").Optional(),
		field.Int("tenant_id").Comment("租户ID").Positive(),
		field.Time("created_at").Comment("创建时间").Default(time.Now),
		field.Time("updated_at").Comment("更新时间").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (SLAViolation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("sla_definition", SLADefinition.Type).
			Ref("violations").
			Field("sla_definition_id").
			Unique().
			Required().
			Comment("SLA定义"),
		edge.From("ticket", Ticket.Type).
			Ref("sla_violations").
			Field("ticket_id").
			Unique().
			Required().
			Comment("工单"),
	}
}
