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

		// 部门ID（0表示全局）
		field.Int("department_id").
			Comment("部门ID (0表示全局)").
			Optional().
			Default(0),

		// 团队ID（0表示全局）
		field.Int("team_id").
			Comment("团队ID (0表示全局)").
			Optional().
			Default(0),

		// 场景标识（alert_handling/change_release/code_release等）
		field.String("scenario").
			Comment("场景标识: alert_handling, change_release, code_release, expense_approval").
			Optional().
			Default(""),

		// 流程分类（operations/rd/finance/hr）
		field.String("category").
			Comment("流程分类: operations, rd, finance, hr").
			Optional().
			Default(""),

		// 匹配条件JSON
		field.JSON("conditions", map[string]interface{}{}).
			Comment("匹配条件JSON").
			Optional().
			Default(map[string]interface{}{}),

		// 审批链ID
		field.String("approval_chain_id").
			Comment("审批链ID").
			Optional().
			Default(""),

		// SLA策略ID
		field.String("sla_policy_id").
			Comment("SLA策略ID").
			Optional().
			Default(""),

		// 覆盖配置
		field.JSON("overrides", map[string]interface{}{}).
			Comment("覆盖配置").
			Optional().
			Default(map[string]interface{}{}),

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
		// 流程路由复合索引
		index.Fields("tenant_id", "business_type", "is_active", "department_id", "team_id", "scenario"),
	}
}
