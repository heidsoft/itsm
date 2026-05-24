package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"time"
)

type Contract struct {
	ent.Schema
}

func (Contract) Fields() []ent.Field {
	return []ent.Field{
		field.String("contract_number").NotEmpty().Unique(),
		field.String("title"),
		field.String("contract_type"), // 服务协议, 维护合同, 采购合同
		field.Float("value").Default(0),
		field.Time("start_date"),
		field.Time("end_date"),
		field.String("status").Default("active"), // 生效, 到期, 终止
		field.String("description"),
		field.Int("vendor_id").Optional(),
		field.Int("tenant_id"),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now),
	}
}

func (Contract) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("vendor", Vendor.Type).Field("vendor_id").Ref("contracts").Unique(),
	}
}
