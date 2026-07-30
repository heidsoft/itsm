package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/edge"
)

// BootstrapToken holds the schema definition for the BootstrapToken entity.
type BootstrapToken struct {
	ent.Schema
}

// Fields of the BootstrapToken.
func (BootstrapToken) Fields() []ent.Field {
	return []ent.Field{
		field.String("token_hash").
			Comment("bootstrap token bcrypt哈希").
			NotEmpty(),
		field.Time("expires_at").
			Comment("token过期时间"),
		field.Bool("used").
			Comment("是否已使用").
			Default(false),
		field.Int("used_by").
			Comment("创建的管理员用户ID").
			Optional(),
		field.Time("created_at").
			Comment("创建时间").
			Default(time.Now),
		field.Int("tenant_id").
			Comment("租户ID").
			Positive(),
	}
}

// Edges of the BootstrapToken.
func (BootstrapToken) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("tenant", Tenant.Type).
			Required(),
	}
}
