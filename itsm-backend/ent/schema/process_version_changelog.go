package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ProcessVersionChangelog holds the schema definition for tracking BPMN process version changes.
type ProcessVersionChangelog struct {
	ent.Schema
}

// Fields of the ProcessVersionChangelog.
func (ProcessVersionChangelog) Fields() []ent.Field {
	return []ent.Field{
		field.Int("process_definition_id").
			Comment("流程定义 ID").
			Positive(),
		field.String("version").
			Comment("版本号").
			NotEmpty(),
		field.Text("change_log").
			Comment("变更日志描述").
			NotEmpty(),
		field.JSON("change_details", []map[string]interface{}{}).
			Comment("变更详情列表").
			Optional(),
		field.String("change_type").
			Comment("变更类型：create/update/delete/rollback").
			Default("update"),
		field.Int("created_by").
			Comment("创建人 ID").
			Positive().
			Optional(),
		field.String("created_by_name").
			Comment("创建人姓名").
			Optional(),
		field.Int("tenant_id").
			Comment("租户 ID").
			Positive(),
		field.Time("created_at").
			Comment("创建时间").
			Default(time.Now),
	}
}

// Edges of the ProcessVersionChangelog.
func (ProcessVersionChangelog) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("process_definition", ProcessDefinition.Type).
			Ref("version_changelogs").
			Field("process_definition_id").
			Required().
			Unique(),
		edge.From("user", User.Type).
			Ref("version_changelogs").
			Field("created_by").
			Unique(),
	}
}

// Indexes of the ProcessVersionChangelog.
func (ProcessVersionChangelog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("process_definition_id"),
		index.Fields("tenant_id"),
		index.Fields("process_definition_id", "created_at"),
		index.Fields("tenant_id", "process_definition_id"),
	}
}
