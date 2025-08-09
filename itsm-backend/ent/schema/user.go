package schema

import (
	"time"

	"entgo.io/ent"
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
		field.String("department").
			Comment("部门").
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
	return nil
}
