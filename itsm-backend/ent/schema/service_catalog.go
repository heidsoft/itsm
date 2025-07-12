package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

// ServiceCatalogStatus defines the status enum for service catalog
type ServiceCatalogStatus string

const (
	ServiceCatalogStatusEnabled  ServiceCatalogStatus = "enabled"  // 启用
	ServiceCatalogStatusDisabled ServiceCatalogStatus = "disabled" // 禁用
)

// ServiceCatalog holds the schema definition for the ServiceCatalog entity.
type ServiceCatalog struct {
	ent.Schema
}

// Fields of the ServiceCatalog.
func (ServiceCatalog) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty().
			MaxLen(255).
			Comment("服务名称"),
		field.String("category").
			NotEmpty().
			MaxLen(100).
			Comment("服务分类"),
		field.Text("description").
			Optional().
			Comment("服务描述"),
		field.String("delivery_time").
			NotEmpty().
			MaxLen(50).
			Comment("交付时间"),
		field.Enum("status").
			Values(
				string(ServiceCatalogStatusEnabled),
				string(ServiceCatalogStatusDisabled),
			).
			Default(string(ServiceCatalogStatusEnabled)).
			Comment("服务状态"),
		field.Int("tenant_id").
			Positive().
			Comment("租户ID"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("创建时间"),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			Comment("更新时间"),
	}
}

// Edges of the ServiceCatalog.
func (ServiceCatalog) Edges() []ent.Edge {
	return []ent.Edge{
		// 租户关联
		edge.From("tenant", Tenant.Type).
			Ref("service_catalogs").
			Field("tenant_id").
			Required().
			Unique(),
		// 服务请求
		edge.To("service_requests", ServiceRequest.Type),
	}
}

// Indexes of the ServiceCatalog.
func (ServiceCatalog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("category"),
		index.Fields("status"),
		index.Fields("created_at"),
		// 租户相关索引
		index.Fields("tenant_id"),
		index.Fields("tenant_id", "category"),
		// 复合索引
		index.Fields("category", "status"),
	}
}
