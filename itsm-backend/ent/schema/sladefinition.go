package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type SLADefinition struct{ ent.Schema }

func (SLADefinition) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").Comment("SLA名称").NotEmpty(),
		field.Text("description").Comment("SLA描述").Optional(),
		field.String("service_type").Comment("服务类型").Optional(),
		field.String("priority").Comment("优先级").Optional(),
		field.Int("response_time").Comment("响应时间(分钟)").Positive().Default(30),
		field.Int("resolution_time").Comment("解决时间(分钟)").Positive().Default(240),
		field.JSON("business_hours", map[string]interface{}{}).Comment("营业时间配置").Optional(),
		field.JSON("escalation_rules", map[string]interface{}{}).Comment("升级规则").Optional(),
		field.JSON("conditions", map[string]interface{}{}).Comment("适用条件").Optional(),
		field.Bool("is_active").Comment("是否激活").Default(true),
		field.Int("tenant_id").Comment("租户ID").Positive(),
		field.Time("created_at").Comment("创建时间").Default(time.Now),
		field.Time("updated_at").Comment("更新时间").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (SLADefinition) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("violations", SLAViolation.Type).Comment("SLA违规记录"),
		edge.To("metrics", SLAMetric.Type).Comment("SLA指标"),
		edge.To("tickets", Ticket.Type).Comment("关联工单"),
		edge.To("alert_rules", SLAAlertRule.Type).Comment("SLA预警规则"),
	}
}
