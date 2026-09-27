package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"itsm-backend/ent/mixin"
)

// SLAState 统一 SLA 运行时状态。
//
// Phase 3 Step 3.3（2026-09-27）：收敛 Ticket/Incident 内嵌 SLA 字段到独立表。
// 三阶段迁移：
//
//	阶段 1 — 双写：新建 SLAState 记录，旧内嵌字段继续写入
//	阶段 2 — 存量迁移：从内嵌字段回填 sla_states
//	阶段 3 — 切读：读取切到 sla_states，移除内嵌字段
//
// 多态关联：aggregate_type + aggregate_id 指向 ticket / incident（后续扩展 change / problem）。
type SLAState struct{ ent.Schema }

func (SLAState) Mixins() []ent.Mixin {
	return []ent.Mixin{
		mixin.TenantTimestamps{},
	}
}

func (SLAState) Fields() []ent.Field {
	return []ent.Field{
		field.Int("tenant_id").
			Comment("租户ID"),
		field.Time("created_at").
			Comment("创建时间").
			Default(time.Now),

		// ── 多态关联 ──
		field.String("aggregate_type").
			Comment("聚合根类型: ticket, incident, change, problem, service_request").
			NotEmpty(),
		field.Int("aggregate_id").
			Comment("聚合根ID").
			Positive(),

		// ── SLA 绑定 ──
		field.Int("sla_definition_id").
			Comment("SLA定义ID").
			Optional(),
		field.String("sla_policy_id").
			Comment("SLA策略标识").
			Optional(),

		// ── 计算截止时间 ──
		field.Time("response_deadline").
			Comment("响应截止时间").
			Optional(),
		field.Time("resolution_deadline").
			Comment("解决截止时间").
			Optional(),

		// ── 状态 ──
		field.String("status").
			Comment("SLA状态: pending, active, paused, breached, met").
			Default("pending"),
		field.Time("paused_at").
			Comment("暂停时间").
			Optional(),
		field.String("pause_reason").
			Comment("暂停原因").
			Optional(),
		field.Int("paused_duration_seconds").
			Comment("累计暂停秒数").
			Default(0),

		// ── 实际时间戳 ──
		field.Time("first_response_at").
			Comment("首次响应时间").
			Optional(),
		field.Time("resolved_at").
			Comment("解决时间").
			Optional(),
	}
}

func (SLAState) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("sla_definition", SLADefinition.Type).
			Ref("states").
			Field("sla_definition_id").
			Unique().
			Comment("SLA定义"),
	}
}

func (SLAState) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "aggregate_type", "aggregate_id").
			Unique(),
		index.Fields("tenant_id", "status"),
		index.Fields("response_deadline"),
		index.Fields("resolution_deadline"),
	}
}
