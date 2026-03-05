package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ProcessBinding holds the schema definition for the ProcessBinding entity.
type ProcessBinding struct {
	ent.Schema
}

// Fields of the ProcessBinding.
func (ProcessBinding) Fields() []ent.Field {
	return []ent.Field{
		// 业务类型（ticket/change/incident/problem/service_request）
		field.String("business_type").
			Comment("业务类型").
			NotEmpty(),

		// 子类型（如普通变更/紧急变更）
		field.String("business_sub_type").
			Comment("业务子类型").
			Optional(),

		// 流程定义Key
		field.String("process_definition_key").
			Comment("流程定义Key").
			NotEmpty(),

		// 流程版本
		field.Int("process_version").
			Comment("流程版本").
			Default(1),

		// 是否默认流程
		field.Bool("is_default").
			Comment("是否默认流程").
			Default(false),

		// 优先级
		field.Int("priority").
			Comment("优先级（数值越大优先级越高）").
			Default(0),

		// 是否激活
		field.Bool("is_active").
			Comment("是否激活").
			Default(true),

		// 租户ID
		field.Int("tenant_id").
			Comment("租户ID").
			Positive(),

		// 创建时间
		field.Time("created_at").
			Comment("创建时间").
			Default(time.Now),

		// 更新时间
		field.Time("updated_at").
			Comment("更新时间").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the ProcessBinding.
func (ProcessBinding) Edges() []ent.Edge {
	return []ent.Edge{
		// 关联到流程定义
		edge.From("process_definition", ProcessDefinition.Type).
			Ref("bindings").
			Unique(),
	}
}

// Indexes of the ProcessBinding.
func (ProcessBinding) Indexes() []ent.Index {
	return []ent.Index{
		// 业务类型索引
		index.Fields("business_type", "business_sub_type"),
		// 流程定义索引
		index.Fields("process_definition_key"),
		// 租户索引
		index.Fields("tenant_id"),
	}
}
