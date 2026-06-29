package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ConfigurationItemHistory holds the schema definition for the ConfigurationItemHistory entity.
type ConfigurationItemHistory struct {
	ent.Schema
}

// Fields of the ConfigurationItemHistory.
func (ConfigurationItemHistory) Fields() []ent.Field {
	return []ent.Field{
		field.Int("ci_id").
			Comment("CI ID").
			Positive(),
		field.Int("version").
			Comment("版本号").
			Positive(),
		field.String("operation").
			Comment("操作类型: create/update/delete").
			NotEmpty(),
		field.JSON("before", map[string]interface{}{}).
			Comment("变更前数据").
			Optional(),
		field.JSON("after", map[string]interface{}{}).
			Comment("变更后数据").
			Optional(),
		field.JSON("changed_fields", []string{}).
			Comment("变更的字段列表").
			Optional(),
		field.Int("operator_id").
			Comment("操作人ID").
			Positive(),
		field.String("operator_name").
			Comment("操作人姓名").
			Optional(),
		field.String("remark").
			Comment("变更备注").
			Optional(),
		field.Int("tenant_id").
			Comment("租户ID").
			Positive(),
		field.Time("created_at").
			Comment("创建时间").
			Default(time.Now),
	}
}

// Edges of the ConfigurationItemHistory.
func (ConfigurationItemHistory) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("ci", ConfigurationItem.Type).
			Ref("history").
			Unique().
			Field("ci_id").
			Required(),
	}
}

// Indexes of the ConfigurationItemHistory.
func (ConfigurationItemHistory) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("ci_id", "version").
			Unique(),
		index.Fields("tenant_id"),
		index.Fields("operation"),
		index.Fields("operator_id"),
		index.Fields("created_at"),
	}
}
