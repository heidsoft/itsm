package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// ToolInvocation stores each tool call during a conversation.
type ToolInvocation struct{ ent.Schema }

func (ToolInvocation) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").Default(time.Now),
		field.Int("tenant_id"),
		field.Int("conversation_id").Optional(),
		field.String("tool_name"),
		field.Text("arguments").Default(""),
		field.Text("result").Optional().Nillable(),
		field.String("status").Default("success"),
		field.String("request_id").Optional(),
		field.Bool("needs_approval").Default(false),
		field.String("approval_state").Default("none"), // none|pending|approved|rejected
		field.String("approval_reason").Default(""),
		field.Int("approved_by").Optional(),
		field.Time("approved_at").Optional(),
		field.Bool("dry_run").Default(false),
		field.Text("error").Optional().Nillable(),
		// P2-6 AI 工具 RBAC 校验审计字段（向后兼容，所有字段均带默认值）
		field.Int("user_id").Optional().Comment("工具触发者用户 ID"),
		field.String("permission_check").Default("skipped").Comment("权限校验结果: passed|denied|skipped"),
		field.String("permission_reason").Default("").Comment("权限校验原因/拒绝原因"),
		field.String("role_snapshot").Default("").Comment("调用时角色快照，便于事后审计"),
		// Phase B6（2026-11-17）：业务单据强关联（可空，0 表示无关联）。
		// 替代 AI 审计底账 inputRef 字符串弱关联，可按工单/变更反查工具调用链。
		field.Int("ticket_id").Optional().Comment("关联工单 ID"),
		field.Int("change_id").Optional().Comment("关联变更 ID"),
	}
}

func (ToolInvocation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("conversation", Conversation.Type).Ref("tool_invocations").Unique().Field("conversation_id"),
		edge.From("user", User.Type).Ref("tool_invocations").Unique().Field("user_id"),
	}
}
