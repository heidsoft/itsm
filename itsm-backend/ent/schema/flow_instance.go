package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

// FlowStatus defines the flow instance status enum
type FlowStatus string

const (
	FlowStatusActive     FlowStatus = "active"     // 活跃
	FlowStatusCompleted  FlowStatus = "completed"  // 已完成
	FlowStatusTerminated FlowStatus = "terminated" // 已终止
	FlowStatusSuspended  FlowStatus = "suspended"  // 已暂停
)

// FlowInstance holds the schema definition for the FlowInstance entity.
type FlowInstance struct {
	ent.Schema
}

// Fields of the FlowInstance.
func (FlowInstance) Fields() []ent.Field {
	return []ent.Field{
		field.String("flow_definition_id").
			NotEmpty().
			MaxLen(100).
			Comment("流程定义ID"),
		field.String("flow_name").
			NotEmpty().
			MaxLen(200).
			Comment("流程名称"),
		field.String("flow_version").
			Optional().
			MaxLen(20).
			Default("1.0").
			Comment("流程版本"),
		field.Enum("status").
			Values(
				string(FlowStatusActive),
				string(FlowStatusCompleted),
				string(FlowStatusTerminated),
				string(FlowStatusSuspended),
			).
			Default(string(FlowStatusActive)).
			Comment("流程状态"),
		field.Int("current_step").
			Default(1).
			Positive().
			Comment("当前步骤"),
		field.Int("total_steps").
			Positive().
			Comment("总步骤数"),
		field.JSON("step_config", map[string]interface{}{}).
			Optional().
			Comment("步骤配置信息"),
		field.JSON("variables", map[string]interface{}{}).
			Optional().
			Comment("流程变量"),
		field.JSON("execution_history", []map[string]interface{}{}).
			Optional().
			Comment("执行历史记录"),
		field.Int("ticket_id").
			Positive().
			Comment("关联工单ID"),
		field.Time("started_at").
			Default(time.Now).
			Immutable().
			Comment("开始时间"),
		field.Time("completed_at").
			Optional().
			Nillable().
			Comment("完成时间"),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			Comment("更新时间"),
	}
}

// Edges of the FlowInstance.
func (FlowInstance) Edges() []ent.Edge {
	return []ent.Edge{
		// 关联的工单
		edge.From("ticket", Ticket.Type).
			Ref("flow_instance").
			Field("ticket_id").
			Required().
			Unique(),
	}
}

// Indexes of the FlowInstance.
func (FlowInstance) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("ticket_id"),
		index.Fields("status"),
		index.Fields("flow_definition_id"),
		index.Fields("current_step"),
		// 复合索引
		index.Fields("flow_definition_id", "status"),
	}
}
