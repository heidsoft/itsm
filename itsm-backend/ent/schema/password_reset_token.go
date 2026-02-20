package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// PasswordResetToken holds the schema definition for the PasswordResetToken entity.
type PasswordResetToken struct {
	ent.Schema
}

// Fields of the PasswordResetToken.
func (PasswordResetToken) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id").
			Comment("用户ID"),
		field.String("email").
			Comment("邮箱地址"),
		field.String("token").
			Comment("重置令牌").
			NotEmpty(),
		field.Time("expires_at").
			Comment("过期时间"),
		field.Bool("used").
			Comment("是否已使用").
			Default(false),
		field.Time("created_at").
			Comment("创建时间").
			Default(time.Now),
	}
}
