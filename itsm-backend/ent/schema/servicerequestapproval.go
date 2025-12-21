package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ServiceRequestApproval holds the schema definition for the ServiceRequestApproval entity.
// 说明：V0 仅用于 ServiceRequest 的三段审批记录留痕（主管→IT→安全）。
// V1 将扩展为支持可配置工作流。
type ServiceRequestApproval struct{ ent.Schema }

func (ServiceRequestApproval) Fields() []ent.Field {
	return []ent.Field{
		field.Int("tenant_id").Comment("租户ID").Positive(),
		field.Int("service_request_id").Comment("服务请求ID").Positive(),
		field.Int("level").Comment("审批级别（从1开始）").Positive(),
		field.String("step").Comment("审批步骤名称").Optional(), // 步骤名称，可从 node.name 获取

		// 存储该审批步骤对应的工作流节点配置
		field.JSON("node", map[string]interface{}{}).
			Comment("工作流节点配置的快照").
			Optional(),

		field.String("status").Comment("状态：pending|approved|rejected|delegated").Default("pending"),

		// 审批人信息（pending 时可为空）
		field.Int("approver_id").Comment("审批人ID").Optional().Nillable(),
		field.String("approver_name").Comment("审批人姓名").Optional(),

		field.String("action").Comment("动作：approve|reject|delegate").Optional(),
		field.Text("comment").Comment("审批意见").Optional(),

		// V1 增强字段：审批时限和超时处理
		field.Int("timeout_hours").Comment("审批时限（小时）").Default(24),
		field.Time("due_at").Comment("到期时间（自动计算）").Optional(),
		field.Bool("is_escalated").Comment("是否已升级").Default(false),
		field.Int("delegated_to_id").Comment("转交审批人ID").Optional().Nillable(),
		field.String("escalation_reason").Comment("升级原因").Optional(),

		field.Time("created_at").Comment("创建时间").Default(time.Now),
		field.Time("processed_at").Comment("处理时间").Optional(),
	}
}

func (ServiceRequestApproval) Edges() []ent.Edge { return nil }

func (ServiceRequestApproval) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "service_request_id", "level"),
		index.Fields("tenant_id", "service_request_id", "status"),
		// V1 新增索引：用于查询超时审批
		index.Fields("due_at"),
		index.Fields("status", "due_at"),
	}
}
