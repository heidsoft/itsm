package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AssetLicense represents a software license in Asset Management
type AssetLicense struct {
	ent.Schema
}

func (AssetLicense) Fields() []ent.Field {
	return []ent.Field{
		field.String("license_key").
			Comment("许可证密钥").
			Optional(),
		field.String("name").
			Comment("许可证名称").
			NotEmpty(),
		field.Text("description").
			Comment("许可证描述").
			Optional(),
		field.String("vendor").
			Comment("供应商").
			Optional(),
		field.String("license_type").
			Comment("许可证类型: perpetual/subscription/per-user/per-seat/site").
			Default("perpetual"),
		field.Int("total_quantity").
			Comment("总数量").
			Default(1),
		field.Int("used_quantity").
			Comment("已使用数量").
			Default(0),
		field.Int("tenant_id").
			Comment("租户ID").
			Positive(),
		field.Int("asset_id").
			Comment("关联的资产ID").
			Optional(),
		field.String("purchase_date").
			Comment("采购日期").
			Optional(),
		field.Float("purchase_price").
			Comment("采购价格").
			Optional(),
		field.String("expiry_date").
			Comment("到期日期").
			Optional(),
		field.String("support_vendor").
			Comment("支持供应商").
			Optional(),
		field.String("support_contact").
			Comment("支持联系方式").
			Optional(),
		field.String("renewal_cost").
			Comment("续费成本").
			Optional(),
		field.String("status").
			Comment("状态: active/expired/expiring-soon/depleted").
			Default("active"),
		field.Text("notes").
			Comment("备注").
			Optional(),
		field.JSON("users", []int{}).
			Comment("授权用户列表").
			Optional(),
		field.JSON("tags", []string{}).
			Comment("标签").
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

func (AssetLicense) Edges() []ent.Edge {
	return nil
}

func (AssetLicense) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("name"),
		index.Fields("license_type"),
		index.Fields("status"),
		index.Fields("tenant_id"),
		index.Fields("asset_id"),
	}
}
