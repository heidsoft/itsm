package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// NotificationDelivery records the durable delivery state for one recipient/channel command.
type NotificationDelivery struct{ ent.Schema }

func (NotificationDelivery) Fields() []ent.Field {
	return []ent.Field{
		field.Int("tenant_id").Positive(),
		field.Int("operational_command_id").Positive().Unique(),
		field.Int("ticket_id").Optional().Nillable(),
		field.Int("ticket_notification_id").Optional().Nillable(),
		field.Int("recipient_id").Positive(),
		field.String("channel").NotEmpty().MaxLen(50),
		field.String("target_masked").Optional().MaxLen(200),
		field.String("status").Default("pending").MaxLen(32),
		field.Int("attempt").Default(0).NonNegative(),
		field.String("provider_message_id").Optional().MaxLen(200),
		field.String("error_code").Optional().MaxLen(100),
		field.String("error_message").Optional().MaxLen(1000),
		field.Time("sent_at").Optional().Nillable(),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (NotificationDelivery) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "recipient_id", "created_at"),
		index.Fields("tenant_id", "status", "created_at"),
		index.Fields("tenant_id", "ticket_id"),
	}
}
