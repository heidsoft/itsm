package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// IncidentEscalationRule holds the schema definition for the IncidentEscalationRule entity.
// 事件升级规则，支持多级升级 (L1/L2/L3)
type IncidentEscalationRule struct {
	ent.Schema
}

// Fields of the IncidentEscalationRule.
func (IncidentEscalationRule) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("规则名称").
			NotEmpty(),
		field.Text("description").
			Comment("规则描述").
			Optional(),
		field.String("trigger_type").
			Comment("触发类型: time_based/sla_breach/manual").
			NotEmpty(),
		field.Int("escalation_level").
			Comment("升级级别: 1/2/3").
			Default(1),
		field.Int("trigger_minutes").
			Comment("触发时间(分钟)").
			Positive(),
		field.String("from_status").
			Comment("原状态").
			Optional(),
		field.String("to_status").
			Comment("目标状态").
			Optional(),
		field.String("target_assignee_type").
			Comment("目标分配类型: group/role/user").
			NotEmpty(),
		field.Int("target_assignee_id").
			Comment("目标分配人ID").
			Optional(),
		field.String("target_group").
			Comment("目标组").
			Optional(),
		field.Bool("auto_escalate").
			Comment("是否自动升级").
			Default(true),
		field.JSON("notification_config", map[string]interface{}{}).
			Comment("通知配置: 邮件/短信/站内信").
			Optional(),
		field.Bool("is_active").
			Comment("是否激活").
			Default(true),
		field.String("priority_match").
			Comment("匹配的优先级: critical/high/medium/low").
			Optional(),
		field.String("category_match").
			Comment("匹配的事件分类").
			Optional(),
		field.Int("tenant_id").
			Comment("租户ID").
			Positive(),
		field.Time("created_at").
			Comment("创建时间").
			Default(time.Now),
		field.Time("updated_at").
			Comment("更新时间").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the IncidentEscalationRule.
func (IncidentEscalationRule) Edges() []ent.Edge {
	return nil
}
