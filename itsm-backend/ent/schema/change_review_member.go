package schema

import (
	"fmt"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// ChangeReviewMember holds the schema definition for the ChangeReviewMember entity.
// 变更评审组成员，负责审批常规变更
// 紧急评审组成员，负责审批紧急变更
type ChangeReviewMember struct {
	ent.Schema
}

// Fields of the ChangeReviewMember.
func (ChangeReviewMember) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id").
			Comment("用户ID").
			Positive(),
		field.String("type").
			Comment("评审组类型: REVIEW（常规变更审批）, EREVIEW（紧急变更审批）").
			Validate(func(s string) error {
				if s != "REVIEW" && s != "EREVIEW" {
					return fmt.Errorf("invalid type: must be REVIEW or EREVIEW")
				}
				return nil
			}),
		field.String("role").
			Comment("角色: member, chair, secretary").
			Default("member"),
		field.Bool("is_active").
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

// Edges of the ChangeReviewMember.
func (ChangeReviewMember) Edges() []ent.Edge {
	return nil
}
