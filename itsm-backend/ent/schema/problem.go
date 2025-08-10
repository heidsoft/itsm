package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Problem holds the schema definition for the Problem entity.
type Problem struct {
	ent.Schema
}

// Fields of the Problem.
func (Problem) Fields() []ent.Field {
	return []ent.Field{
		field.String("title").
			Comment("问题标题").
			NotEmpty(),
		field.Text("description").
			Comment("问题描述").
			Optional(),
		field.String("status").
			Comment("状态").
			Default("open"),
		field.String("priority").
			Comment("优先级").
			Default("medium"),
		field.String("category").
			Comment("问题分类").
			Optional(),
		field.Text("root_cause").
			Comment("根本原因").
			Optional(),
		field.Text("impact").
			Comment("影响范围").
			Optional(),
		field.Int("assignee_id").
			Comment("处理人ID").
			Optional(),
		field.Int("created_by").
			Comment("创建人ID").
			Positive(),
		field.Int("tenant_id").
			Comment("租户ID").
			Positive(),
		field.Time("created_at").
			Comment("创建时间").
			Default(time.Now),
		field.Time("updated_at").
			Comment("更新时间").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the Problem.
func (Problem) Edges() []ent.Edge {
	return nil
}
