package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"
)

// SLADefinition holds the schema definition for the SLADefinition entity.
type SLADefinition struct {
	ent.Schema
}

// Fields of the SLADefinition.
func (SLADefinition) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").Comment("SLA名称"),
		field.String("description").Optional().Comment("SLA描述"),
		field.String("service_type").Comment("服务类型：incident/service_request/problem/change"),
		field.String("priority").Comment("优先级：low/medium/high/critical"),
		field.String("impact").Comment("影响范围：low/medium/high"),
		field.Int("response_time").Comment("响应时间（分钟）"),
		field.Int("resolution_time").Comment("解决时间（分钟）"),
		field.String("business_hours").Comment("工作时间配置JSON"),
		field.String("holidays").Comment("节假日配置JSON"),
		field.Bool("is_active").Default(true).Comment("是否启用"),
		field.Int("tenant_id").Comment("租户ID"),
		field.String("created_by").Comment("创建人"),
		field.Time("created_at").Comment("创建时间"),
		field.Time("updated_at").Comment("更新时间"),
	}
}

// Edges of the SLADefinition.
func (SLADefinition) Edges() []ent.Edge {
	return nil
}

// Indexes of the SLADefinition.
func (SLADefinition) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "service_type", "priority", "impact"),
		index.Fields("tenant_id", "is_active"),
	}
}

// Mixin of the SLADefinition.
func (SLADefinition) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.Time{},
	}
}
