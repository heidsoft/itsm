package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// TicketAttachment holds the schema definition for the TicketAttachment entity.
type TicketAttachment struct {
	ent.Schema
}

// Fields of the TicketAttachment.
func (TicketAttachment) Fields() []ent.Field {
	return []ent.Field{
		field.Int("ticket_id").
			Comment("工单ID").
			Positive(),
		field.String("file_name").
			Comment("文件名").
			NotEmpty(),
		field.String("file_path").
			Comment("文件路径").
			NotEmpty(),
		field.String("file_url").
			Comment("文件URL（用于访问）").
			Optional(),
		field.Int("file_size").
			Comment("文件大小（字节）").
			Positive(),
		field.String("file_type").
			Comment("文件类型（MIME类型）").
			NotEmpty(),
		field.String("mime_type").
			Comment("MIME类型").
			Optional(),
		field.Int("uploaded_by").
			Comment("上传人ID").
			Positive(),
		field.Int("tenant_id").
			Comment("租户ID").
			Positive(),
		field.Time("created_at").
			Comment("创建时间").
			Default(time.Now),
	}
}

// Edges of the TicketAttachment.
func (TicketAttachment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("ticket", Ticket.Type).
			Ref("attachments").
			Field("ticket_id").
			Required().
			Unique().
			Comment("所属工单"),
		edge.From("uploader", User.Type).
			Ref("ticket_attachments").
			Field("uploaded_by").
			Required().
			Unique().
			Comment("上传人"),
	}
}

