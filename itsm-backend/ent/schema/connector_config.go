package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ConnectorConfig persists tenant connector lifecycle configuration.
// Credentials are stored only as authenticated ciphertext.
type ConnectorConfig struct{ ent.Schema }

func (ConnectorConfig) Fields() []ent.Field {
	return []ent.Field{
		field.Int("tenant_id").Positive(),
		field.String("name").NotEmpty().MaxLen(100),
		field.String("provider").NotEmpty().MaxLen(100),
		field.String("connector_type").Optional().MaxLen(50),
		field.Bool("enabled").Default(false),
		field.Text("encrypted_credentials").Optional(),
		field.JSON("settings", map[string]interface{}{}).Optional(),
		field.JSON("labels", map[string]string{}).Optional(),
		field.String("status").Default("configured").MaxLen(40),
		field.String("last_error").Optional().MaxLen(2000),
		field.Time("last_health_check_at").Optional().Nillable(),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (ConnectorConfig) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "name", "provider").Unique(),
		index.Fields("tenant_id", "enabled"),
	}
}
