package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ProcessExecutionHistory holds the schema definition for the BPMN Process Execution History entity.
type ProcessExecutionHistory struct {
	ent.Schema
}

// Fields of the ProcessExecutionHistory.
func (ProcessExecutionHistory) Fields() []ent.Field {
	return []ent.Field{
		field.String("history_id").
			Comment("历史记录ID").
			Unique().
			NotEmpty(),
		field.String("process_instance_id").
			Comment("流程实例ID").
			NotEmpty(),
		field.String("process_definition_key").
			Comment("流程定义Key").
			NotEmpty(),
		field.String("activity_id").
			Comment("活动ID").
			Optional(),
		field.String("activity_name").
			Comment("活动名称").
			Optional(),
		field.String("activity_type").
			Comment("活动类型：start_event, user_task, service_task, gateway, end_event").
			NotEmpty(),
		field.String("event_type").
			Comment("事件类型：start, complete, cancel, error").
			NotEmpty(),
		field.String("event_detail").
			Comment("事件详情").
			Optional(),
		field.JSON("variables", map[string]interface{}{}).
			Comment("相关变量").
			Optional(),
		field.String("user_id").
			Comment("操作用户ID").
			Optional(),
		field.String("user_name").
			Comment("操作用户名称").
			Optional(),
		field.Time("timestamp").
			Comment("时间戳").
			Default(time.Now),
		field.String("comment").
			Comment("备注").
			Optional(),
		field.String("error_message").
			Comment("错误信息").
			Optional(),
		field.String("error_code").
			Comment("错误代码").
			Optional(),
		field.Int("tenant_id").
			Comment("租户ID").
			Positive(),
		field.Time("created_at").
			Comment("创建时间").
			Default(time.Now),
	}
}

// Edges of the ProcessExecutionHistory.
func (ProcessExecutionHistory) Edges() []ent.Edge {
	return []ent.Edge{
		// TODO: 添加相关实体的edge定义
	}
}

// Indexes of the ProcessExecutionHistory.
func (ProcessExecutionHistory) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("history_id").
			Unique(),
		index.Fields("process_instance_id"),
		index.Fields("process_definition_key"),
		index.Fields("activity_id"),
		index.Fields("activity_type"),
		index.Fields("event_type"),
		index.Fields("user_id"),
		index.Fields("timestamp"),
		index.Fields("tenant_id"),
		index.Fields("created_at"),
	}
}
