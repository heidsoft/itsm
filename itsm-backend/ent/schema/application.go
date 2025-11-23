package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Application holds the schema definition for the Application entity.
type Application struct {
	ent.Schema
}

// Fields of the Application.
func (Application) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("应用名称").
			NotEmpty(),
		field.String("code").
			Comment("应用代码").
			Unique().
			NotEmpty(),
		field.Text("description").
			Comment("应用描述").
			Optional(),
		field.String("type").
			Comment("类型: web, mobile, backend, desktop").
			Default("web"),
		field.String("status").
			Comment("状态: active, inactive, deprecated").
			Default("active"),
		field.Int("owner_id").
			Comment("负责人ID").
			Optional(),
		field.Int("project_id").
			Comment("所属项目ID").
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

// Edges of the Application.
func (Application) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).
			Ref("applications").
			Field("project_id").
			Unique(),
		edge.To("microservices", Microservice.Type).
			Comment("包含的微服务"),
		edge.To("tags", Tag.Type).
			Comment("应用标签"),
	}
}
