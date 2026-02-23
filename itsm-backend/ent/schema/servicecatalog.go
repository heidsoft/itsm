package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type ServiceCatalog struct{ ent.Schema }

func (ServiceCatalog) Fields() []ent.Field {
	return []ent.Field{
		// 基础信息
		field.String("name").Comment("服务名称").NotEmpty(),
		field.Text("description").Comment("服务描述").Optional(),
		field.String("category").Comment("服务分类").Optional(),
		field.String("icon").Comment("服务图标").Optional(),
		field.String("service_type").Comment("服务类型: vm|rds|oss|network|storage|security|custom").Default("custom"),

		// 价格与交付
		field.Float("price").Comment("价格").Optional(),
		field.Int("delivery_time").Comment("交付时间（天）").Optional(),
		field.String("unit").Comment("计价单位: 月|次|用户").Optional(),

		// 审批配置
		field.Bool("requires_approval").Comment("是否需要审批").Default(true),
		field.Int("approval_level").Comment("审批级别: 1-3").Default(1),
		field.JSON("approvers", []int{}).Comment("审批人ID列表").Optional(),

		// SLA配置
		field.Int("sla_response_time").Comment("SLA响应时间(分钟)").Optional(),
		field.Int("sla_resolution_time").Comment("SLA解决时间(分钟)").Optional(),

		// CI/云服务关联
		field.Int("ci_type_id").Comment("关联CI类型ID").Optional(),
		field.Int("cloud_service_id").Comment("关联云服务ID").Optional(),

		// 表单配置
		field.JSON("form_schema", map[string]interface{}{}).Comment("表单JSON配置").Optional(),
		field.JSON("available_regions", []string{}).Comment("可选区域").Optional(),
		field.JSON("available_specs", []string{}).Comment("可选规格").Optional(),

		// 状态
		field.String("status").Comment("状态").Default("active"),
		field.Int("tenant_id").Comment("租户ID").Positive(),
		field.Bool("is_active").Comment("是否激活").Default(true),
		field.Int("sort_order").Comment("排序").Default(0),

		// 时间戳
		field.Time("created_at").Comment("创建时间").Default(time.Now),
		field.Time("updated_at").Comment("更新时间").Default(time.Now).UpdateDefault(time.Now),
	}
}
func (ServiceCatalog) Edges() []ent.Edge { return nil }

func (ServiceCatalog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("ci_type_id"),
		index.Fields("cloud_service_id"),
		index.Fields("service_type"),
		index.Fields("category"),
		index.Fields("tenant_id", "status"),
	}
}
