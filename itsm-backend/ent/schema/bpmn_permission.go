package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// BPMNPermission holds the schema definition for the BPMN Permission entity.
type BPMNPermission struct {
	ent.Schema
}

// Fields of the BPMNPermission.
func (BPMNPermission) Fields() []ent.Field {
	return []ent.Field{
		field.String("resource_type").
			Comment("资源类型: process_definition, process_instance, task").
			NotEmpty(),
		field.Int("resource_id").
			Comment("资源ID (process_definition_id, process_instance_id, or task_id)").
			Positive(),
		field.String("resource_key").
			Comment("资源Key (process_definition_key)").
			Optional(),
		field.String("permission_type").
			Comment("权限类型: read, write, execute, admin, assign, complete, delegate, escalate").
			NotEmpty(),
		field.String("principal_type").
			Comment("授权主体类型: user, role, group, department").
			NotEmpty(),
		field.Int("principal_id").
			Comment("授权主体ID (user_id, role_id, group_id, or department_id)").
			Positive(),
		field.Bool("is_granted").
			Comment("是否授予权限").
			Default(true),
		field.String("conditions").
			Comment("权限条件 (JSON): 如数据过滤条件").
			Optional(),
		field.String("field_permissions").
			Comment("字段级权限 (JSON): 如 {\"field1\": \"read\", \"field2\": \"readwrite\"}").
			Optional(),
		field.String("description").
			Comment("权限描述").
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
		field.Time("expires_at").
			Comment("过期时间").
			Optional(),
	}
}

// Edges of the BPMNPermission.
func (BPMNPermission) Edges() []ent.Edge {
	return []ent.Edge{}
}

// Indexes of the BPMNPermission.
func (BPMNPermission) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("resource_type"),
		index.Fields("resource_id"),
		index.Fields("resource_key"),
		index.Fields("permission_type"),
		index.Fields("principal_type"),
		index.Fields("principal_id"),
		index.Fields("tenant_id"),
		index.Fields("resource_type", "principal_type", "principal_id"),
		index.Fields("resource_key", "principal_type", "principal_id"),
		index.Fields("tenant_id", "principal_id"),
	}
}
