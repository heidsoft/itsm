package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// CloudService holds the schema definition for the CloudService entity.
type CloudService struct {
	ent.Schema
}

// Fields of the CloudService.
func (CloudService) Fields() []ent.Field {
	return []ent.Field{
		field.Int("parent_id").
			Comment("父级服务ID").
			Optional(),
		field.String("provider").
			Comment("云厂商标识（aliyun/huawei/tencent/azure/onprem）").
			NotEmpty(),
		field.String("category").
			Comment("服务分类").
			Optional(),
		field.String("service_code").
			Comment("云服务代码，例如 ecs/rds/oss").
			NotEmpty(),
		field.String("service_name").
			Comment("云服务名称").
			NotEmpty(),
		field.String("resource_type_code").
			Comment("资源类型代码，例如 instance/volume/vpc").
			NotEmpty(),
		field.String("resource_type_name").
			Comment("资源类型名称").
			NotEmpty(),
		field.String("api_version").
			Comment("API版本").
			Optional(),
		field.JSON("attribute_schema", map[string]interface{}{}).
			Comment("动态属性Schema").
			Optional(),
		field.Bool("is_system").
			Comment("是否系统预置").
			Default(false),
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

// Edges of the CloudService.
func (CloudService) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("parent", CloudService.Type).
			Ref("children").
			Unique().
			Field("parent_id"),
		edge.To("children", CloudService.Type),
		edge.To("resources", CloudResource.Type),
	}
}

// Indexes of the CloudService.
func (CloudService) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id"),
		index.Fields("parent_id"),
		index.Fields("category"),
		index.Fields("provider", "service_code", "resource_type_code").
			Unique(),
	}
}
