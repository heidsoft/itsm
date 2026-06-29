package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// ServiceCatalogItem holds the schema definition for the ServiceCatalogItem entity.
// 服务目录项，代表一个可以申请的服务
type ServiceCatalogItem struct {
	ent.Schema
}

// Fields of the ServiceCatalogItem.
func (ServiceCatalogItem) Fields() []ent.Field {
	return []ent.Field{
		field.Int("catalog_id").
			Comment("服务目录ID").
			Positive(),
		field.String("name").
			Comment("服务名称").
			NotEmpty(),
		field.String("description").
			Comment("服务描述").
			Optional(),
		field.Text("details").
			Comment("服务详细说明").
			Optional(),
		field.String("category").
			Comment("服务分类").
			Optional(),
		field.String("icon").
			Comment("图标URL").
			Optional(),
		field.JSON("form_schema", map[string]interface{}{}).
			Comment("表单Schema，用于动态渲染申请表单").
			Optional(),
		field.Int("sla_id").
			Comment("关联的SLA ID").
			Optional(),
		field.Int("approval_chain_id").
			Comment("关联的审批链ID").
			Optional(),
		field.Bool("is_active").
			Comment("是否激活").
			Default(true),
		field.Bool("requires_approval").
			Comment("是否需要审批").
			Default(true),
		field.Int("estimated_days").
			Comment("预计完成天数").
			Default(1),
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

// Edges of the ServiceCatalogItem.
func (ServiceCatalogItem) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("catalog", ServiceCatalog.Type).
			Ref("items").
			Field("catalog_id").
			Required().
			Unique(),
	}
}
