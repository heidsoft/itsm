package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// CloudResource holds the schema definition for the CloudResource entity.
type CloudResource struct {
	ent.Schema
}

// Fields of the CloudResource.
func (CloudResource) Fields() []ent.Field {
	return []ent.Field{
		field.Int("cloud_account_id").
			Comment("云账号ID").
			Positive(),
		field.Int("service_id").
			Comment("云服务类型ID").
			Positive(),
		field.String("resource_id").
			Comment("云资源唯一ID").
			NotEmpty(),
		field.String("resource_name").
			Comment("云资源名称").
			Optional(),
		field.String("region").
			Comment("Region").
			Optional(),
		field.String("zone").
			Comment("Zone").
			Optional(),
		field.String("status").
			Comment("资源状态").
			Optional(),
		field.JSON("tags", map[string]string{}).
			Comment("资源标签").
			Optional(),
		field.JSON("metadata", map[string]interface{}{}).
			Comment("资源元数据").
			Optional(),
		field.Time("first_seen_at").
			Comment("首次发现时间").
			Optional(),
		field.Time("last_seen_at").
			Comment("最近发现时间").
			Optional(),
		field.String("lifecycle_state").
			Comment("生命周期状态").
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

// Edges of the CloudResource.
func (CloudResource) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("account", CloudAccount.Type).
			Ref("resources").
			Unique().
			Field("cloud_account_id").
			Required(),
		edge.From("service", CloudService.Type).
			Ref("resources").
			Unique().
			Field("service_id").
			Required(),
		edge.To("cis", ConfigurationItem.Type),
	}
}

// Indexes of the CloudResource.
func (CloudResource) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id"),
		index.Fields("cloud_account_id", "resource_id").
			Unique(),
		index.Fields("service_id"),
		index.Fields("region"),
	}
}
