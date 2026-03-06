package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// SLAPolicy holds the schema definition for the SLAPolicy entity.
// 多层级SLA策略，支持按客户等级、工单类型等维度配置
type SLAPolicy struct {
	ent.Schema
}

// Fields of the SLAPolicy.
func (SLAPolicy) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("SLA策略名称").
			NotEmpty(),
		field.Text("description").
			Comment("SLA策略描述").
			Optional(),
		field.String("customer_tier").
			Comment("客户等级: platinum/gold/silver/bronze").
			Optional(),
		field.String("ticket_type").
			Comment("工单类型: incident/problem/change/request").
			Optional(),
		field.String("priority").
			Comment("优先级: critical/high/medium/low").
			Optional(),
		field.Int("response_time_minutes").
			Comment("响应时间(分钟)").
			Positive(),
		field.Int("resolution_time_minutes").
			Comment("解决时间(分钟)").
			Positive(),
		field.JSON("business_hours", map[string]interface{}{}).
			Comment("业务时间配置: 工作日、时段、时区").
			Optional(),
		field.Bool("exclude_weekends").
			Comment("是否排除周末").
			Default(false),
		field.Bool("exclude_holidays").
			Comment("是否排除节假日").
			Default(false),
		field.JSON("escalation_rules", map[string]interface{}{}).
			Comment("升级规则配置").
			Optional(),
		field.Bool("is_active").
			Comment("是否激活").
			Default(true),
		field.Int("priority_score").
			Comment("优先级分数(用于排序)").
			Default(0),
		field.Int("tenant_id").
			Comment("租户ID").
			Positive(),
		field.Time("created_at").
			Comment("创建时间").
			Default(time.Now),
		field.Time("updated_at").
			Comment("更新时间").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the SLAPolicy.
func (SLAPolicy) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("sla_definition", SLADefinition.Type).
			Comment("关联的SLA定义"),
		edge.To("tickets", Ticket.Type).
			Comment("应用此策略的工单"),
	}
}

// BusinessHoursConfig 业务时间配置
type BusinessHoursConfig struct {
	WorkDays     []int  `json:"work_days"`      // 工作日: 1-7 (周一到周日)
	StartTime    string `json:"start_time"`    // 开始时间: "09:00"
	EndTime      string `json:"end_time"`      // 结束时间: "18:00"
	TimeZone     string `json:"time_zone"`     // 时区: "Asia/Shanghai"
	HolidayList  []string `json:"holiday_list"` // 节假日列表
}
