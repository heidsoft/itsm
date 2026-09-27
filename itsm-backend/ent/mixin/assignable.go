package mixin

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
)

// Assignable 提供处理人/负责人字段。
// 接入 Schema：Ticket, Incident, Problem, Change
//
// Release 使用 owner_id、ServiceRequest 使用 processor_id，语义不同，不接入。
type Assignable struct {
	mixin.Schema
}

func (Assignable) Fields() []ent.Field {
	return []ent.Field{
		field.Int("assignee_id").
			Comment("处理人ID").
			Optional(),
	}
}
