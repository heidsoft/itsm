package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Release represents a software/hardware release in Release Management
type Release struct {
	ent.Schema
}

func (Release) Fields() []ent.Field {
	return []ent.Field{
		field.String("release_number").
			Comment("发布编号").
			NotEmpty(),
		field.String("title").
			Comment("发布标题").
			NotEmpty(),
		field.Text("description").
			Comment("发布描述").
			Optional(),
		field.String("type").
			Comment("发布类型: major/minor/patch/hotfix").
			Default("minor"),
		field.String("status").
			Comment("状态: draft/scheduled/in-progress/completed/cancelled").
			Default("draft"),
		field.String("severity").
			Comment("严重程度: low/medium/high/critical").
			Default("medium"),
		field.String("environment").
			Comment("目标环境: dev/staging/production").
			Default("staging"),
		field.Int("change_id").
			Comment("关联的变更ID").
			Optional().
			Nillable(),
		field.Int("owner_id").
			Comment("负责人ID").
			Optional().
			Nillable(),
		field.Int("created_by").
			Comment("创建人ID").
			Positive(),
		field.Int("tenant_id").
			Comment("租户ID").
			Positive(),
		field.Time("planned_release_date").
			Comment("计划发布日期").
			Optional(),
		field.Time("actual_release_date").
			Comment("实际发布日期").
			Optional(),
		field.Time("planned_start_date").
			Comment("计划开始时间").
			Optional(),
		field.Time("planned_end_date").
			Comment("计划结束时间").
			Optional(),
		field.Text("release_notes").
			Comment("发布说明").
			Optional(),
		field.Text("rollback_procedure").
			Comment("回滚程序").
			Optional(),
		field.Text("validation_criteria").
			Comment("验证标准").
			Optional(),
		field.JSON("affected_systems", []string{}).
			Comment("受影响的系统").
			Optional(),
		field.JSON("affected_components", []string{}).
			Comment("受影响的组件").
			Optional(),
		field.JSON("deployment_steps", []string{}).
			Comment("部署步骤").
			Optional(),
		field.JSON("tags", []string{}).
			Comment("标签").
			Optional(),
		field.Bool("is_emergency").
			Comment("是否紧急发布").
			Default(false),
		field.Bool("requires_approval").
			Comment("是否需要审批").
			Default(true),
		field.Time("created_at").
			Comment("创建时间").
			Default(time.Now),
		field.Time("updated_at").
			Comment("更新时间").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (Release) Edges() []ent.Edge {
	return nil
}

func (Release) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("release_number"),
		index.Fields("change_id"),
		index.Fields("status"),
		index.Fields("tenant_id"),
	}
}
