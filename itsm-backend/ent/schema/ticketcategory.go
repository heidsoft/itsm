package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/edge"
)

// TicketCategory holds the schema definition for the TicketCategory entity.
type TicketCategory struct {
	ent.Schema
}

// Fields of the TicketCategory.
func (TicketCategory) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("分类名称").
			NotEmpty(),
		field.Text("description").
			Comment("分类描述").
			Optional(),
		field.String("code").
			Comment("分类代码").
			Unique().
			NotEmpty(),
		field.Int("parent_id").
			Comment("父分类ID").
			Optional(),
		field.Int("level").
			Comment("分类层级").
			Default(1),
		field.Int("sort_order").
			Comment("排序顺序").
			Default(0),
		field.Bool("is_active").
			Comment("是否启用").
			Default(true),
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

// Edges of the TicketCategory.
func (TicketCategory) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("tickets", Ticket.Type).
			Comment("此分类下的工单"),
		edge.To("children", TicketCategory.Type).
			Comment("子分类"),
		edge.From("parent", TicketCategory.Type).
			Ref("children").
			Field("parent_id").
			Unique().
			Comment("父分类"),
	}
}
