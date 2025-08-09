package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// AuditLog holds the schema definition for the AuditLog entity.
type AuditLog struct {
	ent.Schema
}

// Fields of the AuditLog.
func (AuditLog) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").Default(time.Now),
		field.Int("tenant_id").Optional(),
		field.Int("user_id").Optional(),
		field.String("request_id").Optional(),
		field.String("ip").Default(""),
		field.String("resource").Default(""),
		field.String("action").Default(""),
		field.String("path"),
		field.String("method"),
		field.Int("status_code").Default(0),
		field.Text("request_body").Optional().Nillable(),
	}
}

// Edges of the AuditLog.
func (AuditLog) Edges() []ent.Edge { return nil }
