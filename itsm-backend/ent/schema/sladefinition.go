package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type SLADefinition struct{ ent.Schema }

func (SLADefinition) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").Comment("SLA名称").NotEmpty(),
		field.Text("description").Comment("SLA描述").Optional(),
		field.Int("tenant_id").Comment("租户ID").Positive(),
		field.Time("created_at").Comment("创建时间").Default(time.Now),
		field.Time("updated_at").Comment("更新时间").Default(time.Now).UpdateDefault(time.Now),
	}
}
func (SLADefinition) Edges() []ent.Edge { return nil }
