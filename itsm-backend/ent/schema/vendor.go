package schema

import (
    "entgo.io/ent"
    "entgo.io/ent/schema/field"
    "entgo.io/ent/schema/edge"
    "time"
)

type Vendor struct {
    ent.Schema
}

func (Vendor) Fields() []ent.Field {
    return []ent.Field{
        field.String("name").NotEmpty(),
        field.String("code").NotEmpty().Unique(),
        field.String("vendor_type"),  // IT服务, 硬件供应商, 软件供应商
        field.String("contact_name"),
        field.String("contact_email"),
        field.String("contact_phone"),
        field.String("address"),
        field.String("website"),
        field.Float("rating").Default(0),
        field.String("status").Default("active"),
        field.Int("tenant_id"),
        field.Time("created_at").Default(time.Now),
        field.Time("updated_at").Default(time.Now),
    }
}

func (Vendor) Edges() []ent.Edge {
    return []ent.Edge{
        edge.To("contracts", Contract.Type),
        edge.To("assets", Asset.Type),
    }
}