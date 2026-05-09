package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// EndpointACL holds the schema definition for the EndpointACL entity.
// This replaces the hardcoded ResourceActionMap for dynamic URL-to-permission mapping.
type EndpointACL struct {
	ent.Schema
}

// Fields of the EndpointACL.
func (EndpointACL) Fields() []ent.Field {
	return []ent.Field{
		field.Int("tenant_id").
			Comment("租户ID").
			Positive(),
		field.String("path_pattern").
			Comment("URL路径模式，支持通配符").
			NotEmpty(),
		field.String("method").
			Comment("HTTP方法：GET, POST, PUT, DELETE, null=ALL").
			Optional(),
		field.String("resource").
			Comment("资源类型").
			NotEmpty(),
		field.String("action").
			Comment("操作类型：read, write, delete").
			NotEmpty(),
		field.String("description").
			Comment("描述").
			Optional(),
		field.Int("priority").
			Comment("优先级").
			Default(100),
		field.Bool("is_active").
			Comment("是否启用").
			Default(true),
		field.Bool("is_whitelist").
			Comment("是否白名单（无需权限检查）").
			Default(false),
		field.Time("created_at").
			Comment("创建时间").
			Default(time.Now),
		field.Time("updated_at").
			Comment("更新时间").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the EndpointACL.
func (EndpointACL) Edges() []ent.Edge {
	return nil
}

// Index of EndpointACL.
func (EndpointACL) Index() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "path_pattern", "method").Unique(),
		index.Fields("tenant_id", "priority"),
	}
}
