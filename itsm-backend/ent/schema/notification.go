package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Notification holds the schema definition for the Notification entity.
type Notification struct {
	ent.Schema
}

// Fields of the Notification.
func (Notification) Fields() []ent.Field {
	return []ent.Field{
		field.String("title").
			Comment("通知标题").
			NotEmpty(),
		field.Text("message").
			Comment("通知内容").
			NotEmpty(),
		field.String("type").
			Comment("通知类型: info, success, warning, error").
			Default("info"),
		field.Bool("read").
			Comment("是否已读").
			Default(false),
		field.String("action_url").
			Comment("操作链接").
			Optional(),
		field.String("action_text").
			Comment("操作文本").
			Optional(),
		field.Int("user_id").
			Comment("用户ID").
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

// Edges of the Notification.
func (Notification) Edges() []ent.Edge {
	return nil
}
