package mixin

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
)

// TenantTimestamps 提供乐观锁和审计时间戳字段。
// tenant_id 和 created_at 保留在各 Schema 的 Fields() 中，
// 因为 Ent v0.14.6 不支持 Mixin 字段出现在 Indexes() 定义中。
// 接入 Schema：SLAState
type TenantTimestamps struct {
	mixin.Schema
}

func (TenantTimestamps) Fields() []ent.Field {
	return []ent.Field{
		field.Int("version").
			Comment("版本号（乐观锁）").
			Default(1).
			Positive(),
		field.Time("updated_at").
			Comment("更新时间").
			Default(time.Now).
			UpdateDefault(time.Now),
		field.Time("deleted_at").
			Comment("软删除时间").
			Optional().
			Nillable(),
	}
}
