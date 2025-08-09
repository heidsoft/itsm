package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type ServiceRequest struct{ ent.Schema }

func (ServiceRequest) Fields() []ent.Field {
	return []ent.Field{
		field.Int("catalog_id").Comment("服务目录ID").Positive(),
		field.Int("requester_id").Comment("申请人ID").Positive(),
		field.String("status").Comment("状态").Default("pending"),
		field.Text("reason").Comment("申请原因").Optional(),
		field.Time("created_at").Comment("创建时间").Default(time.Now),
		field.Time("updated_at").Comment("更新时间").Default(time.Now).UpdateDefault(time.Now),
	}
}
func (ServiceRequest) Edges() []ent.Edge { return nil }
