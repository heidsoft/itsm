// 创建知识库文章数据模型
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

type KnowledgeArticle struct {
	ent.Schema
}

func (KnowledgeArticle) Fields() []ent.Field {
	return []ent.Field{
		field.String("title").NotEmpty(),
		field.Text("content").Optional(), // 使用Text类型支持长文本
		field.String("category").NotEmpty(),
		field.String("status").Default("draft"), // draft, published, archived
		field.String("author").NotEmpty(),
		field.Int("views").Default(0),
		field.JSON("tags", []string{}).Optional(),
		field.Int("tenant_id"),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (KnowledgeArticle) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("tenant", Tenant.Type).Ref("knowledge_articles").Field("tenant_id").Unique().Required(),
	}
}

func (KnowledgeArticle) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id"),
		index.Fields("category"),
		index.Fields("status"),
		index.Fields("created_at"),
		index.Fields("tenant_id", "status"),
	}
}
