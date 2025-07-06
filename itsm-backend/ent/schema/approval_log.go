package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

// ApprovalStatus defines the approval status enum
type ApprovalStatus string

const (
	ApprovalStatusPending  ApprovalStatus = "pending"  // 待审批
	ApprovalStatusApproved ApprovalStatus = "approved" // 已批准
	ApprovalStatusRejected ApprovalStatus = "rejected" // 已拒绝
	ApprovalStatusSkipped  ApprovalStatus = "skipped"  // 已跳过
)

// ApprovalLog holds the schema definition for the ApprovalLog entity.
type ApprovalLog struct {
	ent.Schema
}

// Fields of the ApprovalLog.
func (ApprovalLog) Fields() []ent.Field {
	return []ent.Field{
		field.Int("approver_id").
			Positive().
			Comment("审批人ID"),
		field.Text("comment").
			Optional().
			Comment("审批意见"),
		field.Enum("status").
			Values(
				string(ApprovalStatusPending),
				string(ApprovalStatusApproved),
				string(ApprovalStatusRejected),
				string(ApprovalStatusSkipped),
			).
			Default(string(ApprovalStatusPending)).
			Comment("审批状态"),
		field.Int("ticket_id").
			Positive().
			Comment("所属工单ID"),
		field.Int("step_order").
			Positive().
			Comment("审批步骤顺序"),
		field.String("step_name").
			Optional().
			MaxLen(100).
			Comment("审批步骤名称"),
		field.JSON("metadata", map[string]interface{}{}).
			Optional().
			Comment("审批元数据"),
		field.Time("approved_at").
			Optional().
			Nillable().
			Comment("审批时间"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("创建时间"),
	}
}

// Edges of the ApprovalLog.
func (ApprovalLog) Edges() []ent.Edge {
	return []ent.Edge{
		// 审批人
		edge.From("approver", User.Type).
			Ref("approval_logs").
			Field("approver_id").
			Required().
			Unique(),
		// 所属工单
		edge.From("ticket", Ticket.Type).
			Ref("approval_logs").
			Field("ticket_id").
			Required().
			Unique(),
	}
}

// Indexes of the ApprovalLog.
func (ApprovalLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("ticket_id"),
		index.Fields("approver_id"),
		index.Fields("status"),
		index.Fields("step_order"),
		// 复合索引
		index.Fields("ticket_id", "step_order"),
		index.Fields("approver_id", "status"),
	}
}
