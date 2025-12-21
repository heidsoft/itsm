package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// TicketView holds the schema definition for the TicketView entity.
type TicketView struct {
	ent.Schema
}

// Fields of the TicketView.
func (TicketView) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("视图名称").
			NotEmpty(),
		field.Text("description").
			Comment("视图描述").
			Optional(),
		field.JSON("filters", map[string]interface{}{}).
			Comment("筛选条件（JSON格式）").
			Optional(),
		field.JSON("columns", []string{}).
			Comment("显示的列（JSON数组）").
			Optional(),
		field.JSON("sort_config", map[string]interface{}{}).
			Comment("排序配置（JSON格式）").
			Optional(),
		field.JSON("group_config", map[string]interface{}{}).
			Comment("分组配置（JSON格式）").
			Optional(),
		field.Bool("is_shared").
			Comment("是否共享").
			Default(false),
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

// Edges of the TicketView.
func (TicketView) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("creator", User.Type).
			Unique().
			Required().
			Field("created_by").
			Comment("创建人"),
	}
}
