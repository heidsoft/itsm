package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ProcessAuditLog holds the schema definition for the BPMN Process Audit Log entity.
type ProcessAuditLog struct {
	ent.Schema
}

// Fields of the ProcessAuditLog.
func (ProcessAuditLog) Fields() []ent.Field {
	return []ent.Field{
		field.Int("process_instance_id").
			Comment("流程实例ID").
			Positive(),
		field.String("process_instance_key").
			Comment("流程实例Key").
			NotEmpty(),
		field.String("process_definition_key").
			Comment("流程定义Key").
			NotEmpty(),
		field.Int("process_definition_id").
			Comment("流程定义ID").
			Positive(),
		field.String("activity_id").
			Comment("活动ID").
			NotEmpty(),
		field.String("activity_name").
			Comment("活动名称").
			Optional(),
		field.String("activity_type").
			Comment("活动类型：startEvent, endEvent, userTask, serviceTask, scriptTask, manualTask, gateway, subProcess").
			NotEmpty(),
		field.String("action").
			Comment("操作类型：started, completed, assigned, unassigned, claimed, completed, cancelled, suspended, resumed, terminated, variable_changed, escalated, reassigned").
			NotEmpty(),
		field.Int("user_id").
			Comment("操作用户ID").
			Optional(),
		field.String("user_name").
			Comment("操作用户名称").
			Optional(),
		field.Int("assignee_id").
			Comment("任务受理人ID").
			Optional(),
		field.String("assignee_name").
			Comment("任务受理人名称").
			Optional(),
		field.JSON("variables_before", map[string]interface{}{}).
			Comment("操作前变量").
			Optional(),
		field.JSON("variables_after", map[string]interface{}{}).
			Comment("操作后变量").
			Optional(),
		field.String("comment").
			Comment("备注/原因").
			Optional(),
		field.String("ip_address").
			Comment("IP地址").
			Optional(),
		field.String("user_agent").
			Comment("用户代理").
			Optional(),
		field.Int("tenant_id").
			Comment("租户ID").
			Positive(),
		field.Time("timestamp").
			Comment("操作时间").
			Default(time.Now),
		field.Int("duration_ms").
			Comment("操作耗时(毫秒)").
			Optional(),
		field.JSON("metadata", map[string]interface{}{}).
			Comment("扩展元数据").
			Optional(),
	}
}

// Edges of the ProcessAuditLog.
func (ProcessAuditLog) Edges() []ent.Edge {
	return []ent.Edge{}
}

// Indexes of the ProcessAuditLog.
func (ProcessAuditLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("process_instance_id"),
		index.Fields("process_instance_key"),
		index.Fields("process_definition_key"),
		index.Fields("activity_id"),
		index.Fields("activity_type"),
		index.Fields("action"),
		index.Fields("user_id"),
		index.Fields("assignee_id"),
		index.Fields("tenant_id"),
		index.Fields("timestamp"),
		index.Fields("process_definition_key", "action"),
		index.Fields("tenant_id", "timestamp"),
	}
}
