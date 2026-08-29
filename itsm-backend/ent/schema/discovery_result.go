package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// DiscoveryResult holds the schema definition for the DiscoveryResult entity.
type DiscoveryResult struct {
	ent.Schema
}

// Fields of the DiscoveryResult.
func (DiscoveryResult) Fields() []ent.Field {
	return []ent.Field{
		field.Int("job_id").
			Comment("发现任务ID").
			Positive(),
		field.Int("ci_id").
			Comment("关联CI ID").
			Optional(),
		field.String("action").
			Comment("变更动作（create/update/delete）").
			NotEmpty(),
		field.String("resource_type").
			Comment("资源类型").
			Optional(),
		field.String("resource_id").
			Comment("资源ID").
			Optional(),
		field.String("resource_identity").Comment("规范资源身份哈希").Optional(),
		field.Int("identity_version").Comment("身份算法版本").Default(1).Positive(),
		field.JSON("resource_snapshot", map[string]interface{}{}).Comment("标准化资源事实快照").Optional(),
		field.String("before_hash").Comment("应用前内容哈希").Optional(),
		field.String("after_hash").Comment("发现后内容哈希").Optional(),
		field.JSON("diff", map[string]interface{}{}).
			Comment("差异信息").
			Optional(),
		field.String("status").
			Comment("处理状态（pending/applying/applied/rejected/apply_failed）").
			Default("pending"),
		field.String("error_code").Comment("稳定错误码").Optional(),
		field.String("error_message").Comment("净化后的错误摘要").Optional(),
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

// Edges of the DiscoveryResult.
func (DiscoveryResult) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("job", DiscoveryJob.Type).
			Ref("results").
			Unique().
			Field("job_id").
			Required(),
	}
}

// Indexes of the DiscoveryResult.
func (DiscoveryResult) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id"),
		index.Fields("job_id"),
		index.Fields("status"),
		index.Fields("tenant_id", "job_id", "resource_identity"),
	}
}
