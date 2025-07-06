package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

// Fields of the User.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("username").
			Unique().
			NotEmpty().
			MaxLen(50).
			Comment("用户名"),
		field.String("email").
			Unique().
			NotEmpty().
			MaxLen(100).
			Comment("邮箱地址"),
		field.String("name").
			NotEmpty().
			MaxLen(100).
			Comment("真实姓名"),
		field.String("department").
			Optional().
			MaxLen(100).
			Comment("部门"),
		field.String("phone").
			Optional().
			MaxLen(20).
			Comment("联系电话"),
		field.String("password_hash").
			Sensitive().
			Comment("密码哈希"),
		field.Bool("active").
			Default(true).
			Comment("是否激活"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("创建时间"),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			Comment("更新时间"),
	}
}

// Edges of the User.
func (User) Edges() []ent.Edge {
	return []ent.Edge{
		// 用户作为申请人的工单
		edge.To("submitted_tickets", Ticket.Type),
		// 用户作为处理人的工单
		edge.To("assigned_tickets", Ticket.Type),
		// 用户的审批记录
		edge.To("approval_logs", ApprovalLog.Type),
	}
}

// Indexes of the User.
func (User) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("email"),
		index.Fields("username"),
		index.Fields("department"),
	}
}
