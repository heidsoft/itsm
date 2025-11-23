package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Department holds the schema definition for the Department entity.
type Department struct {
	ent.Schema
}

// Fields of the Department.
func (Department) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("部门名称").
			NotEmpty(),
		field.String("code").
			Comment("部门代码").
			Unique().
			NotEmpty(),
		field.Text("description").
			Comment("部门描述").
			Optional(),
		field.Int("manager_id").
			Comment("部门经理ID").
			Optional(),
		field.Int("parent_id").
			Comment("父部门ID").
			Optional(),
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

// Edges of the Department.
func (Department) Edges() []ent.Edge {
	return []ent.Edge{
		// 树形结构关系
		edge.To("children", Department.Type).
			From("parent").
			Field("parent_id").
			Unique(),

		edge.To("users", User.Type).
			Comment("部门成员"),
		edge.To("tickets", Ticket.Type).
			Comment("部门工单"),
		edge.To("workflows", Workflow.Type).
			Comment("部门工作流"),
		edge.To("categories", TicketCategory.Type).
			Comment("部门工单分类"),
		edge.To("projects", Project.Type).
			Comment("部门项目"),
		edge.To("tags", Tag.Type).
			Comment("部门标签"),
	}
}
