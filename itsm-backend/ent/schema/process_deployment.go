package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ProcessDeployment holds the schema definition for the BPMN Process Deployment entity.
type ProcessDeployment struct {
	ent.Schema
}

// Fields of the ProcessDeployment.
func (ProcessDeployment) Fields() []ent.Field {
	return []ent.Field{
		field.String("deployment_id").
			Comment("部署ID，BPMN标准").
			Unique().
			NotEmpty(),
		field.String("deployment_name").
			Comment("部署名称").
			NotEmpty(),
		field.Text("deployment_source").
			Comment("部署来源").
			Optional(),
		field.Time("deployment_time").
			Comment("部署时间").
			Default(time.Now),
		field.String("deployed_by").
			Comment("部署人").
			Optional(),
		field.String("deployment_comment").
			Comment("部署备注").
			Optional(),
		field.Bool("is_active").
			Comment("是否激活").
			Default(true),
		field.String("deployment_category").
			Comment("部署分类").
			Default("default"),
		field.JSON("deployment_metadata", map[string]interface{}{}).
			Comment("部署元数据").
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

// Edges of the ProcessDeployment.
func (ProcessDeployment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("definitions", ProcessDefinition.Type).
			Comment("流程定义"),
	}
}

// Indexes of the ProcessDeployment.
func (ProcessDeployment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("deployment_id").
			Unique(),
		index.Fields("deployment_name"),
		index.Fields("deployment_time"),
		index.Fields("deployed_by"),
		index.Fields("is_active"),
		index.Fields("deployment_category"),
		index.Fields("tenant_id"),
		index.Fields("created_at"),
	}
}
