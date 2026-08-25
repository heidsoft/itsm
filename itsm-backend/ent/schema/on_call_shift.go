package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type OnCallShift struct{ ent.Schema }

func (OnCallShift) Fields() []ent.Field {
	return []ent.Field{
		field.Int("tenant_id").Positive(),
		field.Int("schedule_id").Positive(),
		field.Int("user_id").Positive(),
		field.Time("start_at"),
		field.Time("end_at"),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (OnCallShift) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("schedule", OnCallSchedule.Type).Ref("shifts").Field("schedule_id").Unique().Required(),
		edge.From("user", User.Type).Ref("on_call_shifts").Field("user_id").Unique().Required(),
	}
}

func (OnCallShift) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "schedule_id", "start_at", "end_at"),
		index.Fields("tenant_id", "user_id", "start_at"),
	}
}
