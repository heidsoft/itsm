package schema

import (
	"fmt"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// CABMember holds the schema definition for the CABMember entity.
// CAB (Change Advisory Board) 成员，负责审批普通变更
// ECAB (Emergency Change Advisory Board) 成员，负责审批紧急变更
type CABMember struct {
	ent.Schema
}

// Fields of the CABMember.
func (CABMember) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id").
			Comment("用户ID").
			Positive(),
		field.String("type").
			Comment("委员会类型: CAB（普通变更审批）, ECAB（紧急变更审批）").
			Validate(func(s string) error {
				if s != "CAB" && s != "ECAB" {
					return fmt.Errorf("invalid type: must be CAB or ECAB")
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

// Edges of the CABMember.
func (CABMember) Edges() []ent.Edge {
	return nil
}
