package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Tenant holds the schema definition for the Tenant entity.
type Tenant struct {
	ent.Schema
}

// Fields of the Tenant.
func (Tenant) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("租户名称").
			NotEmpty(),
		field.String("code").
			Comment("租户代码").
			Unique().
			NotEmpty(),
		field.String("domain").
			Comment("域名").
			Optional(),
		field.Enum("type").
			Comment("租户类型: internal|saas_customer|msp_provider|msp_customer，保留 standard|msp|customer 兼容历史数据").
			Values("standard", "msp", "customer", "internal", "saas_customer", "msp_provider", "msp_customer").
			Default("standard"),
		field.String("status").
			Comment("状态").
			Default("active"),
		field.Int("parent_tenant_id").
			Comment("父租户ID (MSP客户指向MSP提供商)").
			Optional(),
		field.Int("msp_provider_id").
			Comment("MSP服务提供商ID").
			Optional(),
		field.Time("expires_at").
			Comment("过期时间").
			Optional(),
		field.String("plan_code").
			Comment("订阅或套餐编码").
			Optional(),
		field.Bool("billing_enabled").
			Comment("是否启用计费/核算").
			Default(false),
		field.String("cost_center_code").
			Comment("成本中心编码").
			Optional(),
		field.String("legal_entity_code").
			Comment("法人实体编码").
			Optional(),
		field.String("currency").
			Comment("结算币种").
			Optional(),
		field.String("service_tier").
			Comment("服务等级").
			Optional(),
		field.String("owner_contact").
			Comment("租户负责人联系方式").
			Optional(),
		field.Time("created_at").
			Comment("创建时间").
			Default(time.Now),
		field.Time("updated_at").
			Comment("更新时间").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the Tenant.
func (Tenant) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("users", User.Type).
			Comment("租户用户"),
		edge.To("msp_customer_allocations", MSPAllocation.Type).
			Comment("MSP客户分配"),
	}
}
