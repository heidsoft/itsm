package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type CIRelationship struct{ ent.Schema }

func (CIRelationship) Fields() []ent.Field {
	return []ent.Field{
		field.Int("source_ci_id").Comment("源CI ID").Positive(),
		field.Int("target_ci_id").Comment("目标CI ID").Positive(),
		field.Int("relationship_type_id").Comment("关系类型ID").Positive(),
		field.Text("description").Comment("关系描述").Optional(),
		field.Int("tenant_id").Comment("租户ID").Positive(),
		field.Time("created_at").Comment("创建时间").Default(time.Now),
		field.Time("updated_at").Comment("更新时间").Default(time.Now).UpdateDefault(time.Now),
	}
}
func (CIRelationship) Edges() []ent.Edge { return nil }
