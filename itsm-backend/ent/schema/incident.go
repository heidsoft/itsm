package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Incident holds the schema definition for the Incident entity.
type Incident struct {
	ent.Schema
}

// Fields of the Incident.
func (Incident) Fields() []ent.Field {
	return []ent.Field{
		field.String("title").
			Comment("事件标题").
			NotEmpty(),
		field.Text("description").
			Comment("事件描述").
			Optional(),
		field.String("status").
			Comment("状态").
			Default("new"),
		field.String("priority").
			Comment("优先级").
			Default("medium"),
		field.String("incident_number").
			Comment("事件编号").
			Unique().
			NotEmpty(),
		field.Int("reporter_id").
			Comment("报告人ID").
			Positive(),
		field.Int("assignee_id").
			Comment("处理人ID").
			Optional(),
		field.Int("configuration_item_id").
			Comment("配置项ID").
			Optional(),
		field.Time("resolved_at").
			Comment("解决时间").
			Optional(),
		field.Time("closed_at").
			Comment("关闭时间").
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

// Edges of the Incident.
func (Incident) Edges() []ent.Edge {
	return nil
}
