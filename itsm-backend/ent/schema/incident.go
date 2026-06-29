package schema

import (
	"fmt"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Incident holds the schema definition for the Incident entity.
type Incident struct {
	ent.Schema
}

// Fields of the Incident.
func (Incident) Fields() []ent.Field {
	return []ent.Field{
		field.String("title").
			Comment("事件标题").
			NotEmpty(),
		field.Text("description").
			Comment("事件描述").
			Optional(),
		field.String("status").
			Comment("状态").
			Default("new"),
		field.String("type").
			Comment("事件类型").
			Default("incident"),
		field.String("priority").
			Comment("优先级").
			Validate(func(s string) error {
				valid := map[string]bool{"low": true, "medium": true, "high": true, "critical": true}
				if !valid[s] {
					return fmt.Errorf("invalid priority value: %s", s)
				}
				return nil
			}).
			Default("medium"),
		field.String("severity").
			Comment("严重程度").
			Default("medium"),
		field.String("impact").
			Comment("影响范围：low/medium/high/critical").
			Validate(func(s string) error {
				valid := map[string]bool{"low": true, "medium": true, "high": true, "critical": true}
				if !valid[s] {
					return fmt.Errorf("invalid impact value: %s", s)
				}
				return nil
			}).
			Default("medium"),
		field.String("urgency").
			Comment("紧急程度：low/medium/high/critical").
			Validate(func(s string) error {
				valid := map[string]bool{"low": true, "medium": true, "high": true, "critical": true}
				if !valid[s] {
					return fmt.Errorf("invalid urgency value: %s", s)
				}
				return nil
			}).
			Default("medium"),
		field.String("incident_number").
			Comment("事件编号").
			Unique().
			NotEmpty(),
		field.Int("reporter_id").
			Comment("报告人ID").
			Positive(),
		field.Int("assignee_id").
			Comment("处理人ID").
			Optional(),
		field.Int("configuration_item_id").
			Comment("配置项ID").
			Optional(),
		field.String("category").
			Comment("事件分类").
			Optional(),
		field.String("subcategory").
			Comment("事件子分类").
			Optional(),
		field.JSON("impact_analysis", map[string]interface{}{}).
			Comment("影响分析").
			Optional(),
		field.JSON("root_cause", map[string]interface{}{}).
			Comment("根本原因").
			Optional(),
		field.JSON("resolution_steps", []map[string]interface{}{}).
			Comment("解决步骤").
			Optional(),
		field.Time("detected_at").
			Comment("检测时间").
			Default(time.Now),
		field.Time("resolved_at").
			Comment("解决时间").
			Optional(),
		field.Time("closed_at").
			Comment("关闭时间").
			Optional(),
		field.Time("escalated_at").
			Comment("升级时间").
			Optional(),
		field.Int("escalation_level").
			Comment("升级级别").
			Default(0),
		field.Bool("is_automated").
			Comment("是否自动化处理").
			Default(false),
		field.Bool("is_major_incident").
			Comment("是否重大事件").
			Default(false),
		field.String("source").
			Comment("事件来源").
			Default("manual"),
		field.JSON("metadata", map[string]interface{}{}).
			Comment("元数据").
			Optional(),
		field.Int("tenant_id").
			Comment("租户ID").
			Positive(),
		field.Int("version").
			Comment("版本号（乐观锁）").
			Default(1).
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

// Edges of the Incident.
func (Incident) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("related_incidents", Incident.Type).
			Comment("关联事件"),
		edge.To("incident_events", IncidentEvent.Type).
			Comment("事件活动记录"),
		edge.To("incident_alerts", IncidentAlert.Type).
			Comment("事件告警"),
		edge.To("incident_metrics", IncidentMetric.Type).
			Comment("事件指标"),
		edge.From("parent_incident", Incident.Type).
			Ref("related_incidents").
			Comment("父事件"),
		edge.From("configuration_items", ConfigurationItem.Type).
			Ref("incidents").
			Comment("关联的配置项"),
		edge.From("problems", Problem.Type).
			Ref("incidents").
			Comment("关联的问题"),
	}
}
