package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

// Fields of the User.
func (User) Fields() []ent.Field {
    return []ent.Field{
        field.String("username").
            Comment("用户名").
            Unique().
            NotEmpty(),
        field.String("email").
            Comment("邮箱").
            Unique().
            NotEmpty(),
        field.String("name").
            Comment("姓名").
            NotEmpty(),
        field.Enum("role").
            Comment("角色").
            Values("super_admin", "admin", "manager", "agent", "technician", "security", "end_user").
            Default("end_user"),
		field.String("department").
			Comment("部门").
			Optional(),
		field.Int("department_id").
			Comment("部门ID").
			Optional(),
		field.String("phone").
            Comment("电话").
            Optional(),
        field.String("password_hash").
            Comment("密码哈希").
            NotEmpty(),
        field.Bool("active").
            Comment("是否激活").
            Default(true),
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

// Edges of the User.
func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("department_ref", Department.Type).
			Ref("users").
			Field("department_id").
			Unique(),
		edge.To("ticket_comments", TicketComment.Type).
			Comment("工单评论"),
		edge.To("ticket_attachments", TicketAttachment.Type).
			Comment("工单附件"),
		edge.To("ticket_notifications", TicketNotification.Type).
			Comment("工单通知"),
		edge.To("notification_preferences", NotificationPreference.Type).
			Comment("通知偏好"),
		edge.To("roles", Role.Type).
			Comment("用户角色"),
	}
}
