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
		// 策略版本（2026-11-21）：放行判据的内容指纹，取自当时命中的
		// permissions.description 策略信封（见 authz.PermissionPolicy.PolicyVersion）。
		//
		// 为什么必须是独立列而不是塞进 permission_check：
		// permission_check 是 passed|denied|skipped 的三态语义字段，前端按它
		// 渲染「权限校验结果」徽标；把 32 位指纹混进去会让该字段既不是状态也
		// 不是版本，两边都不可读。审计要能回答「这次放行依据哪一版策略」，
		// 就必须有一个只承载版本语义的列。
		//
		// 默认 persisted 空串 = 「当时没有策略参与判定」（human 主体走角色 RBAC）；
		// 无法取到指纹时写 authz.PolicyVersionUnknown 而不是空串，二者语义不同。
		field.String("policy_version").Default("").Comment("放行判据的策略指纹（决策输入哈希）"),
		// 运行轮次标识（2026-11-21）：把「一轮 Agent 会话里的多次工具调用」
		// 收敛成一个可整体取出的证据链。
		//
		// 与 request_id 的分工必须写清楚，否则两个字段会互相冒充：
		//   - request_id = 单次 HTTP 请求（由 middleware 注入），与 auditlog
		//     的 request_id 对齐，用于「这次请求打了哪些审计点」；
		//   - run_id     = 一次逻辑运行（一轮对话 / 一次调查编排 / 一次 MCP
		//     会话），可跨多个 HTTP 请求，用于「这个 Agent 这一轮做了什么」。
		//
		// 空串表示「这次调用不属于任何可标识的运行轮次」（如单发 HTTP 工具调用）。
		field.String("run_id").Default("").Comment("运行轮次标识，跨请求串联同一轮 Agent 行为"),
	}
}

func (ToolInvocation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("conversation", Conversation.Type).Ref("tool_invocations").Unique().Field("conversation_id"),
		edge.From("user", User.Type).Ref("tool_invocations").Unique().Field("user_id"),
	}
}
