package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// ProblemComment holds the schema definition for the ProblemComment entity.
//
// 2026-10-21：问题域 GA 阻断项「评论功能尚未实现」的落地表。与 TicketComment
// 同构（同租户 + 内部备注 + @提及 + 附件引用），但独立成表而非复用 ticket_comments：
// problem_id 与 ticket_id 是不同聚合根，混用会让「问题的处理纪要」与「工单的客户
// 沟通」共享一套可见性规则，而两者的读者角色并不相同。
type ProblemComment struct {
	ent.Schema
}

// Fields of the ProblemComment.
func (ProblemComment) Fields() []ent.Field {
	return []ent.Field{
		field.Int("problem_id").
			Comment("问题ID").
			Positive(),
		field.Int("user_id").
			Comment("评论人ID").
			Positive(),
		field.Text("content").
			Comment("评论内容").
			NotEmpty(),
		field.Bool("is_internal").
			Comment("是否内部备注").
			Default(false),
		field.JSON("mentions", []int{}).
			Comment("@的用户ID列表").
			Optional(),
		field.JSON("attachments", []int{}).
			Comment("附件ID列表").
			Optional(),
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

// Edges of the ProblemComment.
func (ProblemComment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("problem", Problem.Type).
			Ref("comments").
			Field("problem_id").
			Required().
			Unique().
			Comment("所属问题"),
		edge.From("user", User.Type).
			Ref("problem_comments").
			Field("user_id").
			Required().
			Unique().
			Comment("评论人"),
	}
}
