package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// WorkflowTask holds the schema definition for the WorkflowTask entity.
type WorkflowTask struct {
	ent.Schema
}

// Fields of the WorkflowTask.
func (WorkflowTask) Fields() []ent.Field {
	return []ent.Field{
		field.String("task_id").
			Comment("任务ID").
			NotEmpty(),
		field.Int("instance_id").
			Comment("实例ID").
			Positive(),
		field.String("activity_id").
			Comment("活动ID").
			NotEmpty(),
		field.String("name").
			Comment("任务名称").
			NotEmpty(),
		field.String("type").
			Comment("任务类型: user_task, service_task, script_task, call_activity").
			Default("user_task"),
		field.String("assignee").
			Comment("处理人").
			Optional(),
		field.String("candidate_users").
			Comment("候选人用户，多个用逗号分隔").
			Optional(),
		field.String("candidate_groups").
			Comment("候选人组，多个用逗号分隔").
			Optional(),
		field.String("status").
			Comment("任务状态: pending, in_progress, completed, failed, cancelled").
			Default("pending"),
		field.String("priority").
			Comment("优先级: low, medium, high, urgent").
			Default("medium"),
		field.JSON("form_data", []byte{}).
			Comment("表单数据").
			Optional(),
		field.JSON("variables", []byte{}).
			Comment("流程变量").
			Optional(),
		field.String("comment").
			Comment("备注").
			Optional(),
		field.Int("tenant_id").
			Comment("租户ID").
			Positive(),
		field.Time("due_date").
			Comment("截止时间").
			Optional(),
		field.Time("created_at").
			Comment("创建时间").
			Default(time.Now),
		field.Time("updated_at").
			Comment("更新时间").
			Default(time.Now).
			UpdateDefault(time.Now),
		field.Time("completed_at").
			Comment("完成时间").
			Optional(),
		field.String("completed_by").
			Comment("完成人").
			Optional(),
	}
}

// Edges of the WorkflowTask.
func (WorkflowTask) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("instance", WorkflowInstance.Type).
			Ref("workflow_tasks").
			Field("instance_id").
			Unique().
			Required(),
	}
}
