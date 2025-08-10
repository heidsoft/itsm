package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ProcessVariable holds the schema definition for the BPMN Process Variable entity.
type ProcessVariable struct {
	ent.Schema
}

// Fields of the ProcessVariable.
func (ProcessVariable) Fields() []ent.Field {
	return []ent.Field{
		field.String("variable_id").
			Comment("变量ID").
			Unique().
			NotEmpty(),
		field.String("process_instance_id").
			Comment("流程实例ID").
			NotEmpty(),
		field.String("task_id").
			Comment("任务ID，可选").
			Optional(),
		field.String("variable_name").
			Comment("变量名称").
			NotEmpty(),
		field.String("variable_type").
			Comment("变量类型：string, integer, long, double, boolean, date, object").
			Default("string"),
		field.Text("variable_value").
			Comment("变量值").
			Optional(),
		field.String("scope").
			Comment("变量作用域：process, task, global").
			Default("process"),
		field.Bool("is_transient").
			Comment("是否临时变量").
			Default(false),
		field.String("serialization_format").
			Comment("序列化格式：json, xml, binary").
			Default("json"),
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

// Edges of the ProcessVariable.
func (ProcessVariable) Edges() []ent.Edge {
	return []ent.Edge{
		// TODO: 添加相关实体的edge定义
	}
}

// Indexes of the ProcessVariable.
func (ProcessVariable) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("variable_id").
			Unique(),
		index.Fields("process_instance_id"),
		index.Fields("task_id"),
		index.Fields("variable_name"),
		index.Fields("scope"),
		index.Fields("tenant_id"),
		index.Fields("created_at"),
	}
}
