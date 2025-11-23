package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Microservice holds the schema definition for the Microservice entity.
type Microservice struct {
	ent.Schema
}

// Fields of the Microservice.
func (Microservice) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("微服务名称").
			NotEmpty(),
		field.String("code").
			Comment("微服务代码").
			Unique().
			NotEmpty(),
		field.Text("description").
			Comment("微服务描述").
			Optional(),
		field.String("language").
			Comment("开发语言").
			Optional(),
		field.String("framework").
			Comment("开发框架").
			Optional(),
		field.String("git_repo").
			Comment("Git仓库地址").
			Optional(),
		field.String("ci_pipeline").
			Comment("CI流水线地址").
			Optional(),
		field.String("status").
			Comment("状态: active, maintenance, deprecated").
			Default("active"),
		field.Int("application_id").
			Comment("所属应用ID").
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

// Edges of the Microservice.
func (Microservice) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("application", Application.Type).
			Ref("microservices").
			Field("application_id").
			Unique(),
		edge.To("tags", Tag.Type).
			Comment("微服务标签"),
	}
}
