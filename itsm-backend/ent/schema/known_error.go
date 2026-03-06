package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// KnownError holds the schema definition for the KnownError entity.
// 已知错误库，用于问题管理和知识库集成
type KnownError struct {
	ent.Schema
}

// Fields of the KnownError.
func (KnownError) Fields() []ent.Field {
	return []ent.Field{
		field.String("title").
			Comment("错误标题").
			NotEmpty(),
		field.Text("description").
			Comment("错误描述").
			Optional(),
		field.Text("symptoms").
			Comment("症状描述").
			Optional(),
		field.Text("root_cause").
			Comment("根本原因").
			Optional(),
		field.Text("workaround").
			Comment("临时解决方案").
			Optional(),
		field.Text("resolution").
			Comment("永久解决方案").
			Optional(),
		field.String("status").
			Comment("状态: draft/active/resolved/deprecated").
			Default("active"),
		field.String("category").
			Comment("分类").
			Optional(),
		field.String("severity").
			Comment("严重程度: critical/high/medium/low").
			Default("medium"),
		field.JSON("affected_products", []string{}).
			Comment("影响的产品/服务").
			Optional(),
		field.JSON("affected_cis", []string{}).
			Comment("受影响的配置项").
			Optional(),
		field.JSON("keywords", []string{}).
			Comment("关键词(用于搜索)").
			Optional(),
		field.Int("occurrence_count").
			Comment("发生次数").
			Default(0),
		field.Int("created_by").
			Comment("创建人ID").
			Positive(),
		field.Int("approved_by").
			Comment("审批人ID").
			Optional(),
		field.Time("approved_at").
			Comment("审批时间").
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

// Edges of the KnownError.
func (KnownError) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("problem", Problem.Type).
			Comment("关联的问题"),
		edge.To("knowledge_articles", KnowledgeArticle.Type).
			Comment("关联的知识库文章"),
	}
}
