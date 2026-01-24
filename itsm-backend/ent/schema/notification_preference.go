package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// NotificationPreference holds the schema definition for the NotificationPreference entity.
type NotificationPreference struct {
	ent.Schema
}

// Fields of the NotificationPreference.
func (NotificationPreference) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id").
			Comment("用户ID").
			Positive(),
		field.Int("tenant_id").
			Comment("租户ID").
			Positive(),
		field.String("event_type").
			Comment("事件类型: ticket_created, ticket_assigned, ticket_updated, ticket_resolved, ticket_closed, sla_warning, sla_violated, comment_added, approval_required, mention").
			NotEmpty(),
		field.Bool("email_enabled").
			Comment("是否启用邮件通知").
			Default(true),
		field.Bool("sms_enabled").
			Comment("是否启用短信通知").
			Default(false),
		field.Bool("in_app_enabled").
			Comment("是否启用站内通知").
			Default(true),
		field.Bool("push_enabled").
			Comment("是否启用推送通知").
			Default(false),
		field.String("frequency").
			Comment("通知频率: immediate, hourly_digest, daily_digest").
			Default("immediate"),
		field.Time("quiet_hours_start").
			Comment("免打扰开始时间").
			Optional(),
		field.Time("quiet_hours_end").
			Comment("免打扰结束时间").
			Optional(),
		field.String("timezone").
			Comment("时区").
			Default("UTC"),
		field.Time("created_at").
			Comment("创建时间").
			Default(time.Now),
		field.Time("updated_at").
			Comment("更新时间").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the NotificationPreference.
func (NotificationPreference) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("notification_preferences").
			Field("user_id").
			Required().
			Unique().
			Comment("所属用户"),
	}
}
