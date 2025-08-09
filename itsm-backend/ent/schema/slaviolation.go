package schema

import (
	"time"

	"entgo.io/ent"
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
		field.Int("tenant_id").Comment("租户ID").Positive(),
		field.Time("created_at").Comment("创建时间").Default(time.Now),
		field.Time("updated_at").Comment("更新时间").Default(time.Now).UpdateDefault(time.Now),
	}
}
func (SLAViolation) Edges() []ent.Edge { return nil }
