package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// FeishuTicketSync holds the schema definition for the FeishuTicketSync entity.
type FeishuTicketSync struct {
	ent.Schema
}

// Fields of the FeishuTicketSync.
func (FeishuTicketSync) Fields() []ent.Field {
	return []ent.Field{
		field.Int("tenant_id").
			Comment("租户ID").
			Positive(),
		field.Int("ticket_id").
			Comment("ITSM工单ID").
			Positive(),
		field.String("feishu_task_id").
			Comment("飞书任务ID").
			NotEmpty(),
		field.String("feishu_task_guid").
			Comment("飞书任务GUID（用于API调用）").
			Optional(),
		field.String("sync_status").
			Comment("同步状态: pending/synced/failed").
			Default("pending"),
		field.String("last_sync_direction").
			Comment("最后同步方向: itsm_to_feishu / feishu_to_itsm").
			Optional(),
		field.Time("last_synced_at").
			Comment("最后同步时间").
			Optional(),
		field.Text("error_message").
			Comment("同步错误信息").
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

// Edges of the FeishuTicketSync.
func (FeishuTicketSync) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("ticket", Ticket.Type).
			Ref("feishu_syncs").
			Field("ticket_id").
			Unique().
			Required(),
	}
}

// Indexes of the FeishuTicketSync.
func (FeishuTicketSync) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "ticket_id").
			Unique(),
		index.Fields("tenant_id", "feishu_task_id").
			Unique(),
	}
}
