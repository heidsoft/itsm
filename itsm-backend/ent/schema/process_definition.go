package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ProcessDefinition holds the schema definition for the BPMN Process Definition entity.
type ProcessDefinition struct {
	ent.Schema
}

// Fields of the ProcessDefinition.
func (ProcessDefinition) Fields() []ent.Field {
	return []ent.Field{
		field.String("key").
			Comment("流程定义Key，BPMN标准").
			Unique().
			NotEmpty(),
		field.String("name").
			Comment("流程定义名称").
			NotEmpty(),
		field.Text("description").
			Comment("流程描述").
			Optional(),
		field.String("version").
			Comment("版本号").
			Default("1.0.0"),
		field.String("category").
			Comment("流程分类").
			Default("default"),
		field.JSON("bpmn_xml", []byte{}).
			Comment("BPMN XML定义内容"),
		field.JSON("process_variables", map[string]interface{}{}).
			Comment("流程变量定义").
			Optional(),
		field.Bool("is_active").
			Comment("是否激活").
			Default(true),
		field.Bool("is_latest").
			Comment("是否最新版本").
			Default(true),
		field.Int("deployment_id").
			Comment("部署ID").
			Positive(),
		field.String("deployment_name").
			Comment("部署名称").
			Optional(),
		field.Time("deployed_at").
			Comment("部署时间").
			Default(time.Now),
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

// Edges of the ProcessDefinition.
func (ProcessDefinition) Edges() []ent.Edge {
	return []ent.Edge{
		// TODO: 添加相关实体的edge定义
		// edge.To("process_instances", ProcessInstance.Type).
		// 	Comment("流程实例"),
		// edge.To("process_tasks", ProcessTask.Type).
		// 	Comment("流程任务定义"),
		// edge.To("process_events", ProcessEvent.Type).
		// 	Comment("流程事件定义"),
		// edge.To("process_gateways", ProcessGateway.Type).
		// 	Comment("流程网关定义"),
	}
}

// Indexes of the ProcessDefinition.
func (ProcessDefinition) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("key", "version").
			Unique(),
		index.Fields("tenant_id", "key"),
		index.Fields("deployment_id"),
		index.Fields("is_active"),
		index.Fields("is_latest"),
	}
}
