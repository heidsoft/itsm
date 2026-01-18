package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// WorkflowVersion holds the schema definition for the WorkflowVersion entity.
type WorkflowVersion struct {
	ent.Schema
}

// Fields of the WorkflowVersion.
func (WorkflowVersion) Fields() []ent.Field {
	return []ent.Field{
		field.Int("workflow_id").
			Comment("工作流ID").
			Positive(),
		field.String("version").
			Comment("版本号，如 1.0.0").
			NotEmpty(),
		field.Text("bpmn_xml").
			Comment("BPMN XML内容").
			Optional(),
		field.JSON("process_variables", []byte{}).
			Comment("流程变量定义").
			Optional(),
		field.String("status").
			Comment("版本状态: draft, active, deprecated").
			Default("draft"),
		field.String("change_log").
			Comment("变更日志").
			Optional(),
		field.String("created_by").
			Comment("创建人").
			Optional(),
		field.Bool("is_current").
			Comment("是否为当前版本").
			Default(false),
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

// Edges of the WorkflowVersion.
func (WorkflowVersion) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("workflow", Workflow.Type).
			Ref("workflow_versions").
			Field("workflow_id").
			Unique().
			Required(),
	}
}
