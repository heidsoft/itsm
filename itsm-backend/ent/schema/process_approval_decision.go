package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ProcessApprovalDecision is the immutable audit fact produced by an approval task.
type ProcessApprovalDecision struct{ ent.Schema }

func (ProcessApprovalDecision) Fields() []ent.Field {
	return []ent.Field{
		field.Int("process_instance_id").Positive(),
		field.Int("process_task_id").Positive(),
		field.String("process_instance_key").NotEmpty(),
		field.String("task_id").NotEmpty(),
		field.String("process_definition_key").NotEmpty(),
		field.String("node_key").NotEmpty(),
		field.String("business_type").Optional(),
		field.String("business_id").Optional(),
		field.Int("actor_id").Positive(),
		field.String("actor_name").Optional(),
		field.String("action").Comment("approve, reject, delegate, transfer, add_approver, withdraw, timeout, system_decision"),
		field.String("decision").Comment("approved, rejected, delegated, withdrawn, timeout"),
		field.Text("comment").Optional(),
		field.Int("delegated_from").Optional().Nillable(),
		field.JSON("variables_snapshot", map[string]interface{}{}).Optional(),
		field.String("client_ip").Optional(),
		field.String("user_agent").Optional(),
		field.Int("tenant_id").Positive(),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

func (ProcessApprovalDecision) Indexes() []ent.Index {
	return []ent.Index{
		// 审计事实表：一个任务可合法产生多条决策记录（delegate→受托人完成、add_approver
		// 加签、多次委托链、withdraw/timeout 重入等），故 (tenant_id, process_task_id)
		// 绝不能唯一——曾经 UNIQUE 导致"委托后受托人完成审批"必然撞唯一约束。
		// 防重由任务状态 CAS（completeTaskWithClient 事务内 updated!=1 检查）保证。
		// 兼容升级：internal/bootstrap 的 prepareProcessApprovalDecisionIndexMigration
		// 会在 Schema.Create 前删除历史库中的旧唯一索引。
		index.Fields("tenant_id", "process_task_id"),
		index.Fields("tenant_id", "process_instance_id", "created_at"),
		index.Fields("tenant_id", "business_type", "business_id"),
		index.Fields("tenant_id", "actor_id", "created_at"),
	}
}
