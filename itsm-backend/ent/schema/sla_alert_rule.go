package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// SLAAlertRule holds the schema definition for the SLAAlertRule entity.
type SLAAlertRule struct {
	ent.Schema
}

// Fields of the SLAAlertRule.
func (SLAAlertRule) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("预警规则名称").
			NotEmpty(),
		field.Int("sla_definition_id").
			Comment("SLA定义ID").
			Positive(),
		field.String("alert_level").
			Comment("预警级别: warning, critical, severe").
			Default("warning"),
		field.Int("threshold_percentage").
			Comment("阈值百分比(0-100)").
			Default(70).
			Min(0).
			Max(100),
		field.JSON("notification_channels", []string{}).
			Comment("通知渠道: email, sms, in_app").
			Default([]string{"in_app"}),
		field.Bool("escalation_enabled").
			Comment("是否启用升级").
			Default(false),
		field.JSON("escalation_levels", []map[string]interface{}{}).
			Comment("升级级别配置").
			Optional(),
		field.Bool("is_active").
			Comment("是否启用").
			Default(true),
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

// Edges of the SLAAlertRule.
func (SLAAlertRule) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("sla_definition", SLADefinition.Type).
			Ref("alert_rules").
			Field("sla_definition_id").
			Unique().
			Required().
			Comment("关联的SLA定义"),
		edge.To("alert_history", SLAAlertHistory.Type).
			Comment("预警历史记录"),
	}
}
