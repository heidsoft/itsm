package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// EngineerSkill holds the schema definition for the EngineerSkill entity.
// 工程师技能表，用于智能工单分配
type EngineerSkill struct {
	ent.Schema
}

// Fields of the EngineerSkill.
func (EngineerSkill) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id").
			Comment("用户ID").
			Positive(),
		field.String("category").
			Comment("技能分类: network/security/database/middleware/application").
			NotEmpty(),
		field.String("skill_name").
			Comment("技能名称").
			NotEmpty(),
		field.Int("proficiency_level").
			Comment("熟练程度: 1-5").
			Default(3),
		field.Int("experience_years").
			Comment("经验年限").
			Default(0),
		field.JSON("certifications", []string{}).
			Comment("相关认证").
			Optional(),
		field.Bool("is_available").
			Comment("是否可用(可接单)").
			Default(true),
		field.Int("current_load").
			Comment("当前负载(未完成工单数)").
			Default(0),
		field.Int("max_load").
			Comment("最大负载").
			Default(10),
		field.String("preferred_shift").
			Comment("首选班次: morning/afternoon/night").
			Optional(),
		field.JSON("working_hours", map[string]interface{}{}).
			Comment("工作时间配置").
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

// Edges of the EngineerSkill.
func (EngineerSkill) Edges() []ent.Edge {
	return nil
}
