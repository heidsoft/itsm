package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

type ConfigurationItem struct {
	ent.Schema
}

func (ConfigurationItem) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty(),
		field.String("display_name").Optional(),
		field.Text("description").Optional(),
		field.Int("ci_type_id"), // 关联到CI类型
		field.String("serial_number").Optional(),
		field.String("asset_tag").Optional(),
		field.String("status").Default("active"),           // active, inactive, maintenance, retired
		field.String("lifecycle_state").Default("planned"), // 当前生命周期状态
		field.String("business_service").Optional(),
		field.String("owner").Optional(),
		field.String("environment").Optional(),
		field.String("location").Optional(),
		field.JSON("attributes", map[string]interface{}{}).Optional(), // 动态属性
		field.JSON("monitoring_data", map[string]interface{}{}).Optional(),
		field.JSON("discovery_source", map[string]interface{}{}).Optional(), // 发现来源信息
		field.Time("last_discovered").Optional(),
		field.String("version").Default("1.0"), // 版本控制
		field.Int("tenant_id"),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (ConfigurationItem) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("tenant", Tenant.Type).Ref("configuration_items").Field("tenant_id").Unique().Required(),
		edge.From("ci_type", CIType.Type).Ref("configuration_items").Field("ci_type_id").Unique().Required(),
		// 关系管理
		edge.To("outgoing_relationships", CIRelationship.Type),
		edge.To("incoming_relationships", CIRelationship.Type),
		// 生命周期状态
		edge.To("lifecycle_states", CILifecycleState.Type),
		// 变更历史
		edge.To("change_records", CIChangeRecord.Type),
		// 关联工单
		edge.To("incidents", Ticket.Type),
		edge.To("changes", Ticket.Type),
	}
}

func (ConfigurationItem) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id"),
		index.Fields("ci_type_id"),
		index.Fields("status"),
		index.Fields("lifecycle_state"),
		index.Fields("business_service"),
		index.Fields("tenant_id", "name").Unique(),
		index.Fields("tenant_id", "serial_number"),
		index.Fields("tenant_id", "asset_tag"),
	}
}
