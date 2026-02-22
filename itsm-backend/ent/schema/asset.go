package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Asset represents an IT asset in Asset Management
type Asset struct {
	ent.Schema
}

func (Asset) Fields() []ent.Field {
	return []ent.Field{
		field.String("asset_number").
			Comment("资产编号").
			NotEmpty(),
		field.String("name").
			Comment("资产名称").
			NotEmpty(),
		field.Text("description").
			Comment("资产描述").
			Optional(),
		field.String("type").
			Comment("资产类型: hardware/software/cloud/license").
			Default("hardware"),
		field.String("status").
			Comment("状态: available/in-use/maintenance/retired/disposed").
			Default("available"),
		field.String("category").
			Comment("资产分类").
			Optional(),
		field.String("subcategory").
			Comment("资产子分类").
			Optional(),
		field.Int("tenant_id").
			Comment("租户ID").
			Positive(),
		field.Int("ci_id").
			Comment("关联的CMDB配置项ID").
			Optional(),
		field.Int("assigned_to").
			Comment("分配给的用户ID").
			Optional(),
		field.Int("location_id").
			Comment("位置ID").
			Optional(),
		field.String("serial_number").
			Comment("序列号").
			Optional(),
		field.String("model").
			Comment("型号").
			Optional(),
		field.String("manufacturer").
			Comment("制造商").
			Optional(),
		field.String("vendor").
			Comment("供应商").
			Optional(),
		field.String("purchase_date").
			Comment("采购日期").
			Optional(),
		field.Float("purchase_price").
			Comment("采购价格").
			Optional(),
		field.String("warranty_expiry").
			Comment("保修期到期").
			Optional(),
		field.String("support_expiry").
			Comment("支持期到期").
			Optional(),
		field.String("location").
			Comment("物理位置").
			Optional(),
		field.String("department").
			Comment("所属部门").
			Optional(),
		field.Int("parent_asset_id").
			Comment("父资产ID").
			Optional(),
		field.JSON("specifications", map[string]string{}).
			Comment("规格参数").
			Optional(),
		field.JSON("custom_fields", map[string]string{}).
			Comment("自定义字段").
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

func (Asset) Edges() []ent.Edge {
	return nil
}

func (Asset) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("asset_number"),
		index.Fields("type"),
		index.Fields("status"),
		index.Fields("tenant_id"),
		index.Fields("assigned_to"),
		index.Fields("ci_id"),
	}
}
