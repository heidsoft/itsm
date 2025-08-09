package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Tenant holds the schema definition for the Tenant entity.
type Tenant struct {
	ent.Schema
}

// Fields of the Tenant.
func (Tenant) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("租户名称").
			NotEmpty(),
		field.String("code").
			Comment("租户代码").
			Unique().
			NotEmpty(),
		field.String("domain").
			Comment("域名").
			Optional(),
		field.String("type").
			Comment("租户类型").
			Default("standard"),
		field.String("status").
			Comment("状态").
			Default("active"),
		field.Time("expires_at").
			Comment("过期时间").
			Optional(),
		field.Time("created_at").
			Comment("创建时间").
			Default(time.Now),
		field.Time("updated_at").
			Comment("更新时间").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the Tenant.
func (Tenant) Edges() []ent.Edge {
	return nil
}
