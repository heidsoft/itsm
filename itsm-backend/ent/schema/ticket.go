package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Ticket holds the schema definition for the Ticket entity.
type Ticket struct {
	ent.Schema
}

// Fields of the Ticket.
func (Ticket) Fields() []ent.Field {
	return []ent.Field{
		field.String("title").
			Comment("工单标题").
			NotEmpty(),
		field.Text("description").
			Comment("工单描述").
			Optional(),
		field.String("status").
			Comment("状态").
			Default("open"),
		field.String("priority").
			Comment("优先级").
			Default("medium"),
		field.String("ticket_number").
			Comment("工单编号").
			Unique().
			NotEmpty(),
		field.Int("requester_id").
			Comment("申请人ID").
			Positive(),
		field.Int("assignee_id").
			Comment("处理人ID").
			Optional(),
		field.Int("tenant_id").
			Comment("租户ID").
			Positive(),
		field.Int("template_id").
			Comment("模板ID").
			Optional(),
		field.Int("category_id").
			Comment("分类ID").
			Optional(),
		field.Int("parent_ticket_id").
			Comment("父工单ID").
			Optional(),
		field.Int("sla_definition_id").
			Comment("SLA定义ID").
			Optional(),
		field.Time("sla_response_deadline").
			Comment("SLA响应截止时间").
			Optional(),
		field.Time("sla_resolution_deadline").
			Comment("SLA解决截止时间").
			Optional(),
		field.Time("first_response_at").
			Comment("首次响应时间").
			Optional(),
		field.Time("resolved_at").
			Comment("解决时间").
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

// Edges of the Ticket.
func (Ticket) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("template", TicketTemplate.Type).
			Ref("tickets").
			Field("template_id").
			Unique().
			Comment("工单模板"),
		edge.From("category", TicketCategory.Type).
			Ref("tickets").
			Field("category_id").
			Unique().
			Comment("工单分类"),
		edge.To("tags", TicketTag.Type).
			Comment("工单标签"),
		edge.To("related_tickets", Ticket.Type).
			Comment("关联工单"),
		edge.From("parent_ticket", Ticket.Type).
			Ref("related_tickets").
			Field("parent_ticket_id").
			Unique().
			Comment("父工单"),
		edge.To("workflow_instances", WorkflowInstance.Type).
			Comment("工作流实例"),
		edge.From("sla_definition", SLADefinition.Type).
			Ref("tickets").
			Field("sla_definition_id").
			Unique().
			Comment("SLA定义"),
		edge.To("sla_violations", SLAViolation.Type).
			Comment("SLA违规记录"),
	}
}
