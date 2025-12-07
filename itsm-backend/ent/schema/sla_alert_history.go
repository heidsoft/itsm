package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/edge"
)

// SLAAlertHistory holds the schema definition for the SLAAlertHistory entity.
type SLAAlertHistory struct {
	ent.Schema
}

// Fields of the SLAAlertHistory.
func (SLAAlertHistory) Fields() []ent.Field {
	return []ent.Field{
		field.Int("ticket_id").
			Comment("工单ID").
			Positive(),
		field.String("ticket_number").
			Comment("工单编号").
			NotEmpty(),
		field.String("ticket_title").
			Comment("工单标题").
			NotEmpty(),
		field.Int("alert_rule_id").
			Comment("预警规则ID").
			Positive(),
		field.String("alert_rule_name").
			Comment("预警规则名称").
			NotEmpty(),
		field.String("alert_level").
			Comment("预警级别").
			Default("warning"),
		field.Int("threshold_percentage").
			Comment("阈值百分比").
			Default(70),
		field.Float("actual_percentage").
			Comment("实际百分比").
			Default(0),
		field.Bool("notification_sent").
			Comment("是否已发送通知").
			Default(false),
		field.Int("escalation_level").
			Comment("升级级别").
			Default(0),
		field.Int("tenant_id").
			Comment("租户ID").
			Positive(),
		field.Time("created_at").
			Comment("创建时间").
			Default(time.Now),
		field.Time("resolved_at").
			Comment("解决时间").
			Optional(),
	}
}

// Edges of the SLAAlertHistory.
func (SLAAlertHistory) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("ticket", Ticket.Type).
			Ref("sla_alert_history").
			Field("ticket_id").
			Unique().
			Required().
			Comment("关联的工单"),
		edge.From("alert_rule", SLAAlertRule.Type).
			Ref("alert_history").
			Field("alert_rule_id").
			Unique().
			Required().
			Comment("关联的预警规则"),
	}
}

