package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Tag holds the schema definition for the Tag entity.
type Tag struct {
	ent.Schema
}

// Fields of the Tag.
func (Tag) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("标签名称").
			NotEmpty(),
		field.String("code").
			Comment("标签代码").
			Unique().
			NotEmpty(),
		field.Text("description").
			Comment("标签描述").
			Optional(),
		field.String("color").
			Comment("标签颜色").
			Default("#1890ff"),
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

// Edges of the Tag.
func (Tag) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("projects", Project.Type).
			Ref("tags").
			Comment("关联的项目"),
		edge.From("applications", Application.Type).
			Ref("tags").
			Comment("关联的应用"),
		edge.From("microservices", Microservice.Type).
			Ref("tags").
			Comment("关联的微服务"),
		edge.From("departments", Department.Type).
			Ref("tags").
			Comment("关联的部门"),
		edge.From("teams", Team.Type).
			Ref("tags").
			Comment("关联的团队"),
	}
}
