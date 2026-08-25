package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type OnCallSchedule struct{ ent.Schema }

func (OnCallSchedule) Fields() []ent.Field {
	return []ent.Field{
		field.Int("tenant_id").Positive(),
		field.Int("group_id").Positive(),
		field.String("name").NotEmpty().MaxLen(160),
		field.String("timezone").Default("Asia/Shanghai").MaxLen(64),
		field.String("status").Default("active").MaxLen(32),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (OnCallSchedule) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("group", Group.Type).Ref("on_call_schedules").Field("group_id").Unique().Required(),
		edge.To("shifts", OnCallShift.Type),
	}
}

func (OnCallSchedule) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "group_id", "name").Unique(),
		index.Fields("tenant_id", "status"),
	}
}
