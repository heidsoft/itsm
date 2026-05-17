package schema

import (
	"time"
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// MSPAllocation holds the schema definition for the MSPAllocation entity.
type MSPAllocation struct {
	ent.Schema
}

// Fields of the MSPAllocation.
func (MSPAllocation) Fields() []ent.Field {
	return []ent.Field{
		field.Int("msp_user_id").
			Comment("MSP 员工ID（属于MSP租户）"),
		field.Int("customer_tenant_id").
			Comment("客户租户ID（支持单客户模式）").
			Optional(),
		field.String("role").
			Comment("分配角色: primary|backup|specialist").
			Default("primary"),
		field.Time("assigned_at").
			Comment("分配时间").
			Default(time.Now),
		field.Time("deassigned_at").
			Comment("解除分配时间").
			Optional(),
		field.Time("created_at").
			Comment("创建时间").
			Default(time.Now),
	}
}

// Edges of the MSPAllocation.
func (MSPAllocation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("msp_user", User.Type).
			Ref("msp_allocations").
			Field("msp_user_id").
			Required().
			Unique(),
		edge.From("customer_tenant", Tenant.Type).
			Ref("msp_customer_allocations").
			Required(),
	}
}