package mixin

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
)

// SLATracking 提供 SLA 绑定与截止时间字段。
// 接入 Schema：Ticket, Incident
//
// 当前仅 Ticket 和 Incident 有 SLA 绑定；其他域后续按需接入。
type SLATracking struct {
	mixin.Schema
}

func (SLATracking) Fields() []ent.Field {
	return []ent.Field{
		field.Int("sla_definition_id").
			Comment("SLA定义ID").
			Optional(),
		field.Time("sla_response_deadline").
			Comment("SLA响应截止时间").
			Optional(),
		field.Time("sla_resolution_deadline").
			Comment("SLA解决截止时间").
			Optional(),
	}
}
