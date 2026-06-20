package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// DomainConfig holds the schema definition for the DomainConfig entity.
// This entity supports hierarchical configuration with inheritance.
type DomainConfig struct {
	ent.Schema
}

// Fields of the DomainConfig.
func (DomainConfig) Fields() []ent.Field {
	return []ent.Field{
		field.String("config_key").
			Comment("配置键").
			NotEmpty(),

		field.String("config_type").
			Comment("配置类型: process_binding, approval_workflow, sla_rule, notification").
			NotEmpty(),

		field.JSON("config_value", map[string]interface{}{}).
			Comment("配置值JSON").
			Default(map[string]interface{}{}),

		field.String("inherit_mode").
			Comment("继承模式: inherit, override, extend").
			Default("inherit"),

		field.Int("tenant_id").
			Comment("租户ID").
			NonNegative(),

		field.Int("department_id").
			Comment("部门ID (0=租户级配置)").
			Default(0),

		field.Int("team_id").
			Comment("团队ID (0=部门级配置)").
			Default(0),

		field.Int("parent_config_id").
			Comment("父配置ID").
			Optional(),

		field.Int("version").
			Comment("配置版本").
			Default(1),

		field.Bool("is_active").
			Comment("是否启用").
			Default(true),

		field.String("description").
			Comment("配置描述").
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

// Edges of the DomainConfig.
func (DomainConfig) Edges() []ent.Edge {
	return nil
}

// Indexes of the DomainConfig.
func (DomainConfig) Indexes() []ent.Index {
	return []ent.Index{
		// Unique constraint for config key within scope
		index.Fields("tenant_id", "config_type", "config_key", "department_id", "team_id").
			Unique(),
		// Query index
		index.Fields("tenant_id", "config_type", "config_key"),
		// Department scope index
		index.Fields("tenant_id", "department_id", "team_id"),
	}
}
